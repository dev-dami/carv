package types

import (
	"testing"

	"github.com/carv-lang/carv/pkg/lexer"
	"github.com/carv-lang/carv/pkg/parser"
)

func TestTypeCheckerBasics(t *testing.T) {
	tests := []struct {
		input       string
		shouldError bool
	}{
		{"let x = 5;", false},
		{"let x: int = 5;", false},
		{"let x: float = 5.0;", false},
		{"let x: string = \"hello\";", false},
		{"let x: bool = true;", false},
		{"let x = 5 + 3;", false},
		{"let x = 5.0 + 3.0;", false},
		{"let x = 5 < 10;", false},
		{"let x = true && false;", false},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := parser.New(l)
		program := p.ParseProgram()

		checker := NewChecker()
		ok := checker.Check(program)

		if tt.shouldError && ok {
			t.Errorf("expected error for %q, got none", tt.input)
		}
		if !tt.shouldError && !ok {
			t.Errorf("unexpected error for %q: %v", tt.input, checker.Errors())
		}
	}
}

func TestTypeCheckerFunctions(t *testing.T) {
	input := `
fn add(a: int, b: int) -> int {
	return a + b;
}
let result = add(5, 3);
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	checker := NewChecker()
	ok := checker.Check(program)

	if !ok {
		t.Errorf("unexpected errors: %v", checker.Errors())
	}
}

func TestTypeCheckerPipes(t *testing.T) {
	input := `
fn double(x: int) -> int {
	return x * 2;
}
5 |> double |> print;
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	checker := NewChecker()
	ok := checker.Check(program)

	if !ok {
		t.Errorf("unexpected errors: %v", checker.Errors())
	}
}

func TestTypeCheckerArrays(t *testing.T) {
	input := `
let arr = [1, 2, 3];
let x = arr[0];
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	checker := NewChecker()
	ok := checker.Check(program)

	if !ok {
		t.Errorf("unexpected errors: %v", checker.Errors())
	}
}

func TestTypeCheckerLoops(t *testing.T) {
	input := `
for (let i = 0; i < 10; i = i + 1) {
	print(i);
}

mut x = 0;
while x < 5 {
	x = x + 1;
}
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	checker := NewChecker()
	ok := checker.Check(program)

	if !ok {
		t.Errorf("unexpected errors: %v", checker.Errors())
	}
}

func TestTypeCheckerErrors(t *testing.T) {
	tests := []struct {
		input string
		desc  string
	}{
		{"let x = y;", "undefined variable"},
		{"let x = 5 && 3;", "boolean op on ints"},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := parser.New(l)
		program := p.ParseProgram()

		checker := NewChecker()
		ok := checker.Check(program)

		if ok {
			t.Errorf("expected error for %s: %q", tt.desc, tt.input)
		}
	}
}
