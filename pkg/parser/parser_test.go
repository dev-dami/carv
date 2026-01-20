package parser

import (
	"testing"

	"github.com/carv-lang/carv/pkg/ast"
	"github.com/carv-lang/carv/pkg/lexer"
)

func TestLetStatements(t *testing.T) {
	input := `let x = 5;
let y = 10;
mut z = 15;`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 3 {
		t.Fatalf("expected 3 statements, got %d", len(program.Statements))
	}

	tests := []struct {
		expectedName string
		mutable      bool
	}{
		{"x", false},
		{"y", false},
		{"z", true},
	}

	for i, tt := range tests {
		stmt := program.Statements[i]
		letStmt, ok := stmt.(*ast.LetStatement)
		if !ok {
			t.Fatalf("stmt not *ast.LetStatement. got=%T", stmt)
		}
		if letStmt.Name.Value != tt.expectedName {
			t.Fatalf("expected name %s, got %s", tt.expectedName, letStmt.Name.Value)
		}
		if letStmt.Mutable != tt.mutable {
			t.Fatalf("expected mutable %v, got %v", tt.mutable, letStmt.Mutable)
		}
	}
}

func TestFunctionStatement(t *testing.T) {
	input := `fn add(a: int, b: int) -> int {
	return a + b;
}`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	fnStmt, ok := program.Statements[0].(*ast.FunctionStatement)
	if !ok {
		t.Fatalf("stmt not *ast.FunctionStatement. got=%T", program.Statements[0])
	}

	if fnStmt.Name.Value != "add" {
		t.Fatalf("expected function name 'add', got %s", fnStmt.Name.Value)
	}

	if len(fnStmt.Parameters) != 2 {
		t.Fatalf("expected 2 parameters, got %d", len(fnStmt.Parameters))
	}
}

func TestPipeExpression(t *testing.T) {
	input := `x |> double |> print;`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*ast.ExpressionStatement)
	if !ok {
		t.Fatalf("stmt not *ast.ExpressionStatement. got=%T", program.Statements[0])
	}

	pipe, ok := stmt.Expression.(*ast.PipeExpression)
	if !ok {
		t.Fatalf("expression not *ast.PipeExpression. got=%T", stmt.Expression)
	}

	rightIdent, ok := pipe.Right.(*ast.Identifier)
	if !ok {
		t.Fatalf("right not identifier. got=%T", pipe.Right)
	}
	if rightIdent.Value != "print" {
		t.Fatalf("expected 'print', got %s", rightIdent.Value)
	}
}

func TestComparisonExpression(t *testing.T) {
	tests := []struct {
		input    string
		left     int64
		operator string
		right    int64
	}{
		{"5 == 5;", 5, "==", 5},
		{"5 != 4;", 5, "!=", 4},
		{"5 < 10;", 5, "<", 10},
		{"10 > 5;", 10, ">", 5},
		{"5 <= 5;", 5, "<=", 5},
		{"5 >= 5;", 5, ">=", 5},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		stmt := program.Statements[0].(*ast.ExpressionStatement)
		exp, ok := stmt.Expression.(*ast.InfixExpression)
		if !ok {
			t.Fatalf("expression not *ast.InfixExpression. got=%T", stmt.Expression)
		}

		if exp.Operator != tt.operator {
			t.Fatalf("expected operator %s, got %s", tt.operator, exp.Operator)
		}
	}
}

func TestCallExpression(t *testing.T) {
	input := `print("hello", 42);`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	call, ok := stmt.Expression.(*ast.CallExpression)
	if !ok {
		t.Fatalf("expression not *ast.CallExpression. got=%T", stmt.Expression)
	}

	if len(call.Arguments) != 2 {
		t.Fatalf("expected 2 arguments, got %d", len(call.Arguments))
	}
}

func TestForLoop(t *testing.T) {
	input := `for (let i = 0; i < 10; i = i + 1) {
	print(i);
}`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	forStmt, ok := program.Statements[0].(*ast.ForStatement)
	if !ok {
		t.Fatalf("stmt not *ast.ForStatement. got=%T", program.Statements[0])
	}

	if forStmt.Init == nil {
		t.Fatal("expected init statement")
	}
	if forStmt.Condition == nil {
		t.Fatal("expected condition")
	}
	if forStmt.Post == nil {
		t.Fatal("expected post statement")
	}
}

func TestIfExpression(t *testing.T) {
	input := `if x > 5 {
	print("big");
} else {
	print("small");
}`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	ifExp, ok := stmt.Expression.(*ast.IfExpression)
	if !ok {
		t.Fatalf("expression not *ast.IfExpression. got=%T", stmt.Expression)
	}

	if ifExp.Alternative == nil {
		t.Fatal("expected alternative block")
	}
}

func TestOkErrExpressions(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Ok(42);", "Ok"},
		{"Err(\"failed\");", "Err"},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		stmt := program.Statements[0].(*ast.ExpressionStatement)
		if tt.expected == "Ok" {
			_, ok := stmt.Expression.(*ast.OkExpression)
			if !ok {
				t.Fatalf("expected OkExpression, got %T", stmt.Expression)
			}
		} else {
			_, ok := stmt.Expression.(*ast.ErrExpression)
			if !ok {
				t.Fatalf("expected ErrExpression, got %T", stmt.Expression)
			}
		}
	}
}

func TestTryExpression(t *testing.T) {
	input := `read_file("test.txt")?;`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	tryExp, ok := stmt.Expression.(*ast.TryExpression)
	if !ok {
		t.Fatalf("expected TryExpression, got %T", stmt.Expression)
	}

	call, ok := tryExp.Value.(*ast.CallExpression)
	if !ok {
		t.Fatalf("expected CallExpression inside try, got %T", tryExp.Value)
	}

	fn, ok := call.Function.(*ast.Identifier)
	if !ok || fn.Value != "read_file" {
		t.Fatalf("expected read_file call")
	}
}

func TestMatchExpression(t *testing.T) {
	input := `match result {
	Ok(x) => x,
	Err(e) => 0,
};`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	matchExp, ok := stmt.Expression.(*ast.MatchExpression)
	if !ok {
		t.Fatalf("expected MatchExpression, got %T", stmt.Expression)
	}

	if len(matchExp.Arms) != 2 {
		t.Fatalf("expected 2 match arms, got %d", len(matchExp.Arms))
	}
}

func checkParserErrors(t *testing.T, p *Parser) {
	errors := p.Errors()
	if len(errors) == 0 {
		return
	}

	t.Errorf("parser has %d errors", len(errors))
	for _, msg := range errors {
		t.Errorf("parser error: %s", msg)
	}
	t.FailNow()
}
