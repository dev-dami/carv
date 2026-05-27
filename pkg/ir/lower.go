package ir

import (
	"fmt"
	"strconv"

	"github.com/dev-dami/carv/pkg/ast"
	"github.com/dev-dami/carv/pkg/types"
)

type Lowerer struct {
	typeInfo      map[ast.Expression]types.Type
	fnReturns     map[string]IRType
	scope         *lowerScope
	fn            *Function
	labelCounter  int
	loopBreaks    []string
	loopContinues []string
	errors        []string
	classes       map[string]*ast.ClassStatement
}

type lowerScope struct {
	parent *lowerScope
	slots  map[string]int
	names  []string
}

func newLowerScope(parent *lowerScope) *lowerScope {
	return &lowerScope{parent: parent, slots: make(map[string]int)}
}

func NewLowerer(typeInfo map[ast.Expression]types.Type) *Lowerer {
	return &Lowerer{
		typeInfo:  typeInfo,
		fnReturns: make(map[string]IRType),
		classes:   make(map[string]*ast.ClassStatement),
	}
}

func (l *Lowerer) Lower(program *ast.Program) *Module {
	mod := &Module{
		Functions: make(map[string]*Function),
		Entry:     "main",
	}

	for _, stmt := range program.Statements {
		if cls, ok := stmt.(*ast.ClassStatement); ok {
			l.classes[cls.Name.Value] = cls
		}
	}

	for _, stmt := range program.Statements {
		if fn, ok := stmt.(*ast.FunctionStatement); ok {
			l.fnReturns[fn.Name.Value] = l.resolveReturnType(fn)
		}
	}

	for _, stmt := range program.Statements {
		if fn, ok := stmt.(*ast.FunctionStatement); ok {
			mod.Functions[fn.Name.Value] = l.lowerFunction(fn)
		}
	}

	// Collect top-level (script-style) statements that are not function/class/interface/impl declarations.
	// These are wrapped in an implicit main() function, matching the C codegen behavior.
	var topLevel []ast.Statement
	for _, stmt := range program.Statements {
		switch stmt.(type) {
		case *ast.FunctionStatement, *ast.ClassStatement, *ast.InterfaceStatement, *ast.ImplStatement:
			// skip declarations
		default:
			topLevel = append(topLevel, stmt)
		}
	}

	if len(topLevel) > 0 {
		if _, exists := mod.Functions["main"]; exists {
			l.error("cannot define fn main() together with top-level script statements")
		} else {
			l.scope = newLowerScope(nil)
			l.fn = &Function{
				Name:    "main",
				Returns: IRVoid,
				Body:    make([]Inst, 0),
			}

			for _, stmt := range topLevel {
				l.lowerStatement(stmt)
			}

			l.emit(Inst{Op: OpReturn})
			mod.Functions["main"] = l.fn
		}
	}

	// Lower class methods as {ClassName}_{MethodName} standalone functions
	for _, stmt := range program.Statements {
		if cls, ok := stmt.(*ast.ClassStatement); ok {
			for _, method := range cls.Methods {
				l.lowerMethodDecl(mod, cls.Name.Value, method)
			}
		}
	}

	// Lower impl-block methods
	for _, stmt := range program.Statements {
		if impl, ok := stmt.(*ast.ImplStatement); ok {
			typeName := impl.Type.Value
			for _, method := range impl.Methods {
				l.lowerMethodDecl(mod, typeName, method)
			}
		}
	}

	if _, ok := mod.Functions["main"]; ok {
		mod.Entry = "main"
	}

	return mod
}

func (l *Lowerer) lowerFunction(fn *ast.FunctionStatement) *Function {
	l.scope = newLowerScope(nil)
	l.fn = &Function{
		Name:    fn.Name.Value,
		Returns: l.fnReturns[fn.Name.Value],
		Body:    make([]Inst, 0),
	}

	for _, p := range fn.Parameters {
		irt := l.resolveIRType(p.Type)
		l.allocSlot(p.Name.Value, irt)
		l.fn.Params = append(l.fn.Params, Local{Name: p.Name.Value, Type: irt})
	}

	if fn.Body != nil {
		l.lowerBlock(fn.Body)
	}

	if l.fn.Returns == IRVoid || l.fn.Returns == IRNil {
		l.emit(Inst{Op: OpReturn})
	} else {
		l.emit(Inst{Op: OpConstNil, Type: l.fn.Returns})
		l.emit(Inst{Op: OpReturnVal, Type: l.fn.Returns})
	}

	return l.fn
}

func (l *Lowerer) lowerBlock(block *ast.BlockStatement) {
	l.enterScope()
	for _, stmt := range block.Statements {
		l.lowerStatement(stmt)
	}
	l.exitScope()
}

