package types

import (
	"fmt"

	"github.com/dev-dami/carv/pkg/ast"
)

// OwnershipState tracks whether a variable's value has been moved
type OwnershipState int

const (
	Owned OwnershipState = iota // variable holds its value
	Moved                       // value has been moved to another binding
)

// VarOwnership tracks ownership metadata for a variable
type VarOwnership struct {
	State   OwnershipState
	MovedAt int    // line number where the move occurred
	MovedTo string // what it was moved to (variable name or "function call")
}

type BorrowInfo struct {
	ImmutableCount int
	MutableActive  bool
}

type Checker struct {
	errors         []string
	warnings       []string
	ownership      map[string]*VarOwnership
	borrows        map[string]*BorrowInfo
	scope          *Scope
	nodeTypes      map[ast.Expression]Type
	impls          map[string]map[string]bool
	ifaceReceivers map[string]map[string]ast.ReceiverKind
	inAsyncFn      bool
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
		errors:         []string{},
		warnings:       []string{},
		ownership:      make(map[string]*VarOwnership),
		borrows:        make(map[string]*BorrowInfo),
		scope:          NewScope(nil),
		nodeTypes:      make(map[ast.Expression]Type),
		impls:          make(map[string]map[string]bool),
		ifaceReceivers: make(map[string]map[string]ast.ReceiverKind),
	}
	c.defineBuiltins()
	return c
}

func (c *Checker) TypeInfo() map[ast.Expression]Type {
	return c.nodeTypes
}

func (c *Checker) recordType(expr ast.Expression, t Type) Type {
	if expr != nil && t != nil {
		c.nodeTypes[expr] = t
	}
	return t
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
	c.scope.Define("remove_file", &FunctionType{Params: []Type{String}, Return: Void})
	c.scope.Define("rename_file", &FunctionType{Params: []Type{String, String}, Return: Void})
	c.scope.Define("read_dir", &FunctionType{Params: []Type{String}, Return: &ArrayType{Element: String}})
	c.scope.Define("cwd", &FunctionType{Params: []Type{}, Return: String})
	c.scope.Define("getenv", &FunctionType{Params: []Type{String}, Return: String})
	c.scope.Define("setenv", &FunctionType{Params: []Type{String, String}, Return: Void})
	c.scope.Define("tcp_listen", &FunctionType{Params: []Type{String, Int}, Return: Int})
	c.scope.Define("tcp_accept", &FunctionType{Params: []Type{Int}, Return: Int})
	c.scope.Define("tcp_read", &FunctionType{Params: []Type{Int, Int}, Return: String})
	c.scope.Define("tcp_write", &FunctionType{Params: []Type{Int, String}, Return: Int})
	c.scope.Define("tcp_close", &FunctionType{Params: []Type{Int}, Return: Bool})
}

func builtinModuleMemberTypes(moduleName string) map[string]Type {
	switch moduleName {
	case "net", "web":
		return map[string]Type{
			"tcp_listen": &FunctionType{Params: []Type{String, Int}, Return: Int},
			"tcp_accept": &FunctionType{Params: []Type{Int}, Return: Int},
			"tcp_read":   &FunctionType{Params: []Type{Int, Int}, Return: String},
			"tcp_write":  &FunctionType{Params: []Type{Int, String}, Return: Int},
			"tcp_close":  &FunctionType{Params: []Type{Int}, Return: Bool},
		}
	default:
		return nil
	}
}

func (c *Checker) Errors() []string {
	return c.errors
}

func (c *Checker) Warnings() []string {
	return c.warnings
}

func (c *Checker) error(line, col int, format string, args ...interface{}) {
	msg := fmt.Sprintf("type error at %d:%d: %s", line, col, fmt.Sprintf(format, args...))
	c.errors = append(c.errors, msg)
}

func (c *Checker) warning(line, col int, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	c.warnings = append(c.warnings, fmt.Sprintf("warning at line %d, col %d: %s", line, col, msg))
}

func (c *Checker) pushOwnership() map[string]*VarOwnership {
	prev := c.ownership
	next := make(map[string]*VarOwnership, len(prev))
	for name, ownership := range prev {
		next[name] = ownership
	}
	c.ownership = next
	return prev
}

func (c *Checker) popOwnership(prev map[string]*VarOwnership) {
	c.ownership = prev
}

func (c *Checker) pushBorrows() map[string]*BorrowInfo {
	prev := c.borrows
	next := make(map[string]*BorrowInfo, len(prev))
	for name, info := range prev {
		cp := *info
		next[name] = &cp
	}
	c.borrows = next
	return prev
}

