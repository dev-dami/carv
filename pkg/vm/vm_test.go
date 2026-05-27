package vm

import (
	"bytes"
	"strings"
	"testing"

	"github.com/dev-dami/carv/pkg/ir"
	"github.com/dev-dami/carv/pkg/lexer"
	"github.com/dev-dami/carv/pkg/parser"
	"github.com/dev-dami/carv/pkg/types"
)

func runVMOpt(t *testing.T, source string, trace bool) (string, error) {
	t.Helper()
	l := lexer.New(source)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		for _, e := range p.Errors() {
			t.Logf("parse error: %s", e)
		}
		t.Fatalf("parse failed")
	}
	checker := types.NewChecker()
	if !checker.Check(prog) {
		for _, e := range checker.Errors() {
			t.Logf("type error: %s", e)
		}
		t.Fatalf("type check failed")
	}
	lowerer := ir.NewLowerer(checker.TypeInfo())
	mod := lowerer.Lower(prog)
	v := New(mod)
	v.SetTrace(trace)
	var buf bytes.Buffer
	v.SetOutput(&buf)
	_, err := v.Run()
	return buf.String(), err
}

func runVM(t *testing.T, source string) string {
	t.Helper()
	out, err := runVMOpt(t, source, false)
	if err != nil {
		t.Fatalf("vm error: %v\nsource:\n%s\noutput: %s", err, source, out)
	}
	return out
}

func runVMErr(t *testing.T, source string) error {
	t.Helper()
	_, err := runVMOpt(t, source, false)
	return err
}

func expectExact(t *testing.T, source string, expected string) {
	t.Helper()
	out := strings.TrimSpace(runVM(t, source))
	if out != expected {
		t.Errorf("expected output %q, got %q", expected, out)
	}
}

// --- Literals ---

func TestVMIntLiteral(t *testing.T) {
	expectExact(t, `fn main() { println(0); println(-1); println(255); }`, "0\n-1\n255")
}

func TestVMFloatLiteral(t *testing.T) {
	expectExact(t, `fn main() { println(3.14); }`, "3.14")
}

func TestVMBoolLiteral(t *testing.T) {
	expectExact(t, `fn main() { println(true); println(false); }`, "true\nfalse")
}

func TestVMStringLiteral(t *testing.T) {
	expectExact(t, `fn main() { println("hello"); }`, "hello")
}

func TestVMNilLiteral(t *testing.T) {
	expectExact(t, `fn main() { println(nil); }`, "nil")
}

func TestVMCharLiteral(t *testing.T) {
	expectExact(t, `fn main() { let c = 'A'; println(c); }`, "65")
}

// --- Arithmetic ---

func TestVMIntegerArithmetic(t *testing.T) {
	expectExact(t, `fn main() { println(1 + 2); }`, "3")
	expectExact(t, `fn main() { println(10 - 3); }`, "7")
	expectExact(t, `fn main() { println(4 * 5); }`, "20")
	expectExact(t, `fn main() { println(15 / 3); }`, "5")
	expectExact(t, `fn main() { println(17 % 5); }`, "2")
	expectExact(t, `fn main() { println(-10); }`, "-10")
}

func TestVMFloatArithmetic(t *testing.T) {
	expectExact(t, `fn main() { println(1.5 + 2.5); }`, "4")
	expectExact(t, `fn main() { println(10.0 - 3.5); }`, "6.5")
	expectExact(t, `fn main() { println(3.0 * 1.5); }`, "4.5")
	expectExact(t, `fn main() { println(7.0 / 2.0); }`, "3.5")
}

func TestVMMixedArithmetic(t *testing.T) {
	expectExact(t, `fn main() { println(1 + 2.5); }`, "3.5")
	expectExact(t, `fn main() { println(3.0 * 2); }`, "6")
}

// --- Comparison ---