func (l *Lowerer) lowerStatement(stmt ast.Statement) {
	switch s := stmt.(type) {
	case *ast.ExpressionStatement:
		l.lowerExpression(s.Expression)
		l.emit(Inst{Op: OpNop})

	case *ast.LetStatement:
		irt := l.resolveIRType(s.Type)
		if irt == IRAny {
			irt = l.inferExprType(s.Value)
		}
		l.allocSlot(s.Name.Value, irt)
		l.lowerExpression(s.Value)
		l.emit(Inst{Op: OpStore, Label: s.Name.Value})

	case *ast.ConstStatement:
		irt := l.resolveIRType(s.Type)
		if irt == IRAny {
			irt = l.inferExprType(s.Value)
		}
		l.allocSlot(s.Name.Value, irt)
		l.lowerExpression(s.Value)
		l.emit(Inst{Op: OpStore, Label: s.Name.Value})

	case *ast.ReturnStatement:
		if s.ReturnValue != nil {
			l.lowerExpression(s.ReturnValue)
			l.emit(Inst{Op: OpReturnVal})
		} else {
			l.emit(Inst{Op: OpReturn})
		}

	case *ast.ForStatement:
		l.lowerForStatement(s)

	case *ast.ForInStatement:
		l.lowerForInStatement(s)

	case *ast.WhileStatement:
		l.lowerWhileStatement(s)

	case *ast.LoopStatement:
		l.lowerLoopStatement(s)

	case *ast.BreakStatement:
		if len(l.loopBreaks) > 0 {
			l.emit(Inst{Op: OpJmp, Label: l.loopBreaks[len(l.loopBreaks)-1]})
		}

	case *ast.ContinueStatement:
		if len(l.loopContinues) > 0 {
			l.emit(Inst{Op: OpJmp, Label: l.loopContinues[len(l.loopContinues)-1]})
		}

	case *ast.BlockStatement:
		l.lowerBlock(s)

	case *ast.FunctionStatement:

	case *ast.UnsafeStatement:
		l.lowerBlock(s.Body)

	default:
		l.error("unknown statement %T", stmt)
	}
}

func (l *Lowerer) lowerForStatement(s *ast.ForStatement) {
	startLabel := l.newLabel("for_start")
	endLabel := l.newLabel("for_end")
	contLabel := l.newLabel("for_cont")

	l.loopBreaks = append(l.loopBreaks, endLabel)
	l.loopContinues = append(l.loopContinues, contLabel)

	if s.Init != nil {
		l.lowerStatement(s.Init)
	}

	l.emit(Inst{Op: OpLabel, Label: startLabel})

	if s.Condition != nil {
		l.lowerExpression(s.Condition)
		l.emit(Inst{Op: OpJmpIf, Label: endLabel})
	}

	l.lowerBlock(s.Body)
	l.emit(Inst{Op: OpLabel, Label: contLabel})

	if s.Post != nil {
		l.lowerStatement(s.Post)
	}

	l.emit(Inst{Op: OpJmp, Label: startLabel})
	l.emit(Inst{Op: OpLabel, Label: endLabel})

	l.loopBreaks = l.loopBreaks[:len(l.loopBreaks)-1]
	l.loopContinues = l.loopContinues[:len(l.loopContinues)-1]
}

