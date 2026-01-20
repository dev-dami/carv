package types

import (
	"fmt"

	"github.com/dev-dami/carv/pkg/ast"
)

type Checker struct {
	errors []string
	scope  *Scope
}

type Scope struct {
	symbols map[string]Type
	parent  *Scope
}

func NewScope(parent *Scope) *Scope {
	return &Scope{symbols: make(map[string]Type), parent: parent}
}

func (s *Scope) Define(name string, t Type) {
	s.symbols[name] = t
}

func (s *Scope) Lookup(name string) (Type, bool) {
	if t, ok := s.symbols[name]; ok {
		return t, true
	}
	if s.parent != nil {
		return s.parent.Lookup(name)
	}
	return nil, false
}

func NewChecker() *Checker {
	c := &Checker{
		errors: []string{},
		scope:  NewScope(nil),
	}
	c.defineBuiltins()
	return c
}

func (c *Checker) defineBuiltins() {
	c.scope.Define("print", &FunctionType{Params: []Type{Any}, Return: Void})
	c.scope.Define("println", &FunctionType{Params: []Type{Any}, Return: Void})
	c.scope.Define("len", &FunctionType{Params: []Type{Any}, Return: Int})
	c.scope.Define("str", &FunctionType{Params: []Type{Any}, Return: String})
	c.scope.Define("int", &FunctionType{Params: []Type{Any}, Return: Int})
	c.scope.Define("float", &FunctionType{Params: []Type{Any}, Return: Float})
	c.scope.Define("parse_int", &FunctionType{Params: []Type{String}, Return: Int})
	c.scope.Define("parse_float", &FunctionType{Params: []Type{String}, Return: Float})
	c.scope.Define("push", &FunctionType{Params: []Type{Any, Any}, Return: Any})
	c.scope.Define("head", &FunctionType{Params: []Type{Any}, Return: Any})
	c.scope.Define("tail", &FunctionType{Params: []Type{Any}, Return: Any})
	c.scope.Define("read_file", &FunctionType{Params: []Type{String}, Return: String})
	c.scope.Define("write_file", &FunctionType{Params: []Type{String, String}, Return: Void})
	c.scope.Define("file_exists", &FunctionType{Params: []Type{String}, Return: Bool})
	c.scope.Define("split", &FunctionType{Params: []Type{String, String}, Return: &ArrayType{Element: String}})
	c.scope.Define("join", &FunctionType{Params: []Type{Any, String}, Return: String})
	c.scope.Define("trim", &FunctionType{Params: []Type{String}, Return: String})
	c.scope.Define("substr", &FunctionType{Params: []Type{String, Int, Int}, Return: String})
	c.scope.Define("ord", &FunctionType{Params: []Type{Any}, Return: Int})
	c.scope.Define("chr", &FunctionType{Params: []Type{Int}, Return: Char})
	c.scope.Define("char_at", &FunctionType{Params: []Type{String, Int}, Return: Char})
	c.scope.Define("contains", &FunctionType{Params: []Type{String, String}, Return: Bool})
	c.scope.Define("starts_with", &FunctionType{Params: []Type{String, String}, Return: Bool})
	c.scope.Define("ends_with", &FunctionType{Params: []Type{String, String}, Return: Bool})
	c.scope.Define("replace", &FunctionType{Params: []Type{String, String, String}, Return: String})
	c.scope.Define("index_of", &FunctionType{Params: []Type{String, String}, Return: Int})
	c.scope.Define("to_upper", &FunctionType{Params: []Type{String}, Return: String})
	c.scope.Define("to_lower", &FunctionType{Params: []Type{String}, Return: String})
	c.scope.Define("exit", &FunctionType{Params: []Type{Int}, Return: Void})
	c.scope.Define("panic", &FunctionType{Params: []Type{Any}, Return: Void})
	c.scope.Define("type_of", &FunctionType{Params: []Type{Any}, Return: String})
	c.scope.Define("keys", &FunctionType{Params: []Type{Any}, Return: &ArrayType{Element: Any}})
	c.scope.Define("values", &FunctionType{Params: []Type{Any}, Return: &ArrayType{Element: Any}})
	c.scope.Define("has_key", &FunctionType{Params: []Type{Any, Any}, Return: Bool})
	c.scope.Define("delete", &FunctionType{Params: []Type{Any, Any}, Return: Any})
	c.scope.Define("set", &FunctionType{Params: []Type{Any, Any, Any}, Return: Any})
	c.scope.Define("args", &FunctionType{Params: []Type{}, Return: &ArrayType{Element: String}})
	c.scope.Define("exec", &FunctionType{Params: []Type{Any}, Return: Int})
	c.scope.Define("exec_output", &FunctionType{Params: []Type{Any}, Return: Any})
	c.scope.Define("mkdir", &FunctionType{Params: []Type{String}, Return: Void})
	c.scope.Define("append_file", &FunctionType{Params: []Type{String, String}, Return: Void})
	c.scope.Define("getenv", &FunctionType{Params: []Type{String}, Return: String})
	c.scope.Define("setenv", &FunctionType{Params: []Type{String, String}, Return: Void})
}

