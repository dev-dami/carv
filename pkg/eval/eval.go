package eval

import (
	"strings"

	"github.com/dev-dami/carv/pkg/ast"
	"github.com/dev-dami/carv/pkg/module"
)

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
		env.Set(node.Name.Value, val)
		return val

	case *ast.ConstStatement:
		val := Eval(node.Value, env)
		if isError(val) {
			return val
		}
		env.Set(node.Name.Value, val)
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
		return &Integer{Value: node.Value}

	case *ast.FloatLiteral:
		return &Float{Value: node.Value}

	case *ast.StringLiteral:
		return &String{Value: node.Value}

	case *ast.CharLiteral:
		return &Char{Value: node.Value}

	case *ast.BoolLiteral:
		return nativeBoolToBooleanObject(node.Value)

	case *ast.NilLiteral:
		return NIL

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

func evalIdentifier(node *ast.Identifier, env *Environment) Object {
	if val, ok := env.Get(node.Value); ok {
		return val
	}

	if builtin, ok := builtins[node.Value]; ok {
		return builtin
	}

	return &Error{Message: "identifier not found: " + node.Value, Line: node.Token.Line, Column: node.Token.Column}
}

func evalPrefixExpression(operator string, right Object) Object {
	switch operator {
	case "!":
		return evalBangOperatorExpression(right)
	case "-":
		return evalMinusPrefixOperatorExpression(right)
	case "~":
		return evalBitwiseNotExpression(right)
	default:
		return &Error{Message: "unknown operator: " + operator + right.Type().string()}
	}
}

func (ot ObjectType) string() string {
	return string(ot)
}

func evalBangOperatorExpression(right Object) Object {
	switch right {
	case TRUE:
		return FALSE
	case FALSE:
		return TRUE
	case NIL:
		return TRUE
	default:
		return FALSE
	}
}

func evalMinusPrefixOperatorExpression(right Object) Object {
	switch obj := right.(type) {
	case *Integer:
		return &Integer{Value: -obj.Value}
	case *Float:
		return &Float{Value: -obj.Value}
	default:
		return &Error{Message: "unknown operator: -" + string(right.Type())}
	}
}

func evalBitwiseNotExpression(right Object) Object {
	if obj, ok := right.(*Integer); ok {
		return &Integer{Value: ^obj.Value}
	}
	return &Error{Message: "bitwise not requires integer"}
}

func evalInfixExpression(operator string, left, right Object) Object {
	switch {
	case left.Type() == INTEGER_OBJ && right.Type() == INTEGER_OBJ:
		return evalIntegerInfixExpression(operator, left, right)
	case left.Type() == FLOAT_OBJ || right.Type() == FLOAT_OBJ:
		return evalFloatInfixExpression(operator, left, right)
	case left.Type() == STRING_OBJ && right.Type() == STRING_OBJ:
		return evalStringInfixExpression(operator, left, right)
	case left.Type() == CHAR_OBJ && right.Type() == CHAR_OBJ:
		return evalCharInfixExpression(operator, left, right)
	case operator == "==":
		return nativeBoolToBooleanObject(left == right)
	case operator == "!=":
		return nativeBoolToBooleanObject(left != right)
	case operator == "&&":
		return nativeBoolToBooleanObject(isTruthy(left) && isTruthy(right))
	case operator == "||":
		return nativeBoolToBooleanObject(isTruthy(left) || isTruthy(right))
	default:
		return &Error{Message: "unknown operator: " + string(left.Type()) + " " + operator + " " + string(right.Type())}
	}
}

func evalIntegerInfixExpression(operator string, left, right Object) Object {
	leftVal := left.(*Integer).Value
	rightVal := right.(*Integer).Value

	switch operator {
	case "+":
		return &Integer{Value: leftVal + rightVal}
	case "-":
		return &Integer{Value: leftVal - rightVal}
	case "*":
		return &Integer{Value: leftVal * rightVal}
	case "/":
		if rightVal == 0 {
			return &Error{Message: "division by zero"}
		}
		return &Integer{Value: leftVal / rightVal}
	case "%":
		if rightVal == 0 {
			return &Error{Message: "modulo by zero"}
		}
		return &Integer{Value: leftVal % rightVal}
	case "<":
		return nativeBoolToBooleanObject(leftVal < rightVal)
	case ">":
		return nativeBoolToBooleanObject(leftVal > rightVal)
	case "<=":
		return nativeBoolToBooleanObject(leftVal <= rightVal)
	case ">=":
		return nativeBoolToBooleanObject(leftVal >= rightVal)
	case "==":
		return nativeBoolToBooleanObject(leftVal == rightVal)
	case "!=":
		return nativeBoolToBooleanObject(leftVal != rightVal)
	case "&":
		return &Integer{Value: leftVal & rightVal}
	case "|":
		return &Integer{Value: leftVal | rightVal}
	case "^":
		return &Integer{Value: leftVal ^ rightVal}
	default:
		return &Error{Message: "unknown operator: INTEGER " + operator + " INTEGER"}
	}
}

