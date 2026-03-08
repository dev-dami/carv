package codegen

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

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
