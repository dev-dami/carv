package eval

import "github.com/dev-dami/carv/pkg/ast"

func evalIndexExpression(left, index Object) Object {
	switch {
	case left.Type() == ARRAY_OBJ && index.Type() == INTEGER_OBJ:
		return evalArrayIndexExpression(left, index)
	case left.Type() == STRING_OBJ && index.Type() == INTEGER_OBJ:
		return evalStringIndexExpression(left, index)
	case left.Type() == MAP_OBJ:
		return evalMapIndexExpression(left, index)
	default:
		return &Error{Message: "index operator not supported: " + string(left.Type())}
	}
}

func evalMapIndexExpression(mapObj, index Object) Object {
	m := mapObj.(*Map)

	key, ok := index.(Hashable)
	if !ok {
		return &Error{Message: "unusable as hash key: " + string(index.Type())}
	}

	pair, ok := m.Pairs[key.HashKey()]
	if !ok {
		return NIL
	}

	return pair.Value
}

func evalArrayIndexExpression(array, index Object) Object {
	arrayObject := array.(*Array)
	idx := index.(*Integer).Value
	maxIdx := int64(len(arrayObject.Elements) - 1)

	if idx < 0 || idx > maxIdx {
		return NIL
	}

	return arrayObject.Elements[idx]
}

func evalStringIndexExpression(str, index Object) Object {
	stringObject := str.(*String)
	idx := index.(*Integer).Value
	maxIdx := int64(len(stringObject.Value) - 1)

	if idx < 0 || idx > maxIdx {
		return NIL
	}

	return &Char{Value: rune(stringObject.Value[idx])}
}

func evalMemberExpression(node *ast.MemberExpression, env *Environment) Object {
	obj := Eval(node.Object, env)
	if isError(obj) {
		return obj
	}

	if mod, ok := obj.(*Module); ok {
		if val, exists := mod.Exports[node.Member.Value]; exists {
			return val
		}
		return &Error{Message: "undefined member: " + node.Member.Value}
	}

	instance, ok := obj.(*Instance)
	if !ok {
		return &Error{Message: "member access requires an instance, got " + string(obj.Type())}
	}

	memberName := node.Member.Value

	if val, exists := instance.Fields[memberName]; exists {
		return val
	}

	if method, exists := instance.Class.Methods[memberName]; exists {
		return &Method{Instance: instance, Fn: method}
	}

	return &Error{Message: "undefined member: " + memberName}
}
