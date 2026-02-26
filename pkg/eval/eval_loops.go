package eval

import "github.com/dev-dami/carv/pkg/ast"

func evalIfExpression(ie *ast.IfExpression, env *Environment) Object {
	condition := Eval(ie.Condition, env)
	if isError(condition) {
		return condition
	}

	if isTruthy(condition) {
		return Eval(ie.Consequence, env)
	}
	if ie.Alternative != nil {
		return Eval(ie.Alternative, env)
	}
	return NIL
}

func evalForStatement(node *ast.ForStatement, env *Environment) Object {
	loopEnv := NewEnclosedEnvironment(env)

	if node.Init != nil {
		initResult := Eval(node.Init, loopEnv)
		if isError(initResult) {
			return initResult
		}
	}

	for {
		if node.Condition != nil {
			condition := Eval(node.Condition, loopEnv)
			if isError(condition) {
				return condition
			}
			if !isTruthy(condition) {
				break
			}
		}

		result := Eval(node.Body, loopEnv)
		if result != nil {
			if result.Type() == RETURN_VALUE_OBJ || result.Type() == ERROR_OBJ {
				return result
			}
			if result.Type() == BREAK_OBJ {
				break
			}
		}

		if node.Post != nil {
			postResult := Eval(node.Post, loopEnv)
			if isError(postResult) {
				return postResult
			}
		}
	}

	return NIL
}

func evalForInStatement(node *ast.ForInStatement, env *Environment) Object {
	iterable := Eval(node.Iterable, env)
	if isError(iterable) {
		return iterable
	}

	loopEnv := NewEnclosedEnvironment(env)

	switch obj := iterable.(type) {
	case *Array:
		for _, elem := range obj.Elements {
			loopEnv.Set(node.Value.Value, elem)
			result := Eval(node.Body, loopEnv)
			if result != nil {
				if result.Type() == RETURN_VALUE_OBJ || result.Type() == ERROR_OBJ {
					return result
				}
				if result.Type() == BREAK_OBJ {
					break
				}
			}
		}
	case *String:
		for _, ch := range obj.Value {
			loopEnv.Set(node.Value.Value, &Char{Value: ch})
			result := Eval(node.Body, loopEnv)
			if result != nil {
				if result.Type() == RETURN_VALUE_OBJ || result.Type() == ERROR_OBJ {
					return result
				}
				if result.Type() == BREAK_OBJ {
					break
				}
			}
		}
	case *Map:
		for _, pair := range obj.Pairs {
			loopEnv.Set(node.Value.Value, pair.Key)
			result := Eval(node.Body, loopEnv)
			if result != nil {
				if result.Type() == RETURN_VALUE_OBJ || result.Type() == ERROR_OBJ {
					return result
				}
				if result.Type() == BREAK_OBJ {
					break
				}
			}
		}
	default:
		return &Error{Message: "for-in requires array, string, or map, got " + string(iterable.Type())}
	}

	return NIL
}

func evalWhileStatement(node *ast.WhileStatement, env *Environment) Object {
	for {
		condition := Eval(node.Condition, env)
		if isError(condition) {
			return condition
		}
		if !isTruthy(condition) {
			break
		}

		result := Eval(node.Body, env)
		if result != nil {
			if result.Type() == RETURN_VALUE_OBJ || result.Type() == ERROR_OBJ {
				return result
			}
			if result.Type() == BREAK_OBJ {
				break
			}
		}
	}

	return NIL
}

func evalLoopStatement(node *ast.LoopStatement, env *Environment) Object {
	for {
		result := Eval(node.Body, env)
		if result != nil {
			if result.Type() == RETURN_VALUE_OBJ || result.Type() == ERROR_OBJ {
				return result
			}
			if result.Type() == BREAK_OBJ {
				break
			}
		}
	}

	return NIL
}
