package eval

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dev-dami/carv/pkg/lexer"
	"github.com/dev-dami/carv/pkg/parser"
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
mut sum = 0;
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

func TestElseIfExpression(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{`if true { 1; } else if false { 2; } else { 3; }`, 1},
		{`if false { 1; } else if true { 2; } else { 3; }`, 2},
		{`if false { 1; } else if false { 2; } else { 3; }`, 3},
	}

	for _, tt := range tests {
		evaluated := testEval(tt.input)
		testIntegerObject(t, evaluated, tt.expected)
	}
}

func TestChainedElseIf(t *testing.T) {
	input := `
let x = 15;
if x > 20 { 1; } else if x > 10 { 2; } else if x > 5 { 3; } else { 4; }
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 2)
}

func TestImmutableLet(t *testing.T) {
	input := `let x = 5; x = 10;`
	evaluated := testEval(input)
	errObj, ok := evaluated.(*Error)
	if !ok {
		t.Fatalf("expected Error for immutable reassignment, got %T (%s)", evaluated, evaluated.Inspect())
	}
	if errObj.Message != "cannot assign to immutable variable: x" {
		t.Fatalf("unexpected error message: %s", errObj.Message)
	}
}

func TestImmutableConst(t *testing.T) {
	input := `const PI = 3.14; PI = 2.0;`
	evaluated := testEval(input)
	errObj, ok := evaluated.(*Error)
	if !ok {
		t.Fatalf("expected Error for const reassignment, got %T (%s)", evaluated, evaluated.Inspect())
	}
	if errObj.Message != "cannot assign to immutable variable: PI" {
		t.Fatalf("unexpected error message: %s", errObj.Message)
	}
}

func TestMutableVariable(t *testing.T) {
	input := `mut x = 5; x = 10; x;`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 10)
}

func TestCompoundAssignment(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"mut x = 10; x += 5; x;", 15},
		{"mut x = 10; x -= 3; x;", 7},
		{"mut x = 10; x *= 2; x;", 20},
		{"mut x = 10; x /= 2; x;", 5},
		{"mut x = 10; x %= 3; x;", 1},
		{"mut x = 7; x &= 3; x;", 3},
		{"mut x = 3; x |= 8; x;", 11},
		{"mut x = 11; x ^= 5; x;", 14},
	}

	for _, tt := range tests {
		evaluated := testEval(tt.input)
		testIntegerObject(t, evaluated, tt.expected)
	}
}

func TestForLoopWithImmutableFix(t *testing.T) {
	input := `
mut sum = 0;
for (let i = 0; i < 5; i = i + 1) {
	sum = sum + i;
}
sum;
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 10)
}

func TestForInArray(t *testing.T) {
	input := `
let arr = [10, 20, 30];
mut total = 0;
for item in arr {
	total = total + item;
}
total;
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 60)
}

func TestForInString(t *testing.T) {
	input := `
mut count = 0;
for c in "hello" {
	count = count + 1;
}
count;
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 5)
}

func TestInfiniteForWithBreak(t *testing.T) {
	input := `
mut x = 0;
for {
	x = x + 1;
	if x == 5 { break; }
}
x;
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 5)
}

func TestContinueInLoop(t *testing.T) {
	input := `
mut sum = 0;
for (let i = 0; i < 10; i = i + 1) {
	if i % 2 != 0 { continue; }
	sum = sum + i;
}
sum;
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 20)
}

func TestOkErrMatch(t *testing.T) {
	input := `
fn divide(a: int, b: int) {
	if b == 0 { return Err("zero"); }
	return Ok(a / b);
}
let r = divide(10, 2);
match r {
	Ok(x) => x,
	Err(e) => 0,
};
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 5)
}

func TestTryOperator(t *testing.T) {
	input := `
fn safeFn() {
	let v = Ok(42)?;
	return Ok(v + 1);
}
safeFn();
`
	evaluated := testEval(input)
	ok, isOk := evaluated.(*Ok)
	if !isOk {
		t.Fatalf("expected Ok, got %T (%s)", evaluated, evaluated.Inspect())
	}
	testIntegerObject(t, ok.Value, 43)
}

func TestClassBasic(t *testing.T) {
	input := `
class Counter {
	value: int = 0
	fn increment() { self.value = self.value + 1; }
	fn get() -> int { return self.value; }
}
let c = new Counter;
c.increment();
c.increment();
c.get();
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 2)
}