func TestVMComparisons(t *testing.T) {
	expectExact(t, `fn main() { println(5 == 5); }`, "true")
	expectExact(t, `fn main() { println(5 == 3); }`, "false")
	expectExact(t, `fn main() { println(5 != 3); }`, "true")
	expectExact(t, `fn main() { println(5 != 5); }`, "false")
	expectExact(t, `fn main() { println(5 < 10); }`, "true")
	expectExact(t, `fn main() { println(5 < 3); }`, "false")
	expectExact(t, `fn main() { println(5 > 3); }`, "true")
	expectExact(t, `fn main() { println(5 > 10); }`, "false")
	expectExact(t, `fn main() { println(5 <= 5); }`, "true")
	expectExact(t, `fn main() { println(5 <= 3); }`, "false")
	expectExact(t, `fn main() { println(5 >= 5); }`, "true")
	expectExact(t, `fn main() { println(5 >= 10); }`, "false")
}

func TestVMFloatComparisons(t *testing.T) {
	expectExact(t, `fn main() { println(3.0 < 4.0); }`, "true")
	expectExact(t, `fn main() { println(3.0 > 4.0); }`, "false")
	expectExact(t, `fn main() { println(3.0 == 3.0); }`, "true")
}

func TestVMStringComparison(t *testing.T) {
	expectExact(t, `fn main() { println("abc" == "abc"); }`, "true")
	expectExact(t, `fn main() { println("abc" == "xyz"); }`, "false")
}

// --- Logical ---

func TestVMLogical(t *testing.T) {
	expectExact(t, `fn main() { println(true && true); }`, "true")
	expectExact(t, `fn main() { println(true && false); }`, "false")
	expectExact(t, `fn main() { println(false || true); }`, "true")
	expectExact(t, `fn main() { println(false || false); }`, "false")
	expectExact(t, `fn main() { println(!true); }`, "false")
	expectExact(t, `fn main() { println(!false); }`, "true")
}

// --- Variables ---

func TestVMVariables(t *testing.T) {
	expectExact(t, `fn main() { let x = 10; println(x); }`, "10")
	expectExact(t, `fn main() { mut x = 5; x = x + 1; println(x); }`, "6")
}

func TestVMConst(t *testing.T) {
	expectExact(t, `fn main() { const PI = 3.14; println(PI); }`, "3.14")
}

// --- String operations ---

func TestVMStringConcat(t *testing.T) {
	expectExact(t, `fn main() { println("a" + "b"); }`, "ab")
	expectExact(t, `fn main() { println("hello " + "world"); }`, "hello world")
}

func TestVMStringLen(t *testing.T) {
	expectExact(t, `fn main() { println(len("hello")); }`, "5")
	expectExact(t, `fn main() { println(len("")); }`, "0")
}

func TestVMStringIndex(t *testing.T) {
	expectExact(t, `fn main() { println("carv"[0]); }`, "99")
	expectExact(t, `fn main() { println("carv"[3]); }`, "118")
}

func TestVMStringContains(t *testing.T) {
	expectExact(t, `fn main() { println(contains("hello", "el")); }`, "true")
	expectExact(t, `fn main() { println(contains("hello", "xyz")); }`, "false")
}

// --- Arrays ---

func TestVMArrayLiteral(t *testing.T) {
	expectExact(t, `fn main() { let a = [1, 2, 3]; println(len(a)); }`, "3")
	expectExact(t, `fn main() { let a = []; println(len(a)); }`, "0")
}

func TestVMArrayIndex(t *testing.T) {
	expectExact(t, `fn main() { let a = [10, 20, 30]; println(a[0]); println(a[2]); }`, "10\n30")
}

// --- Maps ---

func TestVMMapLiteral(t *testing.T) {
	expectExact(t, `fn main() { let m = {"a": 1, "b": 2}; println(len(m)); }`, "2")
	expectExact(t, `fn main() { let m = {}; println(len(m)); }`, "0")
}

func TestVMMapIndex(t *testing.T) {
	expectExact(t, `fn main() { let m = {"x": 42}; println(m["x"]); }`, "42")
}

func TestVMMapContains(t *testing.T) {
	expectExact(t, `fn main() { let m = {"k": 1}; println(contains(m, "k")); println(contains(m, "z")); }`, "true\nfalse")
}

