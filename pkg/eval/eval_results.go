package eval

import "github.com/dev-dami/carv/pkg/ast"

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