func TestMapLiteral(t *testing.T) {
	input := `
let m = {"a": 1, "b": 2};
m["a"];
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 1)
}

func TestMapAssignment(t *testing.T) {
	input := `
mut m = {"x": 1};
m["y"] = 2;
m["y"];
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 2)
}

func TestArrayMutation(t *testing.T) {
	input := `
mut arr = [10, 20, 30];
arr[1] = 99;
arr[1];
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 99)
}

func TestClosures(t *testing.T) {
	input := `
fn makeAdder(x: int) {
	return fn(y: int) -> int { return x + y; };
}
let add5 = makeAdder(5);
add5(3);
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 8)
}

func TestRecursion(t *testing.T) {
	input := `
fn fib(n: int) -> int {
	if n <= 1 { return n; }
	return fib(n - 1) + fib(n - 2);
}
fib(10);
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 55)
}

func TestInterpolatedString(t *testing.T) {
	input := `
let name = "World";
f"Hello, {name}!";
`
	evaluated := testEval(input)
	str, ok := evaluated.(*String)
	if !ok {
		t.Fatalf("expected String, got %T", evaluated)
	}
	if str.Value != "Hello, World!" {
		t.Errorf("expected 'Hello, World!', got %q", str.Value)
	}
}

func TestInterpolatedStringWithExpr(t *testing.T) {
	input := `f"{2 + 3}";`
	evaluated := testEval(input)
	str, ok := evaluated.(*String)
	if !ok {
		t.Fatalf("expected String, got %T", evaluated)
	}
	if str.Value != "5" {
		t.Errorf("expected '5', got %q", str.Value)
	}
}

func TestTypeConversionBuiltins(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"int(3.99);", 3},
	}

	for _, tt := range tests {
		evaluated := testEval(tt.input)
		testIntegerObject(t, evaluated, tt.expected)
	}
}

func TestFloatConversionBuiltin(t *testing.T) {
	input := `float(5);`
	evaluated := testEval(input)
	f, ok := evaluated.(*Float)
	if !ok {
		t.Fatalf("expected Float, got %T (%s)", evaluated, evaluated.Inspect())
	}
	if f.Value != 5.0 {
		t.Errorf("expected 5.0, got %g", f.Value)
	}
}

func TestStringBuiltins(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`trim("  hello  ");`, "hello"},
		{`to_upper("hello");`, "HELLO"},
		{`to_lower("HELLO");`, "hello"},
		{`replace("hello world", "world", "carv");`, "hello carv"},
		{`substr("hello", 1, 3);`, "el"},
		{`substr("hello", 2);`, "llo"},
	}

	for _, tt := range tests {
		evaluated := testEval(tt.input)
		str, ok := evaluated.(*String)
		if !ok {
			t.Errorf("expected String for %q, got %T (%s)", tt.input, evaluated, evaluated.Inspect())
			continue
		}
		if str.Value != tt.expected {
			t.Errorf("for %q: expected %q, got %q", tt.input, tt.expected, str.Value)
		}
	}
}

func TestBitwiseOperations(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"5 & 3;", 1},
		{"5 | 3;", 7},
		{"5 ^ 3;", 6},
		{"~0;", -1},
	}

	for _, tt := range tests {
		evaluated := testEval(tt.input)
		testIntegerObject(t, evaluated, tt.expected)
	}
}

func TestBuiltinStr(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`str(42);`, "42"},
		{`str(true);`, "true"},
		{`str([1, 2, 3]);`, "[1, 2, 3]"},
	}
	for _, tt := range tests {
		evaluated := testEval(tt.input)
		str, ok := evaluated.(*String)
		if !ok {
			t.Errorf("for %q: expected String, got %T (%s)", tt.input, evaluated, evaluated.Inspect())
			continue
		}
		if str.Value != tt.expected {
			t.Errorf("for %q: expected %q, got %q", tt.input, tt.expected, str.Value)
		}
	}
}

func TestBuiltinIntConversions(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"int(3.14);", 3},
		{"int(true);", 1},
		{"int(false);", 0},
	}
	for _, tt := range tests {
		evaluated := testEval(tt.input)
		testIntegerObject(t, evaluated, tt.expected)
	}
}

func TestBuiltinTypeOf(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`type_of(42);`, "INTEGER"},
		{`type_of("hello");`, "STRING"},
		{`type_of([1,2,3]);`, "ARRAY"},
		{`type_of(true);`, "BOOLEAN"},
		{`type_of(3.14);`, "FLOAT"},
	}
	for _, tt := range tests {
		evaluated := testEval(tt.input)
		str, ok := evaluated.(*String)
		if !ok {
			t.Errorf("for %q: expected String, got %T (%s)", tt.input, evaluated, evaluated.Inspect())
			continue
		}
		if str.Value != tt.expected {
			t.Errorf("for %q: expected %q, got %q", tt.input, tt.expected, str.Value)
		}
	}
}

func TestBuiltinLen(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{`len("hello");`, 5},
		{`len([1, 2, 3]);`, 3},
		{`len("");`, 0},
		{`len([]);`, 0},
	}
	for _, tt := range tests {
		evaluated := testEval(tt.input)
		testIntegerObject(t, evaluated, tt.expected)
	}
}

func TestBuiltinPush(t *testing.T) {
	input := `
let a = [1, 2];
let b = push(a, 3);
len(b);
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 3)

	input2 := `
let a = [1, 2];
let b = push(a, 3);
len(a);
`
	evaluated2 := testEval(input2)
	testIntegerObject(t, evaluated2, 2)
}

