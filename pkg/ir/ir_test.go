package ir

import (
	"testing"

	"github.com/dev-dami/carv/pkg/types"
)

func TestResolveType(t *testing.T) {
	tests := []struct {
		name string
		typ  types.Type
		want IRType
	}{
		{"int", types.Int, IRInt},
		{"float", types.Float, IRFloat},
		{"bool", types.Bool, IRBool},
		{"string", types.String, IRString},
		{"char", types.Char, IRInt},
		{"nil", types.Nil, IRNil},
		{"void", types.Void, IRVoid},
		{"any", types.Any, IRAny},
		{"array", &types.ArrayType{Element: types.Int}, IRArray},
		{"map", &types.MapType{Key: types.String, Value: types.Int}, IRMap},
		{"class", &types.ClassType{Name: "Point", Fields: map[string]types.Type{}}, IRClass},
		{"func", &types.FunctionType{Params: []types.Type{types.Int}, Return: types.Int}, IRFunc},
		{"nil input", nil, IRNil},
		{"ref type", &types.RefType{Inner: types.String}, IRAny},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveType(tt.typ)
			if got != tt.want {
				t.Errorf("ResolveType(%v) = %v, want %v", tt.typ, got, tt.want)
			}
		})
	}
}

func TestIRTypeString(t *testing.T) {
	tests := []struct {
		irt  IRType
		want string
	}{
		{IRAny, "any"},
		{IRInt, "int"},
		{IRFloat, "float"},
		{IRBool, "bool"},
		{IRString, "string"},
		{IRChar, "char"},
		{IRArray, "array"},
		{IRMap, "map"},
		{IRResult, "result"},
		{IRClass, "class"},
		{IRFunc, "fn"},
		{IRVoid, "void"},
		{IRNil, "nil"},
		{IRType(99), "?"},
	}
	for _, tt := range tests {
		if got := tt.irt.String(); got != tt.want {
			t.Errorf("IRType(%d).String() = %q, want %q", tt.irt, got, tt.want)
		}
	}
}

func TestOpString(t *testing.T) {
	ops := []Op{
		OpNop, OpConstInt, OpConstFloat, OpConstString, OpConstBool, OpConstNil, OpConstChar,
		OpAlloc, OpLoad, OpStore, OpLoadRef,
		OpNeg, OpNot, OpBitNot,
		OpAdd, OpSub, OpMul, OpDiv, OpMod, OpPow,
		OpEq, OpNe, OpLt, OpGt, OpLe, OpGe, OpAnd, OpOr,
		OpArrayLit, OpArrayGet, OpArraySet, OpArrayLen,
		OpMapLit, OpMapGet, OpMapSet, OpMapLen, OpMapKeys, OpMapContains,
		OpStrLen, OpStrIndex, OpStrCat, OpStrContains, OpStrRepeat,
		OpCall, OpReturn, OpReturnVal,
		OpOk, OpErr, OpTry,
		OpLabel, OpJmp, OpJmpIf,
		OpMatchArmEq, OpMatchResult,
		OpNew, OpGetField, OpSetField,
		OpBorrow, OpDeref,
		OpPrint, OpPrintln, OpReadLine, OpLen, OpContains, OpKeys, OpAssert,
		OpToInt, OpToFloat, OpToString, OpInterp, OpCastInt, OpCastFloat,
	}
	for _, op := range ops {
		if s := op.String(); s == "?" || s == "unknown" || s == "" {
			t.Errorf("Op(%d).String() returned %q", op, s)
		}
	}
	// unknown op
	if Op(255).String() != "?" {
		t.Errorf("expected '?' for unknown op, got %q", Op(255).String())
	}
}

func TestPrettyPrintEmptyModule(t *testing.T) {
	m := &Module{
		Functions: map[string]*Function{},
		Entry:     "main",
	}
	s := m.PrettyPrint()
	if s != "" {
		t.Errorf("expected empty output, got %q", s)
	}
}

func TestPrettyPrintFunction(t *testing.T) {
	m := &Module{
		Functions: map[string]*Function{
			"main": {
				Name:    "main",
				Returns: IRInt,
				Locals:  []Local{{Name: "x", Type: IRInt}},
				Body: []Inst{
					{Op: OpConstInt, Arg: Operand{Int: 42}, Type: IRInt},
					{Op: OpStore, Label: "x"},
					{Op: OpLoad, Label: "x"},
					{Op: OpReturnVal},
				},
			},
		},
		Entry: "main",
	}
	s := m.PrettyPrint()
	if s == "" {
		t.Fatal("expected non-empty output")
	}
	if !contains(s, "main") {
		t.Errorf("expected 'main' in output, got: %s", s)
	}
	if !contains(s, "42") {
		t.Errorf("expected '42' in output, got: %s", s)
	}
}

func TestLowererErrors(t *testing.T) {
	l := NewLowerer(nil)
	l.error("test error %d", 1)
	errors := l.Errors()
	if len(errors) != 1 || errors[0] != "test error 1" {
		t.Errorf("unexpected errors: %v", errors)
	}
}

func TestNewLabel(t *testing.T) {
	l := NewLowerer(nil)
	l1 := l.newLabel("test")
	l2 := l.newLabel("test")
	if l1 == l2 {
		t.Errorf("expected different labels, got same: %s", l1)
	}
	if !contains(l1, "test") {
		t.Errorf("expected label to contain 'test', got: %s", l1)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsStr(s, substr)
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestFormatInst(t *testing.T) {
	tests := []struct {
		name  string
		inst  Inst
		check string
	}{
		{"label", Inst{Op: OpLabel, Label: "loop"}, "loop:"},
		{"const_int", Inst{Op: OpConstInt, Arg: Operand{Int: 42}}, "const_int 42"},
		{"const_float", Inst{Op: OpConstFloat, Arg: Operand{Float: 3.14}}, "const_float"},
		{"const_string", Inst{Op: OpConstString, Arg: Operand{Str: "hello"}}, `const_string "hello"`},
		{"const_bool_true", Inst{Op: OpConstBool, Arg: Operand{Bool: true}}, "const_bool true"},
		{"const_bool_false", Inst{Op: OpConstBool, Arg: Operand{Bool: false}}, "const_bool false"},
		{"alloc", Inst{Op: OpAlloc, Label: "x"}, "alloc x"},
		{"load", Inst{Op: OpLoad, Label: "x"}, "load x"},
		{"store", Inst{Op: OpStore, Label: "x"}, "store x"},
		{"call", Inst{Op: OpCall, Label: "add", Arg: Operand{Int: 2}}, "call add(2)"},
		{"jmp", Inst{Op: OpJmp, Label: "end"}, "jmp end"},
		{"jmp_if", Inst{Op: OpJmpIf, Label: "else"}, "jmp_if else"},
		{"match_arm", Inst{Op: OpMatchArmEq, Arg: Operand{Int: 5}, Label: "arm"}, "match_arm_eq 5 -> arm"},
		{"match_result", Inst{Op: OpMatchResult, Label: "ok_l", Arg: Operand{Str: "err_l"}}, "match_result ok=ok_l err=err_l"},
		{"add", Inst{Op: OpAdd}, "add"},
		{"return", Inst{Op: OpReturn}, "return"},
		{"nop", Inst{Op: OpNop}, "nop"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatInst(0, &tt.inst)
			if !containsStr(got, tt.check) {
				t.Errorf("FormatInst = %q, want it to contain %q", got, tt.check)
			}
		})
	}
}