func (l *Lowerer) lowerForInStatement(s *ast.ForInStatement) {
	endLabel := l.newLabel("forin_end")
	bodyLabel := l.newLabel("forin_body")
	contLabel := l.newLabel("forin_cont")
	iterVar := "_iter_" + s.Value.Value
	idxVar := "_idx_" + s.Value.Value

	l.loopBreaks = append(l.loopBreaks, endLabel)
	l.loopContinues = append(l.loopContinues, contLabel)

	irt := l.inferExprType(s.Iterable)
	l.allocSlot(iterVar, irt)
	l.lowerExpression(s.Iterable)
	l.emit(Inst{Op: OpStore, Label: iterVar})

	switch irt {
	case IRString, IRArray:
		valType := IRChar
		if irt == IRArray {
			valType = IRAny
		}
		l.allocSlot(idxVar, IRInt)
		l.allocSlot(s.Value.Value, valType)

		l.emit(Inst{Op: OpConstInt, Arg: Operand{Int: 0}, Type: IRInt})
		l.emit(Inst{Op: OpStore, Label: idxVar})

		l.emit(Inst{Op: OpLabel, Label: bodyLabel})
		l.emit(Inst{Op: OpLoad, Label: idxVar})
		l.emit(Inst{Op: OpLoad, Label: iterVar})
		if irt == IRString {
			l.emit(Inst{Op: OpStrLen, Type: IRInt})
		} else {
			l.emit(Inst{Op: OpArrayLen, Type: IRInt})
		}
		l.emit(Inst{Op: OpLt, Type: IRBool})
		l.emit(Inst{Op: OpJmpIf, Label: endLabel})

		l.emit(Inst{Op: OpLoad, Label: iterVar})
		l.emit(Inst{Op: OpLoad, Label: idxVar})
		if irt == IRString {
			l.emit(Inst{Op: OpStrIndex, Type: valType})
		} else {
			l.emit(Inst{Op: OpArrayGet, Type: valType})
		}
		l.emit(Inst{Op: OpStore, Label: s.Value.Value})

		l.lowerBlock(s.Body)
		l.emit(Inst{Op: OpLabel, Label: contLabel})

		l.emit(Inst{Op: OpLoad, Label: idxVar})
		l.emit(Inst{Op: OpConstInt, Arg: Operand{Int: 1}, Type: IRInt})
		l.emit(Inst{Op: OpAdd, Type: IRInt})
		l.emit(Inst{Op: OpStore, Label: idxVar})
		l.emit(Inst{Op: OpJmp, Label: bodyLabel})
		l.emit(Inst{Op: OpLabel, Label: endLabel})

	case IRMap:
		l.emit(Inst{Op: OpLoad, Label: iterVar})
		l.emit(Inst{Op: OpMapKeys, Type: IRArray})
		l.emit(Inst{Op: OpStore, Label: idxVar})
		l.allocSlot(idxVar, IRArray)
		iVar := idxVar + "_i"
		l.allocSlot(iVar, IRInt)
		l.emit(Inst{Op: OpConstInt, Arg: Operand{Int: 0}, Type: IRInt})
		l.emit(Inst{Op: OpStore, Label: iVar})
		l.allocSlot(s.Value.Value, IRString)

		l.emit(Inst{Op: OpLabel, Label: bodyLabel})
		l.emit(Inst{Op: OpLoad, Label: iVar})
		l.emit(Inst{Op: OpLoad, Label: idxVar})
		l.emit(Inst{Op: OpArrayLen, Type: IRInt})
		l.emit(Inst{Op: OpLt, Type: IRBool})
		l.emit(Inst{Op: OpJmpIf, Label: endLabel})

		l.emit(Inst{Op: OpLoad, Label: idxVar})
		l.emit(Inst{Op: OpLoad, Label: iVar})
		l.emit(Inst{Op: OpArrayGet, Type: IRString})
		l.emit(Inst{Op: OpStore, Label: s.Value.Value})

		l.lowerBlock(s.Body)
		l.emit(Inst{Op: OpLabel, Label: contLabel})

		l.emit(Inst{Op: OpLoad, Label: iVar})
		l.emit(Inst{Op: OpConstInt, Arg: Operand{Int: 1}, Type: IRInt})
		l.emit(Inst{Op: OpAdd, Type: IRInt})
		l.emit(Inst{Op: OpStore, Label: iVar})
		l.emit(Inst{Op: OpJmp, Label: bodyLabel})
		l.emit(Inst{Op: OpLabel, Label: endLabel})
	}

	l.loopBreaks = l.loopBreaks[:len(l.loopBreaks)-1]
	l.loopContinues = l.loopContinues[:len(l.loopContinues)-1]
}

func (l *Lowerer) lowerWhileStatement(s *ast.WhileStatement) {
	startLabel := l.newLabel("while_start")
	endLabel := l.newLabel("while_end")

	l.loopBreaks = append(l.loopBreaks, endLabel)
	l.loopContinues = append(l.loopContinues, startLabel)

	l.emit(Inst{Op: OpLabel, Label: startLabel})
	l.lowerExpression(s.Condition)
	l.emit(Inst{Op: OpJmpIf, Label: endLabel})
	l.lowerBlock(s.Body)
	l.emit(Inst{Op: OpJmp, Label: startLabel})
	l.emit(Inst{Op: OpLabel, Label: endLabel})

	l.loopBreaks = l.loopBreaks[:len(l.loopBreaks)-1]
	l.loopContinues = l.loopContinues[:len(l.loopContinues)-1]
}

func (l *Lowerer) lowerLoopStatement(s *ast.LoopStatement) {
	startLabel := l.newLabel("loop_start")
	endLabel := l.newLabel("loop_end")

	l.loopBreaks = append(l.loopBreaks, endLabel)
	l.loopContinues = append(l.loopContinues, startLabel)

	l.emit(Inst{Op: OpLabel, Label: startLabel})
	l.lowerBlock(s.Body)
	l.emit(Inst{Op: OpJmp, Label: startLabel})
	l.emit(Inst{Op: OpLabel, Label: endLabel})

	l.loopBreaks = l.loopBreaks[:len(l.loopBreaks)-1]
	l.loopContinues = l.loopContinues[:len(l.loopContinues)-1]
}

