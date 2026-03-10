package workflow

import (
	"workflow-engine/internal/workflow/context"

	"github.com/antonmedv/expr"
)

// EvaluateCondition parses and executes a boolean expression against the global context.
// It uses the expr library for evaluation and returns false if the expression is invalid or fails.
func EvaluateCondition(condition string, ctx context.GlobalContext) bool {
	env := map[string]interface{}{"payload": ctx.AsMap()}
	program, err := expr.Compile(condition, expr.AsBool())
	if err != nil {
		return false
	}
	result, err := expr.Run(program, env)
	if err != nil {
		return false
	}
	return result.(bool)
}