func TestVMMapKeys(t *testing.T) {
	out := runVM(t, `fn main() { let m = {"a": 1, "b": 2}; let k = keys(m); println(len(k)); }`)
	if !strings.Contains(out, "2") {
		t.Errorf("expected 2 keys, got: %s", out)
	}
}

// --- Control flow ---

func TestVMIf(t *testing.T) {
	expectExact(t, `fn main() { if true { println("yes"); } }`, "yes")
	expectExact(t, `fn main() { if false { println("yes"); } }`, "")
}

func TestVMIfElse(t *testing.T) {
	expectExact(t, `fn main() { if true { println("t"); } else { println("f"); } }`, "t")
	expectExact(t, `fn main() { if false { println("t"); } else { println("f"); } }`, "f")
}

func TestVMIfElseIf(t *testing.T) {
	expectExact(t, `fn main() { let x = 5; if x > 10 { println("a"); } else if x > 3 { println("b"); } else { println("c"); } }`, "b")
}

func TestVMWhile(t *testing.T) {
	expectExact(t, `fn main() { mut i = 0; while i < 3 { println(i); i = i + 1; } }`, "0\n1\n2")
}

func TestVMWhileBreak(t *testing.T) {
	expectExact(t, `fn main() { mut i = 0; while true { if i >= 2 { break; } println(i); i = i + 1; } }`, "0\n1")
}

func TestVMWhileContinue(t *testing.T) {
	expectExact(t, `fn main() { mut i = 0; while i < 5 { i = i + 1; if i == 3 { continue; } println(i); } }`, "1\n2\n4\n5")
}

func TestVMForCstyle(t *testing.T) {
	expectExact(t, `fn main() { for (let i = 0; i < 3; i = i + 1) { println(i); } }`, "0\n1\n2")
}

func TestVMForInArray(t *testing.T) {
	expectExact(t, `fn main() { let a = [10, 20, 30]; for n in a { println(n); } }`, "10\n20\n30")
}

func TestVMForInString(t *testing.T) {
	expectExact(t, `fn main() { for ch in "ab" { println(ch); } }`, "97\n98")
}

func TestVMForInMap(t *testing.T) {
	out := runVM(t, `fn main() { let m = {"x": 1, "y": 2}; for k in m { println(k); } }`)
	if !strings.Contains(out, "x") || !strings.Contains(out, "y") {
		t.Errorf("expected x and y in output, got: %s", out)
	}
}

func TestVMLoopBreak(t *testing.T) {
	expectExact(t, `fn main() { mut i = 0; loop { if i >= 3 { break; } println(i); i = i + 1; } }`, "0\n1\n2")
}

// --- Functions ---

func TestVMFnNoArgs(t *testing.T) {
	expectExact(t, `fn greet() { println("hi"); } fn main() { greet(); }`, "hi")
}

func TestVMFnArgs(t *testing.T) {
	expectExact(t, `fn add(a: int, b: int) -> int { return a + b; } fn main() { println(add(3, 4)); }`, "7")
}

func TestVMFnNested(t *testing.T) {
	expectExact(t, `fn add(a: int, b: int) -> int { return a + b; } fn double(x: int) -> int { return x * 2; } fn main() { println(double(add(2, 3))); }`, "10")
}

func TestVMFnRecursive(t *testing.T) {
	expectExact(t, `fn fib(n: int) -> int { if n <= 1 { return n; } return fib(n - 1) + fib(n - 2); } fn main() { println(fib(10)); }`, "55")
}

func TestVMFnReturnVoid(t *testing.T) {
	expectExact(t, `fn noop() { return; } fn main() { noop(); println("ok"); }`, "ok")
}

func TestVMFnMultipleReturns(t *testing.T) {
	expectExact(t, `fn abs(n: int) -> int { if n < 0 { return -n; } return n; } fn main() { println(abs(-5)); println(abs(3)); }`, "5\n3")
}

// --- Result / Match ---