func TestBuiltinHead(t *testing.T) {
	input := `head([1, 2, 3]);`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 1)

	input2 := `head([]);`
	evaluated2 := testEval(input2)
	testNilObject(t, evaluated2)
}

func TestBuiltinTail(t *testing.T) {
	input := `
let t = tail([1, 2, 3]);
len(t);
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 2)

	input2 := `
let t = tail([1, 2, 3]);
head(t);
`
	evaluated2 := testEval(input2)
	testIntegerObject(t, evaluated2, 2)

	input3 := `len(tail([1]));`
	evaluated3 := testEval(input3)
	testIntegerObject(t, evaluated3, 0)
}

func TestBuiltinKeys(t *testing.T) {
	input := `
let m = {"a": 1, "b": 2};
let k = keys(m);
len(k);
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 2)
}

func TestBuiltinValues(t *testing.T) {
	input := `
let m = {"a": 1, "b": 2};
let v = values(m);
len(v);
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 2)
}

func TestBuiltinHasKey(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{`let m = {"a": 1}; has_key(m, "a");`, true},
		{`let m = {"a": 1}; has_key(m, "b");`, false},
	}
	for _, tt := range tests {
		evaluated := testEval(tt.input)
		testBooleanObject(t, evaluated, tt.expected)
	}
}

func TestBuiltinSet(t *testing.T) {
	input := `
let m = {"a": 1};
let m2 = set(m, "b", 2);
m2["b"];
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 2)

	input2 := `
let m = {"a": 1};
let m2 = set(m, "b", 2);
has_key(m, "b");
`
	evaluated2 := testEval(input2)
	testBooleanObject(t, evaluated2, false)
}

func TestBuiltinDelete(t *testing.T) {
	input := `
let m = {"a": 1, "b": 2};
let m2 = delete(m, "a");
has_key(m2, "a");
`
	evaluated := testEval(input)
	testBooleanObject(t, evaluated, false)

	input2 := `
let m = {"a": 1, "b": 2};
let m2 = delete(m, "a");
has_key(m, "a");
`
	evaluated2 := testEval(input2)
	testBooleanObject(t, evaluated2, true)
}

func TestBuiltinSplit(t *testing.T) {
	input := `
let parts = split("a,b,c", ",");
len(parts);
`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 3)

	input2 := `
let parts = split("a,b,c", ",");
head(parts);
`
	evaluated2 := testEval(input2)
	str, ok := evaluated2.(*String)
	if !ok {
		t.Fatalf("expected String, got %T", evaluated2)
	}
	if str.Value != "a" {
		t.Errorf("expected 'a', got %q", str.Value)
	}
}

