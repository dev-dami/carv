package codegen

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dev-dami/carv/pkg/ast"
	"github.com/dev-dami/carv/pkg/lexer"
	"github.com/dev-dami/carv/pkg/parser"
	"github.com/dev-dami/carv/pkg/types"
)

func generateOutputFromSource(t *testing.T, input string) string {
	t.Helper()
	gen := NewCGenerator()
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	return gen.Generate(program)
}

func compileGeneratedC(t *testing.T, source string) {
	t.Helper()
	if _, err := exec.LookPath("gcc"); err != nil {
		t.Skip("gcc not found; skipping emitted C compile test")
	}

	tmpDir := t.TempDir()
	cFile := filepath.Join(tmpDir, "out.c")
	outBin := filepath.Join(tmpDir, "out")

	if err := os.WriteFile(cFile, []byte(source), 0o644); err != nil {
		t.Fatalf("failed to write generated C file: %v", err)
	}

	cmd := exec.Command("gcc", "-O2", "-o", outBin, cFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gcc failed to compile emitted C: %v\n%s", err, string(output))
	}
}

func TestGenerateEmptyProgram(t *testing.T) {
	gen := NewCGenerator()
	input := ""

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	output := gen.Generate(program)

	if !strings.Contains(output, "#include <stdio.h>") {
		t.Error("expected stdio.h include")
	}
	if !strings.Contains(output, "int main(void)") {
		t.Error("expected main function")
	}
	if !strings.Contains(output, "return 0;") {
		t.Error("expected return 0")
	}
}

func TestGenerateLetStatement(t *testing.T) {
	gen := NewCGenerator()
	input := `let x = 42;`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	output := gen.Generate(program)

	if !strings.Contains(output, "carv_int x = 42;") {
		t.Errorf("expected 'carv_int x = 42;' in output, got:\n%s", output)
	}
}

func TestGenerateConstStatement(t *testing.T) {
	gen := NewCGenerator()
	input := `const PI = 3.14;`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	output := gen.Generate(program)

	if !strings.Contains(output, "const carv_float PI = 3.14") {
		t.Errorf("expected 'const carv_float PI = 3.14' in output, got:\n%s", output)
	}
}

func TestGenerateFunction(t *testing.T) {
	gen := NewCGenerator()
	input := `
fn add(a: int, b: int) -> int {
    return a + b;
}
`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	output := gen.Generate(program)

	if !strings.Contains(output, "carv_int add(carv_int a, carv_int b)") {
		t.Errorf("expected function declaration in output, got:\n%s", output)
	}
	if !strings.Contains(output, "__carv_retval = (a + b);") {
		t.Errorf("expected return assignment in output, got:\n%s", output)
	}
	if !strings.Contains(output, "goto __carv_exit;") {
		t.Errorf("expected single-exit goto in output, got:\n%s", output)
	}
}

func TestGenerateIfStatement(t *testing.T) {
	gen := NewCGenerator()
	input := `
if true {
    let x = 1;
}
`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	output := gen.Generate(program)

	if !strings.Contains(output, "if (true)") {
		t.Errorf("expected 'if (true)' in output, got:\n%s", output)
	}
}

func TestGenerateForLoop(t *testing.T) {
	gen := NewCGenerator()
	input := `
for (let i = 0; i < 10; i = i + 1) {
    let x = i;
}
`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	output := gen.Generate(program)

	if !strings.Contains(output, "for (") {
		t.Errorf("expected 'for (' in output, got:\n%s", output)
	}
}