func (c *Checker) popBorrows(prev map[string]*BorrowInfo) {
	c.borrows = prev
}

func (c *Checker) trackOwnership(name string, t Type) {
	if IsMoveType(t) {
		c.ownership[name] = &VarOwnership{State: Owned}
		return
	}
	delete(c.ownership, name)
}

func (c *Checker) markMoved(name string, line int, movedTo string) {
	if own, exists := c.ownership[name]; exists && own.State == Owned {
		own.State = Moved
		own.MovedAt = line
		own.MovedTo = movedTo
	}
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
	case *ast.ClassStatement:
		c.checkClassStatement(s)
	case *ast.InterfaceStatement:
		c.checkInterfaceStatement(s)
	case *ast.ImplStatement:
		c.checkImplStatement(s)
	}
}

func (c *Checker) checkLetStatement(s *ast.LetStatement) {
	valType := c.checkExpression(s.Value)
	if valType == nil {
		return
	}

	boundType := valType

	if s.Type != nil {
		declType := c.resolveTypeExpr(s.Type)
		if declType != nil && !c.isAssignable(declType, valType) {
			line, col := s.Pos()
			c.error(line, col, "cannot assign %s to %s", valType.String(), declType.String())
		}
		c.scope.Define(s.Name.Value, declType)
		boundType = declType
	} else {
		c.scope.Define(s.Name.Value, valType)
	}

	c.trackOwnership(s.Name.Value, boundType)

	if IsMoveType(valType) {
		if ident, ok := s.Value.(*ast.Identifier); ok {
			line, _ := s.Pos()
			c.markMoved(ident.Value, line, s.Name.Value)
		}
	}
}

func (c *Checker) checkConstStatement(s *ast.ConstStatement) {
	valType := c.checkExpression(s.Value)
	if valType == nil {
		return
	}

	boundType := valType

	if s.Type != nil {
		declType := c.resolveTypeExpr(s.Type)
		if declType != nil && !c.isAssignable(declType, valType) {
			line, col := s.Pos()
			c.error(line, col, "cannot assign %s to %s", valType.String(), declType.String())
		}
		c.scope.Define(s.Name.Value, declType)
		boundType = declType
	} else {
		c.scope.Define(s.Name.Value, valType)
	}

	c.trackOwnership(s.Name.Value, boundType)

	if IsMoveType(valType) {
		if ident, ok := s.Value.(*ast.Identifier); ok {
			line, _ := s.Pos()
			c.markMoved(ident.Value, line, s.Name.Value)
		}
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

	fnRetType := retType
	if s.Async {
		fnRetType = &FutureType{Inner: retType}
	}

	fnType := &FunctionType{Params: paramTypes, Return: fnRetType}
	c.scope.Define(s.Name.Value, fnType)

	prevScope := c.scope
	prevOwnership := c.pushOwnership()
	prevBorrows := c.pushBorrows()
	prevAsync := c.inAsyncFn
	c.scope = NewScope(prevScope)
	c.inAsyncFn = s.Async

	for i, p := range s.Parameters {
		c.scope.Define(p.Name.Value, paramTypes[i])
		c.trackOwnership(p.Name.Value, paramTypes[i])
	}

	c.checkBlockStatement(s.Body)
	c.scope = prevScope
	c.inAsyncFn = prevAsync
	c.popOwnership(prevOwnership)
	c.popBorrows(prevBorrows)
}

func (c *Checker) checkReturnStatement(s *ast.ReturnStatement) {
	if s.ReturnValue != nil {
		retType := c.checkExpression(s.ReturnValue)
		if IsMoveType(retType) {
			if ident, ok := s.ReturnValue.(*ast.Identifier); ok {
				line, _ := s.ReturnValue.Pos()
				c.markMoved(ident.Value, line, "return")
			}
		}
		if _, isRef := retType.(*RefType); isRef {
			line, col := s.ReturnValue.Pos()
			c.warning(line, col, "reference cannot escape function scope")
		}
	}
}

func (c *Checker) checkForStatement(s *ast.ForStatement) {
	prevScope := c.scope
	prevOwnership := c.pushOwnership()
	prevBorrows := c.pushBorrows()
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
	c.popOwnership(prevOwnership)
	c.popBorrows(prevBorrows)
}

func (c *Checker) checkForInStatement(s *ast.ForInStatement) {
	iterType := c.checkExpression(s.Iterable)

	prevScope := c.scope
	prevOwnership := c.pushOwnership()
	prevBorrows := c.pushBorrows()
	c.scope = NewScope(prevScope)

	if arr, ok := iterType.(*ArrayType); ok {
		c.scope.Define(s.Value.Value, arr.Element)
		c.trackOwnership(s.Value.Value, arr.Element)
	} else {
		c.scope.Define(s.Value.Value, Any)
		c.trackOwnership(s.Value.Value, Any)
	}

	c.checkBlockStatement(s.Body)
	c.scope = prevScope
	c.popOwnership(prevOwnership)
	c.popBorrows(prevBorrows)
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
		if members := builtinModuleMemberTypes(s.Path.Value); members != nil {
			for _, name := range s.Names {
				if t, ok := members[name.Value]; ok {
					c.scope.Define(name.Value, t)
				} else {
					line, col := name.Pos()
					c.error(line, col, "undefined export: %s", name.Value)
				}
			}
			return
		}
		for _, n := range s.Names {
			c.scope.Define(n.Value, Any)
		}
	} else if s.All {
		if members := builtinModuleMemberTypes(s.Path.Value); members != nil {
			for name, t := range members {
				c.scope.Define(name, t)
			}
			return
		}
		// Wildcard imports for non-builtin modules are resolved at runtime.
		c.scope.Define(s.Path.Value, &ModuleType{Name: s.Path.Value})
	}
}