func (c *Checker) Errors() []string {
	return c.errors
}

func (c *Checker) error(line, col int, format string, args ...interface{}) {
	msg := fmt.Sprintf("type error at %d:%d: %s", line, col, fmt.Sprintf(format, args...))
	c.errors = append(c.errors, msg)
}

func (c *Checker) Check(program *ast.Program) bool {
	for _, stmt := range program.Statements {
		c.checkStatement(stmt)
	}
	return len(c.errors) == 0
}

func (c *Checker) checkStatement(stmt ast.Statement) {
	switch s := stmt.(type) {
	case *ast.LetStatement:
		c.checkLetStatement(s)
	case *ast.ConstStatement:
		c.checkConstStatement(s)
	case *ast.FunctionStatement:
		c.checkFunctionStatement(s)
	case *ast.ReturnStatement:
		c.checkReturnStatement(s)
	case *ast.ExpressionStatement:
		c.checkExpression(s.Expression)
	case *ast.ForStatement:
		c.checkForStatement(s)
	case *ast.ForInStatement:
		c.checkForInStatement(s)
	case *ast.WhileStatement:
		c.checkWhileStatement(s)
	case *ast.LoopStatement:
		c.checkLoopStatement(s)
	case *ast.BlockStatement:
		c.checkBlockStatement(s)
	case *ast.RequireStatement:
		c.checkRequireStatement(s)
	}
}

func (c *Checker) checkLetStatement(s *ast.LetStatement) {
	valType := c.checkExpression(s.Value)
	if valType == nil {
		return
	}

	if s.Type != nil {
		declType := c.resolveTypeExpr(s.Type)
		if declType != nil && !c.isAssignable(declType, valType) {
			line, col := s.Pos()
			c.error(line, col, "cannot assign %s to %s", valType.String(), declType.String())
		}
		c.scope.Define(s.Name.Value, declType)
	} else {
		c.scope.Define(s.Name.Value, valType)
	}
}

func (c *Checker) checkConstStatement(s *ast.ConstStatement) {
	valType := c.checkExpression(s.Value)
	if valType == nil {
		return
	}

	if s.Type != nil {
		declType := c.resolveTypeExpr(s.Type)
		if declType != nil && !c.isAssignable(declType, valType) {
			line, col := s.Pos()
			c.error(line, col, "cannot assign %s to %s", valType.String(), declType.String())
		}
		c.scope.Define(s.Name.Value, declType)
	} else {
		c.scope.Define(s.Name.Value, valType)
	}
}

func (c *Checker) checkFunctionStatement(s *ast.FunctionStatement) {
	paramTypes := make([]Type, len(s.Parameters))
	for i, p := range s.Parameters {
		if p.Type != nil {
			paramTypes[i] = c.resolveTypeExpr(p.Type)
		} else {
			paramTypes[i] = Any
		}
	}

	var retType Type = Void
	if s.ReturnType != nil {
		retType = c.resolveTypeExpr(s.ReturnType)
	}

	fnType := &FunctionType{Params: paramTypes, Return: retType}
	c.scope.Define(s.Name.Value, fnType)

	prevScope := c.scope
	c.scope = NewScope(prevScope)

	for i, p := range s.Parameters {
		c.scope.Define(p.Name.Value, paramTypes[i])
	}

	c.checkBlockStatement(s.Body)
	c.scope = prevScope
}