func (l *Lowerer) lowerExpression(expr ast.Expression) IRType {
	switch e := expr.(type) {
	case *ast.IntegerLiteral:
		irt := IRInt
		if ti, ok := l.typeInfo[e]; ok {
			irt = ResolveType(ti)
		}
		l.emit(Inst{Op: OpConstInt, Arg: Operand{Int: e.Value}, Type: irt})
		return irt

	case *ast.FloatLiteral:
		l.emit(Inst{Op: OpConstFloat, Arg: Operand{Float: e.Value}, Type: IRFloat})
		return IRFloat

	case *ast.StringLiteral:
		l.emit(Inst{Op: OpConstString, Arg: Operand{Str: e.Value}, Type: IRString})
		return IRString

	case *ast.BoolLiteral:
		l.emit(Inst{Op: OpConstBool, Arg: Operand{Bool: e.Value}, Type: IRBool})
		return IRBool

	case *ast.NilLiteral:
		l.emit(Inst{Op: OpConstNil, Type: IRNil})
		return IRNil

	case *ast.CharLiteral:
		l.emit(Inst{Op: OpConstInt, Arg: Operand{Int: int64(e.Value)}, Type: IRInt})
		return IRInt

	case *ast.Identifier:
		irt := l.lookupSlotType(e.Value)
		if irt == IRAny {
			if ti, ok := l.typeInfo[e]; ok {
				irt = ResolveType(ti)
			}
		}
		l.emit(Inst{Op: OpLoad, Label: e.Value, Type: irt})
		return irt

	case *ast.PrefixExpression:
		return l.lowerPrefix(e)

	case *ast.InfixExpression:
		return l.lowerInfix(e)

	case *ast.AssignExpression:
		return l.lowerAssign(e)

	case *ast.CallExpression:
		return l.lowerCall(e)

	case *ast.IfExpression:
		return l.lowerIfExpression(e)

	case *ast.IndexExpression:
		return l.lowerIndex(e)

	case *ast.MemberExpression:
		return l.lowerMember(e)

	case *ast.ArrayLiteral:
		irt := IRArray
		if ti, ok := l.typeInfo[e]; ok {
			irt = ResolveType(ti)
		}
		for _, el := range e.Elements {
			l.lowerExpression(el)
		}
		l.emit(Inst{Op: OpArrayLit, Arg: Operand{Int: int64(len(e.Elements))}, Type: irt})
		return irt

	case *ast.MapLiteral:
		irt := IRMap
		if ti, ok := l.typeInfo[e]; ok {
			irt = ResolveType(ti)
		}
		for k, v := range e.Pairs {
			l.lowerExpression(k)
			l.lowerExpression(v)
		}
		l.emit(Inst{Op: OpMapLit, Arg: Operand{Int: int64(len(e.Pairs))}, Type: irt})
		return irt

	case *ast.MatchExpression:
		return l.lowerMatch(e)

	case *ast.OkExpression:
		l.lowerExpression(e.Value)
		l.emit(Inst{Op: OpOk, Type: IRResult})
		return IRResult

	case *ast.ErrExpression:
		l.lowerExpression(e.Value)
		l.emit(Inst{Op: OpErr, Type: IRResult})
		return IRResult

	case *ast.TryExpression:
		l.lowerExpression(e.Value)
		l.emit(Inst{Op: OpTry, Type: IRResult})
		return IRResult

	case *ast.BorrowExpression:
		irt := l.lowerExpression(e.Value)
		l.emit(Inst{Op: OpBorrow, Type: irt})
		return irt

	case *ast.DerefExpression:
		irt := l.lowerExpression(e.Value)
		l.emit(Inst{Op: OpDeref, Type: irt})
		return irt

	case *ast.InterpolatedString:
		count := int64(len(e.Parts))
		for _, p := range e.Parts {
			l.lowerExpression(p)
		}
		l.emit(Inst{Op: OpInterp, Arg: Operand{Int: count}, Type: IRString})
		return IRString

	case *ast.NewExpression:
		irt := IRClass
		if ti, ok := l.typeInfo[e]; ok {
			irt = ResolveType(ti)
		}
		className := "?"
		if named, ok := e.Type.(*ast.NamedType); ok {
			className = named.Name.Value
		}
		l.emit(Inst{Op: OpNew, Label: className, Type: irt})
		return irt

	case *ast.FunctionLiteral:
		irt := IRFunc
		if ti, ok := l.typeInfo[e]; ok {
			irt = ResolveType(ti)
		}
		l.emit(Inst{Op: OpConstNil, Type: irt})
		return irt

	case *ast.CastExpression:
		return l.lowerCast(e)

	default:
		l.error("unknown expression %T", expr)
		l.emit(Inst{Op: OpConstNil, Type: IRNil})
		return IRNil
	}
}

func (l *Lowerer) lowerPrefix(e *ast.PrefixExpression) IRType {
	irt := l.lowerExpression(e.Right)
	switch e.Operator {
	case "-":
		l.emit(Inst{Op: OpNeg, Type: irt})
	case "!":
		l.emit(Inst{Op: OpNot, Type: IRBool})
	case "~":
		l.emit(Inst{Op: OpBitNot, Type: irt})
	default:
		l.error("unknown prefix operator %s", e.Operator)
	}
	return irt
}

