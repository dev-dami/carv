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
	mod           *Module
	labelCounter  int
	spawnCounter  int
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
	l.mod = mod

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

	// First pass: register all named functions as empty stubs so that any
	// function body can reference any other function regardless of definition
	// order (handles cross-recursion: fn a() calls fn b() and vice versa).
	var topLevelFns []string
	for _, stmt := range program.Statements {
		if fn, ok := stmt.(*ast.FunctionStatement); ok {
			mod.Functions[fn.Name.Value] = nil
			topLevelFns = append(topLevelFns, fn.Name.Value)
		}
	}

	// Second pass: lower each function body.  The function is already in
	// mod.Functions (from the first pass) so isNamedFunction finds it,
	// and lowerFunction overwrites the nil stub with the real function.
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
	} else if len(topLevel) == 0 && len(topLevelFns) == 1 {
		mod.Entry = topLevelFns[0]
	}

	return mod
}

func (l *Lowerer) lowerFunction(fn *ast.FunctionStatement) *Function {
	l.scope = newLowerScope(nil)
	l.fn = &Function{
		Name:    fn.Name.Value,
		Returns: l.fnReturns[fn.Name.Value],
		Body:    make([]Inst, 0),
		Async:   fn.Async,
	}

	// Register early so recursive calls can find this function
	if l.mod != nil {
		l.mod.Functions[fn.Name.Value] = l.fn
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
		if l.fn.Async {
			l.emit(Inst{Op: OpMakeFuture})
		}
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
		idx := l.allocSlot(s.Name.Value, irt)
		l.lowerExpression(s.Value)
		l.emit(Inst{Op: OpStore, Arg: Operand{Idx: idx}})

	case *ast.ConstStatement:
		irt := l.resolveIRType(s.Type)
		if irt == IRAny {
			irt = l.inferExprType(s.Value)
		}
		idx := l.allocSlot(s.Name.Value, irt)
		l.lowerExpression(s.Value)
		l.emit(Inst{Op: OpStore, Arg: Operand{Idx: idx}})

	case *ast.ReturnStatement:
		if s.ReturnValue != nil {
			l.lowerExpression(s.ReturnValue)
			if l.fn != nil && l.fn.Async {
				l.emit(Inst{Op: OpMakeFuture})
			}
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
	iterIdx := l.allocSlot(iterVar, irt)
	l.lowerExpression(s.Iterable)
	l.emit(Inst{Op: OpStore, Arg: Operand{Idx: iterIdx}})

	switch irt {
	case IRString, IRArray:
		valType := IRChar
		if irt == IRArray {
			valType = IRAny
		}
		idxIdx := l.allocSlot(idxVar, IRInt)
		valIdx := l.allocSlot(s.Value.Value, valType)

		l.emit(Inst{Op: OpConstInt, Arg: Operand{Int: 0}, Type: IRInt})
		l.emit(Inst{Op: OpStore, Arg: Operand{Idx: idxIdx}})

		l.emit(Inst{Op: OpLabel, Label: bodyLabel})
		l.emit(Inst{Op: OpLoad, Arg: Operand{Idx: idxIdx}})
		l.emit(Inst{Op: OpLoad, Arg: Operand{Idx: iterIdx}})
		if irt == IRString {
			l.emit(Inst{Op: OpStrLen, Type: IRInt})
		} else {
			l.emit(Inst{Op: OpArrayLen, Type: IRInt})
		}
		l.emit(Inst{Op: OpLt, Type: IRBool})
		l.emit(Inst{Op: OpJmpIf, Label: endLabel})

		l.emit(Inst{Op: OpLoad, Arg: Operand{Idx: iterIdx}})
		l.emit(Inst{Op: OpLoad, Arg: Operand{Idx: idxIdx}})
		if irt == IRString {
			l.emit(Inst{Op: OpStrIndex, Type: valType})
		} else {
			l.emit(Inst{Op: OpArrayGet, Type: valType})
		}
		l.emit(Inst{Op: OpStore, Arg: Operand{Idx: valIdx}})

		l.lowerBlock(s.Body)
		l.emit(Inst{Op: OpLabel, Label: contLabel})

		l.emit(Inst{Op: OpLoad, Arg: Operand{Idx: idxIdx}})
		l.emit(Inst{Op: OpConstInt, Arg: Operand{Int: 1}, Type: IRInt})
		l.emit(Inst{Op: OpAdd, Type: IRInt})
		l.emit(Inst{Op: OpStore, Arg: Operand{Idx: idxIdx}})
		l.emit(Inst{Op: OpJmp, Label: bodyLabel})
		l.emit(Inst{Op: OpLabel, Label: endLabel})

	case IRMap:
		l.emit(Inst{Op: OpLoad, Arg: Operand{Idx: iterIdx}})
		l.emit(Inst{Op: OpMapKeys, Type: IRArray})
		idxIdx := l.allocSlot(idxVar, IRArray)
		l.emit(Inst{Op: OpStore, Arg: Operand{Idx: idxIdx}})
		iVar := idxVar + "_i"
		iIdx := l.allocSlot(iVar, IRInt)
		l.emit(Inst{Op: OpConstInt, Arg: Operand{Int: 0}, Type: IRInt})
		l.emit(Inst{Op: OpStore, Arg: Operand{Idx: iIdx}})
		valIdx := l.allocSlot(s.Value.Value, IRString)

		l.emit(Inst{Op: OpLabel, Label: bodyLabel})
		l.emit(Inst{Op: OpLoad, Arg: Operand{Idx: iIdx}})
		l.emit(Inst{Op: OpLoad, Arg: Operand{Idx: idxIdx}})
		l.emit(Inst{Op: OpArrayLen, Type: IRInt})
		l.emit(Inst{Op: OpLt, Type: IRBool})
		l.emit(Inst{Op: OpJmpIf, Label: endLabel})

		l.emit(Inst{Op: OpLoad, Arg: Operand{Idx: idxIdx}})
		l.emit(Inst{Op: OpLoad, Arg: Operand{Idx: iIdx}})
		l.emit(Inst{Op: OpArrayGet, Type: IRString})
		l.emit(Inst{Op: OpStore, Arg: Operand{Idx: valIdx}})

		l.lowerBlock(s.Body)
		l.emit(Inst{Op: OpLabel, Label: contLabel})

		l.emit(Inst{Op: OpLoad, Arg: Operand{Idx: iIdx}})
		l.emit(Inst{Op: OpConstInt, Arg: Operand{Int: 1}, Type: IRInt})
		l.emit(Inst{Op: OpAdd, Type: IRInt})
		l.emit(Inst{Op: OpStore, Arg: Operand{Idx: iIdx}})
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
		l.emit(Inst{Op: OpLoad, Arg: Operand{Idx: l.lookupSlot(e.Value)}, Type: irt})
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

		// Resolve return type from type info
		returnType := IRVoid
		if ti, ok := l.typeInfo[e]; ok {
			if ft, ok := ti.(*types.FunctionType); ok {
				returnType = ResolveType(ft.Return)
			}
		}

		// Generate unique name for the closure function
		fnName := l.newLabel("closure")

		// Save current lowering state
		oldScope := l.scope
		oldFn := l.fn

		// Create new function for the closure body
		l.scope = newLowerScope(nil)
		l.fn = &Function{
			Name:    fnName,
			Returns: returnType,
			Body:    make([]Inst, 0),
		}

		// Add parameters as locals
		for _, p := range e.Parameters {
			paramIrt := l.resolveIRType(p.Type)
			l.allocSlot(p.Name.Value, paramIrt)
			l.fn.Params = append(l.fn.Params, Local{Name: p.Name.Value, Type: paramIrt})
		}

		// Find free variables (identifiers referencing enclosing-scope variables)
		captures := findFreeVariables(e.Body, e.Parameters, oldScope)

		// Add captured variables as locals in the closure function
		for _, name := range captures {
			t := lookupSlotTypeInScope(oldScope, oldFn, name)
			idx := l.allocSlot(name, t)
			l.fn.Captures = append(l.fn.Captures, name)
			l.fn.CaptureIdx = append(l.fn.CaptureIdx, idx)
		}

		// Lower function body
		if e.Body != nil {
			l.lowerBlock(e.Body)
		}

		// Ensure return instruction
		if returnType == IRVoid || returnType == IRNil {
			l.emit(Inst{Op: OpReturn})
		} else {
			l.emit(Inst{Op: OpConstNil, Type: returnType})
			l.emit(Inst{Op: OpReturnVal, Type: returnType})
		}

		// Register the closure function in the module
		l.mod.Functions[fnName] = l.fn

		// Restore previous lowering state
		l.scope = oldScope
		l.fn = oldFn

		// Emit capture loads — each OpLoad reads from the enclosing frame's locals
		for _, name := range captures {
			t := l.lookupSlotType(name)
			l.emit(Inst{Op: OpLoad, Arg: Operand{Idx: l.lookupSlot(name)}, Type: t})
		}

		// Emit closure creation — pops capture values from stack
		l.emit(Inst{Op: OpMakeClosure, Label: fnName, Arg: Operand{Int: int64(len(captures))}, Type: irt})
		return irt

	case *ast.PipeExpression:
		return l.lowerPipeExpression(e)

	case *ast.CastExpression:
		return l.lowerCast(e)

	case *ast.AwaitExpression:
		irt := l.lowerExpression(e.Value)
		l.emit(Inst{Op: OpAwait, Type: irt})
		return irt

	case *ast.SpawnExpression:
		l.lowerSpawn(e)
		return IRNil

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
		if leftIdent, ok := e.Left.(*ast.Identifier); ok {
			if call, ok := e.Right.(*ast.CallExpression); ok {
				if fnIdent, ok := call.Function.(*ast.Identifier); ok && fnIdent.Value == "push" && len(call.Arguments) == 2 {
					if argIdent, ok := call.Arguments[0].(*ast.Identifier); ok && argIdent.Value == leftIdent.Value {
						l.lowerExpression(call.Arguments[1])
						l.emit(Inst{Op: OpArrayPushLocal, Arg: Operand{Idx: l.lookupSlot(leftIdent.Value)}, Type: IRArray})
						return irt
					}
				}
			}
		}

		// Simple assignment: rhs → store lhs
		l.lowerExpression(e.Right)
		switch left := e.Left.(type) {
		case *ast.Identifier:
			l.emit(Inst{Op: OpStore, Arg: Operand{Idx: l.lookupSlot(left.Value)}, Type: irt})
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

		l.emit(Inst{Op: OpLoad, Arg: Operand{Idx: l.lookupSlot(leftIdent.Value)}, Type: irt})
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
		l.emit(Inst{Op: OpStore, Arg: Operand{Idx: l.lookupSlot(leftIdent.Value)}, Type: irt})
	}
	return irt
}

func (l *Lowerer) isNamedFunction(name string) bool {
	if l.mod == nil {
		return false
	}
	_, ok := l.mod.Functions[name]
	return ok
}

func (l *Lowerer) lowerCall(e *ast.CallExpression) IRType {
	irt := IRAny
	if ti, ok := l.typeInfo[e]; ok {
		irt = ResolveType(ti)
	}

	_, isMemberExpr := e.Function.(*ast.MemberExpression)

	// Check for builtin calls first — applies to any identifier, not just named functions
	if ident, ok := e.Function.(*ast.Identifier); ok {
		switch ident.Value {
		case "print":
			for _, arg := range e.Arguments {
				l.lowerExpression(arg)
			}
			l.emit(Inst{Op: OpPrint})
			return IRVoid
		case "println":
			for _, arg := range e.Arguments {
				l.lowerExpression(arg)
			}
			l.emit(Inst{Op: OpPrintln})
			return IRVoid
		case "len":
			for _, arg := range e.Arguments {
				l.lowerExpression(arg)
			}
			l.emit(Inst{Op: OpLen, Type: IRInt})
			return IRInt
		case "contains":
			for _, arg := range e.Arguments {
				l.lowerExpression(arg)
			}
			l.emit(Inst{Op: OpContains, Type: IRBool})
			return IRBool
		case "keys":
			for _, arg := range e.Arguments {
				l.lowerExpression(arg)
			}
			l.emit(Inst{Op: OpKeys, Type: IRArray})
			return IRArray
		case "assert":
			for _, arg := range e.Arguments {
				l.lowerExpression(arg)
			}
			l.emit(Inst{Op: OpAssert})
			return IRVoid
		case "int":
			for _, arg := range e.Arguments {
				l.lowerExpression(arg)
			}
			l.emit(Inst{Op: OpToInt, Type: IRInt})
			return IRInt
		case "float":
			for _, arg := range e.Arguments {
				l.lowerExpression(arg)
			}
			l.emit(Inst{Op: OpToFloat, Type: IRFloat})
			return IRFloat
		case "string":
			for _, arg := range e.Arguments {
				l.lowerExpression(arg)
			}
			l.emit(Inst{Op: OpToString, Type: IRString})
			return IRString
		case "readline":
			l.emit(Inst{Op: OpReadLine, Type: IRString})
			return IRString
		case "str":
			for _, arg := range e.Arguments {
				l.lowerExpression(arg)
			}
			l.emit(Inst{Op: OpToString, Type: IRString})
			return IRString
		case "type_of":
			for _, arg := range e.Arguments {
				l.lowerExpression(arg)
			}
			l.emit(Inst{Op: OpTypeOf, Type: IRString})
			return IRString
		case "parse_int":
			for _, arg := range e.Arguments {
				l.lowerExpression(arg)
			}
			l.emit(Inst{Op: OpParseInt, Type: IRInt})
			return IRInt
		case "parse_float":
			for _, arg := range e.Arguments {
				l.lowerExpression(arg)
			}
			l.emit(Inst{Op: OpParseFloat, Type: IRFloat})
			return IRFloat
		case "push":
			for _, arg := range e.Arguments {
				l.lowerExpression(arg)
			}
			l.emit(Inst{Op: OpArrayPush, Type: IRArray})
			return IRArray
		case "head":
			for _, arg := range e.Arguments {
				l.lowerExpression(arg)
			}
			l.emit(Inst{Op: OpArrayHead, Type: IRAny})
			return IRAny
		case "tail":
			for _, arg := range e.Arguments {
				l.lowerExpression(arg)
			}
			l.emit(Inst{Op: OpArrayTail, Type: IRArray})
			return IRArray
		case "split":
			for _, arg := range e.Arguments {
				l.lowerExpression(arg)
			}
			l.emit(Inst{Op: OpStrSplit, Type: IRArray})
			return IRArray
		case "join":
			for _, arg := range e.Arguments {
				l.lowerExpression(arg)
			}
			l.emit(Inst{Op: OpStrJoin, Type: IRString})
			return IRString
		case "trim":
			for _, arg := range e.Arguments {
				l.lowerExpression(arg)
			}
			l.emit(Inst{Op: OpStrTrim, Type: IRString})
			return IRString
		case "substr":
			for _, arg := range e.Arguments {
				l.lowerExpression(arg)
			}
			l.emit(Inst{Op: OpStrSubstr, Type: IRString})
			return IRString
		case "starts_with":
			for _, arg := range e.Arguments {
				l.lowerExpression(arg)
			}
			l.emit(Inst{Op: OpStrStartsWith, Type: IRBool})
			return IRBool
		case "ends_with":
			for _, arg := range e.Arguments {
				l.lowerExpression(arg)
			}
			l.emit(Inst{Op: OpStrEndsWith, Type: IRBool})
			return IRBool
		case "replace":
			for _, arg := range e.Arguments {
				l.lowerExpression(arg)
			}
			l.emit(Inst{Op: OpStrReplace, Type: IRString})
			return IRString
		case "index_of":
			for _, arg := range e.Arguments {
				l.lowerExpression(arg)
			}
			l.emit(Inst{Op: OpStrIndexOf, Type: IRInt})
			return IRInt
		case "to_upper":
			for _, arg := range e.Arguments {
				l.lowerExpression(arg)
			}
			l.emit(Inst{Op: OpStrToUpper, Type: IRString})
			return IRString
		case "to_lower":
			for _, arg := range e.Arguments {
				l.lowerExpression(arg)
			}
			l.emit(Inst{Op: OpStrToLower, Type: IRString})
			return IRString
		case "ord":
			for _, arg := range e.Arguments {
				l.lowerExpression(arg)
			}
			l.emit(Inst{Op: OpOrd, Type: IRInt})
			return IRInt
		case "chr":
			for _, arg := range e.Arguments {
				l.lowerExpression(arg)
			}
			l.emit(Inst{Op: OpChr, Type: IRChar})
			return IRChar
		case "char_at":
			for _, arg := range e.Arguments {
				l.lowerExpression(arg)
			}
			l.emit(Inst{Op: OpCharAt, Type: IRChar})
			return IRChar
		case "values":
			for _, arg := range e.Arguments {
				l.lowerExpression(arg)
			}
			l.emit(Inst{Op: OpMapValues, Type: IRArray})
			return IRArray
		case "has_key":
			for _, arg := range e.Arguments {
				l.lowerExpression(arg)
			}
			l.emit(Inst{Op: OpContains, Type: IRBool})
			return IRBool
		case "set":
			for _, arg := range e.Arguments {
				l.lowerExpression(arg)
			}
			l.emit(Inst{Op: OpMapSet, Type: IRMap})
			return IRMap
		case "delete":
			for _, arg := range e.Arguments {
				l.lowerExpression(arg)
			}
			l.emit(Inst{Op: OpMapDelete, Type: IRMap})
			return IRMap
		case "read_file":
			for _, arg := range e.Arguments {
				l.lowerExpression(arg)
			}
			l.emit(Inst{Op: OpReadFile, Type: IRString})
			return IRString
		case "write_file":
			for _, arg := range e.Arguments {
				l.lowerExpression(arg)
			}
			l.emit(Inst{Op: OpWriteFile})
			return IRVoid
		case "append_file":
			for _, arg := range e.Arguments {
				l.lowerExpression(arg)
			}
			l.emit(Inst{Op: OpAppendFile})
			return IRVoid
		case "file_exists":
			for _, arg := range e.Arguments {
				l.lowerExpression(arg)
			}
			l.emit(Inst{Op: OpFileExists, Type: IRBool})
			return IRBool
		case "mkdir":
			for _, arg := range e.Arguments {
				l.lowerExpression(arg)
			}
			l.emit(Inst{Op: OpMkDir})
			return IRVoid
		case "remove_file":
			for _, arg := range e.Arguments {
				l.lowerExpression(arg)
			}
			l.emit(Inst{Op: OpRemoveFile})
			return IRVoid
		case "rename_file":
			for _, arg := range e.Arguments {
				l.lowerExpression(arg)
			}
			l.emit(Inst{Op: OpRenameFile})
			return IRVoid
		case "read_dir":
			for _, arg := range e.Arguments {
				l.lowerExpression(arg)
			}
			l.emit(Inst{Op: OpReadDir, Type: IRArray})
			return IRArray
		case "cwd":
			l.emit(Inst{Op: OpCwd, Type: IRString})
			return IRString
		case "args":
			l.emit(Inst{Op: OpArgs, Type: IRArray})
			return IRArray
		case "exec":
			for _, arg := range e.Arguments {
				l.lowerExpression(arg)
			}
			l.emit(Inst{Op: OpExec, Type: IRInt, Arg: Operand{Int: int64(len(e.Arguments))}})
			return IRInt
		case "exec_output":
			for _, arg := range e.Arguments {
				l.lowerExpression(arg)
			}
			l.emit(Inst{Op: OpExecOutput, Type: IRString, Arg: Operand{Int: int64(len(e.Arguments))}})
			return IRString
		case "exit":
			for _, arg := range e.Arguments {
				l.lowerExpression(arg)
			}
			l.emit(Inst{Op: OpExit})
			return IRVoid
		case "panic":
			for _, arg := range e.Arguments {
				l.lowerExpression(arg)
			}
			l.emit(Inst{Op: OpPanic})
			return IRVoid
		case "getenv":
			for _, arg := range e.Arguments {
				l.lowerExpression(arg)
			}
			l.emit(Inst{Op: OpGetEnv, Type: IRString})
			return IRString
		case "setenv":
			for _, arg := range e.Arguments {
				l.lowerExpression(arg)
			}
			l.emit(Inst{Op: OpSetEnv})
			return IRVoid
		case "tcp_listen":
			for _, arg := range e.Arguments {
				l.lowerExpression(arg)
			}
			l.emit(Inst{Op: OpTCPListen, Type: IRInt})
			return IRInt
		case "tcp_accept":
			for _, arg := range e.Arguments {
				l.lowerExpression(arg)
			}
			l.emit(Inst{Op: OpTCPAccept, Type: IRInt})
			return IRInt
		case "tcp_read":
			for _, arg := range e.Arguments {
				l.lowerExpression(arg)
			}
			l.emit(Inst{Op: OpTCPRead, Type: IRString})
			return IRString
		case "tcp_write":
			for _, arg := range e.Arguments {
				l.lowerExpression(arg)
			}
			l.emit(Inst{Op: OpTCPWrite, Type: IRInt})
			return IRInt
		case "tcp_close":
			for _, arg := range e.Arguments {
				l.lowerExpression(arg)
			}
			l.emit(Inst{Op: OpTCPClose, Type: IRBool})
			return IRBool
		}
	}

	// Named function call: push args, emit OpCall
	if ident, ok := e.Function.(*ast.Identifier); ok && l.isNamedFunction(ident.Value) {
		for _, arg := range e.Arguments {
			l.lowerExpression(arg)
		}
		argc := int64(len(e.Arguments))
		l.emit(Inst{Op: OpCall, Label: ident.Value, Arg: Operand{Int: argc}, Type: irt})
		return irt
	}

	// Method call
	if isMemberExpr {
		return l.lowerMethodCall(e.Function.(*ast.MemberExpression), e.Arguments, true)
	}

	// Function value call: push fn on stack first, then args
	l.lowerExpression(e.Function)
	for _, arg := range e.Arguments {
		l.lowerExpression(arg)
	}
	argc := int64(len(e.Arguments))
	l.emit(Inst{Op: OpCallFunc, Arg: Operand{Int: argc}, Type: irt})
	return irt
}

func (l *Lowerer) lowerMethodCall(member *ast.MemberExpression, args []ast.Expression, receiverFirst bool) IRType {
	irt := IRAny
	methodName := member.Member.Value

	if receiverFirst {
		l.lowerExpression(member.Object)
	}

	for _, arg := range args {
		l.lowerExpression(arg)
	}

	if !receiverFirst {
		l.lowerExpression(member.Object)
	}

	argc := int64(len(args) + 1)

	// Determine how to dispatch this method call.
	className, isVirtual := l.classNameForMethodCall(member)
	if isVirtual {
		// Interface dispatch: resolve the class at runtime via OpCallVirt
		l.emit(Inst{Op: OpCallVirt, Label: methodName, Arg: Operand{Int: argc}, Type: irt})
	} else if className != "" {
		// Direct class dispatch: call {ClassName}_{MethodName}
		l.emit(Inst{Op: OpCall, Label: className + "_" + methodName, Arg: Operand{Int: argc}, Type: irt})
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
		Async:   method.Async,
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
		if method.Async {
			l.emit(Inst{Op: OpMakeFuture})
		}
		l.emit(Inst{Op: OpReturnVal, Type: retType})
	}

	mod.Functions[fnName] = l.fn
}

func (l *Lowerer) lowerSpawn(expr *ast.SpawnExpression) {
	l.spawnCounter++
	fnName := fmt.Sprintf("__spawn_%d", l.spawnCounter)

	oldScope := l.scope
	oldFn := l.fn

	l.scope = newLowerScope(nil)
	l.fn = &Function{
		Name:    fnName,
		Returns: IRVoid,
		Body:    make([]Inst, 0),
	}

	for _, stmt := range expr.Body.Statements {
		l.lowerStatement(stmt)
	}

	l.emit(Inst{Op: OpReturn})

	l.mod.Functions[fnName] = l.fn

	l.scope = oldScope
	l.fn = oldFn

	l.emit(Inst{Op: OpSpawn, Label: fnName})
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

func (l *Lowerer) lowerPipeExpression(e *ast.PipeExpression) IRType {
	irt := IRAny
	if ti, ok := l.typeInfo[e]; ok {
		irt = ResolveType(ti)
	}

	// Stack convention: OpCallFunc expects [fn, arg0, arg1, ...] —
	// fn on bottom, args on top. Named functions (OpCall) expect
	// [arg0, arg1, ...] — OpCall label replaces fn.

	switch right := e.Right.(type) {
	case *ast.CallExpression:
		if ident, ok := right.Function.(*ast.Identifier); ok && l.isNamedFunction(ident.Value) {
			l.lowerExpression(e.Left)
			for _, arg := range right.Arguments {
				l.lowerExpression(arg)
			}
			argc := int64(len(right.Arguments) + 1)
			l.emit(Inst{Op: OpCall, Label: ident.Value, Arg: Operand{Int: argc}, Type: irt})
			return irt
		}
		if _, ok := right.Function.(*ast.MemberExpression); ok {
			return l.lowerMethodCall(right.Function.(*ast.MemberExpression),
				append([]ast.Expression{e.Left}, right.Arguments...), false)
		}
		l.lowerExpression(right.Function)
		l.lowerExpression(e.Left)
		for _, arg := range right.Arguments {
			l.lowerExpression(arg)
		}
		argc := int64(len(right.Arguments) + 1)
		l.emit(Inst{Op: OpCallFunc, Arg: Operand{Int: argc}, Type: irt})
		return irt

	case *ast.Identifier:
		if l.isNamedFunction(right.Value) {
			l.lowerExpression(e.Left)
			l.emit(Inst{Op: OpCall, Label: right.Value, Arg: Operand{Int: 1}, Type: irt})
			return irt
		}
		l.lowerExpression(right)
		l.lowerExpression(e.Left)
		l.emit(Inst{Op: OpCallFunc, Arg: Operand{Int: 1}, Type: irt})
		return irt

	case *ast.MemberExpression:
		return l.lowerMethodCall(right, []ast.Expression{e.Left}, false)

	default:
		l.lowerExpression(right)
		l.lowerExpression(e.Left)
		l.emit(Inst{Op: OpCallFunc, Arg: Operand{Int: 1}, Type: irt})
		return irt
	}
}

func (l *Lowerer) lowerMatch(e *ast.MatchExpression) IRType {
	endLabel := l.newLabel("match_end")

	l.lowerExpression(e.Value)
	matchIdx := l.allocSlot("__match_val", IRAny)
	l.emit(Inst{Op: OpStore, Arg: Operand{Idx: matchIdx}})

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

		l.emit(Inst{Op: OpLoad, Arg: Operand{Idx: matchIdx}})
		l.emit(Inst{Op: OpMatchResult, Label: okLabel, Arg: Operand{Str: errLabel}})

		for _, arm := range e.Arms {
			if okPat, ok := arm.Pattern.(*ast.OkExpression); ok {
				l.emit(Inst{Op: OpLabel, Label: okLabel})
				if okPat.Value != nil {
					if ident, ok := okPat.Value.(*ast.Identifier); ok {
						identIdx := l.allocSlot(ident.Value, IRAny)
						l.emit(Inst{Op: OpLoad, Arg: Operand{Idx: matchIdx}})
						l.emit(Inst{Op: OpGetField, Label: "ok"})
						l.emit(Inst{Op: OpStore, Arg: Operand{Idx: identIdx}})
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
						identIdx := l.allocSlot(ident.Value, IRAny)
						l.emit(Inst{Op: OpLoad, Arg: Operand{Idx: matchIdx}})
						l.emit(Inst{Op: OpGetField, Label: "err"})
						l.emit(Inst{Op: OpStore, Arg: Operand{Idx: identIdx}})
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

			l.emit(Inst{Op: OpLoad, Arg: Operand{Idx: matchIdx}})

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

// lookupSlot returns the local variable index for the given name by traversing
// scopes from innermost to outermost. Returns -1 if not found.
func (l *Lowerer) lookupSlot(name string) int {
	for s := l.scope; s != nil; s = s.parent {
		if idx, ok := s.slots[name]; ok {
			return idx
		}
	}
	return -1
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

// lookupSlotTypeInScope looks up the IRType of a variable in an arbitrary
// scope / function pair (used when the current l.scope/l.fn differ from the
// enclosing context, e.g. during closure capture analysis).
func lookupSlotTypeInScope(scope *lowerScope, fn *Function, name string) IRType {
	for s := scope; s != nil; s = s.parent {
		if idx, ok := s.slots[name]; ok {
			if idx >= 0 && idx < len(fn.Locals) {
				return fn.Locals[idx].Type
			}
		}
	}
	return IRAny
}

// findFreeVariables collects all identifier references in body that are not
// closure parameters and are defined in an enclosing scope.  These variables
// must be captured (bundled) when the closure is created at runtime.
func findFreeVariables(body *ast.BlockStatement, params []*ast.Parameter, enclosingScope *lowerScope) []string {
	paramSet := make(map[string]bool)
	for _, p := range params {
		paramSet[p.Name.Value] = true
	}

	var refs []string
	collectIdentifiersInBlock(body, &refs)

	seen := make(map[string]bool)
	var captures []string
	for _, name := range refs {
		if paramSet[name] || seen[name] {
			continue
		}
		// Only capture variables that exist in an enclosing scope
		if variableExistsInScope(name, enclosingScope) {
			captures = append(captures, name)
			seen[name] = true
		}
	}
	return captures
}

func variableExistsInScope(name string, scope *lowerScope) bool {
	for s := scope; s != nil; s = s.parent {
		if _, ok := s.slots[name]; ok {
			return true
		}
	}
	return false
}

// collectIdentifiersInBlock walks a BlockStatement and collects all
// Identifier Values.  It does NOT descend into nested FunctionLiteral bodies
// (those have their own scope).
func collectIdentifiersInBlock(block *ast.BlockStatement, out *[]string) {
	if block == nil {
		return
	}
	for _, stmt := range block.Statements {
		collectIdentifiersInStmt(stmt, out)
	}
}

func collectIdentifiersInStmt(stmt ast.Statement, out *[]string) {
	switch s := stmt.(type) {
	case *ast.ExpressionStatement:
		collectIdentifiersInExpr(s.Expression, out)
	case *ast.ReturnStatement:
		if s.ReturnValue != nil {
			collectIdentifiersInExpr(s.ReturnValue, out)
		}
	case *ast.LetStatement:
		if s.Value != nil {
			collectIdentifiersInExpr(s.Value, out)
		}
	case *ast.ConstStatement:
		if s.Value != nil {
			collectIdentifiersInExpr(s.Value, out)
		}
	case *ast.BlockStatement:
		collectIdentifiersInBlock(s, out)
	case *ast.ForStatement:
		if s.Init != nil {
			collectIdentifiersInStmt(s.Init, out)
		}
		if s.Condition != nil {
			collectIdentifiersInExpr(s.Condition, out)
		}
		if s.Post != nil {
			collectIdentifiersInStmt(s.Post, out)
		}
		if s.Body != nil {
			collectIdentifiersInBlock(s.Body, out)
		}
	case *ast.ForInStatement:
		if s.Iterable != nil {
			collectIdentifiersInExpr(s.Iterable, out)
		}
		if s.Body != nil {
			collectIdentifiersInBlock(s.Body, out)
		}
	case *ast.WhileStatement:
		if s.Condition != nil {
			collectIdentifiersInExpr(s.Condition, out)
		}
		if s.Body != nil {
			collectIdentifiersInBlock(s.Body, out)
		}
	case *ast.LoopStatement:
		if s.Body != nil {
			collectIdentifiersInBlock(s.Body, out)
		}
	}
}

func collectIdentifiersInExpr(expr ast.Expression, out *[]string) {
	switch e := expr.(type) {
	case *ast.Identifier:
		*out = append(*out, e.Value)
	case *ast.PrefixExpression:
		collectIdentifiersInExpr(e.Right, out)
	case *ast.InfixExpression:
		collectIdentifiersInExpr(e.Left, out)
		collectIdentifiersInExpr(e.Right, out)
	case *ast.PipeExpression:
		collectIdentifiersInExpr(e.Left, out)
		collectIdentifiersInExpr(e.Right, out)
	case *ast.CallExpression:
		collectIdentifiersInExpr(e.Function, out)
		for _, arg := range e.Arguments {
			collectIdentifiersInExpr(arg, out)
		}
	case *ast.IndexExpression:
		collectIdentifiersInExpr(e.Left, out)
		collectIdentifiersInExpr(e.Index, out)
	case *ast.MemberExpression:
		collectIdentifiersInExpr(e.Object, out)
	case *ast.AssignExpression:
		collectIdentifiersInExpr(e.Left, out)
		collectIdentifiersInExpr(e.Right, out)
	case *ast.IfExpression:
		collectIdentifiersInExpr(e.Condition, out)
		if e.Consequence != nil {
			collectIdentifiersInBlock(e.Consequence, out)
		}
		if e.Alternative != nil {
			collectIdentifiersInBlock(e.Alternative, out)
		}
	case *ast.MatchExpression:
		collectIdentifiersInExpr(e.Value, out)
		for _, arm := range e.Arms {
			if arm.Pattern != nil {
				collectIdentifiersInExpr(arm.Pattern, out)
			}
			if arm.Body != nil {
				collectIdentifiersInExpr(arm.Body, out)
			}
		}
	case *ast.ArrayLiteral:
		for _, el := range e.Elements {
			collectIdentifiersInExpr(el, out)
		}
	case *ast.MapLiteral:
		for k, v := range e.Pairs {
			collectIdentifiersInExpr(k, out)
			collectIdentifiersInExpr(v, out)
		}
	case *ast.FunctionLiteral:
		// Do NOT descend into nested FunctionLiteral bodies — those have
		// their own scope and will be handled when they are lowered.
	case *ast.CastExpression:
		collectIdentifiersInExpr(e.Value, out)
	case *ast.BorrowExpression:
		collectIdentifiersInExpr(e.Value, out)
	case *ast.DerefExpression:
		collectIdentifiersInExpr(e.Value, out)
	case *ast.TryExpression:
		collectIdentifiersInExpr(e.Value, out)
	case *ast.OkExpression:
		collectIdentifiersInExpr(e.Value, out)
	case *ast.ErrExpression:
		collectIdentifiersInExpr(e.Value, out)
	case *ast.SpawnExpression:
		if e.Body != nil {
			collectIdentifiersInBlock(e.Body, out)
		}
	case *ast.AwaitExpression:
		collectIdentifiersInExpr(e.Value, out)
	case *ast.NewExpression:
		for _, arg := range e.Arguments {
			collectIdentifiersInExpr(arg, out)
		}
	case *ast.BlockExpression:
		collectIdentifiersInBlock(e.Block, out)
	case *ast.InterpolatedString:
		for _, part := range e.Parts {
			collectIdentifiersInExpr(part, out)
		}
	case *ast.IsExpression:
		collectIdentifiersInExpr(e.Value, out)
	case *ast.SendExpression:
		collectIdentifiersInExpr(e.Channel, out)
		collectIdentifiersInExpr(e.Value, out)
	case *ast.RecvExpression:
		collectIdentifiersInExpr(e.Channel, out)
	}
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
