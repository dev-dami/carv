package ir

import (
	"fmt"
	"strconv"

	"github.com/dev-dami/carv/pkg/types"
)

type Op byte

const (
	OpNop Op = iota

	OpConstInt
	OpConstFloat
	OpConstString
	OpConstBool
	OpConstNil
	OpConstChar

	OpAlloc
	OpLoad
	OpStore
	OpLoadRef

	OpNeg
	OpNot
	OpBitNot

	OpAdd
	OpSub
	OpMul
	OpDiv
	OpMod
	OpPow
	OpEq
	OpNe
	OpLt
	OpGt
	OpLe
	OpGe
	OpAnd
	OpOr

	OpArrayLit
	OpArrayGet
	OpArraySet
	OpArrayLen
	OpMapLit
	OpMapGet
	OpMapSet
	OpMapLen
	OpMapKeys
	OpMapContains

	OpStrLen
	OpStrIndex
	OpStrCat
	OpStrContains
	OpStrRepeat

	OpCall
	OpCallVirt
	OpReturn
	OpReturnVal

	OpOk
	OpErr
	OpTry

	OpLabel
	OpJmp
	OpJmpIf

	OpMatchArmEq
	OpMatchResult

	OpNew
	OpGetField
	OpSetField

	OpBorrow
	OpDeref

	OpPrint
	OpPrintln
	OpReadLine
	OpLen
	OpContains
	OpKeys
	OpAssert
	OpToInt
	OpToFloat
	OpToString

	OpInterp
	OpCastInt
	OpCastFloat
)

func (o Op) String() string {
	switch o {
	case OpNop:
		return "nop"
	case OpConstInt:
		return "const_int"
	case OpConstFloat:
		return "const_float"
	case OpConstString:
		return "const_string"
	case OpConstBool:
		return "const_bool"
	case OpConstNil:
		return "const_nil"
	case OpConstChar:
		return "const_char"
	case OpAlloc:
		return "alloc"
	case OpLoad:
		return "load"
	case OpStore:
		return "store"
	case OpLoadRef:
		return "load_ref"
	case OpNeg:
		return "neg"
	case OpNot:
		return "not"
	case OpBitNot:
		return "bit_not"
	case OpAdd:
		return "add"
	case OpSub:
		return "sub"
	case OpMul:
		return "mul"
	case OpDiv:
		return "div"
	case OpMod:
		return "mod"
	case OpPow:
		return "pow"
	case OpEq:
		return "eq"
	case OpNe:
		return "ne"
	case OpLt:
		return "lt"
	case OpGt:
		return "gt"
	case OpLe:
		return "le"
	case OpGe:
		return "ge"
	case OpAnd:
		return "and"
	case OpOr:
		return "or"
	case OpArrayLit:
		return "array_lit"
	case OpArrayGet:
		return "array_get"
	case OpArraySet:
		return "array_set"
	case OpArrayLen:
		return "array_len"
	case OpMapLit:
		return "map_lit"
	case OpMapGet:
		return "map_get"
	case OpMapSet:
		return "map_set"
	case OpMapLen:
		return "map_len"
	case OpMapKeys:
		return "map_keys"
	case OpMapContains:
		return "map_contains"
	case OpStrLen:
		return "str_len"
	case OpStrIndex:
		return "str_index"
	case OpStrCat:
		return "str_cat"
	case OpStrContains:
		return "str_contains"
	case OpStrRepeat:
		return "str_repeat"
	case OpCall:
		return "call"
	case OpCallVirt:
		return "call_virt"
	case OpReturn:
		return "return"
	case OpReturnVal:
		return "return_val"
	case OpOk:
		return "ok"
	case OpErr:
		return "err"
	case OpTry:
		return "try"
	case OpLabel:
		return "label"
	case OpJmp:
		return "jmp"
	case OpJmpIf:
		return "jmp_if"
	case OpMatchArmEq:
		return "match_arm_eq"
	case OpMatchResult:
		return "match_result"
	case OpNew:
		return "new"
	case OpGetField:
		return "get_field"
	case OpSetField:
		return "set_field"
	case OpBorrow:
		return "borrow"
	case OpDeref:
		return "deref"
	case OpPrint:
		return "print"
	case OpPrintln:
		return "println"
	case OpReadLine:
		return "readline"
	case OpLen:
		return "len"
	case OpContains:
		return "contains"
	case OpKeys:
		return "keys"
	case OpAssert:
		return "assert"
	case OpToInt:
		return "to_int"
	case OpToFloat:
		return "to_float"
	case OpToString:
		return "to_string"
	case OpInterp:
		return "interp"
	case OpCastInt:
		return "cast_int"
	case OpCastFloat:
		return "cast_float"
	}
	return "?"
}