func (c *Checker) checkReturnStatement(s *ast.ReturnStatement) {
	if s.ReturnValue != nil {
		c.checkExpression(s.ReturnValue)
	}
}

func (c *Checker) checkForStatement(s *ast.ForStatement) {
	prevScope := c.scope
	c.scope = NewScope(prevScope)

	if s.Init != nil {
		c.checkStatement(s.Init)
	}
	if s.Condition != nil {
		condType := c.checkExpression(s.Condition)
		if condType != nil && !condType.Equals(Bool) {
			line, col := s.Condition.Pos()
			c.error(line, col, "for condition must be bool, got %s", condType.String())
		}
	}
	if s.Post != nil {
		c.checkStatement(s.Post)
	}
	c.checkBlockStatement(s.Body)

	c.scope = prevScope
}

func (c *Checker) checkForInStatement(s *ast.ForInStatement) {
	iterType := c.checkExpression(s.Iterable)

	prevScope := c.scope
	c.scope = NewScope(prevScope)

	if arr, ok := iterType.(*ArrayType); ok {
		c.scope.Define(s.Value.Value, arr.Element)
	} else {
		c.scope.Define(s.Value.Value, Any)
	}

	c.checkBlockStatement(s.Body)
	c.scope = prevScope
}

func (c *Checker) checkWhileStatement(s *ast.WhileStatement) {
	condType := c.checkExpression(s.Condition)
	if condType != nil && !condType.Equals(Bool) {
		line, col := s.Condition.Pos()
		c.error(line, col, "while condition must be bool, got %s", condType.String())
	}
	c.checkBlockStatement(s.Body)
}

func (c *Checker) checkLoopStatement(s *ast.LoopStatement) {
	c.checkBlockStatement(s.Body)
}

func (c *Checker) checkBlockStatement(s *ast.BlockStatement) {
	for _, stmt := range s.Statements {
		c.checkStatement(stmt)
	}
}

func (c *Checker) checkRequireStatement(s *ast.RequireStatement) {
	if s.Alias != nil {
		c.scope.Define(s.Alias.Value, &ModuleType{Name: s.Path.Value})
	} else if len(s.Names) > 0 {
		for _, name := range s.Names {
			c.scope.Define(name.Value, Any)
		}
	} else if s.All {
		// Wildcard imports resolved at runtime; define synthetic binding to avoid undefined errors
		c.scope.Define(s.Path.Value, &ModuleType{Name: s.Path.Value})
	}
}

func (c *Checker) checkExpression(expr ast.Expression) Type {
	if expr == nil {
		return nil
	}

	switch e := expr.(type) {
	case *ast.IntegerLiteral:
		return Int
	case *ast.FloatLiteral:
		return Float
	case *ast.StringLiteral:
		return String
	case *ast.CharLiteral:
		return Char
	case *ast.BoolLiteral:
		return Bool
	case *ast.NilLiteral:
		return Nil
	case *ast.Identifier:
		return c.checkIdentifier(e)
	case *ast.PrefixExpression:
		return c.checkPrefixExpression(e)
	case *ast.InfixExpression:
		return c.checkInfixExpression(e)
	case *ast.PipeExpression:
		return c.checkPipeExpression(e)
	case *ast.AssignExpression:
		return c.checkAssignExpression(e)
	case *ast.CallExpression:
		return c.checkCallExpression(e)
	case *ast.ArrayLiteral:
		return c.checkArrayLiteral(e)
	case *ast.MapLiteral:
		return c.checkMapLiteral(e)
	case *ast.IndexExpression:
		return c.checkIndexExpression(e)
	case *ast.IfExpression:
		return c.checkIfExpression(e)
	case *ast.FunctionLiteral:
		return c.checkFunctionLiteral(e)
	case *ast.MemberExpression:
		return c.checkMemberExpression(e)
	case *ast.SpawnExpression:
		return c.checkSpawnExpression(e)
	case *ast.InterpolatedString:
		return c.checkInterpolatedString(e)
	}

	return Any
}