func TestBuiltinJoin(t *testing.T) {
	input := `join(["a", "b", "c"], "-");`
	evaluated := testEval(input)
	str, ok := evaluated.(*String)
	if !ok {
		t.Fatalf("expected String, got %T (%s)", evaluated, evaluated.Inspect())
	}
	if str.Value != "a-b-c" {
		t.Errorf("expected 'a-b-c', got %q", str.Value)
	}
}

func TestBuiltinContains(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{`contains("hello", "ell");`, true},
		{`contains("hello", "xyz");`, false},
	}
	for _, tt := range tests {
		evaluated := testEval(tt.input)
		testBooleanObject(t, evaluated, tt.expected)
	}
}

func TestBuiltinStartsWith(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{`starts_with("hello", "he");`, true},
		{`starts_with("hello", "lo");`, false},
	}
	for _, tt := range tests {
		evaluated := testEval(tt.input)
		testBooleanObject(t, evaluated, tt.expected)
	}
}

func TestBuiltinEndsWith(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{`ends_with("hello", "lo");`, true},
		{`ends_with("hello", "he");`, false},
	}
	for _, tt := range tests {
		evaluated := testEval(tt.input)
		testBooleanObject(t, evaluated, tt.expected)
	}
}

func TestBuiltinReplace(t *testing.T) {
	input := `replace("hello", "l", "L");`
	evaluated := testEval(input)
	str, ok := evaluated.(*String)
	if !ok {
		t.Fatalf("expected String, got %T", evaluated)
	}
	if str.Value != "heLLo" {
		t.Errorf("expected 'heLLo', got %q", str.Value)
	}
}

func TestBuiltinIndexOf(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{`index_of("hello", "l");`, 2},
		{`index_of("hello", "x");`, -1},
	}
	for _, tt := range tests {
		evaluated := testEval(tt.input)
		testIntegerObject(t, evaluated, tt.expected)
	}
}

func TestBuiltinToUpperToLower(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`to_upper("hello");`, "HELLO"},
		{`to_lower("HELLO");`, "hello"},
	}
	for _, tt := range tests {
		evaluated := testEval(tt.input)
		str, ok := evaluated.(*String)
		if !ok {
			t.Errorf("for %q: expected String, got %T", tt.input, evaluated)
			continue
		}
		if str.Value != tt.expected {
			t.Errorf("for %q: expected %q, got %q", tt.input, tt.expected, str.Value)
		}
	}
}

func TestBuiltinOrd(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{`ord("A");`, 65},
	}
	for _, tt := range tests {
		evaluated := testEval(tt.input)
		testIntegerObject(t, evaluated, tt.expected)
	}
}

func TestBuiltinOrdChar(t *testing.T) {
	input := `ord('A');`
	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 65)
}

func TestBuiltinChr(t *testing.T) {
	input := `chr(65);`
	evaluated := testEval(input)
	ch, ok := evaluated.(*Char)
	if !ok {
		t.Fatalf("expected Char, got %T (%s)", evaluated, evaluated.Inspect())
	}
	if ch.Value != 'A' {
		t.Errorf("expected 'A', got %c", ch.Value)
	}
}

func TestBuiltinCharAt(t *testing.T) {
	input := `char_at("hello", 1);`
	evaluated := testEval(input)
	ch, ok := evaluated.(*Char)
	if !ok {
		t.Fatalf("expected Char, got %T (%s)", evaluated, evaluated.Inspect())
	}
	if ch.Value != 'e' {
		t.Errorf("expected 'e', got %c", ch.Value)
	}

	input2 := `char_at("hello", 10);`
	evaluated2 := testEval(input2)
	testNilObject(t, evaluated2)
}

func TestBuiltinSubstr(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`substr("hello", 0, 2);`, "he"},
		{`substr("hello", 2);`, "llo"},
	}
	for _, tt := range tests {
		evaluated := testEval(tt.input)
		str, ok := evaluated.(*String)
		if !ok {
			t.Errorf("for %q: expected String, got %T (%s)", tt.input, evaluated, evaluated.Inspect())
			continue
		}
		if str.Value != tt.expected {
			t.Errorf("for %q: expected %q, got %q", tt.input, tt.expected, str.Value)
		}
	}
}