type IRType byte

const (
	IRAny IRType = iota
	IRInt
	IRFloat
	IRBool
	IRString
	IRChar
	IRArray
	IRMap
	IRResult
	IRClass
	IRFunc
	IRVoid
	IRNil
)

func (t IRType) String() string {
	switch t {
	case IRAny:
		return "any"
	case IRInt:
		return "int"
	case IRFloat:
		return "float"
	case IRBool:
		return "bool"
	case IRString:
		return "string"
	case IRChar:
		return "char"
	case IRArray:
		return "array"
	case IRMap:
		return "map"
	case IRResult:
		return "result"
	case IRClass:
		return "class"
	case IRFunc:
		return "fn"
	case IRVoid:
		return "void"
	case IRNil:
		return "nil"
	}
	return "?"
}

type Operand struct {
	Int   int64
	Float float64
	Str   string
	Bool  bool
	Idx   int
}

type Inst struct {
	Op    Op
	Type  IRType
	Arg   Operand
	Label string
}

type Local struct {
	Name string
	Type IRType
}

type Function struct {
	Name      string
	Params    []Local
	Returns   IRType
	Locals    []Local
	Body      []Inst
	Variadic  bool
}

type Module struct {
	Functions map[string]*Function
	Entry     string
}

func ResolveType(t types.Type) IRType {
	if t == nil {
		return IRNil
	}
	switch {
	case t.Equals(types.Int), t.Equals(types.Char):
		return IRInt
	case t.Equals(types.Float):
		return IRFloat
	case t.Equals(types.Bool):
		return IRBool
	case t.Equals(types.String):
		return IRString
	case t.Equals(types.Nil):
		return IRNil
	case t.Equals(types.Void):
		return IRVoid
	}
	if _, ok := t.(*types.ArrayType); ok {
		return IRArray
	}
	if _, ok := t.(*types.MapType); ok {
		return IRMap
	}
	if _, ok := t.(*types.ClassType); ok {
		return IRClass
	}
	if _, ok := t.(*types.FunctionType); ok {
		return IRFunc
	}
	return IRAny
}

func (m *Module) PrettyPrint() string {
	var s string
	for name, fn := range m.Functions {
		s += "--- " + name + " ---\n"
		for _, l := range fn.Locals {
			s += "  local " + l.Name + ": " + l.Type.String() + "\n"
		}
		for i, inst := range fn.Body {
			s += fmt.Sprintf("  %4d  %s\n", i, FormatInst(i, &inst))
		}
		s += "\n"
	}
	return s
}

func FormatInst(i int, inst *Inst) string {
	switch inst.Op {
	case OpLabel:
		return inst.Label + ":"
	case OpAlloc:
		return "alloc " + inst.Label
	case OpConstInt:
		return "const_int " + strconv.FormatInt(inst.Arg.Int, 10)
	case OpConstFloat:
		return "const_float " + strconv.FormatFloat(inst.Arg.Float, 'g', -1, 64)
	case OpConstString:
		return "const_string " + strconv.Quote(inst.Arg.Str)
	case OpConstBool:
		if inst.Arg.Bool {
			return "const_bool true"
		}
		return "const_bool false"
	case OpConstChar:
		return "const_char " + strconv.FormatInt(inst.Arg.Int, 10)
	case OpLoad, OpStore, OpLoadRef:
		return inst.Op.String() + " " + inst.Label
	case OpCall:
		return "call " + inst.Label + "(" + strconv.FormatInt(inst.Arg.Int, 10) + ")"
	case OpCallVirt:
		return "call_virt " + inst.Label + "(" + strconv.FormatInt(inst.Arg.Int, 10) + ")"
	case OpJmp:
		return "jmp " + inst.Label
	case OpJmpIf:
		return "jmp_if " + inst.Label
	case OpMatchArmEq:
		return "match_arm_eq " + strconv.FormatInt(inst.Arg.Int, 10) + " -> " + inst.Label
	case OpMatchResult:
		return "match_result ok=" + inst.Label + " err=" + inst.Arg.Str
	case OpNew, OpGetField, OpSetField:
		return inst.Op.String() + " " + inst.Label
	case OpArrayLit, OpMapLit, OpInterp:
		return inst.Op.String() + " " + strconv.FormatInt(inst.Arg.Int, 10)
	default:
		return inst.Op.String()
	}
}