func (l *Lowerer) lowerInfix(e *ast.InfixExpression) IRType {
	irt := IRAny
	if ti, ok := l.typeInfo[e]; ok {
		irt = ResolveType(ti)
	}

	l.lowerExpression(e.Left)
	l.lowerExpression(e.Right)

	switch e.Operator {
	case "+":
		l.emit(Inst{Op: OpAdd, Type: irt})
	case "-":
		l.emit(Inst{Op: OpSub, Type: irt})
	case "*":
		l.emit(Inst{Op: OpMul, Type: irt})
	case "/":
		l.emit(Inst{Op: OpDiv, Type: irt})
	case "%":
		l.emit(Inst{Op: OpMod, Type: irt})
	case "**":
		l.emit(Inst{Op: OpPow, Type: irt})
	case "==":
		l.emit(Inst{Op: OpEq, Type: IRBool})
	case "!=":
		l.emit(Inst{Op: OpNe, Type: IRBool})
	case "<":
		l.emit(Inst{Op: OpLt, Type: IRBool})
	case ">":
		l.emit(Inst{Op: OpGt, Type: IRBool})
	case "<=":
		l.emit(Inst{Op: OpLe, Type: IRBool})
	case ">=":
		l.emit(Inst{Op: OpGe, Type: IRBool})
	case "&&":
		l.emit(Inst{Op: OpAnd, Type: IRBool})
	case "||":
		l.emit(Inst{Op: OpOr, Type: IRBool})
	default:
		l.error("unknown infix operator %s", e.Operator)
	}
	return irt
}

func (l *Lowerer) lowerAssign(e *ast.AssignExpression) IRType {
	irt := l.inferExprType(e.Right)

	if e.Operator == "=" {
		// Simple assignment: rhs → store lhs
		l.lowerExpression(e.Right)
		switch left := e.Left.(type) {
		case *ast.Identifier:
			l.emit(Inst{Op: OpStore, Label: left.Value, Type: irt})
		case *ast.IndexExpression:
			l.lowerExpression(left.Left)
			l.lowerExpression(left.Index)
			l.emit(Inst{Op: OpArraySet, Type: irt})
		case *ast.MemberExpression:
			l.lowerExpression(left.Object)
			l.emit(Inst{Op: OpSetField, Label: left.Member.Value, Type: irt})
		default:
			l.error("unsupported assignment target %T", e.Left)
		}
	} else {
		// Compound assignment: lhs +=/-=/*= etc rhs → load lhs, eval rhs, op, store lhs
		// Currently only simple identifiers are supported for compound targets.
		leftIdent, ok := e.Left.(*ast.Identifier)
		if !ok {
			l.error("compound assignment target must be a simple identifier, got %T", e.Left)
			return irt
		}

		l.emit(Inst{Op: OpLoad, Label: leftIdent.Value, Type: irt})
		l.lowerExpression(e.Right)

		switch e.Operator {
		case "+=":
			l.emit(Inst{Op: OpAdd, Type: irt})
		case "-=":
			l.emit(Inst{Op: OpSub, Type: irt})
		case "*=":
			l.emit(Inst{Op: OpMul, Type: irt})
		case "/=":
			l.emit(Inst{Op: OpDiv, Type: irt})
		case "%=":
			l.emit(Inst{Op: OpMod, Type: irt})
		default:
			l.error("unknown compound assignment operator %s", e.Operator)
		}
		l.emit(Inst{Op: OpStore, Label: leftIdent.Value, Type: irt})
	}
	return irt
}

func (l *Lowerer) lowerCall(e *ast.CallExpression) IRType {
	irt := IRAny
	if ti, ok := l.typeInfo[e]; ok {
		irt = ResolveType(ti)
	}

	for _, arg := range e.Arguments {
		l.lowerExpression(arg)
	}

	argc := int64(len(e.Arguments))

	if ident, ok := e.Function.(*ast.Identifier); ok {
		switch ident.Value {
		case "print":
			l.emit(Inst{Op: OpPrint})
			return IRVoid
		case "println":
			l.emit(Inst{Op: OpPrintln})
			return IRVoid
		case "len":
			l.emit(Inst{Op: OpLen, Type: IRInt})
			return IRInt
		case "contains":
			l.emit(Inst{Op: OpContains, Type: IRBool})
			return IRBool
		case "keys":
			l.emit(Inst{Op: OpKeys, Type: IRArray})
			return IRArray
		case "assert":
			l.emit(Inst{Op: OpAssert})
			return IRVoid
		case "int":
			l.emit(Inst{Op: OpToInt, Type: IRInt})
			return IRInt
		case "float":
			l.emit(Inst{Op: OpToFloat, Type: IRFloat})
			return IRFloat
		case "string":
			l.emit(Inst{Op: OpToString, Type: IRString})
			return IRString
		case "readline":
			l.emit(Inst{Op: OpReadLine, Type: IRString})
			return IRString
		}
	}

	if member, ok := e.Function.(*ast.MemberExpression); ok {
		return l.lowerMethodCall(member, e.Arguments)
	}

	l.emit(Inst{Op: OpCall, Label: l.extractFuncName(e.Function), Arg: Operand{Int: argc}, Type: irt})
	return irt
}