func TestBuiltinPanic(t *testing.T) {
	input := `panic("something went wrong");`
	evaluated := testEval(input)
	errObj, ok := evaluated.(*Error)
	if !ok {
		t.Fatalf("expected Error, got %T (%s)", evaluated, evaluated.Inspect())
	}
	if errObj.Message != "panic: something went wrong" {
		t.Errorf("expected 'panic: something went wrong', got %q", errObj.Message)
	}
}

func TestBuiltinFileIO(t *testing.T) {
	input := `
write_file("/tmp/carv_test_builtin.txt", "hello carv");
let content = read_file("/tmp/carv_test_builtin.txt");
content;
`
	evaluated := testEval(input)
	str, ok := evaluated.(*String)
	if !ok {
		t.Fatalf("expected String, got %T (%s)", evaluated, evaluated.Inspect())
	}
	if str.Value != "hello carv" {
		t.Errorf("expected 'hello carv', got %q", str.Value)
	}

	input2 := `file_exists("/tmp/carv_test_builtin.txt");`
	evaluated2 := testEval(input2)
	testBooleanObject(t, evaluated2, true)

	input3 := `file_exists("/tmp/nonexistent_carv_test_file_xyz.txt");`
	evaluated3 := testEval(input3)
	testBooleanObject(t, evaluated3, false)
}

func TestBuiltinNetModuleNamedImport(t *testing.T) {
	input := `
require { tcp_close } from "net";
type_of(tcp_close);
`
	evaluated := testEval(input)
	str, ok := evaluated.(*String)
	if !ok {
		t.Fatalf("expected String, got %T (%s)", evaluated, evaluated.Inspect())
	}
	if str.Value != "BUILTIN" {
		t.Fatalf("expected BUILTIN, got %q", str.Value)
	}
}

func TestBuiltinNetModuleAliasMemberAccess(t *testing.T) {
	input := `
require "net" as net;
type_of(net.tcp_close);
`
	evaluated := testEval(input)
	str, ok := evaluated.(*String)
	if !ok {
		t.Fatalf("expected String, got %T (%s)", evaluated, evaluated.Inspect())
	}
	if str.Value != "BUILTIN" {
		t.Fatalf("expected BUILTIN, got %q", str.Value)
	}
}