func TestVMResultOk(t *testing.T) {
	out := runVM(t, `fn main() { let r = Ok(42); match r { Ok(v) => println(v), Err(e) => println(0), }; }`)
	if !strings.Contains(out, "42") {
		t.Errorf("expected 42, got: %s", out)
	}
}

func TestVMResultErr(t *testing.T) {
	out := runVM(t, `fn main() { let r = Err("fail"); match r { Ok(v) => println(1), Err(e) => println(e), }; }`)
	if !strings.Contains(out, "fail") {
		t.Errorf("expected fail, got: %s", out)
	}
}

func TestVMTryExpression(t *testing.T) {
	expectExact(t, `fn main() { let r = Ok(10); let v = r?; println(v); }`, "10")
}

// --- Interpolation ---

func TestVMInterpolation(t *testing.T) {
	expectExact(t, `fn main() { let n = 42; println(f"n={n}"); }`, "n=42")
	expectExact(t, `fn main() { let a = 1; let b = 2; println(f"{a}+{b}={a+b}"); }`, "1+2=3")
}

// --- Multiple statements ---

func TestVMMultipleStatements(t *testing.T) {
	expectExact(t, `fn main() { println(1); println(2); println(3); }`, "1\n2\n3")
}

// --- Block scope ---

func TestVMBlockScope(t *testing.T) {
	expectExact(t, `fn main() { let x = 1; if true { let y = 2; println(y); } println(x); }`, "2\n1")
}


func TestVMNestedIf(t *testing.T) {
	expectExact(t, `fn main() { let x = 10; if x > 0 { if x > 5 { println("big"); } else { println("small"); } } }`, "big")
}

func TestVMNestedLoops(t *testing.T) {
	expectExact(t, `fn main() { mut s = 0; mut i = 0; while i < 3 { mut j = 0; while j < 3 { s = s + 1; j = j + 1; } i = i + 1; } println(s); }`, "9")
}

// --- Error cases ---

func TestVMDivisionByZero(t *testing.T) {
	err := runVMErr(t, `fn main() { println(10 / 0); }`)
	if err == nil {
		t.Error("expected error for division by zero")
	}
}

func TestVMModByZero(t *testing.T) {
	err := runVMErr(t, `fn main() { println(10 % 0); }`)
	if err == nil {
		t.Error("expected error for modulo by zero")
	}
}

func TestVMArrayOOB(t *testing.T) {
	err := runVMErr(t, `fn main() { let a = [1, 2]; println(a[5]); }`)
	if err == nil {
		t.Error("expected error for array index out of bounds")
	}
}

func TestVMStringOOB(t *testing.T) {
	err := runVMErr(t, `fn main() { let s = "hi"; println(s[10]); }`)
	if err == nil {
		t.Error("expected error for string index out of bounds")
	}
}

// --- Builtins ---

func TestVMBuiltinPrint(t *testing.T) {
	buf := runVM(t, `fn main() { print("a"); print("b"); println("c"); }`)
	if !strings.Contains(buf, "abc") {
		t.Errorf("expected 'abc' in output, got: %s", buf)
	}
}

func TestVMBuiltinContainsArray(t *testing.T) {
	expectExact(t, `fn main() { let a = [1, 2, 3]; println(contains(a, 2)); println(contains(a, 5)); }`, "true\nfalse")
}

// --- Conversions ---

func TestVMToInt(t *testing.T) {
	expectExact(t, `fn main() { println(int(3.9)); }`, "3")
}

func TestVMToFloat(t *testing.T) {
	expectExact(t, `fn main() { println(float(5)); }`, "5")
}

func TestVMToString(t *testing.T) {
	expectExact(t, `fn main() { println(string(42)); }`, "42")
}

// --- Cast ---

func TestVMCast(t *testing.T) {
	expectExact(t, `fn main() { let x = 3.14 as int; println(x); }`, "3")
	expectExact(t, `fn main() { let x = 42 as float; println(x); }`, "42")
}

// --- Stress ---