func (c *Checker) checkIdentifier(e *ast.Identifier) Type {
	t, ok := c.scope.Lookup(e.Value)
	if !ok {
		line, col := e.Pos()
		c.error(line, col, "undefined: %s", e.Value)
		return Any
	}
	return t
}

func (c *Checker) checkPrefixExpression(e *ast.PrefixExpression) Type {
	rightType := c.checkExpression(e.Right)

	switch e.Operator {
	case "-":
		if !IsNumeric(rightType) {
			line, col := e.Pos()
			c.error(line, col, "operator - requires numeric type, got %s", rightType.String())
		}
		return rightType
	case "!":
		return Bool
	case "~":
		if !rightType.Equals(Int) {
			line, col := e.Pos()
			c.error(line, col, "operator ~ requires int, got %s", rightType.String())
		}
		return Int
	}

	return Any
}

func (c *Checker) checkInfixExpression(e *ast.InfixExpression) Type {
	leftType := c.checkExpression(e.Left)
	rightType := c.checkExpression(e.Right)

	switch e.Operator {
	case "+", "-", "*", "/", "%":
		if !IsNumeric(leftType) || !IsNumeric(rightType) {
			if e.Operator == "+" && leftType.Equals(String) && rightType.Equals(String) {
				return String
			}
			line, col := e.Pos()
			c.error(line, col, "operator %s requires numeric types, got %s and %s",
				e.Operator, leftType.String(), rightType.String())
		}
		if leftType.Equals(Float) || rightType.Equals(Float) {
			return Float
		}
		return Int

	case "<", ">", "<=", ">=":
		if !IsComparable(leftType) || !IsComparable(rightType) {
			line, col := e.Pos()
			c.error(line, col, "operator %s requires comparable types", e.Operator)
		}
		return Bool

	case "==", "!=":
		return Bool

	case "&&", "||":
		if !leftType.Equals(Bool) || !rightType.Equals(Bool) {
			line, col := e.Pos()
			c.error(line, col, "operator %s requires bool types, got %s and %s",
				e.Operator, leftType.String(), rightType.String())
		}
		return Bool

	case "&", "|", "^":
		if !leftType.Equals(Int) || !rightType.Equals(Int) {
			line, col := e.Pos()
			c.error(line, col, "bitwise operator requires int types")
		}
		return Int
	}

	return Any
}

func (c *Checker) checkPipeExpression(e *ast.PipeExpression) Type {
	leftType := c.checkExpression(e.Left)

	switch right := e.Right.(type) {
	case *ast.Identifier:
		fnType, ok := c.scope.Lookup(right.Value)
		if !ok {
			line, col := right.Pos()
			c.error(line, col, "undefined: %s", right.Value)
			return Any
		}
		if ft, ok := fnType.(*FunctionType); ok {
			return ft.Return
		}
		return Any

	case *ast.CallExpression:
		fnType := c.checkExpression(right.Function)
		if ft, ok := fnType.(*FunctionType); ok {
			if len(ft.Params) > 0 && !c.isAssignable(ft.Params[0], leftType) {
				line, col := e.Pos()
				c.error(line, col, "pipe: cannot pass %s to function expecting %s",
					leftType.String(), ft.Params[0].String())
			}
			return ft.Return
		}
		return Any

	default:
		return c.checkExpression(e.Right)
	}
}

func (c *Checker) checkAssignExpression(e *ast.AssignExpression) Type {
	rightType := c.checkExpression(e.Right)

	if ident, ok := e.Left.(*ast.Identifier); ok {
		leftType, exists := c.scope.Lookup(ident.Value)
		if !exists {
			line, col := ident.Pos()
			c.error(line, col, "undefined: %s", ident.Value)
			return Any
		}

		if e.Operator == "=" {
			if !c.isAssignable(leftType, rightType) {
				line, col := e.Pos()
				c.error(line, col, "cannot assign %s to %s", rightType.String(), leftType.String())
			}
		}
		return leftType
	}

	return Any
}

func (c *Checker) checkCallExpression(e *ast.CallExpression) Type {
	fnType := c.checkExpression(e.Function)

	ft, ok := fnType.(*FunctionType)
	if !ok {
		return Any
	}

	for _, arg := range e.Arguments {
		c.checkExpression(arg)
	}

	return ft.Return
}