func (l *Lowerer) lowerMethodCall(member *ast.MemberExpression, args []ast.Expression) IRType {
	irt := IRAny
	methodName := member.Member.Value

	l.lowerExpression(member.Object)

	for _, arg := range args {
		l.lowerExpression(arg)
	}

	argc := int64(len(args) + 1)

	// Determine how to dispatch this method call.
	className, isVirtual := l.classNameForMethodCall(member)
	if isVirtual {
		// Interface dispatch: resolve the class at runtime via OpCallVirt
		l.emit(Inst{Op: OpCallVirt, Label: methodName, Arg: Operand{Int: argc}, Type: irt})
	} else if className != "" {
		// Direct class dispatch: call {ClassName}_{MethodName}
		l.emit(Inst{Op: OpCall, Label: className+"_"+methodName, Arg: Operand{Int: argc}, Type: irt})
	} else {
		// Unqualified fallback (should not normally happen)
		l.emit(Inst{Op: OpCall, Label: methodName, Arg: Operand{Int: argc}, Type: irt})
	}
	return irt
}

func (l *Lowerer) inferClassName(expr ast.Expression) string {
	if ti, ok := l.typeInfo[expr]; ok {
		if cls, ok := ti.(*types.ClassType); ok {
			return cls.Name
		}
		if ref, ok := ti.(*types.RefType); ok {
			if cls, ok := ref.Inner.(*types.ClassType); ok {
				return cls.Name
			}
		}
	}
	return ""
}

func (l *Lowerer) classNameForMethodCall(member *ast.MemberExpression) (string, bool) {
	// Check if the receiver is an interface reference (needs virtual dispatch)
	if ti, ok := l.typeInfo[member.Object]; ok {
		if ref, ok := ti.(*types.RefType); ok {
			if _, ok := ref.Inner.(*types.InterfaceType); ok {
				return "", true // virtual dispatch needed
			}
		}
		// Direct class or ref-to-class
		if cls, ok := ti.(*types.ClassType); ok {
			return cls.Name, false
		}
		if ref, ok := ti.(*types.RefType); ok {
			if cls, ok := ref.Inner.(*types.ClassType); ok {
				return cls.Name, false
			}
		}
	}

	// Fallback: use the method name to find which class owns this method
	methodName := member.Member.Value
	for className, cls := range l.classes {
		for _, m := range cls.Methods {
			if m.Name.Value == methodName {
				return className, false
			}
		}
	}

	return "", false
}

func (l *Lowerer) lowerMethodDecl(mod *Module, className string, method *ast.MethodDecl) {
	fnName := className + "_" + method.Name.Value

	// Pre-collect return type
	retType := l.resolveIRType(method.ReturnType)
	l.fnReturns[fnName] = retType

	l.scope = newLowerScope(nil)
	l.fn = &Function{
		Name:    fnName,
		Returns: retType,
		Body:    make([]Inst, 0),
	}

	// Self receiver becomes the first parameter
	if method.Receiver != ast.RecvNone {
		selfType := IRClass
		l.allocSlot("self", selfType)
		l.fn.Params = append(l.fn.Params, Local{Name: "self", Type: selfType})
	}

	// Additional parameters
	for _, p := range method.Parameters {
		irt := l.resolveIRType(p.Type)
		l.allocSlot(p.Name.Value, irt)
		l.fn.Params = append(l.fn.Params, Local{Name: p.Name.Value, Type: irt})
	}

	if method.Body != nil {
		for _, stmt := range method.Body.Statements {
			l.lowerStatement(stmt)
		}
	}

	if retType == IRVoid || retType == IRNil {
		l.emit(Inst{Op: OpReturn})
	} else {
		l.emit(Inst{Op: OpConstNil, Type: retType})
		l.emit(Inst{Op: OpReturnVal, Type: retType})
	}

	mod.Functions[fnName] = l.fn
}

func (l *Lowerer) lowerIfExpression(e *ast.IfExpression) IRType {
	irt := IRAny
	if ti, ok := l.typeInfo[e]; ok {
		irt = ResolveType(ti)
	}

	elseLabel := l.newLabel("if_else")
	endLabel := l.newLabel("if_end")

	l.lowerExpression(e.Condition)
	l.emit(Inst{Op: OpJmpIf, Label: elseLabel})

	l.lowerBlock(e.Consequence)
	l.emit(Inst{Op: OpJmp, Label: endLabel})
	l.emit(Inst{Op: OpLabel, Label: elseLabel})

	if e.Alternative != nil {
		l.lowerBlock(e.Alternative)
	}

	l.emit(Inst{Op: OpLabel, Label: endLabel, Type: irt})
	return irt
}

