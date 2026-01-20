package eval

import (
	"testing"

	"github.com/carv-lang/carv/pkg/lexer"
	"github.com/carv-lang/carv/pkg/parser"
)

func TestEvalIntegerExpression(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"5;", 5},
		{"10;", 10},
		{"-5;", -5},
		{"5 + 5 + 5 + 5 - 10;", 10},
		{"2 * 2 * 2 * 2 * 2;", 32},
		{"-50 + 100 + -50;", 0},
		{"5 * 2 + 10;", 20},
		{"20 / 2 * 5;", 50},
		{"10 % 3;", 1},
	}

	for _, tt := range tests {
		evaluated := testEval(tt.input)
		testIntegerObject(t, evaluated, tt.expected)
	}
}

func TestEvalBooleanExpression(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"true;", true},
		{"false;", false},
		{"1 < 2;", true},
		{"1 > 2;", false},
		{"1 == 1;", true},
		{"1 != 1;", false},
		{"1 == 2;", false},
		{"1 != 2;", true},
		{"true == true;", true},
		{"false == false;", true},
		{"true != false;", true},
		{"(1 < 2) == true;", true},
		{"5 <= 5;", true},
		{"5 >= 5;", true},
		{"5 <= 4;", false},
		{"5 >= 6;", false},
	}

	for _, tt := range tests {
		evaluated := testEval(tt.input)
		testBooleanObject(t, evaluated, tt.expected)
	}
}

func TestPipeExpression(t *testing.T) {
	input := `
fn double(x: int) -> int {
	return x * 2;
}
fn add(x: int, y: int) -> int {
	return x + y;
}
5 |> double;
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 10)
}

func TestPipeWithArgs(t *testing.T) {
	input := `
fn add(x: int, y: int) -> int {
	return x + y;
}
5 |> add(3);
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 8)
}

func TestPipeChain(t *testing.T) {
	input := `
fn double(x: int) -> int {
	return x * 2;
}
5 |> double |> double;
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 20)
}

func TestIfExpression(t *testing.T) {
	tests := []struct {
		input    string
		expected interface{}
	}{
		{"if true { 10; }", 10},
		{"if false { 10; }", nil},
		{"if 1 < 2 { 10; }", 10},
		{"if 1 > 2 { 10; }", nil},
		{"if 1 > 2 { 10; } else { 20; }", 20},
		{"if 1 < 2 { 10; } else { 20; }", 10},
	}

	for _, tt := range tests {
		evaluated := testEval(tt.input)
		integer, ok := tt.expected.(int)
		if ok {
			testIntegerObject(t, evaluated, int64(integer))
		} else {
			testNilObject(t, evaluated)
		}
	}
}

func TestForLoop(t *testing.T) {
	input := `
let sum = 0;
for (let i = 0; i < 5; i = i + 1) {
	sum = sum + i;
}
sum;
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 10)
}

func TestWhileLoop(t *testing.T) {
	input := `
mut x = 0;
while x < 5 {
	x = x + 1;
}
x;
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 5)
}

func TestArrayLiteral(t *testing.T) {
	input := "[1, 2 * 2, 3 + 3];"

	evaluated := testEval(input)
	result, ok := evaluated.(*Array)
	if !ok {
		t.Fatalf("expected Array, got %T", evaluated)
	}

	if len(result.Elements) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(result.Elements))
	}

	testIntegerObject(t, result.Elements[0], 1)
	testIntegerObject(t, result.Elements[1], 4)
	testIntegerObject(t, result.Elements[2], 6)
}

func TestArrayIndexExpression(t *testing.T) {
	tests := []struct {
		input    string
		expected interface{}
	}{
		{"[1, 2, 3][0];", 1},
		{"[1, 2, 3][1];", 2},
		{"[1, 2, 3][2];", 3},
		{"let arr = [1, 2, 3]; arr[0];", 1},
		{"let arr = [1, 2, 3]; arr[1] + arr[2];", 5},
	}

	for _, tt := range tests {
		evaluated := testEval(tt.input)
		integer, ok := tt.expected.(int)
		if ok {
			testIntegerObject(t, evaluated, int64(integer))
		} else {
			testNilObject(t, evaluated)
		}
	}
}

func TestStringConcatenation(t *testing.T) {
	input := `"Hello" + " " + "World!";`
	evaluated := testEval(input)

	str, ok := evaluated.(*String)
	if !ok {
		t.Fatalf("expected String, got %T", evaluated)
	}

	if str.Value != "Hello World!" {
		t.Errorf("expected 'Hello World!', got %q", str.Value)
	}
}

func testEval(input string) Object {
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	env := NewEnvironment()
	return Eval(program, env)
}

func testIntegerObject(t *testing.T, obj Object, expected int64) bool {
	result, ok := obj.(*Integer)
	if !ok {
		t.Errorf("expected Integer, got %T (%+v)", obj, obj)
		return false
	}
	if result.Value != expected {
		t.Errorf("expected %d, got %d", expected, result.Value)
		return false
	}
	return true
}

func testBooleanObject(t *testing.T, obj Object, expected bool) bool {
	result, ok := obj.(*Boolean)
	if !ok {
		t.Errorf("expected Boolean, got %T", obj)
		return false
	}
	if result.Value != expected {
		t.Errorf("expected %t, got %t", expected, result.Value)
		return false
	}
	return true
}

func testNilObject(t *testing.T, obj Object) bool {
	if obj != NIL {
		t.Errorf("expected NIL, got %T (%+v)", obj, obj)
		return false
	}
	return true
}
