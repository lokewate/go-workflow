package state

import (
	"fmt"

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
// It returns the result and an error if the expression is invalid or evaluation fails.
func EvaluateCondition(condition string, ctx GlobalContext) (bool, error) {
	tree, err := parser.Parse(condition)
	if err != nil {
		return false, fmt.Errorf("parse condition %q: %w", condition, err)
	}

	visitor := &envVisitor{}
	ast.Walk(&tree.Node, visitor)

	// Build the minimal environment dynamically
	env := make(map[string]any)

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
		return false, fmt.Errorf("compile condition %q: %w", condition, err)
	}
	result, err := expr.Run(program, env)
	if err != nil {
		return false, fmt.Errorf("evaluate condition %q: %w", condition, err)
	}
	return result.(bool), nil
}
