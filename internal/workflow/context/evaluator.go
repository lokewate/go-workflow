package context

import (
	"log"

	"github.com/antonmedv/expr"
	"github.com/antonmedv/expr/ast"
	"github.com/antonmedv/expr/parser"
)

type envVisitor struct {
	keys map[string]bool
}

func (v *envVisitor) Visit(node *ast.Node) {
	if n, ok := (*node).(*ast.IdentifierNode); ok {
		if v.keys == nil {
			v.keys = make(map[string]bool)
		}
		v.keys[n.Value] = true
	}
}

// EvaluateCondition parses and executes a boolean expression against the global context.
// It uses the expr library for evaluation and returns false if the expression is invalid or fails.
func EvaluateCondition(condition string, ctx GlobalContext) bool {
	tree, err := parser.Parse(condition)
	if err != nil {
		return false
	}

	visitor := &envVisitor{}
	ast.Walk(&tree.Node, visitor)

	// Build the minimal environment dynamically
	env := make(map[string]interface{})

	for key := range visitor.keys {
		env[key] = ctx.Get(key)
	}

	var options []expr.Option
	options = append(options, expr.Env(env), expr.AsBool())
	for key := range visitor.keys {
		options = append(options, expr.DisableBuiltin(key))
	}

	program, err := expr.Compile(condition, options...)
	if err != nil {
		log.Printf("[Evaluator]: Compile error: %v", err)
		return false
	}
	result, err := expr.Run(program, env)
	if err != nil {
		log.Printf("[Evaluator]: Run error: %v", err)
		return false
	}
	return result.(bool)
}