func (c *Checker) checkExpression(expr ast.Expression) Type {
	if expr == nil {
		return nil
	}

	var t Type
	switch e := expr.(type) {
	case *ast.IntegerLiteral:
		t = Int
	case *ast.FloatLiteral:
		t = Float
	case *ast.StringLiteral:
		t = String
	case *ast.CharLiteral:
		t = Char
	case *ast.BoolLiteral:
		t = Bool
	case *ast.NilLiteral:
		t = Nil
	case *ast.Identifier:
		t = c.checkIdentifier(e)
	case *ast.PrefixExpression:
		t = c.checkPrefixExpression(e)
	case *ast.InfixExpression:
		t = c.checkInfixExpression(e)
	case *ast.PipeExpression:
		t = c.checkPipeExpression(e)
	case *ast.AssignExpression:
		t = c.checkAssignExpression(e)
	case *ast.CallExpression:
		t = c.checkCallExpression(e)
	case *ast.ArrayLiteral:
		t = c.checkArrayLiteral(e)
	case *ast.MapLiteral:
		t = c.checkMapLiteral(e)
	case *ast.IndexExpression:
		t = c.checkIndexExpression(e)
	case *ast.IfExpression:
		t = c.checkIfExpression(e)
	case *ast.FunctionLiteral:
		t = c.checkFunctionLiteral(e)
	case *ast.MemberExpression:
		t = c.checkMemberExpression(e)
	case *ast.SpawnExpression:
		t = c.checkSpawnExpression(e)
	case *ast.InterpolatedString:
		t = c.checkInterpolatedString(e)
	case *ast.BorrowExpression:
		t = c.checkBorrowExpression(e)
	case *ast.DerefExpression:
		t = c.checkDerefExpression(e)
	case *ast.CastExpression:
		t = c.checkCastExpression(e)
	case *ast.AwaitExpression:
		t = c.checkAwaitExpression(e)
	default:
		t = Any
	}

	return c.recordType(expr, t)
}