func TestGenerateWhileLoop(t *testing.T) {
	gen := NewCGenerator()
	input := `
while true {
    break;
}
`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	output := gen.Generate(program)

	if !strings.Contains(output, "while (true)") {
		t.Errorf("expected 'while (true)' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "break;") {
		t.Errorf("expected 'break;' in output, got:\n%s", output)
	}
}

func TestGenerateClass(t *testing.T) {
	gen := NewCGenerator()
	input := `
class Counter {
    value: int = 0
    
    fn increment() {
        self.value = self.value + 1;
    }
}
`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	output := gen.Generate(program)

	if !strings.Contains(output, "typedef struct Counter Counter;") {
		t.Errorf("expected typedef in output, got:\n%s", output)
	}
	if !strings.Contains(output, "struct Counter {") {
		t.Errorf("expected struct definition in output, got:\n%s", output)
	}
	if !strings.Contains(output, "Counter_new(void)") {
		t.Errorf("expected constructor in output, got:\n%s", output)
	}
	if !strings.Contains(output, "Counter_increment(Counter* self)") {
		t.Errorf("expected method in output, got:\n%s", output)
	}
}

func TestGenerateArray(t *testing.T) {
	gen := NewCGenerator()
	input := `let nums = [1, 2, 3];`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	output := gen.Generate(program)

	if !strings.Contains(output, "carv_int_array nums") {
		t.Errorf("expected array declaration in output, got:\n%s", output)
	}
}

func TestSafeName(t *testing.T) {
	gen := NewCGenerator()

	tests := []struct {
		input    string
		expected string
	}{
		{"foo", "foo"},
		{"int", "carv_int"},
		{"double", "carv_double"},
		{"return", "carv_return"},
		{"if", "carv_if"},
		{"myvar", "myvar"},
	}

	for _, tt := range tests {
		result := gen.safeName(tt.input)
		if result != tt.expected {
			t.Errorf("safeName(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestEscapeString(t *testing.T) {
	gen := NewCGenerator()

	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "hello"},
		{"hello\nworld", "hello\\nworld"},
		{"hello\rworld", "hello\\rworld"},
		{"tab\there", "tab\\there"},
		{`quote"here`, `quote\"here`},
	}

	for _, tt := range tests {
		result := gen.escapeString(tt.input)
		if result != tt.expected {
			t.Errorf("escapeString(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestTypeToC(t *testing.T) {
	gen := NewCGenerator()

	// nil type should return void
	result := gen.typeToC(nil)
	if result != "void" {
		t.Errorf("typeToC(nil) = %q, expected 'void'", result)
	}
}

func TestStringLiteralEmitsStructLit(t *testing.T) {
	gen := NewCGenerator()
	input := `let s = "hello";`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	output := gen.Generate(program)

	if !strings.Contains(output, `carv_string_lit("hello")`) {
		t.Errorf("expected carv_string_lit for string literal, got:\n%s", output)
	}
	if !strings.Contains(output, "carv_string s =") {
		t.Errorf("expected carv_string type for string var, got:\n%s", output)
	}
}

func TestFunctionSingleExit(t *testing.T) {
	gen := NewCGenerator()
	input := `
fn greet(name: string) -> string {
    return name;
}
`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	output := gen.Generate(program)

	if !strings.Contains(output, "__carv_exit:") {
		t.Errorf("expected __carv_exit label, got:\n%s", output)
	}
	if !strings.Contains(output, "__carv_retval") {
		t.Errorf("expected __carv_retval variable, got:\n%s", output)
	}
	if !strings.Contains(output, "goto __carv_exit;") {
		t.Errorf("expected goto __carv_exit, got:\n%s", output)
	}
}

func TestStringStructTypedef(t *testing.T) {
	gen := NewCGenerator()
	input := ""

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	output := gen.Generate(program)

	if !strings.Contains(output, "typedef struct { char* data; size_t len; bool owned; } carv_string;") {
		t.Errorf("expected carv_string struct typedef, got:\n%s", output)
	}
	if !strings.Contains(output, "carv_string_lit") {
		t.Errorf("expected carv_string_lit helper, got:\n%s", output)
	}
	if !strings.Contains(output, "carv_string_clone") {
		t.Errorf("expected carv_string_clone helper, got:\n%s", output)
	}
	if !strings.Contains(output, "carv_string_drop") {
		t.Errorf("expected carv_string_drop helper, got:\n%s", output)
	}
}

func TestCloneBuiltin(t *testing.T) {
	gen := NewCGenerator()
	input := `
fn test() {
    let s = "hello";
    let t = clone(s);
}
`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	output := gen.Generate(program)

	if !strings.Contains(output, "carv_string_clone") {
		t.Errorf("expected carv_string_clone call for clone(), got:\n%s", output)
	}
}

func TestScopeDropsEmitted(t *testing.T) {
	gen := NewCGenerator()
	input := `
fn test() {
    let s = "hello";
}
`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	output := gen.Generate(program)

	if !strings.Contains(output, "carv_string_drop(&s)") {
		t.Errorf("expected carv_string_drop for owned string at scope exit, got:\n%s", output)
	}
}

func TestZeroValue(t *testing.T) {
	gen := NewCGenerator()

	tests := []struct {
		input    string
		expected string
	}{
		{"carv_int", "0"},
		{"carv_float", "0.0"},
		{"carv_bool", "false"},
		{"carv_string", "(carv_string){NULL, 0, false}"},
		{"unknown", "0"},
	}

	for _, tt := range tests {
		result := gen.zeroValue(tt.input)
		if result != tt.expected {
			t.Errorf("zeroValue(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestBorrowExpressionEmitsAddressOf(t *testing.T) {
	gen := NewCGenerator()
	input := `let x = 1; let r = &x;`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	output := gen.Generate(program)

	if !strings.Contains(output, "(&x)") {
		t.Errorf("expected address-of expression in output, got:\n%s", output)
	}
}

func TestDerefExpressionEmitsPointerAccess(t *testing.T) {
	gen := NewCGenerator()
	input := `let x = 1; let r = &x; let y = *r;`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	output := gen.Generate(program)

	if !strings.Contains(output, "(*r)") {
		t.Errorf("expected deref expression in output, got:\n%s", output)
	}
}

func TestInterfaceVtableEmission(t *testing.T) {
	gen := NewCGenerator()
	input := `
interface Printable {
	fn to_string(&self) -> string;
}
`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	output := gen.Generate(program)

	if !strings.Contains(output, "Printable_vtable") {
		t.Errorf("expected Printable_vtable typedef, got:\n%s", output)
	}
	if !strings.Contains(output, "Printable_ref") {
		t.Errorf("expected Printable_ref typedef, got:\n%s", output)
	}
	if !strings.Contains(output, "Printable_mut_ref") {
		t.Errorf("expected Printable_mut_ref typedef, got:\n%s", output)
	}
	if !strings.Contains(output, "const void* data") {
		t.Errorf("expected const void* data in ref struct, got:\n%s", output)
	}
	if !strings.Contains(output, "carv_string (*to_string)(const void* self)") {
		t.Errorf("expected vtable method to use const void* self, got:\n%s", output)
	}
}

func TestImplWrapperAndVtableEmission(t *testing.T) {
	gen := NewCGenerator()
	input := `
class Person {
	name: string
}
interface Printable {
	fn to_string(&self) -> string;
}
impl Printable for Person {
	fn to_string(&self) -> string {
		return self.name;
	}
}
`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	output := gen.Generate(program)

	if !strings.Contains(output, "Printable__Person__to_string") {
		t.Errorf("expected wrapper function, got:\n%s", output)
	}
	if !strings.Contains(output, "Printable__Person__VT") {
		t.Errorf("expected vtable instance, got:\n%s", output)
	}
	if !strings.Contains(output, "Person_to_string(const Person* self)") {
		t.Errorf("expected impl method, got:\n%s", output)
	}
}

func TestCastExpressionEmitsFatPointer(t *testing.T) {
	gen := NewCGenerator()
	input := `
class Person {
	name: string
}
interface Printable {
	fn to_string(&self) -> string;
}
impl Printable for Person {
	fn to_string(&self) -> string {
		return self.name;
	}
}
let p = new Person;
let item = &p as &Printable;
`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	output := gen.Generate(program)

	if !strings.Contains(output, "Printable_ref") {
		t.Errorf("expected Printable_ref type in cast, got:\n%s", output)
	}
	if !strings.Contains(output, "Printable__Person__VT") {
		t.Errorf("expected vtable reference in cast, got:\n%s", output)
	}
}

func TestRefParamEmitsConstPointer(t *testing.T) {
	gen := NewCGenerator()
	input := `fn take(s: &string) {}`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	output := gen.Generate(program)

	if !strings.Contains(output, "const carv_string* s") {
		t.Errorf("expected const ref param in output, got:\n%s", output)
	}
}

func TestConstSelfMethodEmitsConstPointer(t *testing.T) {
	gen := NewCGenerator()
	input := `
class Foo {
	fn get(&self) -> int {
		return 1;
	}
}
`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	output := gen.Generate(program)

	if !strings.Contains(output, "const Foo* self") {
		t.Errorf("expected const self pointer in output, got:\n%s", output)
	}
}

func TestMutSelfMethodEmitsMutablePointer(t *testing.T) {
	gen := NewCGenerator()
	input := `
class Foo {
	value: int = 0
	fn set(&mut self, value: int) {
		self.value = value;
	}
}
`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	output := gen.Generate(program)

	if !strings.Contains(output, "Foo_set(Foo* self") {
		t.Errorf("expected mutable self pointer in output, got:\n%s", output)
	}
}

func TestVtableRefMethodHasConstVoidSelf(t *testing.T) {
	gen := NewCGenerator()
	input := `
interface Readable {
	fn read(&self) -> int;
}
`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	output := gen.Generate(program)

	if !strings.Contains(output, "const void* self") {
		t.Errorf("expected const void* self in vtable, got:\n%s", output)
	}
}

func TestVtableMutRefMethodHasVoidSelf(t *testing.T) {
	gen := NewCGenerator()
	input := `
interface Writable {
	fn write(&mut self, value: int);
}
`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	output := gen.Generate(program)

	if !strings.Contains(output, "void* self") {
		t.Errorf("expected void* self in vtable, got:\n%s", output)
	}
}

func TestImplWrapperConstCast(t *testing.T) {
	gen := NewCGenerator()
	input := `
class Person {
	name: string
}
interface Printable {
	fn to_string(&self) -> string;
}
impl Printable for Person {
	fn to_string(&self) -> string {
		return self.name;
	}
}
`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	output := gen.Generate(program)

	if !strings.Contains(output, "const Person* p = (const Person*)self") {
		t.Errorf("expected const cast in wrapper, got:\n%s", output)
	}
}

func TestMixedReceiverInterface(t *testing.T) {
	gen := NewCGenerator()
	input := `
interface Mixed {
	fn read(&self) -> int;
	fn write(&mut self, value: int);
}
class Doc {
	value: int = 0
}
impl Mixed for Doc {
	fn read(&self) -> int {
		return self.value;
	}
	fn write(&mut self, value: int) {
		self.value = value;
	}
}
`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	output := gen.Generate(program)

	if !strings.Contains(output, "const void* self") {
		t.Errorf("expected const void* self for read, got:\n%s", output)
	}
	if !strings.Contains(output, "void* self") {
		t.Errorf("expected void* self for write, got:\n%s", output)
	}
}

func TestClosureCapturingInt(t *testing.T) {
	gen := NewCGenerator()
	input := `
let x = 10;
let f = fn(y: int) -> int { return x + y; };
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	output := gen.Generate(program)

	if !strings.Contains(output, "__closure_0_env") {
		t.Errorf("expected env struct typedef, got:\n%s", output)
	}
	if !strings.Contains(output, "carv_int x;") {
		t.Errorf("expected captured int x in env struct, got:\n%s", output)
	}
	if !strings.Contains(output, "static carv_int __closure_0_fn") {
		t.Errorf("expected lambda-lifted function, got:\n%s", output)
	}
	if !strings.Contains(output, "__env->x") {
		t.Errorf("expected captured var accessed via __env->x, got:\n%s", output)
	}
	if !strings.Contains(output, "carv_arena_alloc") {
		t.Errorf("expected arena allocation for env, got:\n%s", output)
	}
}

func TestClosureCapturingString(t *testing.T) {
	gen := NewCGenerator()
	input := `
let s = "hello";
let f = fn() -> string { return s; };
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	output := gen.Generate(program)

	if !strings.Contains(output, "carv_string s;") {
		t.Errorf("expected captured string s in env struct, got:\n%s", output)
	}
	if !strings.Contains(output, "carv_string_move") {
		t.Errorf("expected string move for MoveType capture, got:\n%s", output)
	}
}

func TestClosureCall(t *testing.T) {
	gen := NewCGenerator()
	input := `
let x = 10;
let f = fn(y: int) -> int { return x + y; };
let result = f(5);
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	output := gen.Generate(program)

	if !strings.Contains(output, ".fn_ptr(") {
		t.Errorf("expected fat pointer dispatch via .fn_ptr(, got:\n%s", output)
	}
	if !strings.Contains(output, ".env") {
		t.Errorf("expected .env passed to fn_ptr, got:\n%s", output)
	}
}

func TestNonCapturingClosure(t *testing.T) {
	gen := NewCGenerator()
	input := `
let f = fn(a: int, b: int) -> int { return a + b; };
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	output := gen.Generate(program)

	if !strings.Contains(output, "__closure_0_env") {
		t.Errorf("expected env struct even for non-capturing closure, got:\n%s", output)
	}
	if !strings.Contains(output, "__closure_0_fn") {
		t.Errorf("expected lifted function, got:\n%s", output)
	}
	if !strings.Contains(output, "__closure_0") {
		t.Errorf("expected closure value type, got:\n%s", output)
	}
}

func TestClosureEnvStructFields(t *testing.T) {
	gen := NewCGenerator()
	input := `
let x = 10;
let y = 3.14;
let f = fn() -> int { return x; };
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	output := gen.Generate(program)

	if !strings.Contains(output, "typedef struct { carv_int x; } __closure_0_env;") {
		t.Errorf("expected env struct with only captured x, got:\n%s", output)
	}
}

func TestAsyncFnGeneratesFrameStruct(t *testing.T) {
	gen := NewCGenerator()
	input := `
async fn fetch_data() -> int {
	return 42;
}
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	output := gen.Generate(program)

	if !strings.Contains(output, "fetch_data_frame") {
		t.Errorf("expected frame struct typedef, got:\n%s", output)
	}
	if !strings.Contains(output, "__state") {
		t.Errorf("expected __state field in frame struct, got:\n%s", output)
	}
	if !strings.Contains(output, "__result") {
		t.Errorf("expected __result field in frame struct, got:\n%s", output)
	}
}

func TestAsyncFnGeneratesPollFunction(t *testing.T) {
	gen := NewCGenerator()
	input := `
async fn fetch_data() -> int {
	return 42;
}
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	output := gen.Generate(program)

	if !strings.Contains(output, "fetch_data_poll") {
		t.Errorf("expected poll function, got:\n%s", output)
	}
	if !strings.Contains(output, "switch (f->__state)") {
		t.Errorf("expected state machine switch, got:\n%s", output)
	}
}

func TestAsyncFnGeneratesConstructor(t *testing.T) {
	gen := NewCGenerator()
	input := `
async fn compute(x: int) -> int {
	return x * 2;
}
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	output := gen.Generate(program)

	if !strings.Contains(output, "compute_frame* compute(") {
		t.Errorf("expected constructor returning frame pointer, got:\n%s", output)
	}
	if !strings.Contains(output, "f->__state = 0") {
		t.Errorf("expected state initialization, got:\n%s", output)
	}
	if !strings.Contains(output, "f->x = x") {
		t.Errorf("expected parameter copy to frame, got:\n%s", output)
	}
}

func TestAsyncMainGeneratesEventLoop(t *testing.T) {
	gen := NewCGenerator()
	input := `
async fn carv_main() -> int {
	return 0;
}
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	output := gen.Generate(program)

	if !strings.Contains(output, "carv_loop") {
		t.Errorf("expected event loop struct, got:\n%s", output)
	}
	if !strings.Contains(output, "carv_loop_init") {
		t.Errorf("expected carv_loop_init call, got:\n%s", output)
	}
	if !strings.Contains(output, "carv_loop_run") {
		t.Errorf("expected carv_loop_run call, got:\n%s", output)
	}
}

func TestAsyncFrameTypedefEmittedBeforeUse(t *testing.T) {
	output := generateOutputFromSource(t, `
async fn fetch() -> int {
	return 1;
}
`)

	forwardIdx := strings.Index(output, "typedef struct fetch_frame fetch_frame;")
	declIdx := strings.Index(output, "fetch_frame* fetch(void);")
	defIdx := strings.Index(output, "struct fetch_frame {")
	ctorIdx := strings.Index(output, "fetch_frame* fetch(void) {")

	if forwardIdx == -1 {
		t.Fatalf("expected fetch_frame forward typedef in output, got:\n%s", output)
	}
	if declIdx == -1 {
		t.Fatalf("expected fetch declaration using fetch_frame in output, got:\n%s", output)
	}
	if defIdx == -1 || ctorIdx == -1 {
		t.Fatalf("expected frame struct definition and constructor in output, got:\n%s", output)
	}
	if forwardIdx > declIdx {
		t.Fatalf("expected fetch_frame forward typedef before declaration; forwardIdx=%d declIdx=%d", forwardIdx, declIdx)
	}
	if defIdx > ctorIdx {
		t.Fatalf("expected fetch_frame definition before constructor; defIdx=%d ctorIdx=%d", defIdx, ctorIdx)
	}
}

func TestAsyncCarvMainBootstrapUsesCarvMainSymbols(t *testing.T) {
	output := generateOutputFromSource(t, `
async fn carv_main() -> int {
	return 0;
}
`)

	if !strings.Contains(output, "carv_main_frame* mf = carv_main();") {
		t.Fatalf("expected runtime bootstrap to call carv_main, got:\n%s", output)
	}
	if !strings.Contains(output, "carv_main_poll") {
		t.Fatalf("expected runtime bootstrap to reference carv_main_poll, got:\n%s", output)
	}
}

func TestAsyncAwaitUsesFrameLocalInPoll(t *testing.T) {
	output := generateOutputFromSource(t, `
async fn fetch() -> int {
	return 1;
}

async fn carv_main() -> int {
	let x = await fetch();
	println(x);
	return 0;
}
`)

	if strings.Contains(output, "printf(\"%lld\", x)") {
		t.Fatalf("expected async poll path to avoid bare local `x`; got:\n%s", output)
	}
	if !strings.Contains(output, "printf(\"%lld\", f->x)") {
		t.Fatalf("expected async poll path to print frame local `f->x`; got:\n%s", output)
	}
}

func TestAsyncGeneratedCCompiles(t *testing.T) {
	output := generateOutputFromSource(t, `
async fn fetch() -> int {
	return 1;
}

async fn carv_main() -> int {
	let x = await fetch();
	return x;
}
`)

	compileGeneratedC(t, output)
}

func TestEventLoopNotEmittedWithoutAsync(t *testing.T) {
	gen := NewCGenerator()
	input := `
fn sync_func() -> int {
	return 42;
}
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	output := gen.Generate(program)

	if strings.Contains(output, "carv_loop_run") {
		t.Errorf("expected no event loop for sync code, got:\n%s", output)
	}
	if strings.Contains(output, "carv_task") {
		t.Errorf("expected no carv_task for sync code, got:\n%s", output)
	}
}

func TestTCPBuiltinsLowerToRuntimeCalls(t *testing.T) {
	output := generateOutputFromSource(t, `
fn main() {
	let listener = tcp_listen("127.0.0.1", 8080);
	let conn = tcp_accept(listener);
	let req = tcp_read(conn, 64);
	let wrote = tcp_write(conn, req);
	tcp_close(conn);
	tcp_close(listener);
	println(wrote);
}
`)

	if !strings.Contains(output, "carv_tcp_listen(") {
		t.Fatalf("expected tcp_listen lowering to carv_tcp_listen, got:\n%s", output)
	}
	if !strings.Contains(output, "carv_tcp_accept(") {
		t.Fatalf("expected tcp_accept lowering to carv_tcp_accept, got:\n%s", output)
	}
	if !strings.Contains(output, "carv_tcp_read(") {
		t.Fatalf("expected tcp_read lowering to carv_tcp_read, got:\n%s", output)
	}
	if !strings.Contains(output, "carv_tcp_write(") {
		t.Fatalf("expected tcp_write lowering to carv_tcp_write, got:\n%s", output)
	}
	if !strings.Contains(output, "carv_tcp_close(") {
		t.Fatalf("expected tcp_close lowering to carv_tcp_close, got:\n%s", output)
	}
}

func TestTCPBuiltinsGeneratedCCompiles(t *testing.T) {
	output := generateOutputFromSource(t, `
fn main() {
	let listener = tcp_listen("127.0.0.1", 8080);
	tcp_close(listener);
}
`)

	compileGeneratedC(t, output)
}

func TestTCPBuiltinsModuleAliasLowering(t *testing.T) {
	output := generateOutputFromSource(t, `
require "net" as net;
fn main() {
	let listener = net.tcp_listen("127.0.0.1", 8080);
	net.tcp_close(listener);
}
`)

	if !strings.Contains(output, "carv_tcp_listen(") {
		t.Fatalf("expected module alias net.tcp_listen to lower to carv_tcp_listen, got:\n%s", output)
	}
	if !strings.Contains(output, "carv_tcp_close(") {
		t.Fatalf("expected module alias net.tcp_close to lower to carv_tcp_close, got:\n%s", output)
	}
	if strings.Contains(output, "Unknown_tcp_listen(") || strings.Contains(output, "Unknown_tcp_close(") {
		t.Fatalf("expected no Unknown_* module call lowering artifacts, got:\n%s", output)
	}
}

func TestMapLiteralGeneratesMapNew(t *testing.T) {
	output := generateOutputFromSource(t, `let m = {"a": 1, "b": 2};`)

	if !strings.Contains(output, "carv_map") {
		t.Errorf("expected carv_map type, got:\n%s", output)
	}
	if !strings.Contains(output, "carv_map_new(") {
		t.Errorf("expected carv_map_new call, got:\n%s", output)
	}
	if !strings.Contains(output, "carv_map_set_int(") {
		t.Errorf("expected carv_map_set_int calls, got:\n%s", output)
	}
}

func TestMapLiteralGeneratedCCompiles(t *testing.T) {
	output := generateOutputFromSource(t, `
let scores = {"alice": 95, "bob": 87};
println(scores);
`)
	compileGeneratedC(t, output)
}

func TestResultFunctionGeneratedCCompiles(t *testing.T) {
	output := generateOutputFromSource(t, `
fn divide(a: int, b: int) {
    if b == 0 {
        return Err("division by zero");
    }
    return Ok(a / b);
}

let result = divide(10, 2);
match result {
    Ok(v) => println(v),
    Err(e) => println(e),
};
`)
	compileGeneratedC(t, output)
}

func TestResultZeroValue(t *testing.T) {
	gen := NewCGenerator()
	result := gen.zeroValue("carv_result")
	if result != "(carv_result){0}" {
		t.Errorf("zeroValue(carv_result) = %q, expected '(carv_result){0}'", result)
	}
}

func TestMapZeroValue(t *testing.T) {
	gen := NewCGenerator()
	result := gen.zeroValue("carv_map")
	if result != "carv_map_new(8)" {
		t.Errorf("zeroValue(carv_map) = %q, expected 'carv_map_new(8)'", result)
	}
}

func TestGPIOModuleLowering(t *testing.T) {
	output := generateOutputFromSource(t, `
require "gpio" as gpio;
fn main() {
	gpio.digital_write(13, true);
}
`)

	if !strings.Contains(output, "carv_digital_write(13, true)") {
		t.Fatalf("expected gpio.digital_write(13, true) to lower to carv_digital_write(13, true), got:\n%s", output)
	}
}

func TestUARTModuleLowering(t *testing.T) {
	output := generateOutputFromSource(t, `
require "uart" as uart;
fn main() {
	let h = uart.uart_init(1, 9600);
}
`)

	if !strings.Contains(output, "carv_uart_init(1, 9600)") {
		t.Fatalf("expected uart.uart_init(1, 9600) to lower to carv_uart_init(1, 9600), got:\n%s", output)
	}
}

func TestHALModulesGeneratedCCompiles(t *testing.T) {
	input := `
require "gpio" as gpio;
require "uart" as uart;
require "spi" as spi;
require "i2c" as i2c;
require "timer" as timer;
fn main() {
	gpio.pin_mode(13, 1);
	gpio.digital_write(13, true);
	let v = gpio.digital_read(13);
	let a = gpio.analog_read(0);
	gpio.analog_write(9, 128);

	let uh = uart.uart_init(1, 9600);
	let wrote = uart.uart_write(uh, "hello");
	let data = uart.uart_read(uh, 64);
	let avail = uart.uart_available(uh);

	let sh = spi.spi_init(0, 1000000);
	let resp = spi.spi_transfer(sh, "ab");
	let sw = spi.spi_write(sh, "cd");
	let sr = spi.spi_read(sh, 4);

	let ih = i2c.i2c_init(1, 80);
	let iw = i2c.i2c_write(ih, "ef");
	let ir = i2c.i2c_read(ih, 2);

	let th = timer.timer_init(0, 72);
	timer.timer_start(th);
	timer.timer_stop(th);
	let cnt = timer.timer_get_count(th);
	timer.delay_ms(100);
	timer.delay_us(50);
}
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	checker := types.NewChecker()
	if !checker.Check(program) {
		t.Fatalf("type errors: %v", checker.Errors())
	}

	gen := NewCGenerator()
	gen.SetTypeInfo(checker.TypeInfo())
	output := gen.Generate(program)

	compileGeneratedC(t, output)
}

func TestMapPrintGeneratesCarv_print_map(t *testing.T) {
	output := generateOutputFromSource(t, `
let m = {"key": 42};
println(m);
`)
	if !strings.Contains(output, "carv_print_map(") {
		t.Errorf("expected carv_print_map call for map printing, got:\n%s", output)
	}
}

// --- Tests for uncovered codegen paths ---

func TestPrefixExpressionNegation(t *testing.T) {
	output := generateOutputFromSource(t, `let x = 5; let y = -x;`)
	if !strings.Contains(output, "(-x)") {
		t.Errorf("expected negation prefix expression, got:\n%s", output)
	}
}

func TestPrefixExpressionNot(t *testing.T) {
	output := generateOutputFromSource(t, `let x = true; let y = !x;`)
	if !strings.Contains(output, "(!x)") {
		t.Errorf("expected boolean negation prefix expression, got:\n%s", output)
	}
}

func TestForInStatement(t *testing.T) {
	output := generateOutputFromSource(t, `
let arr = [10, 20, 30];
for x in arr {
	println(x);
}
`)
	if !strings.Contains(output, "__idx_") {
		t.Errorf("expected index variable for for-in loop, got:\n%s", output)
	}
	if !strings.Contains(output, ".len") {
		t.Errorf("expected .len comparison in for-in loop, got:\n%s", output)
	}
	if !strings.Contains(output, ".data[") {
		t.Errorf("expected .data[] access in for-in loop, got:\n%s", output)
	}
}

func TestBlockStatement(t *testing.T) {
	// Directly construct a BlockStatement in the AST to exercise generateBlockStatement
	gen := NewCGenerator()
	l := lexer.New(`let x = 42;`)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	// Wrap the parsed statement in a BlockStatement inside a program
	block := &ast.BlockStatement{Statements: program.Statements}
	wrappedProgram := &ast.Program{
		Statements: []ast.Statement{block},
	}
	output := gen.Generate(wrappedProgram)
	if !strings.Contains(output, "carv_int x = 42;") {
		t.Errorf("expected let inside block statement, got:\n%s", output)
	}
}

func TestMethodCallOnClass(t *testing.T) {
	output := generateOutputFromSource(t, `
class Dog {
	name: string
	fn bark(&self) -> string {
		return self.name;
	}
}
let d = new Dog;
let s = d.bark();
`)
	if !strings.Contains(output, "Dog_bark(") {
		t.Errorf("expected Dog_bark method call, got:\n%s", output)
	}
}

func TestArrayPrintInline(t *testing.T) {
	output := generateOutputFromSource(t, `println([1, 2, 3]);`)
	if !strings.Contains(output, "carv_print_int_array(") {
		t.Errorf("expected carv_print_int_array for inline array print, got:\n%s", output)
	}
}

func TestArrayPrintFloat(t *testing.T) {
	output := generateOutputFromSource(t, `println([1.1, 2.2]);`)
	if !strings.Contains(output, "carv_print_float_array(") {
		t.Errorf("expected carv_print_float_array, got:\n%s", output)
	}
}

func TestArrayPrintString(t *testing.T) {
	output := generateOutputFromSource(t, `println(["a", "b"]);`)
	if !strings.Contains(output, "carv_print_string_array(") {
		t.Errorf("expected carv_print_string_array, got:\n%s", output)
	}
}

func TestArrayPrintBool(t *testing.T) {
	output := generateOutputFromSource(t, `println([true, false]);`)
	if !strings.Contains(output, "carv_print_bool_array(") {
		t.Errorf("expected carv_print_bool_array, got:\n%s", output)
	}
}

func TestArrayPrintEmpty(t *testing.T) {
	output := generateOutputFromSource(t, `println([]);`)
	if !strings.Contains(output, "carv_print_int_array(") {
		t.Errorf("expected carv_print_int_array for empty array, got:\n%s", output)
	}
}

func TestIfExpression(t *testing.T) {
	output := generateOutputFromSource(t, `
let x = if true { 1; } else { 2; };
`)
	if !strings.Contains(output, "__if_") {
		t.Errorf("expected __if_ temp variable for if expression, got:\n%s", output)
	}
	if !strings.Contains(output, "if (true)") {
		t.Errorf("expected if (true) in preamble, got:\n%s", output)
	}
}

func TestIfExpressionWithStringType(t *testing.T) {
	output := generateOutputFromSource(t, `
let s = if true { "yes"; } else { "no"; };
`)
	if !strings.Contains(output, "carv_string __if_") {
		t.Errorf("expected carv_string type for if expression, got:\n%s", output)
	}
}

func TestIndexExpressionArray(t *testing.T) {
	output := generateOutputFromSource(t, `
let arr = [10, 20, 30];
let v = arr[1];
`)
	if !strings.Contains(output, "arr.data[1]") {
		t.Errorf("expected arr.data[1] index expression, got:\n%s", output)
	}
}

func TestIndexExpressionMap(t *testing.T) {
	output := generateOutputFromSource(t, `
let m = {"a": 42};
let v = m["a"];
`)
	if !strings.Contains(output, "carv_map_get_int(") {
		t.Errorf("expected carv_map_get_int for map index, got:\n%s", output)
	}
}

func TestInterpolatedStringSimple(t *testing.T) {
	output := generateOutputFromSource(t, `
let name = "world";
let s = f"hello {name}";
`)
	if !strings.Contains(output, "carv_concat(") {
		t.Errorf("expected carv_concat for interpolated string, got:\n%s", output)
	}
}

func TestInterpolatedStringWithInt(t *testing.T) {
	output := generateOutputFromSource(t, `
let x = 42;
let s = f"value is {x}";
`)
	if !strings.Contains(output, "carv_int_to_string(") {
		t.Errorf("expected carv_int_to_string for int interpolation, got:\n%s", output)
	}
}

func TestInterpolatedStringEmpty(t *testing.T) {
	output := generateOutputFromSource(t, `let s = f"";`)
	if !strings.Contains(output, `carv_string_lit("")`) {
		t.Errorf("expected empty carv_string_lit for empty interp string, got:\n%s", output)
	}
}

func TestInterpolatedStringSinglePart(t *testing.T) {
	output := generateOutputFromSource(t, `
let name = "world";
let s = f"{name}";
`)
	// Single part with 1 expression should go through convertToString but no concat
	if !strings.Contains(output, "carv_string") {
		t.Errorf("expected carv_string type for single-part interp string, got:\n%s", output)
	}
}

func TestTryExpressionInFunction(t *testing.T) {
	output := generateOutputFromSource(t, `
fn divide(a: int, b: int) {
	if b == 0 {
		return Err("div by zero");
	}
	return Ok(a / b);
}
fn compute() {
	let val = divide(10, 2)?;
	println(val);
}
`)
	if !strings.Contains(output, "__try_") {
		t.Errorf("expected __try_ temp for try expression, got:\n%s", output)
	}
	if !strings.Contains(output, "is_ok") {
		t.Errorf("expected is_ok check in try expression, got:\n%s", output)
	}
	if !strings.Contains(output, "goto __carv_exit") {
		t.Errorf("expected goto __carv_exit in try expression inside function, got:\n%s", output)
	}
}

func TestTryExpressionTopLevel(t *testing.T) {
	output := generateOutputFromSource(t, `
fn divide(a: int, b: int) {
	if b == 0 {
		return Err("div by zero");
	}
	return Ok(a / b);
}
let val = divide(10, 2)?;
`)
	if !strings.Contains(output, "__try_") {
		t.Errorf("expected __try_ temp for try expression, got:\n%s", output)
	}
	if !strings.Contains(output, "return") {
		t.Errorf("expected return for try expression at top level, got:\n%s", output)
	}
}

func TestReturnStatementNoValue(t *testing.T) {
	output := generateOutputFromSource(t, `
fn doNothing() {
	return;
}
`)
	if !strings.Contains(output, "goto __carv_exit;") {
		t.Errorf("expected goto __carv_exit for void return, got:\n%s", output)
	}
}

func TestAssignExpressionCompound(t *testing.T) {
	output := generateOutputFromSource(t, `
mut x = 10;
x += 5;
x -= 2;
x *= 3;
x /= 2;
`)
	if !strings.Contains(output, "x += 5") {
		t.Errorf("expected x += 5, got:\n%s", output)
	}
	if !strings.Contains(output, "x -= 2") {
		t.Errorf("expected x -= 2, got:\n%s", output)
	}
	if !strings.Contains(output, "x *= 3") {
		t.Errorf("expected x *= 3, got:\n%s", output)
	}
	if !strings.Contains(output, "x /= 2") {
		t.Errorf("expected x /= 2, got:\n%s", output)
	}
}

func TestAssignExpressionMapIndex(t *testing.T) {
	output := generateOutputFromSource(t, `
let m = {"a": 1};
m["b"] = 2;
`)
	if !strings.Contains(output, "carv_map_set_int(") {
		t.Errorf("expected carv_map_set_int for map index assignment, got:\n%s", output)
	}
}

func TestCallExpressionLen(t *testing.T) {
	output := generateOutputFromSource(t, `
let s = "hello";
let n = len(s);
`)
	if !strings.Contains(output, "(carv_int)s.len") {
		t.Errorf("expected (carv_int)s.len for string len, got:\n%s", output)
	}
}

func TestCallExpressionSplit(t *testing.T) {
	output := generateOutputFromSource(t, `
let s = "a,b,c";
let parts = split(s, ",");
`)
	if !strings.Contains(output, "carv_split(") {
		t.Errorf("expected carv_split for split call, got:\n%s", output)
	}
}

func TestCallExpressionJoin(t *testing.T) {
	output := generateOutputFromSource(t, `
let arr = ["a", "b"];
let s = join(arr, ",");
`)
	if !strings.Contains(output, "carv_join(") {
		t.Errorf("expected carv_join for join call, got:\n%s", output)
	}
}

func TestCallExpressionTrim(t *testing.T) {
	output := generateOutputFromSource(t, `
let s = " hello ";
let t = trim(s);
`)
	if !strings.Contains(output, "carv_trim(") {
		t.Errorf("expected carv_trim for trim call, got:\n%s", output)
	}
}

func TestCallExpressionSubstr(t *testing.T) {
	output := generateOutputFromSource(t, `
let s = "hello world";
let sub = substr(s, 0, 5);
`)
	if !strings.Contains(output, "carv_substr(") {
		t.Errorf("expected carv_substr for substr call, got:\n%s", output)
	}
}

func TestCallExpressionWriteFile(t *testing.T) {
	output := generateOutputFromSource(t, `
let ok = write_file("test.txt", "hello");
`)
	if !strings.Contains(output, "carv_write_file(") {
		t.Errorf("expected carv_write_file call, got:\n%s", output)
	}
}

func TestCallExpressionFileExists(t *testing.T) {
	output := generateOutputFromSource(t, `
let exists = file_exists("test.txt");
`)
	if !strings.Contains(output, "carv_file_exists(") {
		t.Errorf("expected carv_file_exists call, got:\n%s", output)
	}
}

func TestPrintCallFloat(t *testing.T) {
	output := generateOutputFromSource(t, `
let x = 3.14;
println(x);
`)
	if !strings.Contains(output, `printf("%g"`) {
		t.Errorf("expected printf with %%g for float printing, got:\n%s", output)
	}
}

func TestPrintCallBool(t *testing.T) {
	output := generateOutputFromSource(t, `
let b = true;
println(b);
`)
	if !strings.Contains(output, `"true" : "false"`) {
		t.Errorf("expected ternary for bool printing, got:\n%s", output)
	}
}

func TestPrintCallString(t *testing.T) {
	output := generateOutputFromSource(t, `
let s = "hello";
println(s);
`)
	if !strings.Contains(output, `printf("%s", s.data)`) {
		t.Errorf("expected printf with %%s and .data for string printing, got:\n%s", output)
	}
}

func TestPrintCallMultipleArgs(t *testing.T) {
	output := generateOutputFromSource(t, `
let x = 1;
let y = 2;
println(x, y);
`)
	if !strings.Contains(output, `printf(" ")`) {
		t.Errorf("expected space separator between print args, got:\n%s", output)
	}
}

func TestPrintCallNoArgs(t *testing.T) {
	output := generateOutputFromSource(t, `println();`)
	if !strings.Contains(output, `printf("\n")`) {
		t.Errorf("expected bare newline for empty println, got:\n%s", output)
	}
}

func TestPrintCallArrayVar(t *testing.T) {
	output := generateOutputFromSource(t, `
let arr = [1, 2, 3];
println(arr);
`)
	if !strings.Contains(output, "carv_print_int_array(") {
		t.Errorf("expected carv_print_int_array for array variable printing, got:\n%s", output)
	}
}

func TestPrintCallFloatArray(t *testing.T) {
	output := generateOutputFromSource(t, `
let arr = [1.1, 2.2];
println(arr);
`)
	if !strings.Contains(output, "carv_print_float_array(") {
		t.Errorf("expected carv_print_float_array, got:\n%s", output)
	}
}

func TestPrintCallStringArray(t *testing.T) {
	output := generateOutputFromSource(t, `
let arr = ["a", "b"];
println(arr);
`)
	if !strings.Contains(output, "carv_print_string_array(") {
		t.Errorf("expected carv_print_string_array, got:\n%s", output)
	}
}

func TestPrintCallBoolArray(t *testing.T) {
	output := generateOutputFromSource(t, `
let arr = [true, false];
println(arr);
`)
	if !strings.Contains(output, "carv_print_bool_array(") {
		t.Errorf("expected carv_print_bool_array, got:\n%s", output)
	}
}

func TestIfStatementWithElse(t *testing.T) {
	output := generateOutputFromSource(t, `
if true {
	let x = 1;
} else {
	let y = 2;
}
`)
	if !strings.Contains(output, "} else {") {
		t.Errorf("expected else branch in if statement, got:\n%s", output)
	}
}

func TestAsyncIfStatement(t *testing.T) {
	output := generateOutputFromSource(t, `
async fn fetch() -> int {
	return 1;
}
async fn carv_main() -> int {
	if true {
		let x = await fetch();
		println(x);
	} else {
		let y = 0;
	}
	return 0;
}
`)
	// The async if statement should be generated within the poll function
	if !strings.Contains(output, "carv_main_poll") {
		t.Errorf("expected carv_main_poll function, got:\n%s", output)
	}
}

func TestAsyncFnWithForInLocals(t *testing.T) {
	output := generateOutputFromSource(t, `
async fn process() -> int {
	let arr = [1, 2, 3];
	for x in arr {
		println(x);
	}
	return 0;
}
`)
	if !strings.Contains(output, "process_frame") {
		t.Errorf("expected process_frame struct, got:\n%s", output)
	}
}

func TestAsyncFnWithWhileLocals(t *testing.T) {
	output := generateOutputFromSource(t, `
async fn looper() -> int {
	let i = 0;
	while i < 10 {
		let x = i;
		i = i + 1;
	}
	return i;
}
`)
	if !strings.Contains(output, "looper_frame") {
		t.Errorf("expected looper_frame struct, got:\n%s", output)
	}
}

func TestAsyncFnWithIfLocals(t *testing.T) {
	output := generateOutputFromSource(t, `
async fn brancher() -> int {
	if true {
		let x = 42;
	}
	return 0;
}
`)
	if !strings.Contains(output, "brancher_frame") {
		t.Errorf("expected brancher_frame struct, got:\n%s", output)
	}
}

func TestClosureCapturingForIn(t *testing.T) {
	output := generateOutputFromSource(t, `
let x = 10;
let f = fn(a: int) -> int {
	let sum = x + a;
	return sum;
};
`)
	if !strings.Contains(output, "__closure_0_env") {
		t.Errorf("expected closure env struct, got:\n%s", output)
	}
}

func TestClosureCapturingWithIfExpr(t *testing.T) {
	output := generateOutputFromSource(t, `
let x = 10;
let f = fn() -> int {
	if x > 5 {
		return x;
	} else {
		return 0;
	}
};
`)
	if !strings.Contains(output, "__env->x") {
		t.Errorf("expected __env->x access in closure with if, got:\n%s", output)
	}
}

func TestClosureWithAssignCapture(t *testing.T) {
	output := generateOutputFromSource(t, `
let x = 10;
let y = 20;
let f = fn() -> int {
	let z = x + y;
	return z;
};
`)
	if !strings.Contains(output, "__env->x") {
		t.Errorf("expected __env->x in closure, got:\n%s", output)
	}
	if !strings.Contains(output, "__env->y") {
		t.Errorf("expected __env->y in closure, got:\n%s", output)
	}
}

func TestCheckerTypeToCStringPrimitives(t *testing.T) {
	tests := []struct {
		typ      types.Type
		expected string
	}{
		{types.Int, "carv_int"},
		{types.Float, "carv_float"},
		{types.Bool, "carv_bool"},
		{types.String, "carv_string"},
		{types.Void, "void"},
		{types.Nil, "void*"},
		{types.U8, "uint8_t"},
		{types.U16, "uint16_t"},
		{types.U32, "uint32_t"},
		{types.U64, "uint64_t"},
		{types.I8, "int8_t"},
		{types.I16, "int16_t"},
		{types.I32, "int32_t"},
		{types.I64, "int64_t"},
		{types.F32, "float"},
		{types.F64, "double"},
		{types.Usize, "size_t"},
		{types.Isize, "ptrdiff_t"},
		{types.Char, "carv_int"},
		{nil, ""},
	}
	for _, tt := range tests {
		result := checkerTypeToCString(tt.typ)
		if result != tt.expected {
			t.Errorf("checkerTypeToCString(%v) = %q, expected %q", tt.typ, result, tt.expected)
		}
	}
}

func TestCheckerTypeToCStringArray(t *testing.T) {
	intArr := &types.ArrayType{Element: types.Int}
	if r := checkerTypeToCString(intArr); r != "carv_int_array" {
		t.Errorf("expected carv_int_array, got %q", r)
	}
	floatArr := &types.ArrayType{Element: types.Float}
	if r := checkerTypeToCString(floatArr); r != "carv_float_array" {
		t.Errorf("expected carv_float_array, got %q", r)
	}
	strArr := &types.ArrayType{Element: types.String}
	if r := checkerTypeToCString(strArr); r != "carv_string_array" {
		t.Errorf("expected carv_string_array, got %q", r)
	}
	boolArr := &types.ArrayType{Element: types.Bool}
	if r := checkerTypeToCString(boolArr); r != "carv_bool_array" {
		t.Errorf("expected carv_bool_array, got %q", r)
	}
}

func TestCheckerTypeToCStringMap(t *testing.T) {
	m := &types.MapType{Key: types.String, Value: types.Int}
	if r := checkerTypeToCString(m); r != "carv_map" {
		t.Errorf("expected carv_map, got %q", r)
	}
}

func TestCheckerTypeToCStringClass(t *testing.T) {
	cls := &types.ClassType{Name: "Foo"}
	if r := checkerTypeToCString(cls); r != "Foo*" {
		t.Errorf("expected Foo*, got %q", r)
	}
}

func TestCheckerTypeToCStringRef(t *testing.T) {
	ref := &types.RefType{Inner: types.Int, Mutable: false}
	if r := checkerTypeToCString(ref); r != "const carv_int*" {
		t.Errorf("expected const carv_int*, got %q", r)
	}
	mutRef := &types.RefType{Inner: types.Int, Mutable: true}
	if r := checkerTypeToCString(mutRef); r != "carv_int*" {
		t.Errorf("expected carv_int*, got %q", r)
	}
}

func TestCheckerTypeToCStringVolatile(t *testing.T) {
	vol := &types.VolatileType{Inner: types.Int}
	if r := checkerTypeToCString(vol); r != "volatile carv_int" {
		t.Errorf("expected volatile carv_int, got %q", r)
	}
}

func TestCheckerTypeToCStringFunction(t *testing.T) {
	fn := &types.FunctionType{Params: []types.Type{types.Int}, Return: types.Int}
	if r := checkerTypeToCString(fn); r != "void*" {
		t.Errorf("expected void*, got %q", r)
	}
}

func TestCheckerTypeToCStringFuture(t *testing.T) {
	fut := &types.FutureType{Inner: types.Int}
	if r := checkerTypeToCString(fut); r != "void*" {
		t.Errorf("expected void*, got %q", r)
	}
}

func TestCheckerTypeToCStringInterfaceRef(t *testing.T) {
	iface := &types.InterfaceType{Name: "Foo"}
	ref := &types.RefType{Inner: iface, Mutable: false}
	if r := checkerTypeToCString(ref); r != "Foo_ref" {
		t.Errorf("expected Foo_ref, got %q", r)
	}
	mutRef := &types.RefType{Inner: iface, Mutable: true}
	if r := checkerTypeToCString(mutRef); r != "Foo_mut_ref" {
		t.Errorf("expected Foo_mut_ref, got %q", r)
	}
}

func TestCheckerTypeToCStringInterface(t *testing.T) {
	iface := &types.InterfaceType{Name: "Bar"}
	if r := checkerTypeToCString(iface); r != "Bar_ref" {
		t.Errorf("expected Bar_ref, got %q", r)
	}
}

func TestBuiltinModuleCallNetRead(t *testing.T) {
	output := generateOutputFromSource(t, `
require "net" as net;
fn main() {
	let conn = net.tcp_accept(5);
	let data = net.tcp_read(conn, 128);
	let wrote = net.tcp_write(conn, data);
}
`)
	if !strings.Contains(output, "carv_tcp_accept(") {
		t.Errorf("expected carv_tcp_accept, got:\n%s", output)
	}
	if !strings.Contains(output, "carv_tcp_read(") {
		t.Errorf("expected carv_tcp_read, got:\n%s", output)
	}
	if !strings.Contains(output, "carv_tcp_write(") {
		t.Errorf("expected carv_tcp_write, got:\n%s", output)
	}
}

func TestConvertToStringFloat(t *testing.T) {
	output := generateOutputFromSource(t, `
let x = 3.14;
let s = f"value: {x}";
`)
	if !strings.Contains(output, "carv_float_to_string(") {
		t.Errorf("expected carv_float_to_string for float interpolation, got:\n%s", output)
	}
}

func TestConvertToStringBool(t *testing.T) {
	output := generateOutputFromSource(t, `
let b = true;
let s = f"flag: {b}";
`)
	if !strings.Contains(output, "carv_bool_to_string(") {
		t.Errorf("expected carv_bool_to_string for bool interpolation, got:\n%s", output)
	}
}

func TestMapAssignFloat(t *testing.T) {
	output := generateOutputFromSource(t, `
let m = {"x": 1.5};
m["y"] = 2.5;
`)
	if !strings.Contains(output, "carv_map_set_float(") {
		t.Errorf("expected carv_map_set_float for float map assignment, got:\n%s", output)
	}
}

func TestMapAssignBool(t *testing.T) {
	output := generateOutputFromSource(t, `
let m = {"x": true};
m["y"] = false;
`)
	if !strings.Contains(output, "carv_map_set_bool(") {
		t.Errorf("expected carv_map_set_bool for bool map assignment, got:\n%s", output)
	}
}

func TestMapAssignString(t *testing.T) {
	output := generateOutputFromSource(t, `
let m = {"x": "hello"};
m["y"] = "world";
`)
	if !strings.Contains(output, "carv_map_set_str(") {
		t.Errorf("expected carv_map_set_str for string map assignment, got:\n%s", output)
	}
}

func TestOkExpressionFloat(t *testing.T) {
	output := generateOutputFromSource(t, `
fn compute() {
	return Ok(3.14);
}
`)
	if !strings.Contains(output, "carv_ok_float(") {
		t.Errorf("expected carv_ok_float, got:\n%s", output)
	}
}

func TestOkExpressionBool(t *testing.T) {
	output := generateOutputFromSource(t, `
fn check() {
	return Ok(true);
}
`)
	if !strings.Contains(output, "carv_ok_bool(") {
		t.Errorf("expected carv_ok_bool, got:\n%s", output)
	}
}

func TestOkExpressionString(t *testing.T) {
	output := generateOutputFromSource(t, `
fn greet() {
	return Ok("hello");
}
`)
	if !strings.Contains(output, "carv_ok_str(") {
		t.Errorf("expected carv_ok_str, got:\n%s", output)
	}
}

func TestErrExpressionInt(t *testing.T) {
	output := generateOutputFromSource(t, `
fn fail() {
	return Err(42);
}
`)
	if !strings.Contains(output, "carv_err_code(") {
		t.Errorf("expected carv_err_code for int error, got:\n%s", output)
	}
}

func TestClosureReturnVoid(t *testing.T) {
	output := generateOutputFromSource(t, `
let f = fn(x: int) { println(x); };
`)
	if !strings.Contains(output, "__closure_0") {
		t.Errorf("expected closure generation, got:\n%s", output)
	}
}

func TestStringConcatInfix(t *testing.T) {
	output := generateOutputFromSource(t, `
let a = "hello";
let b = " world";
let c = a + b;
`)
	if !strings.Contains(output, "carv_concat(") {
		t.Errorf("expected carv_concat for string + string, got:\n%s", output)
	}
}

func TestSubstrTwoArgs(t *testing.T) {
	output := generateOutputFromSource(t, `
let s = "hello";
let sub = substr(s, 2);
`)
	if !strings.Contains(output, "carv_substr(") {
		t.Errorf("expected carv_substr call, got:\n%s", output)
	}
	if !strings.Contains(output, ", -1)") {
		t.Errorf("expected default end=-1, got:\n%s", output)
	}
}

func TestReadFileCall(t *testing.T) {
	output := generateOutputFromSource(t, `let data = read_file("test.txt");`)
	if !strings.Contains(output, "carv_read_file(") {
		t.Errorf("expected carv_read_file call, got:\n%s", output)
	}
}

func TestLenArrayCall(t *testing.T) {
	output := generateOutputFromSource(t, `
let arr = [1, 2, 3];
let n = len(arr);
`)
	if !strings.Contains(output, "arr.len") {
		t.Errorf("expected arr.len for array len, got:\n%s", output)
	}
}

func TestStaticLetStatement(t *testing.T) {
	output := generateOutputFromSource(t, `
fn test() {
	static let x = 42;
}
`)
	if !strings.Contains(output, "static carv_int x = 42;") {
		t.Errorf("expected static let statement, got:\n%s", output)
	}
}

func TestStaticConstStatement(t *testing.T) {
	output := generateOutputFromSource(t, `
fn test() {
	static const X = 42;
}
`)
	if !strings.Contains(output, "static const carv_int X = 42;") {
		t.Errorf("expected static const statement, got:\n%s", output)
	}
}

func TestWalkForCapturesForStatement(t *testing.T) {
	output := generateOutputFromSource(t, `
let x = 10;
let f = fn() -> int {
	let sum = 0;
	for (let i = 0; i < x; i = i + 1) {
		sum = sum + i;
	}
	return sum;
};
`)
	if !strings.Contains(output, "__env->x") {
		t.Errorf("expected captured x via __env->x in for-loop closure, got:\n%s", output)
	}
}

func TestWalkForCapturesWhileStatement(t *testing.T) {
	output := generateOutputFromSource(t, `
let limit = 10;
let f = fn() -> int {
	let i = 0;
	while i < limit {
		i = i + 1;
	}
	return i;
};
`)
	if !strings.Contains(output, "__env->limit") {
		t.Errorf("expected captured limit in while-loop closure, got:\n%s", output)
	}
}

func TestWalkForCapturesIndexExpression(t *testing.T) {
	output := generateOutputFromSource(t, `
let arr = [1, 2, 3];
let f = fn(i: int) -> int { return arr[i]; };
`)
	if !strings.Contains(output, "__env->arr") {
		t.Errorf("expected captured arr via index in closure, got:\n%s", output)
	}
}

func TestWalkForCapturesMemberExpression(t *testing.T) {
	output := generateOutputFromSource(t, `
class Foo {
	val: int = 0
}
let obj = new Foo;
let f = fn() -> int { return obj.val; };
`)
	if !strings.Contains(output, "__env->obj") {
		t.Errorf("expected captured obj via member in closure, got:\n%s", output)
	}
}

func TestWalkForCapturesConstStatement(t *testing.T) {
	output := generateOutputFromSource(t, `
let x = 10;
let f = fn() -> int {
	const y = x;
	return y;
};
`)
	if !strings.Contains(output, "__env->x") {
		t.Errorf("expected captured x in const statement closure, got:\n%s", output)
	}
}

func TestGetArrayTypeFloat(t *testing.T) {
	output := generateOutputFromSource(t, `let arr = [1.1, 2.2, 3.3];`)
	if !strings.Contains(output, "carv_float_array") {
		t.Errorf("expected carv_float_array, got:\n%s", output)
	}
}

func TestGetArrayTypeString(t *testing.T) {
	output := generateOutputFromSource(t, `let arr = ["a", "b", "c"];`)
	if !strings.Contains(output, "carv_string_array") {
		t.Errorf("expected carv_string_array, got:\n%s", output)
	}
}

func TestGetArrayTypeBool(t *testing.T) {
	output := generateOutputFromSource(t, `let arr = [true, false];`)
	if !strings.Contains(output, "carv_bool_array") {
		t.Errorf("expected carv_bool_array, got:\n%s", output)
	}
}

func TestTypeToCBasicTypes(t *testing.T) {
	gen := NewCGenerator()
	tests := []struct {
		name     string
		expected string
	}{
		{"u8", "uint8_t"},
		{"u16", "uint16_t"},
		{"u32", "uint32_t"},
		{"u64", "uint64_t"},
		{"i8", "int8_t"},
		{"i16", "int16_t"},
		{"i32", "int32_t"},
		{"i64", "int64_t"},
		{"f32", "float"},
		{"f64", "double"},
		{"usize", "size_t"},
		{"isize", "ptrdiff_t"},
	}
	for _, tt := range tests {
		result := gen.typeToC(&ast.BasicType{Name: tt.name})
		if result != tt.expected {
			t.Errorf("typeToC(%q) = %q, expected %q", tt.name, result, tt.expected)
		}
	}
}

func TestTypeToCVolatile(t *testing.T) {
	gen := NewCGenerator()
	result := gen.typeToC(&ast.VolatileType{Inner: &ast.BasicType{Name: "int"}})
	if result != "volatile carv_int" {
		t.Errorf("expected volatile carv_int, got %q", result)
	}
}

func TestInferResultPayloadTypesErrBranch(t *testing.T) {
	output := generateOutputFromSource(t, `
fn divide(a: int, b: int) {
	if b == 0 {
		return Err("divide by zero");
	}
	return Ok(a / b);
}
let result = divide(10, 0);
match result {
	Ok(v) => println(v),
	Err(e) => println(e),
};
`)
	// The Err branch should produce carv_err_str
	if !strings.Contains(output, "carv_err_str(") {
		t.Errorf("expected carv_err_str, got:\n%s", output)
	}
}

func TestContinueStatement(t *testing.T) {
	output := generateOutputFromSource(t, `
for (let i = 0; i < 10; i = i + 1) {
	if i == 5 {
		continue;
	}
}
`)
	if !strings.Contains(output, "continue;") {
		t.Errorf("expected continue statement, got:\n%s", output)
	}
}

func TestMapLiteralFloat(t *testing.T) {
	output := generateOutputFromSource(t, `let m = {"x": 1.5, "y": 2.5};`)
	if !strings.Contains(output, "carv_map_set_float(") {
		t.Errorf("expected carv_map_set_float for float map literal, got:\n%s", output)
	}
}

func TestMapLiteralBool(t *testing.T) {
	output := generateOutputFromSource(t, `let m = {"a": true, "b": false};`)
	if !strings.Contains(output, "carv_map_set_bool(") {
		t.Errorf("expected carv_map_set_bool for bool map literal, got:\n%s", output)
	}
}

func TestMapLiteralString(t *testing.T) {
	output := generateOutputFromSource(t, `let m = {"a": "hello"};`)
	if !strings.Contains(output, "carv_map_set_str(") {
		t.Errorf("expected carv_map_set_str for string map literal, got:\n%s", output)
	}
}

func TestIndexExpressionMapWithTypeInfo(t *testing.T) {
	input := `
let m = {"a": 1.5};
let v = m["a"];
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	checker := types.NewChecker()
	checker.Check(program)
	gen := NewCGenerator()
	gen.SetTypeInfo(checker.TypeInfo())
	output := gen.Generate(program)

	if !strings.Contains(output, "carv_map_get_") {
		t.Errorf("expected carv_map_get_ for typed map index, got:\n%s", output)
	}
}

func TestMethodCallClone(t *testing.T) {
	output := generateOutputFromSource(t, `
let s = "hello";
let c = s.clone();
`)
	if !strings.Contains(output, "carv_string_clone(") {
		t.Errorf("expected carv_string_clone for .clone() method, got:\n%s", output)
	}
}

func TestNilLiteral(t *testing.T) {
	output := generateOutputFromSource(t, `let x = nil;`)
	if !strings.Contains(output, "NULL") {
		t.Errorf("expected NULL for nil literal, got:\n%s", output)
	}
}

func TestAsyncFnWithBlockAndForStatement(t *testing.T) {
	output := generateOutputFromSource(t, `
async fn compute() -> int {
	for (let i = 0; i < 5; i = i + 1) {
		let tmp = i * 2;
	}
	return 0;
}
`)
	if !strings.Contains(output, "compute_frame") {
		t.Errorf("expected compute_frame struct, got:\n%s", output)
	}
}

func TestWalkForCapturesPrefixExpression(t *testing.T) {
	output := generateOutputFromSource(t, `
let flag = true;
let f = fn() -> bool { return !flag; };
`)
	if !strings.Contains(output, "__env->flag") {
		t.Errorf("expected captured flag via prefix expression, got:\n%s", output)
	}
}

func TestWalkForCapturesBorrowDeref(t *testing.T) {
	output := generateOutputFromSource(t, `
let x = 42;
let f = fn() -> int {
	let r = &x;
	return *r;
};
`)
	if !strings.Contains(output, "__env->x") {
		t.Errorf("expected captured x via borrow/deref, got:\n%s", output)
	}
}

func TestWalkForCapturesCallExpression(t *testing.T) {
	output := generateOutputFromSource(t, `
let x = 10;
fn double(n: int) -> int { return n * 2; }
let f = fn() -> int { return double(x); };
`)
	if !strings.Contains(output, "__env->x") {
		t.Errorf("expected captured x via call expression arg, got:\n%s", output)
	}
}

func TestGenerateFunctionKeywordAlias(t *testing.T) {
	output := generateOutputFromSource(t, `
function add(a: int, b: int) -> int {
	return a + b;
}
`)
	if !strings.Contains(output, "add(") {
		t.Errorf("expected C function 'add' in output, got:\n%s", output)
	}
}

func TestGenerateUnsafeFunction(t *testing.T) {
	output := generateOutputFromSource(t, `
unsafe fn enable_interrupts() {
	asm("sti");
}
`)
	if !strings.Contains(output, "enable_interrupts") {
		t.Errorf("expected 'enable_interrupts' in output, got:\n%s", output)
	}
	if !strings.Contains(output, `__asm__ volatile("sti")`) {
		t.Errorf("expected inline asm __asm__ volatile(\"sti\") in output, got:\n%s", output)
	}
}

func TestGenerateUnsafeBlock(t *testing.T) {
	output := generateOutputFromSource(t, `
fn main_test() {
	unsafe {
		asm("nop");
	}
}
`)
	if !strings.Contains(output, "/* unsafe */") {
		t.Errorf("expected '/* unsafe */' comment in output, got:\n%s", output)
	}
	if !strings.Contains(output, `__asm__ volatile("nop")`) {
		t.Errorf("expected inline asm __asm__ volatile(\"nop\") in output, got:\n%s", output)
	}
}

// File I/O builtin tests

func TestAppendFileCall(t *testing.T) {
	output := generateOutputFromSource(t, `let ok = append_file("test.txt", "world");`)
	if !strings.Contains(output, "carv_append_file(") {
		t.Errorf("expected carv_append_file call, got:\n%s", output)
	}
}

func TestDeleteFileCall(t *testing.T) {
	output := generateOutputFromSource(t, `let ok = delete_file("test.txt");`)
	if !strings.Contains(output, "carv_delete_file(") {
		t.Errorf("expected carv_delete_file call, got:\n%s", output)
	}
}

func TestListDirCall(t *testing.T) {
	output := generateOutputFromSource(t, `let entries = list_dir(".");`)
	if !strings.Contains(output, "carv_list_dir(") {
		t.Errorf("expected carv_list_dir call, got:\n%s", output)
	}
}

func TestRuntimeIncludesDirent(t *testing.T) {
	output := generateOutputFromSource(t, ``)
	if !strings.Contains(output, "#include <dirent.h>") {
		t.Errorf("expected dirent.h include in runtime, got:\n%s", output)
	}
}

func TestFileIOModuleReadFile(t *testing.T) {
	output := generateOutputFromSource(t, `
require "fs" as fs;
fn main() {
	let data = fs.read_file("test.txt");
}
`)
	if !strings.Contains(output, "carv_read_file(") {
		t.Errorf("expected carv_read_file via fs module, got:\n%s", output)
	}
}

func TestFileIOModuleWriteFile(t *testing.T) {
	output := generateOutputFromSource(t, `
require "fs" as fs;
fn main() {
	let ok = fs.write_file("out.txt", "hello");
}
`)
	if !strings.Contains(output, "carv_write_file(") {
		t.Errorf("expected carv_write_file via fs module, got:\n%s", output)
	}
}

func TestFileIOModuleAppendFile(t *testing.T) {
	output := generateOutputFromSource(t, `
require "fs" as fs;
fn main() {
	let ok = fs.append_file("out.txt", "more");
}
`)
	if !strings.Contains(output, "carv_append_file(") {
		t.Errorf("expected carv_append_file via fs module, got:\n%s", output)
	}
}

func TestFileIOModuleFileExists(t *testing.T) {
	output := generateOutputFromSource(t, `
require "fs" as fs;
fn main() {
	let exists = fs.file_exists("out.txt");
}
`)
	if !strings.Contains(output, "carv_file_exists(") {
		t.Errorf("expected carv_file_exists via fs module, got:\n%s", output)
	}
}

func TestFileIOModuleDeleteFile(t *testing.T) {
	output := generateOutputFromSource(t, `
require "fs" as fs;
fn main() {
	let ok = fs.delete_file("out.txt");
}
`)
	if !strings.Contains(output, "carv_delete_file(") {
		t.Errorf("expected carv_delete_file via fs module, got:\n%s", output)
	}
}

func TestFileIOModuleListDir(t *testing.T) {
	output := generateOutputFromSource(t, `
require "fs" as fs;
fn main() {
	let entries = fs.list_dir(".");
}
`)
	if !strings.Contains(output, "carv_list_dir(") {
		t.Errorf("expected carv_list_dir via fs module, got:\n%s", output)
	}
}

func TestFileIORuntimeEmitted(t *testing.T) {
	output := generateOutputFromSource(t, ``)
	for _, fn := range []string{
		"carv_read_file",
		"carv_write_file",
		"carv_append_file",
		"carv_file_exists",
		"carv_delete_file",
		"carv_list_dir",
	} {
		if !strings.Contains(output, fn) {
			t.Errorf("expected %s to be emitted in runtime, got:\n%s", fn, output)
		}
	}
}