func (c *Checker) checkArrayLiteral(e *ast.ArrayLiteral) Type {
	if len(e.Elements) == 0 {
		return &ArrayType{Element: Any}
	}

	elemType := c.checkExpression(e.Elements[0])
	for _, el := range e.Elements[1:] {
		c.checkExpression(el)
	}

	return &ArrayType{Element: elemType}
}

func (c *Checker) checkMapLiteral(e *ast.MapLiteral) Type {
	if len(e.Pairs) == 0 {
		return &MapType{Key: Any, Value: Any}
	}

	var keyType, valueType Type
	for k, v := range e.Pairs {
		keyType = c.checkExpression(k)
		valueType = c.checkExpression(v)
		break
	}

	for k, v := range e.Pairs {
		c.checkExpression(k)
		c.checkExpression(v)
	}

	return &MapType{Key: keyType, Value: valueType}
}

func (c *Checker) checkIndexExpression(e *ast.IndexExpression) Type {
	leftType := c.checkExpression(e.Left)
	indexType := c.checkExpression(e.Index)

	if arr, ok := leftType.(*ArrayType); ok {
		if !indexType.Equals(Int) {
			line, col := e.Index.Pos()
			c.error(line, col, "array index must be int, got %s", indexType.String())
		}
		return arr.Element
	}
	if leftType.Equals(String) {
		if !indexType.Equals(Int) {
			line, col := e.Index.Pos()
			c.error(line, col, "string index must be int, got %s", indexType.String())
		}
		return Char
	}
	if m, ok := leftType.(*MapType); ok {
		return m.Value
	}

	return Any
}

func (c *Checker) checkIfExpression(e *ast.IfExpression) Type {
	condType := c.checkExpression(e.Condition)
	if !condType.Equals(Bool) {
		line, col := e.Condition.Pos()
		c.error(line, col, "if condition must be bool, got %s", condType.String())
	}

	c.checkBlockStatement(e.Consequence)
	if e.Alternative != nil {
		c.checkBlockStatement(e.Alternative)
	}

	return Void
}

func (c *Checker) checkFunctionLiteral(e *ast.FunctionLiteral) Type {
	paramTypes := make([]Type, len(e.Parameters))
	for i, p := range e.Parameters {
		if p.Type != nil {
			paramTypes[i] = c.resolveTypeExpr(p.Type)
		} else {
			paramTypes[i] = Any
		}
	}

	var retType Type = Void
	if e.ReturnType != nil {
		retType = c.resolveTypeExpr(e.ReturnType)
	}

	return &FunctionType{Params: paramTypes, Return: retType}
}

func (c *Checker) checkMemberExpression(e *ast.MemberExpression) Type {
	c.checkExpression(e.Object)
	return Any
}

func (c *Checker) checkSpawnExpression(e *ast.SpawnExpression) Type {
	c.checkBlockStatement(e.Body)
	return Void
}

func (c *Checker) checkInterpolatedString(e *ast.InterpolatedString) Type {
	for _, part := range e.Parts {
		c.checkExpression(part)
	}
	return String
}

func (c *Checker) resolveTypeExpr(typeExpr ast.TypeExpr) Type {
	switch t := typeExpr.(type) {
	case *ast.BasicType:
		switch t.Name {
		case "int":
			return Int
		case "float":
			return Float
		case "bool":
			return Bool
		case "string":
			return String
		case "char":
			return Char
		case "void":
			return Void
		case "any":
			return Any
		}
	case *ast.ArrayType:
		elemType := c.resolveTypeExpr(t.ElementType)
		return &ArrayType{Element: elemType}
	case *ast.NamedType:
		if typ, ok := c.scope.Lookup(t.Name.Value); ok {
			return typ
		}
		return Any
	}
	return Any
}

func (c *Checker) isAssignable(target, source Type) bool {
	if target.Equals(Any) || source.Equals(Any) {
		return true
	}
	if target.Equals(source) {
		return true
	}
	if target.Equals(Float) && source.Equals(Int) {
		return true
	}
	return false
}