func TestBuiltinTCPServerEcho(t *testing.T) {
	port := getFreeTCPPort(t)

	clientDone := make(chan string, 1)
	go func() {
		addr := fmt.Sprintf("127.0.0.1:%d", port)
		var conn net.Conn
		var err error
		for i := 0; i < 100; i++ {
			conn, err = net.DialTimeout("tcp", addr, 50*time.Millisecond)
			if err == nil {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		if err != nil {
			clientDone <- "dial_failed"
			return
		}
		defer conn.Close()

		_, err = conn.Write([]byte("ping"))
		if err != nil {
			clientDone <- "write_failed"
			return
		}

		buf := make([]byte, 32)
		n, err := conn.Read(buf)
		if err != nil {
			clientDone <- "read_failed"
			return
		}

		clientDone <- string(buf[:n])
	}()

	input := fmt.Sprintf(`
let listener = tcp_listen("127.0.0.1", %d);
let conn = tcp_accept(listener);
let req = tcp_read(conn, 64);
let wrote = tcp_write(conn, req);
tcp_close(conn);
tcp_close(listener);
wrote;
`, port)

	evaluated := testEval(input)
	testIntegerObject(t, evaluated, 4)

	select {
	case got := <-clientDone:
		if got != "ping" {
			t.Fatalf("expected echoed payload 'ping', got %q", got)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for tcp client response")
	}
}

func TestBuiltinPrintAndPrintlnFormatting(t *testing.T) {
	output := captureStdout(t, func() {
		builtins["print"].Fn(&String{Value: "alpha"})
		builtins["println"].Fn(&String{Value: "beta"})
	})

	if output != "alphabeta\n" {
		t.Fatalf("expected print without newline and println with newline, got %q", output)
	}
}

func TestBuiltinMissingFileOperations(t *testing.T) {
	renameFn, ok := builtins["rename_file"]
	if !ok {
		t.Fatal("rename_file builtin not registered")
	}
	removeFn, ok := builtins["remove_file"]
	if !ok {
		t.Fatal("remove_file builtin not registered")
	}
	readDirFn, ok := builtins["read_dir"]
	if !ok {
		t.Fatal("read_dir builtin not registered")
	}
	cwdFn, ok := builtins["cwd"]
	if !ok {
		t.Fatal("cwd builtin not registered")
	}

	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")

	writeRes := builtins["write_file"].Fn(
		&String{Value: src},
		&String{Value: "hello"},
	)
	testNilObject(t, writeRes)

	renameRes := renameFn.Fn(&String{Value: src}, &String{Value: dst})
	testNilObject(t, renameRes)

	oldExists := builtins["file_exists"].Fn(&String{Value: src})
	testBooleanObject(t, oldExists, false)

	newExists := builtins["file_exists"].Fn(&String{Value: dst})
	testBooleanObject(t, newExists, true)

	readDirRes := readDirFn.Fn(&String{Value: dir})
	arr, ok := readDirRes.(*Array)
	if !ok {
		t.Fatalf("expected read_dir to return array, got %T (%s)", readDirRes, readDirRes.Inspect())
	}
	if len(arr.Elements) != 1 {
		t.Fatalf("expected 1 directory entry, got %d", len(arr.Elements))
	}
	entry, ok := arr.Elements[0].(*String)
	if !ok || entry.Value != "dst.txt" {
		t.Fatalf("expected dst.txt entry, got %#v", arr.Elements[0])
	}

	removeRes := removeFn.Fn(&String{Value: dst})
	testNilObject(t, removeRes)

	newExists = builtins["file_exists"].Fn(&String{Value: dst})
	testBooleanObject(t, newExists, false)

	cwdRes := cwdFn.Fn()
	cwd, ok := cwdRes.(*String)
	if !ok || cwd.Value == "" {
		t.Fatalf("expected cwd() to return non-empty string, got %T (%s)", cwdRes, cwdRes.Inspect())
	}
}

func TestTCPReadEOFCleansConnectionHandle(t *testing.T) {
	resetTCPHandlesForTest(t)
	defer resetTCPHandlesForTest(t)

	port := getFreeTCPPort(t)
	listenerObj := builtins["tcp_listen"].Fn(&String{Value: "127.0.0.1"}, &Integer{Value: int64(port)})
	listenerHandle, ok := listenerObj.(*Integer)
	if !ok {
		t.Fatalf("expected listener handle integer, got %T (%s)", listenerObj, listenerObj.Inspect())
	}

	clientDone := make(chan struct{})
	go func() {
		conn, err := dialLocalTCP(port)
		if err == nil {
			_ = conn.Close()
		}
		close(clientDone)
	}()

	connObj := builtins["tcp_accept"].Fn(listenerHandle)
	connHandle, ok := connObj.(*Integer)
	if !ok {
		t.Fatalf("expected connection handle integer, got %T (%s)", connObj, connObj.Inspect())
	}

	readObj := builtins["tcp_read"].Fn(connHandle, &Integer{Value: 64})
	str, ok := readObj.(*String)
	if !ok {
		t.Fatalf("expected string from tcp_read, got %T (%s)", readObj, readObj.Inspect())
	}
	if str.Value != "" {
		t.Fatalf("expected empty read on EOF, got %q", str.Value)
	}

	writeObj := builtins["tcp_write"].Fn(connHandle, &String{Value: "after-eof"})
	errObj, ok := writeObj.(*Error)
	if !ok {
		t.Fatalf("expected invalid-handle error after EOF cleanup, got %T (%s)", writeObj, writeObj.Inspect())
	}
	if !strings.Contains(errObj.Message, "invalid connection handle") {
		t.Fatalf("expected invalid handle error, got %q", errObj.Message)
	}

	closeObj := builtins["tcp_close"].Fn(listenerHandle)
	testBooleanObject(t, closeObj, true)

	<-clientDone
}

func TestTCPClosingListenerCleansAcceptedConnections(t *testing.T) {
	resetTCPHandlesForTest(t)
	defer resetTCPHandlesForTest(t)

	port := getFreeTCPPort(t)
	listenerObj := builtins["tcp_listen"].Fn(&String{Value: "127.0.0.1"}, &Integer{Value: int64(port)})
	listenerHandle, ok := listenerObj.(*Integer)
	if !ok {
		t.Fatalf("expected listener handle integer, got %T (%s)", listenerObj, listenerObj.Inspect())
	}

	clientCh := make(chan net.Conn, 1)
	go func() {
		conn, err := dialLocalTCP(port)
		if err == nil {
			clientCh <- conn
			return
		}
		close(clientCh)
	}()

	connObj := builtins["tcp_accept"].Fn(listenerHandle)
	connHandle, ok := connObj.(*Integer)
	if !ok {
		t.Fatalf("expected connection handle integer, got %T (%s)", connObj, connObj.Inspect())
	}

	clientConn, ok := <-clientCh
	if !ok || clientConn == nil {
		t.Fatal("failed to establish client connection")
	}
	defer clientConn.Close()

	closeObj := builtins["tcp_close"].Fn(listenerHandle)
	testBooleanObject(t, closeObj, true)

	writeObj := builtins["tcp_write"].Fn(connHandle, &String{Value: "ping"})
	errObj, ok := writeObj.(*Error)
	if !ok {
		t.Fatalf("expected invalid-handle error after listener close cleanup, got %T (%s)", writeObj, writeObj.Inspect())
	}
	if !strings.Contains(errObj.Message, "invalid connection handle") {
		t.Fatalf("expected invalid handle error, got %q", errObj.Message)
	}
}

func TestDivisionByZero(t *testing.T) {
	input := `10 / 0;`
	evaluated := testEval(input)
	errObj, ok := evaluated.(*Error)
	if !ok {
		t.Fatalf("expected Error, got %T", evaluated)
	}
	if errObj.Message != "division by zero" {
		t.Fatalf("unexpected error: %s", errObj.Message)
	}
}

func TestModuloByZero(t *testing.T) {
	input := `10 % 0;`
	evaluated := testEval(input)
	errObj, ok := evaluated.(*Error)
	if !ok {
		t.Fatalf("expected Error, got %T", evaluated)
	}
	if errObj.Message != "modulo by zero" {
		t.Fatalf("unexpected error: %s", errObj.Message)
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

func getFreeTCPPort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to reserve tcp port: %v", err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdout pipe: %v", err)
	}
	os.Stdout = w
	defer func() {
		os.Stdout = old
	}()

	fn()

	if closeErr := w.Close(); closeErr != nil {
		t.Fatalf("failed to close pipe writer: %v", closeErr)
	}
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("failed to read stdout capture: %v", err)
	}
	return string(out)
}

func dialLocalTCP(port int) (net.Conn, error) {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	var conn net.Conn
	var err error
	for i := 0; i < 100; i++ {
		conn, err = net.DialTimeout("tcp", addr, 50*time.Millisecond)
		if err == nil {
			return conn, nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	return nil, err
}

func resetTCPHandlesForTest(t *testing.T) {
	t.Helper()

	tcpMu.Lock()
	listeners := make([]net.Listener, 0, len(tcpListeners))
	for handle, ln := range tcpListeners {
		delete(tcpListeners, handle)
		listeners = append(listeners, ln)
	}

	conns := make([]net.Conn, 0, len(tcpConns))
	for handle, conn := range tcpConns {
		delete(tcpConns, handle)
		conns = append(conns, conn)
	}
	for handle := range tcpConnOwners {
		delete(tcpConnOwners, handle)
	}
	for handle := range tcpListenerConns {
		delete(tcpListenerConns, handle)
	}
	tcpNextHandle = 1
	tcpMu.Unlock()

	for _, conn := range conns {
		_ = conn.Close()
	}
	for _, ln := range listeners {
		_ = ln.Close()
	}
}
