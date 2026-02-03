package codegen

import (
	"strings"
	"testing"

	"github.com/dev-dami/carv/pkg/lexer"
	"github.com/dev-dami/carv/pkg/parser"
)

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
