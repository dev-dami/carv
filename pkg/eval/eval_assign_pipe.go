package eval

import "github.com/dev-dami/carv/pkg/ast"

func evalPipeExpression(node *ast.PipeExpression, env *Environment) Object {
	left := Eval(node.Left, env)
	if isError(left) {
		return left
	}

	switch right := node.Right.(type) {
	case *ast.Identifier:
		if fn, ok := env.Get(right.Value); ok {
			return applyFunction(fn, []Object{left})
		}
		if builtin, ok := builtins[right.Value]; ok {
			return builtin.Fn(left)
		}
		return &Error{Message: "unknown function: " + right.Value}

	case *ast.CallExpression:
		function := Eval(right.Function, env)
		if isError(function) {
			return function
		}
		args := []Object{left}
		evalledArgs := evalExpressions(right.Arguments, env)
		if len(evalledArgs) == 1 && isError(evalledArgs[0]) {
			return evalledArgs[0]
		}
		args = append(args, evalledArgs...)
		return applyFunction(function, args)

	default:
		rightVal := Eval(node.Right, env)
		if isError(rightVal) {
			return rightVal
		}
		if fn, ok := rightVal.(*Function); ok {
			return applyFunction(fn, []Object{left})
		}
		if builtin, ok := rightVal.(*Builtin); ok {
			return builtin.Fn(left)
		}
		return &Error{Message: "pipe target must be a function"}
	}
}

func evalAssignExpression(node *ast.AssignExpression, env *Environment) Object {
	val := Eval(node.Right, env)
	if isError(val) {
		return val
	}

	if memberExpr, ok := node.Left.(*ast.MemberExpression); ok {
		return evalMemberAssignment(memberExpr, node.Operator, val, env)
	}

	if indexExpr, ok := node.Left.(*ast.IndexExpression); ok {
		return evalIndexAssignment(indexExpr, node.Operator, val, env)
	}

	ident, ok := node.Left.(*ast.Identifier)
	if !ok {
		return &Error{Message: "cannot assign to non-identifier"}
	}

	existing, exists := env.Get(ident.Value)
	if !exists {
		return &Error{Message: "undefined variable: " + ident.Value}
	}

	if env.IsImmutable(ident.Value) {
		return &Error{Message: "cannot assign to immutable variable: " + ident.Value, Line: node.Token.Line, Column: node.Token.Column}
	}

	var newVal Object
	switch node.Operator {
	case "=":
		newVal = val
	case "+=":
		newVal = evalInfixExpression("+", existing, val)
	case "-=":
		newVal = evalInfixExpression("-", existing, val)
	case "*=":
		newVal = evalInfixExpression("*", existing, val)
	case "/=":
		newVal = evalInfixExpression("/", existing, val)
	case "%=":
		newVal = evalInfixExpression("%", existing, val)
	case "&=":
		newVal = evalInfixExpression("&", existing, val)
	case "|=":
		newVal = evalInfixExpression("|", existing, val)
	case "^=":
		newVal = evalInfixExpression("^", existing, val)
	default:
		return &Error{Message: "unknown assignment operator: " + node.Operator}
	}

	if isError(newVal) {
		return newVal
	}

	env.Update(ident.Value, newVal)
	return newVal
}

func evalMemberAssignment(node *ast.MemberExpression, operator string, val Object, env *Environment) Object {
	obj := Eval(node.Object, env)
	if isError(obj) {
		return obj
	}

	instance, ok := obj.(*Instance)
	if !ok {
		return &Error{Message: "cannot assign to member of non-instance"}
	}

	memberName := node.Member.Value

	existing, exists := instance.Fields[memberName]
	if !exists {
		return &Error{Message: "undefined field: " + memberName}
	}

	var newVal Object
	switch operator {
	case "=":
		newVal = val
	case "+=":
		newVal = evalInfixExpression("+", existing, val)
	case "-=":
		newVal = evalInfixExpression("-", existing, val)
	case "*=":
		newVal = evalInfixExpression("*", existing, val)
	case "/=":
		newVal = evalInfixExpression("/", existing, val)
	case "%=":
		newVal = evalInfixExpression("%", existing, val)
	case "&=":
		newVal = evalInfixExpression("&", existing, val)
	case "|=":
		newVal = evalInfixExpression("|", existing, val)
	case "^=":
		newVal = evalInfixExpression("^", existing, val)
	default:
		return &Error{Message: "unknown assignment operator: " + operator}
	}

	if isError(newVal) {
		return newVal
	}

	instance.Fields[memberName] = newVal
	return newVal
}

func evalIndexAssignment(node *ast.IndexExpression, operator string, val Object, env *Environment) Object {
	left := Eval(node.Left, env)
	if isError(left) {
		return left
	}

	index := Eval(node.Index, env)
	if isError(index) {
		return index
	}

	switch obj := left.(type) {
	case *Array:
		idx, ok := index.(*Integer)
		if !ok {
			return &Error{Message: "array index must be an integer"}
		}
		if idx.Value < 0 || idx.Value >= int64(len(obj.Elements)) {
			return &Error{Message: "array index out of bounds"}
		}

		var newVal Object
		switch operator {
		case "=":
			newVal = val
		case "+=":
			newVal = evalInfixExpression("+", obj.Elements[idx.Value], val)
		case "-=":
			newVal = evalInfixExpression("-", obj.Elements[idx.Value], val)
		case "*=":
			newVal = evalInfixExpression("*", obj.Elements[idx.Value], val)
		case "/=":
			newVal = evalInfixExpression("/", obj.Elements[idx.Value], val)
		case "%=":
			newVal = evalInfixExpression("%", obj.Elements[idx.Value], val)
		case "&=":
			newVal = evalInfixExpression("&", obj.Elements[idx.Value], val)
		case "|=":
			newVal = evalInfixExpression("|", obj.Elements[idx.Value], val)
		case "^=":
			newVal = evalInfixExpression("^", obj.Elements[idx.Value], val)
		default:
			return &Error{Message: "unknown assignment operator: " + operator}
		}

		if isError(newVal) {
			return newVal
		}

		obj.Elements[idx.Value] = newVal
		return newVal

	case *Map:
		hashKey, ok := index.(Hashable)
		if !ok {
			return &Error{Message: "unusable as hash key: " + string(index.Type())}
		}

		var newVal Object
		existing, exists := obj.Pairs[hashKey.HashKey()]
		if operator == "=" {
			newVal = val
		} else {
			if !exists {
				return &Error{Message: "key does not exist in map"}
			}
			switch operator {
			case "+=":
				newVal = evalInfixExpression("+", existing.Value, val)
			case "-=":
				newVal = evalInfixExpression("-", existing.Value, val)
			case "*=":
				newVal = evalInfixExpression("*", existing.Value, val)
			case "/=":
				newVal = evalInfixExpression("/", existing.Value, val)
			case "%=":
				newVal = evalInfixExpression("%", existing.Value, val)
			case "&=":
				newVal = evalInfixExpression("&", existing.Value, val)
			case "|=":
				newVal = evalInfixExpression("|", existing.Value, val)
			case "^=":
				newVal = evalInfixExpression("^", existing.Value, val)
			default:
				return &Error{Message: "unknown assignment operator: " + operator}
			}
		}

		if isError(newVal) {
			return newVal
		}

		obj.Pairs[hashKey.HashKey()] = MapPair{Key: index, Value: newVal}
		return newVal

	default:
		return &Error{Message: "index assignment not supported for " + string(left.Type())}
	}
}
