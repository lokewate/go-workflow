package workflow

import "github.com/antonmedv/expr"

func EvaluateCondition(condition string, payload map[string]interface{}) bool {
	env := map[string]interface{}{"payload": payload}
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