func evalFloatInfixExpression(operator string, left, right Object) Object {
	var leftVal, rightVal float64

	switch l := left.(type) {
	case *Float:
		leftVal = l.Value
	case *Integer:
		leftVal = float64(l.Value)
	}

	switch r := right.(type) {
	case *Float:
		rightVal = r.Value
	case *Integer:
		rightVal = float64(r.Value)
	}

	switch operator {
	case "+":
		return &Float{Value: leftVal + rightVal}
	case "-":
		return &Float{Value: leftVal - rightVal}
	case "*":
		return &Float{Value: leftVal * rightVal}
	case "/":
		if rightVal == 0 {
			return &Error{Message: "division by zero"}
		}
		return &Float{Value: leftVal / rightVal}
	case "<":
		return nativeBoolToBooleanObject(leftVal < rightVal)
	case ">":
		return nativeBoolToBooleanObject(leftVal > rightVal)
	case "<=":
		return nativeBoolToBooleanObject(leftVal <= rightVal)
	case ">=":
		return nativeBoolToBooleanObject(leftVal >= rightVal)
	case "==":
		return nativeBoolToBooleanObject(leftVal == rightVal)
	case "!=":
		return nativeBoolToBooleanObject(leftVal != rightVal)
	default:
		return &Error{Message: "unknown operator: FLOAT " + operator + " FLOAT"}
	}
}

func evalStringInfixExpression(operator string, left, right Object) Object {
	leftVal := left.(*String).Value
	rightVal := right.(*String).Value

	switch operator {
	case "+":
		return &String{Value: leftVal + rightVal}
	case "==":
		return nativeBoolToBooleanObject(leftVal == rightVal)
	case "!=":
		return nativeBoolToBooleanObject(leftVal != rightVal)
	case "<":
		return nativeBoolToBooleanObject(leftVal < rightVal)
	case ">":
		return nativeBoolToBooleanObject(leftVal > rightVal)
	default:
		return &Error{Message: "unknown operator: STRING " + operator + " STRING"}
	}
}

func evalCharInfixExpression(operator string, left, right Object) Object {
	leftVal := left.(*Char).Value
	rightVal := right.(*Char).Value

	switch operator {
	case "==":
		return nativeBoolToBooleanObject(leftVal == rightVal)
	case "!=":
		return nativeBoolToBooleanObject(leftVal != rightVal)
	case "<":
		return nativeBoolToBooleanObject(leftVal < rightVal)
	case ">":
		return nativeBoolToBooleanObject(leftVal > rightVal)
	case "<=":
		return nativeBoolToBooleanObject(leftVal <= rightVal)
	case ">=":
		return nativeBoolToBooleanObject(leftVal >= rightVal)
	default:
		return &Error{Message: "unknown operator: CHAR " + operator + " CHAR"}
	}
}

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