func TestVMLargeLoop(t *testing.T) {
	expectExact(t, `fn main() { mut s = 0; mut i = 0; while i < 100 { s = s + i; i = i + 1; } println(s); }`, "4950")
}

func TestVMDeepRecursion(t *testing.T) {
	expectExact(t, `fn sum(n: int) -> int { if n <= 0 { return 0; } return n + sum(n - 1); } fn main() { println(sum(100)); }`, "5050")
}

// --- Result propagation ---

func TestVMResultPropagation(t *testing.T) {
	out := runVM(t, `
fn safe_div(a: int, b: int) {
	if b == 0 {
		return Err("zero");
	}
	return Ok(a / b);
}
fn main() {
	let r = safe_div(10, 2);
	match r {
		Ok(v) => println(v),
		Err(e) => println(0),
	};
}`)
	if !strings.Contains(out, "5") {
		t.Errorf("expected 5, got: %s", out)
	}
}

func TestVMResultErrPropagation(t *testing.T) {
	out := runVM(t, `
fn safe_div(a: int, b: int) {
	if b == 0 {
		return Err("zero");
	}
	return Ok(a / b);
}
fn main() {
	let r = safe_div(1, 0);
	match r {
		Ok(v) => println(1),
		Err(e) => println(e),
	};
}`)
	if !strings.Contains(out, "zero") {
		t.Errorf("expected 'zero', got: %s", out)
	}
}

// --- valuesEqual ---

func TestValuesEqual(t *testing.T) {
	if !valuesEqual(IntValue(42), IntValue(42)) {
		t.Error("expected 42 == 42")
	}
	if valuesEqual(IntValue(42), IntValue(43)) {
		t.Error("expected 42 != 43")
	}
	if !valuesEqual(FloatValue(3.14), FloatValue(3.14)) {
		t.Error("expected 3.14 == 3.14")
	}
	if !valuesEqual(BoolValue(true), BoolValue(true)) {
		t.Error("expected true == true")
	}
	if !valuesEqual(StrValue("a"), StrValue("a")) {
		t.Error(`expected "a" == "a"`)
	}
	if valuesEqual(StrValue("a"), StrValue("b")) {
		t.Error(`expected "a" != "b"`)
	}
	if !valuesEqual(NilValue(), NilValue()) {
		t.Error("expected nil == nil")
	}
	if !valuesEqual(ArrayValue([]Value{IntValue(1)}), ArrayValue([]Value{IntValue(1)})) {
		t.Error("expected [1] == [1]")
	}
}

// --- SetOutput ---

func TestVMSetOutput(t *testing.T) {
	var buf1, buf2 bytes.Buffer
	l := lexer.New("fn main() { println(\"hello\"); }")
	p := parser.New(l)
	prog := p.ParseProgram()
	checker := types.NewChecker()
	checker.Check(prog)
	lowerer := ir.NewLowerer(checker.TypeInfo())
	mod := lowerer.Lower(prog)

	v := New(mod)
	v.SetOutput(&buf1)
	v.Run()

	v2 := New(mod)
	v2.SetOutput(&buf2)
	v2.Run()

	if buf1.String() != buf2.String() {
		t.Errorf("expected same output, got %q vs %q", buf1.String(), buf2.String())
	}
}

// --- Trace mode ---

func TestVMTrace(t *testing.T) {
	var buf bytes.Buffer
	l := lexer.New("fn main() { let x = 1; println(x); }")
	p := parser.New(l)
	prog := p.ParseProgram()
	checker := types.NewChecker()
	checker.Check(prog)
	lowerer := ir.NewLowerer(checker.TypeInfo())
	mod := lowerer.Lower(prog)

	v := New(mod)
	v.SetOutput(&buf)
	v.SetTrace(true)
	v.err = &buf
	_, err := v.Run()
	if err != nil {
		t.Fatalf("vm error: %v", err)
	}
	trace := buf.String()
	if !strings.Contains(trace, "const_int") {
		t.Errorf("expected trace to contain 'const_int', got: %s", trace)
	}
}
