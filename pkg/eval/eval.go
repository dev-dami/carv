package eval

import "github.com/dev-dami/carv/pkg/ast"

func Eval(node ast.Node, env *Environment) Object {
	switch node := node.(type) {
	case *ast.Program:
		return evalProgram(node, env)

	case *ast.ExpressionStatement:
		return Eval(node.Expression, env)

	case *ast.LetStatement:
		val := Eval(node.Value, env)
		if isError(val) {
			return val
		}
		if node.Mutable {
			env.Set(node.Name.Value, val)
		} else {
			env.SetImmutable(node.Name.Value, val)
		}
		return val

	case *ast.ConstStatement:
		val := Eval(node.Value, env)
		if isError(val) {
			return val
		}
		env.SetImmutable(node.Name.Value, val)
		return val

	case *ast.ReturnStatement:
		if node.ReturnValue == nil {
			return &ReturnValue{Value: NIL}
		}
		val := Eval(node.ReturnValue, env)
		if isError(val) {
			return val
		}
		return &ReturnValue{Value: val}

	case *ast.BlockStatement:
		return evalBlockStatement(node, env)

	case *ast.FunctionStatement:
		fn := &Function{
			Parameters: node.Parameters,
			Body:       node.Body,
			Env:        env,
			Name:       node.Name.Value,
		}
		env.Set(node.Name.Value, fn)
		return fn

	case *ast.ForStatement:
		return evalForStatement(node, env)

	case *ast.ForInStatement:
		return evalForInStatement(node, env)

	case *ast.WhileStatement:
		return evalWhileStatement(node, env)

	case *ast.LoopStatement:
		return evalLoopStatement(node, env)

	case *ast.BreakStatement:
		return BREAK

	case *ast.ContinueStatement:
		return CONT

	case *ast.IntegerLiteral:
		return evalIntegerLiteral(node)

	case *ast.FloatLiteral:
		return evalFloatLiteral(node)

	case *ast.StringLiteral:
		return evalStringLiteral(node)

	case *ast.CharLiteral:
		return evalCharLiteral(node)

	case *ast.BoolLiteral:
		return evalBoolLiteral(node)

	case *ast.NilLiteral:
		return evalNilLiteral(node)

	case *ast.Identifier:
		return evalIdentifier(node, env)

	case *ast.PrefixExpression:
		right := Eval(node.Right, env)
		if isError(right) {
			return right
		}
		return evalPrefixExpression(node.Operator, right)

	case *ast.InfixExpression:
		left := Eval(node.Left, env)
		if isError(left) {
			return left
		}
		right := Eval(node.Right, env)
		if isError(right) {
			return right
		}
		return evalInfixExpression(node.Operator, left, right)

	case *ast.PipeExpression:
		return evalPipeExpression(node, env)

	case *ast.AssignExpression:
		return evalAssignExpression(node, env)

	case *ast.IfExpression:
		return evalIfExpression(node, env)

	case *ast.FunctionLiteral:
		return &Function{
			Parameters: node.Parameters,
			Body:       node.Body,
			Env:        env,
		}

	case *ast.CallExpression:
		function := Eval(node.Function, env)
		if isError(function) {
			return function
		}
		args := evalExpressions(node.Arguments, env)
		if len(args) == 1 && isError(args[0]) {
			return args[0]
		}
		return applyFunction(function, args)

	case *ast.ArrayLiteral:
		elements := evalExpressions(node.Elements, env)
		if len(elements) == 1 && isError(elements[0]) {
			return elements[0]
		}
		return &Array{Elements: elements}

	case *ast.MapLiteral:
		return evalMapLiteral(node, env)

	case *ast.IndexExpression:
		left := Eval(node.Left, env)
		if isError(left) {
			return left
		}
		index := Eval(node.Index, env)
		if isError(index) {
			return index
		}
		return evalIndexExpression(left, index)

	case *ast.MemberExpression:
		return evalMemberExpression(node, env)

	case *ast.OkExpression:
		return evalOkExpression(node, env)

	case *ast.ErrExpression:
		return evalErrExpression(node, env)

	case *ast.MatchExpression:
		return evalMatchExpression(node, env)

	case *ast.TryExpression:
		return evalTryExpression(node, env)

	case *ast.ClassStatement:
		return evalClassStatement(node, env)

	case *ast.NewExpression:
		return evalNewExpression(node, env)

	case *ast.BlockExpression:
		return evalBlockExpression(node, env)

	case *ast.RequireStatement:
		return evalRequireStatement(node, env)

	case *ast.InterpolatedString:
		return evalInterpolatedString(node, env)
	}

	return nil
}
func evalProgram(program *ast.Program, env *Environment) Object {
	var result Object

	for _, statement := range program.Statements {
		result = Eval(statement, env)

		switch result := result.(type) {
		case *ReturnValue:
			return result.Value
		case *Error:
			return result
		}
	}

	return result
}

func evalBlockStatement(block *ast.BlockStatement, env *Environment) Object {
	var result Object

	for _, statement := range block.Statements {
		result = Eval(statement, env)

		if result != nil {
			rt := result.Type()
			if rt == RETURN_VALUE_OBJ || rt == ERROR_OBJ || rt == BREAK_OBJ || rt == CONTINUE_OBJ {
				return result
			}
		}
	}

	return result
}
func nativeBoolToBooleanObject(input bool) *Boolean {
	if input {
		return TRUE
	}
	return FALSE
}

func isTruthy(obj Object) bool {
	switch obj {
	case NIL:
		return false
	case TRUE:
		return true
	case FALSE:
		return false
	default:
		return true
	}
}

func isError(obj Object) bool {
	if obj != nil {
		return obj.Type() == ERROR_OBJ
	}
	return false
}