func (l *Lowerer) lowerIndex(e *ast.IndexExpression) IRType {
	irt := IRAny
	if ti, ok := l.typeInfo[e]; ok {
		irt = ResolveType(ti)
	}

	l.lowerExpression(e.Left)
	l.lowerExpression(e.Index)

	containerType := l.inferExprType(e.Left)
	switch containerType {
	case IRString:
		l.emit(Inst{Op: OpStrIndex, Type: irt})
	case IRMap:
		l.emit(Inst{Op: OpMapGet, Type: irt})
	default:
		l.emit(Inst{Op: OpArrayGet, Type: irt})
	}
	return irt
}

func (l *Lowerer) lowerMember(e *ast.MemberExpression) IRType {
	irt := IRAny
	if ti, ok := l.typeInfo[e]; ok {
		irt = ResolveType(ti)
	}

	l.lowerExpression(e.Object)
	l.emit(Inst{Op: OpGetField, Label: e.Member.Value, Type: irt})
	return irt
}

func (l *Lowerer) lowerMatch(e *ast.MatchExpression) IRType {
	endLabel := l.newLabel("match_end")

	l.lowerExpression(e.Value)
	l.emit(Inst{Op: OpStore, Label: "__match_val"})
	l.allocSlot("__match_val", IRAny)

	isResultMatch := false
	for _, arm := range e.Arms {
		if _, ok := arm.Pattern.(*ast.OkExpression); ok {
			isResultMatch = true
			break
		}
		if _, ok := arm.Pattern.(*ast.ErrExpression); ok {
			isResultMatch = true
			break
		}
	}

	if isResultMatch {
		okLabel := l.newLabel("match_ok")
		errLabel := l.newLabel("match_err")

		l.emit(Inst{Op: OpLoad, Label: "__match_val"})
		l.emit(Inst{Op: OpMatchResult, Label: okLabel, Arg: Operand{Str: errLabel}})

		for _, arm := range e.Arms {
			if okPat, ok := arm.Pattern.(*ast.OkExpression); ok {
				l.emit(Inst{Op: OpLabel, Label: okLabel})
				if okPat.Value != nil {
					if ident, ok := okPat.Value.(*ast.Identifier); ok {
						l.allocSlot(ident.Value, IRAny)
						l.emit(Inst{Op: OpLoad, Label: "__match_val"})
						l.emit(Inst{Op: OpGetField, Label: "ok"})
						l.emit(Inst{Op: OpStore, Label: ident.Value})
					}
				}
				if block, ok := arm.Body.(*ast.BlockExpression); ok {
					l.lowerBlock(block.Block)
				} else {
					l.lowerExpression(arm.Body)
				}
				l.emit(Inst{Op: OpJmp, Label: endLabel})
			} else if errPat, ok := arm.Pattern.(*ast.ErrExpression); ok {
				l.emit(Inst{Op: OpLabel, Label: errLabel})
				if errPat.Value != nil {
					if ident, ok := errPat.Value.(*ast.Identifier); ok {
						l.allocSlot(ident.Value, IRAny)
						l.emit(Inst{Op: OpLoad, Label: "__match_val"})
						l.emit(Inst{Op: OpGetField, Label: "err"})
						l.emit(Inst{Op: OpStore, Label: ident.Value})
					}
				}
				if block, ok := arm.Body.(*ast.BlockExpression); ok {
					l.lowerBlock(block.Block)
				} else {
					l.lowerExpression(arm.Body)
				}
				l.emit(Inst{Op: OpJmp, Label: endLabel})
			}
		}
	} else {
		for _, arm := range e.Arms {
			armLabel := l.newLabel("match_arm")
			nextLabel := l.newLabel("match_next")

			l.emit(Inst{Op: OpLoad, Label: "__match_val"})

			if ident, ok := arm.Pattern.(*ast.Identifier); ok && ident.Value == "_" {
				l.emit(Inst{Op: OpNop})
			} else if lit, ok := arm.Pattern.(*ast.IntegerLiteral); ok {
				l.emit(Inst{Op: OpMatchArmEq, Arg: Operand{Int: lit.Value}, Label: armLabel})
			} else if lit, ok := arm.Pattern.(*ast.StringLiteral); ok {
				l.emit(Inst{Op: OpConstString, Arg: Operand{Str: lit.Value}})
				l.emit(Inst{Op: OpEq, Type: IRBool})
				l.emit(Inst{Op: OpJmpIf, Label: nextLabel})
			} else {
				l.emit(Inst{Op: OpConstBool, Arg: Operand{Bool: true}})
			}
			l.emit(Inst{Op: OpJmp, Label: nextLabel})

			l.emit(Inst{Op: OpLabel, Label: armLabel})
			if block, ok := arm.Body.(*ast.BlockExpression); ok {
				l.lowerBlock(block.Block)
			} else {
				l.lowerExpression(arm.Body)
			}
			l.emit(Inst{Op: OpJmp, Label: endLabel})
			l.emit(Inst{Op: OpLabel, Label: nextLabel})
		}
	}

	l.emit(Inst{Op: OpLabel, Label: endLabel})
	return IRAny
}