func (c *Checker) checkIdentifier(e *ast.Identifier) Type {
	t, ok := c.scope.Lookup(e.Value)
	if !ok {
		line, col := e.Pos()
		c.error(line, col, "undefined: %s", e.Value)
		return Any
	}
	if own, exists := c.ownership[e.Value]; exists && own.State == Moved {
		line, col := e.Pos()
		c.warning(line, col, "use of moved value '%s' (moved at line %d)", e.Value, own.MovedAt)
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
		if bi, exists := c.borrows[ident.Value]; exists && (bi.ImmutableCount > 0 || bi.MutableActive) {
			line, col := e.Pos()
			c.warning(line, col, "cannot assign to '%s' while it is borrowed", ident.Value)
		}
		return leftType
	}

	if member, ok := e.Left.(*ast.MemberExpression); ok {
		if ident, ok := member.Object.(*ast.Identifier); ok && ident.Value == "self" {
			if selfType, exists := c.scope.Lookup("self"); exists {
				if ref, ok := selfType.(*RefType); ok && !ref.Mutable {
					line, col := e.Pos()
					c.warning(line, col, "cannot assign to field through immutable receiver (&self)")
				}
			}
		}
		leftType := c.checkExpression(member)
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

	isVariadic := c.isVariadicFunction(e)

	if !isVariadic && len(e.Arguments) != len(ft.Params) {
		line, col := e.Pos()
		c.error(line, col, "function expects %d arguments, got %d", len(ft.Params), len(e.Arguments))
		return ft.Return
	}

	for i, arg := range e.Arguments {
		argType := c.checkExpression(arg)
		if i < len(ft.Params) {
			paramType := ft.Params[i]
			if !paramType.Equals(Any) && !c.isAssignable(paramType, argType) {
				line, col := arg.Pos()
				c.error(line, col, "argument %d: cannot pass %s as %s", i+1, argType.String(), paramType.String())
			}
		}

		if IsMoveType(argType) {
			if ident, ok := arg.(*ast.Identifier); ok {
				line, _ := arg.Pos()
				c.markMoved(ident.Value, line, "function call")
			}
		}
	}

	return ft.Return
}

func (c *Checker) isVariadicFunction(e *ast.CallExpression) bool {
	if ident, ok := e.Function.(*ast.Identifier); ok {
		switch ident.Value {
		case "print", "println", "exec", "exec_output", "substr", "exit", "panic":
			return true
		}
	}
	return false
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

	prevScope := c.scope
	prevOwnership := c.pushOwnership()
	prevBorrows := c.pushBorrows()
	c.scope = NewScope(prevScope)

	for i, p := range e.Parameters {
		c.scope.Define(p.Name.Value, paramTypes[i])
		c.trackOwnership(p.Name.Value, paramTypes[i])
	}

	if e.Body != nil {
		c.checkBlockStatement(e.Body)
	}

	c.scope = prevScope
	c.popOwnership(prevOwnership)
	c.popBorrows(prevBorrows)

	return &FunctionType{Params: paramTypes, Return: retType}
}

func (c *Checker) checkMemberExpression(e *ast.MemberExpression) Type {
	objType := c.checkExpression(e.Object)

	if result := c.checkMemberExpressionForInterface(e, objType); result != nil {
		return result
	}

	if mod, ok := objType.(*ModuleType); ok {
		if members := builtinModuleMemberTypes(mod.Name); members != nil {
			if t, exists := members[e.Member.Value]; exists {
				return t
			}
			line, col := e.Member.Pos()
			c.error(line, col, "undefined member %s on module %s", e.Member.Value, mod.Name)
			return Any
		}
		return Any
	}

	if cls, ok := objType.(*ClassType); ok {
		if fieldType, exists := cls.Fields[e.Member.Value]; exists {
			return fieldType
		}
	}

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

func (c *Checker) checkBorrowExpression(e *ast.BorrowExpression) Type {
	innerType := c.checkExpression(e.Value)

	varName := ""
	if ident, ok := e.Value.(*ast.Identifier); ok {
		varName = ident.Value
	}

	if varName != "" {
		if own, exists := c.ownership[varName]; exists && own.State == Moved {
			line, col := e.Pos()
			c.warning(line, col, "cannot borrow moved value '%s' (moved at line %d)", varName, own.MovedAt)
			return &RefType{Inner: innerType, Mutable: e.Mutable}
		}

		info := c.borrows[varName]
		if info == nil {
			info = &BorrowInfo{}
			c.borrows[varName] = info
		}

		if e.Mutable {
			if info.ImmutableCount > 0 {
				line, col := e.Pos()
				c.warning(line, col, "cannot mutably borrow '%s': already immutably borrowed", varName)
			}
			if info.MutableActive {
				line, col := e.Pos()
				c.warning(line, col, "cannot mutably borrow '%s': already mutably borrowed", varName)
			}
			info.MutableActive = true
		} else {
			if info.MutableActive {
				line, col := e.Pos()
				c.warning(line, col, "cannot immutably borrow '%s': already mutably borrowed", varName)
			}
			info.ImmutableCount++
		}
	}

	return &RefType{Inner: innerType, Mutable: e.Mutable}
}

func (c *Checker) checkDerefExpression(e *ast.DerefExpression) Type {
	innerType := c.checkExpression(e.Value)
	if ref, ok := innerType.(*RefType); ok {
		return ref.Inner
	}
	line, col := e.Pos()
	c.warning(line, col, "dereference of non-reference type %s", innerType.String())
	return innerType
}

func (c *Checker) checkClassStatement(s *ast.ClassStatement) {
	fields := make(map[string]Type)
	for _, f := range s.Fields {
		var ft Type
		if f.Type != nil {
			ft = c.resolveTypeExpr(f.Type)
		} else {
			ft = Any
		}
		fields[f.Name.Value] = ft
	}

	classType := &ClassType{Name: s.Name.Value, Fields: fields}
	c.scope.Define(s.Name.Value, classType)

	for _, method := range s.Methods {
		prevScope := c.scope
		prevOwnership := c.pushOwnership()
		prevBorrows := c.pushBorrows()
		c.scope = NewScope(prevScope)

		switch method.Receiver {
		case ast.RecvRef:
			c.scope.Define("self", &RefType{Inner: classType, Mutable: false})
		case ast.RecvMutRef:
			c.scope.Define("self", &RefType{Inner: classType, Mutable: true})
		case ast.RecvValue:
			c.scope.Define("self", classType)
		}

		paramTypes := make([]Type, len(method.Parameters))
		for i, p := range method.Parameters {
			if p.Type != nil {
				paramTypes[i] = c.resolveTypeExpr(p.Type)
			} else {
				paramTypes[i] = Any
			}
			c.scope.Define(p.Name.Value, paramTypes[i])
		}

		if method.Body != nil {
			c.checkBlockStatement(method.Body)
		}

		c.scope = prevScope
		c.popOwnership(prevOwnership)
		c.popBorrows(prevBorrows)
	}
}

func (c *Checker) checkInterfaceStatement(s *ast.InterfaceStatement) {
	methods := make(map[string]*FunctionType)
	receivers := make(map[string]ast.ReceiverKind)

	for _, sig := range s.Methods {
		paramTypes := make([]Type, len(sig.Parameters))
		for i, p := range sig.Parameters {
			if p.Type != nil {
				paramTypes[i] = c.resolveTypeExpr(p.Type)
			} else {
				paramTypes[i] = Any
			}
		}

		var retType Type = Void
		if sig.ReturnType != nil {
			retType = c.resolveTypeExpr(sig.ReturnType)
		}

		methods[sig.Name.Value] = &FunctionType{Params: paramTypes, Return: retType}
		receivers[sig.Name.Value] = sig.Receiver
	}

	ifaceType := &InterfaceType{Name: s.Name.Value, Methods: methods}
	c.scope.Define(s.Name.Value, ifaceType)
	c.ifaceReceivers[s.Name.Value] = receivers
}

func (c *Checker) checkImplStatement(s *ast.ImplStatement) {
	ifaceType, ok := c.scope.Lookup(s.Interface.Value)
	if !ok {
		line, col := s.Interface.Pos()
		c.error(line, col, "undefined interface: %s", s.Interface.Value)
		return
	}
	iface, ok := ifaceType.(*InterfaceType)
	if !ok {
		line, col := s.Interface.Pos()
		c.error(line, col, "%s is not an interface", s.Interface.Value)
		return
	}

	classType, ok := c.scope.Lookup(s.Type.Value)
	if !ok {
		line, col := s.Type.Pos()
		c.error(line, col, "undefined type: %s", s.Type.Value)
		return
	}

	implMethods := make(map[string]*FunctionType)
	implReceivers := make(map[string]ast.ReceiverKind)
	implDecls := make(map[string]*ast.MethodDecl)
	for _, method := range s.Methods {
		paramTypes := make([]Type, len(method.Parameters))
		for i, p := range method.Parameters {
			if p.Type != nil {
				paramTypes[i] = c.resolveTypeExpr(p.Type)
			} else {
				paramTypes[i] = Any
			}
		}

		var retType Type = Void
		if method.ReturnType != nil {
			retType = c.resolveTypeExpr(method.ReturnType)
		}

		implMethods[method.Name.Value] = &FunctionType{Params: paramTypes, Return: retType}
		implReceivers[method.Name.Value] = method.Receiver
		implDecls[method.Name.Value] = method

		prevScope := c.scope
		prevOwnership := c.pushOwnership()
		prevBorrows := c.pushBorrows()
		c.scope = NewScope(prevScope)

		switch method.Receiver {
		case ast.RecvRef:
			c.scope.Define("self", &RefType{Inner: classType, Mutable: false})
		case ast.RecvMutRef:
			c.scope.Define("self", &RefType{Inner: classType, Mutable: true})
		case ast.RecvValue:
			c.scope.Define("self", classType)
		}

		for i, p := range method.Parameters {
			c.scope.Define(p.Name.Value, paramTypes[i])
		}

		if method.Body != nil {
			c.checkBlockStatement(method.Body)
		}

		c.scope = prevScope
		c.popOwnership(prevOwnership)
		c.popBorrows(prevBorrows)
	}

	for name, ifaceMethod := range iface.Methods {
		implMethod, exists := implMethods[name]
		if !exists {
			line, col := s.Pos()
			c.error(line, col, "type %s does not implement %s: missing method %s",
				s.Type.Value, s.Interface.Value, name)
			continue
		}
		if len(implMethod.Params) != len(ifaceMethod.Params) {
			line, col := s.Pos()
			c.error(line, col, "method %s has wrong number of parameters", name)
			continue
		}
		for i, p := range ifaceMethod.Params {
			if !c.isAssignable(p, implMethod.Params[i]) {
				line, col := s.Pos()
				c.error(line, col, "method %s parameter %d: expected %s, got %s",
					name, i+1, p.String(), implMethod.Params[i].String())
			}
		}
		if !c.isAssignable(ifaceMethod.Return, implMethod.Return) {
			line, col := s.Pos()
			c.error(line, col, "method %s: return type mismatch: expected %s, got %s",
				name, ifaceMethod.Return.String(), implMethod.Return.String())
		}
		if ifaceReceiverMap, ok := c.ifaceReceivers[s.Interface.Value]; ok {
			if ifaceReceiver, ok := ifaceReceiverMap[name]; ok {
				if implReceiver, ok := implReceivers[name]; ok && ifaceReceiver != implReceiver {
					line, col := s.Pos()
					if decl, ok := implDecls[name]; ok {
						line, col = decl.Name.Pos()
					}
					c.warning(line, col, "receiver mismatch for method %s: interface expects %s, impl has %s",
						name, receiverKindName(ifaceReceiver), receiverKindName(implReceiver))
				}
			}
		}
	}

	if c.impls[s.Type.Value] == nil {
		c.impls[s.Type.Value] = make(map[string]bool)
	}
	c.impls[s.Type.Value][s.Interface.Value] = true
}

func (c *Checker) checkCastExpression(e *ast.CastExpression) Type {
	c.checkExpression(e.Value)
	targetType := c.resolveTypeExpr(e.Type)

	if ref, ok := targetType.(*RefType); ok {
		if iface, ok := ref.Inner.(*InterfaceType); ok {
			_ = iface
			return targetType
		}
	}

	return targetType
}

func (c *Checker) checkAwaitExpression(e *ast.AwaitExpression) Type {
	if !c.inAsyncFn {
		line, col := e.Pos()
		c.error(line, col, "await can only be used inside async functions")
		return Any
	}

	for name, info := range c.borrows {
		if info.ImmutableCount > 0 || info.MutableActive {
			line, col := e.Pos()
			c.error(line, col, "borrow of '%s' cannot be held across await point", name)
		}
	}

	innerType := c.checkExpression(e.Value)
	if ft, ok := innerType.(*FutureType); ok {
		return ft.Inner
	}

	line, col := e.Pos()
	c.error(line, col, "await requires Future type, got %s", innerType.String())
	return Any
}

func (c *Checker) checkMemberExpressionForInterface(e *ast.MemberExpression, objType Type) Type {
	if ref, ok := objType.(*RefType); ok {
		if iface, ok := ref.Inner.(*InterfaceType); ok {
			if methodType, exists := iface.Methods[e.Member.Value]; exists {
				if ifaceReceiverMap, ok := c.ifaceReceivers[iface.Name]; ok {
					if recv, ok := ifaceReceiverMap[e.Member.Value]; ok {
						if recv == ast.RecvMutRef && !ref.Mutable {
							line, col := e.Member.Pos()
							c.error(line, col, "cannot call &mut self method '%s' through immutable interface reference", e.Member.Value)
						}
					}
				}
				return methodType
			}
			line, col := e.Member.Pos()
			c.error(line, col, "interface %s has no method %s", iface.Name, e.Member.Value)
			return Any
		}
	}
	return nil
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
	case *ast.RefType:
		inner := c.resolveTypeExpr(t.Inner)
		return &RefType{Inner: inner, Mutable: t.Mutable}
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

func receiverKindName(kind ast.ReceiverKind) string {
	switch kind {
	case ast.RecvRef:
		return "&self"
	case ast.RecvMutRef:
		return "&mut self"
	case ast.RecvValue:
		return "self"
	case ast.RecvNone:
		return "none"
	default:
		return "unknown"
	}
}