func evalIfExpression(ie *ast.IfExpression, env *Environment) Object {
	condition := Eval(ie.Condition, env)
	if isError(condition) {
		return condition
	}

	if isTruthy(condition) {
		return Eval(ie.Consequence, env)
	} else if ie.Alternative != nil {
		return Eval(ie.Alternative, env)
	} else {
		return NIL
	}
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

	arr, ok := iterable.(*Array)
	if !ok {
		return &Error{Message: "for-in requires array"}
	}

	loopEnv := NewEnclosedEnvironment(env)

	for _, elem := range arr.Elements {
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
	max := int64(len(arrayObject.Elements) - 1)

	if idx < 0 || idx > max {
		return NIL
	}

	return arrayObject.Elements[idx]
}

func evalStringIndexExpression(str, index Object) Object {
	stringObject := str.(*String)
	idx := index.(*Integer).Value
	max := int64(len(stringObject.Value) - 1)

	if idx < 0 || idx > max {
		return NIL
	}

	return &Char{Value: rune(stringObject.Value[idx])}
}

func evalMemberExpression(node *ast.MemberExpression, env *Environment) Object {
	obj := Eval(node.Object, env)
	if isError(obj) {
		return obj
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

func evalExpressions(exps []ast.Expression, env *Environment) []Object {
	var result []Object

	for _, e := range exps {
		evaluated := Eval(e, env)
		if isError(evaluated) {
			return []Object{evaluated}
		}
		result = append(result, evaluated)
	}

	return result
}

func applyFunction(fn Object, args []Object) Object {
	switch fn := fn.(type) {
	case *Function:
		extendedEnv := extendFunctionEnv(fn, args)
		evaluated := Eval(fn.Body, extendedEnv)
		return unwrapReturnValue(evaluated)

	case *Method:
		extendedEnv := extendFunctionEnv(fn.Fn, args)
		extendedEnv.Set("self", fn.Instance)
		evaluated := Eval(fn.Fn.Body, extendedEnv)
		return unwrapReturnValue(evaluated)

	case *Builtin:
		return fn.Fn(args...)

	default:
		return &Error{Message: "not a function: " + string(fn.Type())}
	}
}

func extendFunctionEnv(fn *Function, args []Object) *Environment {
	env := NewEnclosedEnvironment(fn.Env)

	for paramIdx, param := range fn.Parameters {
		if paramIdx < len(args) {
			env.Set(param.Name.Value, args[paramIdx])
		}
	}

	return env
}

func unwrapReturnValue(obj Object) Object {
	if returnValue, ok := obj.(*ReturnValue); ok {
		return returnValue.Value
	}
	return obj
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

func evalOkExpression(node *ast.OkExpression, env *Environment) Object {
	val := Eval(node.Value, env)
	if isError(val) {
		return val
	}
	return &Ok{Value: val}
}

func evalErrExpression(node *ast.ErrExpression, env *Environment) Object {
	val := Eval(node.Value, env)
	if isError(val) {
		return val
	}
	return &Err{Value: val}
}

func evalMatchExpression(node *ast.MatchExpression, env *Environment) Object {
	val := Eval(node.Value, env)
	if isError(val) {
		return val
	}

	for _, arm := range node.Arms {
		if matchPattern(arm.Pattern, val, env) {
			matchEnv := NewEnclosedEnvironment(env)
			bindPattern(arm.Pattern, val, matchEnv)
			return Eval(arm.Body, matchEnv)
		}
	}

	return NIL
}

func matchPattern(pattern ast.Expression, val Object, env *Environment) bool {
	switch p := pattern.(type) {
	case *ast.OkExpression:
		_, isOk := val.(*Ok)
		return isOk
	case *ast.ErrExpression:
		_, isErr := val.(*Err)
		return isErr
	case *ast.CallExpression:
		if ident, ok := p.Function.(*ast.Identifier); ok {
			switch ident.Value {
			case "Ok":
				_, isOk := val.(*Ok)
				return isOk
			case "Err":
				_, isErr := val.(*Err)
				return isErr
			}
		}
	case *ast.Identifier:
		if p.Value == "_" {
			return true
		}
		return true
	}
	return false
}

func bindPattern(pattern ast.Expression, val Object, env *Environment) {
	switch p := pattern.(type) {
	case *ast.OkExpression:
		if okVal, isOk := val.(*Ok); isOk {
			if ident, ok := p.Value.(*ast.Identifier); ok {
				env.Set(ident.Value, okVal.Value)
			}
		}
	case *ast.ErrExpression:
		if errVal, isErr := val.(*Err); isErr {
			if ident, ok := p.Value.(*ast.Identifier); ok {
				env.Set(ident.Value, errVal.Value)
			}
		}
	case *ast.CallExpression:
		if ident, ok := p.Function.(*ast.Identifier); ok {
			switch ident.Value {
			case "Ok":
				if okVal, isOk := val.(*Ok); isOk && len(p.Arguments) > 0 {
					if argIdent, ok := p.Arguments[0].(*ast.Identifier); ok {
						env.Set(argIdent.Value, okVal.Value)
					}
				}
			case "Err":
				if errVal, isErr := val.(*Err); isErr && len(p.Arguments) > 0 {
					if argIdent, ok := p.Arguments[0].(*ast.Identifier); ok {
						env.Set(argIdent.Value, errVal.Value)
					}
				}
			}
		}
	case *ast.Identifier:
		if p.Value != "_" {
			env.Set(p.Value, val)
		}
	}
}

func evalTryExpression(node *ast.TryExpression, env *Environment) Object {
	val := Eval(node.Value, env)
	if isError(val) {
		return val
	}

	switch v := val.(type) {
	case *Ok:
		return v.Value
	case *Err:
		return &ReturnValue{Value: v}
	default:
		return val
	}
}

func evalClassStatement(node *ast.ClassStatement, env *Environment) Object {
	class := &Class{
		Name:    node.Name.Value,
		Fields:  make(map[string]Object),
		Methods: make(map[string]*Function),
	}

	for _, field := range node.Fields {
		if field.Default != nil {
			defaultVal := Eval(field.Default, env)
			if isError(defaultVal) {
				return defaultVal
			}
			class.Fields[field.Name.Value] = defaultVal
		} else {
			class.Fields[field.Name.Value] = NIL
		}
	}

	for _, method := range node.Methods {
		fn := &Function{
			Parameters: method.Parameters,
			Body:       method.Body,
			Env:        env,
			Name:       method.Name.Value,
		}
		class.Methods[method.Name.Value] = fn
	}

	env.Set(node.Name.Value, class)
	return class
}

func evalNewExpression(node *ast.NewExpression, env *Environment) Object {
	namedType, ok := node.Type.(*ast.NamedType)
	if !ok {
		return &Error{Message: "new requires a class type"}
	}

	classObj, exists := env.Get(namedType.Name.Value)
	if !exists {
		return &Error{Message: "undefined class: " + namedType.Name.Value}
	}

	class, ok := classObj.(*Class)
	if !ok {
		return &Error{Message: namedType.Name.Value + " is not a class"}
	}

	instance := &Instance{
		Class:  class,
		Fields: make(map[string]Object),
	}

	for name, defaultVal := range class.Fields {
		instance.Fields[name] = defaultVal
	}

	return instance
}

func evalBlockExpression(node *ast.BlockExpression, env *Environment) Object {
	return evalBlockStatement(node.Block, env)
}

func evalMapLiteral(node *ast.MapLiteral, env *Environment) Object {
	pairs := make(map[HashKey]MapPair)

	for keyNode, valueNode := range node.Pairs {
		key := Eval(keyNode, env)
		if isError(key) {
			return key
		}

		hashKey, ok := key.(Hashable)
		if !ok {
			return &Error{Message: "unusable as hash key: " + string(key.Type())}
		}

		value := Eval(valueNode, env)
		if isError(value) {
			return value
		}

		hashed := hashKey.HashKey()
		pairs[hashed] = MapPair{Key: key, Value: value}
	}

	return &Map{Pairs: pairs}
}

var moduleLoader *module.Loader
var currentFile string

func SetModuleLoader(loader *module.Loader) {
	moduleLoader = loader
}

func SetCurrentFile(file string) {
	currentFile = file
}

func evalRequireStatement(node *ast.RequireStatement, env *Environment) Object {
	if moduleLoader == nil {
		return &Error{Message: "module loader not initialized", Line: node.Token.Line, Column: node.Token.Column}
	}

	mod, err := moduleLoader.Load(node.Path.Value, currentFile)
	if err != nil {
		return &Error{Message: "failed to load module: " + err.Error(), Line: node.Token.Line, Column: node.Token.Column}
	}

	modEnv := NewEnvironment()
	result := Eval(mod.Program, modEnv)
	if isError(result) {
		return result
	}

	if node.Alias != nil {
		modObj := &Module{
			Name:    node.Path.Value,
			Exports: make(map[string]Object),
		}
		for name := range mod.Exports {
			if val, ok := modEnv.Get(name); ok {
				modObj.Exports[name] = val
			}
		}
		env.Set(node.Alias.Value, modObj)
		return NIL
	}

	if len(node.Names) > 0 {
		for _, name := range node.Names {
			if !mod.Exports[name.Value] {
				return &Error{Message: "undefined export: " + name.Value, Line: name.Token.Line, Column: name.Token.Column}
			}
			if val, ok := modEnv.Get(name.Value); ok {
				env.Set(name.Value, val)
			} else {
				return &Error{Message: "undefined export: " + name.Value, Line: name.Token.Line, Column: name.Token.Column}
			}
		}
		return NIL
	}

	if node.All {
		for name := range mod.Exports {
			if val, ok := modEnv.Get(name); ok {
				env.Set(name, val)
			}
		}
		return NIL
	}

	modObj := &Module{
		Name:    node.Path.Value,
		Exports: make(map[string]Object),
	}
	for name := range mod.Exports {
		if val, ok := modEnv.Get(name); ok {
			modObj.Exports[name] = val
		}
	}
	env.Set(getModuleName(node.Path.Value), modObj)
	return NIL
}

func getModuleName(path string) string {
	parts := strings.Split(path, "/")
	name := parts[len(parts)-1]
	name = strings.TrimSuffix(name, ".carv")
	return name
}

func evalInterpolatedString(node *ast.InterpolatedString, env *Environment) Object {
	var builder strings.Builder

	for _, part := range node.Parts {
		evaluated := Eval(part, env)
		if isError(evaluated) {
			return evaluated
		}
		builder.WriteString(evaluated.Inspect())
	}

	return &String{Value: builder.String()}
}