func (l *Lowerer) lowerCast(e *ast.CastExpression) IRType {
	targetType := l.resolveIRType(e.Type)
	l.lowerExpression(e.Value)
	switch targetType {
	case IRInt:
		l.emit(Inst{Op: OpCastInt, Type: IRInt})
	case IRFloat:
		l.emit(Inst{Op: OpCastFloat, Type: IRFloat})
	}
	return targetType
}

func (l *Lowerer) enterScope() {
	l.scope = newLowerScope(l.scope)
}

func (l *Lowerer) exitScope() {
	if l.scope != nil && l.scope.parent != nil {
		l.scope = l.scope.parent
	}
}

func (l *Lowerer) allocSlot(name string, t IRType) int {
	if l.scope == nil {
		l.scope = newLowerScope(nil)
	}
	if idx, ok := l.scope.slots[name]; ok {
		return idx
	}
	idx := len(l.fn.Locals)
	l.scope.slots[name] = idx
	l.scope.names = append(l.scope.names, name)
	l.fn.Locals = append(l.fn.Locals, Local{Name: name, Type: t})
	return idx
}

func (l *Lowerer) lookupSlotType(name string) IRType {
	for s := l.scope; s != nil; s = s.parent {
		if idx, ok := s.slots[name]; ok {
			if idx >= 0 && idx < len(l.fn.Locals) {
				return l.fn.Locals[idx].Type
			}
		}
	}
	return IRAny
}

func (l *Lowerer) resolveIRType(typeExpr ast.TypeExpr) IRType {
	if typeExpr == nil {
		return IRAny
	}
	switch t := typeExpr.(type) {
	case *ast.BasicType:
		switch t.Name {
		case "int", "i64", "char":
			return IRInt
		case "float", "f64":
			return IRFloat
		case "bool":
			return IRBool
		case "string":
			return IRString
		case "void":
			return IRVoid
		}
	case *ast.ArrayType:
		return IRArray
	case *ast.MapType:
		return IRMap
	case *ast.NamedType:
		if _, ok := l.classes[t.Name.Value]; ok {
			return IRClass
		}
	}
	return IRAny
}

func (l *Lowerer) resolveReturnType(fn *ast.FunctionStatement) IRType {
	if fn.ReturnType != nil {
		return l.resolveIRType(fn.ReturnType)
	}
	return IRVoid
}

func (l *Lowerer) inferExprType(expr ast.Expression) IRType {
	if ti, ok := l.typeInfo[expr]; ok {
		return ResolveType(ti)
	}
	switch e := expr.(type) {
	case *ast.IntegerLiteral, *ast.CharLiteral:
		return IRInt
	case *ast.FloatLiteral:
		return IRFloat
	case *ast.StringLiteral, *ast.InterpolatedString:
		return IRString
	case *ast.BoolLiteral:
		return IRBool
	case *ast.NilLiteral:
		return IRNil
	case *ast.ArrayLiteral:
		return IRArray
	case *ast.MapLiteral:
		return IRMap
	case *ast.OkExpression, *ast.ErrExpression:
		return IRResult
	case *ast.Identifier:
		return l.lookupSlotType(e.Value)
	case *ast.InfixExpression:
		if e.Operator == "+" || e.Operator == "-" || e.Operator == "*" || e.Operator == "/" {
			lt := l.inferExprType(e.Left)
			if lt == IRFloat {
				return IRFloat
			}
			return IRInt
		}
		return IRBool
	}
	return IRAny
}

func (l *Lowerer) extractFuncName(expr ast.Expression) string {
	if ident, ok := expr.(*ast.Identifier); ok {
		return ident.Value
	}
	if member, ok := expr.(*ast.MemberExpression); ok {
		if obj, ok := member.Object.(*ast.Identifier); ok {
			return obj.Value + "_" + member.Member.Value
		}
	}
	return "?"
}

func (l *Lowerer) newLabel(prefix string) string {
	id := l.labelCounter
	l.labelCounter++
	return fmt.Sprintf("__%s_%d", prefix, id)
}

func (l *Lowerer) emit(inst Inst) {
	l.fn.Body = append(l.fn.Body, inst)
}

func (l *Lowerer) Errors() []string {
	return l.errors
}

func (l *Lowerer) error(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	l.errors = append(l.errors, msg)
}

var _ = strconv.Itoa
