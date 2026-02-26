package ast

import (
	"testing"

	"github.com/dev-dami/carv/pkg/lexer"
)

func TestProgram(t *testing.T) {
	p := &Program{Statements: []Statement{}}
	if p.TokenLiteral() != "" {
		t.Errorf("expected empty token literal, got %q", p.TokenLiteral())
	}
	line, col := p.Pos()
	if line != 0 || col != 0 {
		t.Errorf("expected position (0,0), got (%d,%d)", line, col)
	}

	token := lexer.Token{Type: lexer.TOKEN_LET, Literal: "let", Line: 1, Column: 1}
	letStmt := &LetStatement{Token: token}
	p.Statements = append(p.Statements, letStmt)

	if p.TokenLiteral() != "let" {
		t.Errorf("expected token literal 'let', got %q", p.TokenLiteral())
	}
	line, col = p.Pos()
	if line != 1 || col != 1 {
		t.Errorf("expected position (1,1), got (%d,%d)", line, col)
	}
}

func TestIdentifier(t *testing.T) {
	token := lexer.Token{Type: lexer.TOKEN_IDENT, Literal: "foo", Line: 5, Column: 10}
	ident := &Identifier{Token: token, Value: "foo"}

	if ident.TokenLiteral() != "foo" {
		t.Errorf("expected token literal 'foo', got %q", ident.TokenLiteral())
	}

	line, col := ident.Pos()
	if line != 5 || col != 10 {
		t.Errorf("expected position (5,10), got (%d,%d)", line, col)
	}
}

func TestIntegerLiteral(t *testing.T) {
	token := lexer.Token{Type: lexer.TOKEN_INT, Literal: "42", Line: 1, Column: 5}
	intLit := &IntegerLiteral{Token: token, Value: 42}

	if intLit.TokenLiteral() != "42" {
		t.Errorf("expected token literal '42', got %q", intLit.TokenLiteral())
	}

	line, col := intLit.Pos()
	if line != 1 || col != 5 {
		t.Errorf("expected position (1,5), got (%d,%d)", line, col)
	}
}

func TestFloatLiteral(t *testing.T) {
	token := lexer.Token{Type: lexer.TOKEN_FLOAT, Literal: "3.14", Line: 2, Column: 3}
	floatLit := &FloatLiteral{Token: token, Value: 3.14}

	if floatLit.TokenLiteral() != "3.14" {
		t.Errorf("expected token literal '3.14', got %q", floatLit.TokenLiteral())
	}

	line, col := floatLit.Pos()
	if line != 2 || col != 3 {
		t.Errorf("expected position (2,3), got (%d,%d)", line, col)
	}
}

func TestStringLiteral(t *testing.T) {
	token := lexer.Token{Type: lexer.TOKEN_STRING, Literal: "hello", Line: 1, Column: 1}
	strLit := &StringLiteral{Token: token, Value: "hello"}

	if strLit.TokenLiteral() != "hello" {
		t.Errorf("expected token literal 'hello', got %q", strLit.TokenLiteral())
	}
}

func TestCharLiteral(t *testing.T) {
	token := lexer.Token{Type: lexer.TOKEN_CHAR, Literal: "a", Line: 1, Column: 1}
	charLit := &CharLiteral{Token: token, Value: 'a'}

	if charLit.TokenLiteral() != "a" {
		t.Errorf("expected token literal 'a', got %q", charLit.TokenLiteral())
	}
}

func TestBoolLiteral(t *testing.T) {
	token := lexer.Token{Type: lexer.TOKEN_TRUE, Literal: "true", Line: 1, Column: 1}
	boolLit := &BoolLiteral{Token: token, Value: true}

	if boolLit.TokenLiteral() != "true" {
		t.Errorf("expected token literal 'true', got %q", boolLit.TokenLiteral())
	}
}

func TestNilLiteral(t *testing.T) {
	token := lexer.Token{Type: lexer.TOKEN_NIL, Literal: "nil", Line: 1, Column: 1}
	nilLit := &NilLiteral{Token: token}

	if nilLit.TokenLiteral() != "nil" {
		t.Errorf("expected token literal 'nil', got %q", nilLit.TokenLiteral())
	}
}

func TestArrayLiteral(t *testing.T) {
	token := lexer.Token{Type: lexer.TOKEN_LBRACKET, Literal: "[", Line: 1, Column: 1}
	arrLit := &ArrayLiteral{Token: token, Elements: []Expression{}}

	if arrLit.TokenLiteral() != "[" {
		t.Errorf("expected token literal '[', got %q", arrLit.TokenLiteral())
	}
}

func TestPrefixExpression(t *testing.T) {
	token := lexer.Token{Type: lexer.TOKEN_BANG, Literal: "!", Line: 1, Column: 1}
	prefix := &PrefixExpression{Token: token, Operator: "!"}

	if prefix.TokenLiteral() != "!" {
		t.Errorf("expected token literal '!', got %q", prefix.TokenLiteral())
	}
}

func TestInfixExpression(t *testing.T) {
	token := lexer.Token{Type: lexer.TOKEN_PLUS, Literal: "+", Line: 1, Column: 5}
	infix := &InfixExpression{Token: token, Operator: "+"}

	if infix.TokenLiteral() != "+" {
		t.Errorf("expected token literal '+', got %q", infix.TokenLiteral())
	}
}

func TestPipeExpression(t *testing.T) {
	token := lexer.Token{Type: lexer.TOKEN_PIPE, Literal: "|>", Line: 1, Column: 1}
	pipe := &PipeExpression{Token: token}

	if pipe.TokenLiteral() != "|>" {
		t.Errorf("expected token literal '|>', got %q", pipe.TokenLiteral())
	}
}

func TestCallExpression(t *testing.T) {
	token := lexer.Token{Type: lexer.TOKEN_LPAREN, Literal: "(", Line: 1, Column: 5}
	call := &CallExpression{Token: token, Arguments: []Expression{}}

	if call.TokenLiteral() != "(" {
		t.Errorf("expected token literal '(', got %q", call.TokenLiteral())
	}
}

func TestIfExpression(t *testing.T) {
	token := lexer.Token{Type: lexer.TOKEN_IF, Literal: "if", Line: 1, Column: 1}
	ifExpr := &IfExpression{Token: token}

	if ifExpr.TokenLiteral() != "if" {
		t.Errorf("expected token literal 'if', got %q", ifExpr.TokenLiteral())
	}
}

func TestMatchExpression(t *testing.T) {
	token := lexer.Token{Type: lexer.TOKEN_MATCH, Literal: "match", Line: 1, Column: 1}
	matchExpr := &MatchExpression{Token: token, Arms: []*MatchArm{}}

	if matchExpr.TokenLiteral() != "match" {
		t.Errorf("expected token literal 'match', got %q", matchExpr.TokenLiteral())
	}
}

func TestFunctionLiteral(t *testing.T) {
	token := lexer.Token{Type: lexer.TOKEN_FN, Literal: "fn", Line: 1, Column: 1}
	fnLit := &FunctionLiteral{Token: token, Parameters: []*Parameter{}}

	if fnLit.TokenLiteral() != "fn" {
		t.Errorf("expected token literal 'fn', got %q", fnLit.TokenLiteral())
	}
}

func TestOkExpression(t *testing.T) {
	token := lexer.Token{Type: lexer.TOKEN_OK, Literal: "Ok", Line: 1, Column: 1}
	okExpr := &OkExpression{Token: token}

	if okExpr.TokenLiteral() != "Ok" {
		t.Errorf("expected token literal 'Ok', got %q", okExpr.TokenLiteral())
	}
}

func TestErrExpression(t *testing.T) {
	token := lexer.Token{Type: lexer.TOKEN_ERR, Literal: "Err", Line: 1, Column: 1}
	errExpr := &ErrExpression{Token: token}

	if errExpr.TokenLiteral() != "Err" {
		t.Errorf("expected token literal 'Err', got %q", errExpr.TokenLiteral())
	}
}

func TestTryExpression(t *testing.T) {
	token := lexer.Token{Type: lexer.TOKEN_QUESTION, Literal: "?", Line: 1, Column: 10}
	tryExpr := &TryExpression{Token: token}

	if tryExpr.TokenLiteral() != "?" {
		t.Errorf("expected token literal '?', got %q", tryExpr.TokenLiteral())
	}
}

func TestNewExpression(t *testing.T) {
	token := lexer.Token{Type: lexer.TOKEN_NEW, Literal: "new", Line: 1, Column: 1}
	newExpr := &NewExpression{Token: token}

	if newExpr.TokenLiteral() != "new" {
		t.Errorf("expected token literal 'new', got %q", newExpr.TokenLiteral())
	}
}

func TestInterpolatedString(t *testing.T) {
	token := lexer.Token{Type: lexer.TOKEN_INTERP_STRING, Literal: "f\"hello\"", Line: 1, Column: 1}
	interpStr := &InterpolatedString{Token: token, Parts: []Expression{}}

	if interpStr.TokenLiteral() != "f\"hello\"" {
		t.Errorf("expected token literal 'f\"hello\"', got %q", interpStr.TokenLiteral())
	}
}

func TestAstNodeLocationHelpers(t *testing.T) {
	token := lexer.Token{Type: lexer.TOKEN_IDENT, Literal: "node", Line: 7, Column: 9}

	matchArm := &MatchArm{Token: token}
	if line, col := matchArm.Pos(); line != 7 || col != 9 {
		t.Fatalf("MatchArm expected pos (7,9), got (%d,%d)", line, col)
	}
	if matchArm.TokenLiteral() != "node" {
		t.Fatalf("MatchArm expected token literal node, got %q", matchArm.TokenLiteral())
	}

	param := &Parameter{Token: token}
	if line, col := param.Pos(); line != 7 || col != 9 {
		t.Fatalf("Parameter expected pos (7,9), got (%d,%d)", line, col)
	}
	if param.TokenLiteral() != "node" {
		t.Fatalf("Parameter expected token literal node, got %q", param.TokenLiteral())
	}

	field := &FieldDecl{Token: token}
	if line, col := field.Pos(); line != 7 || col != 9 {
		t.Fatalf("FieldDecl expected pos (7,9), got (%d,%d)", line, col)
	}
	if field.TokenLiteral() != "node" {
		t.Fatalf("FieldDecl expected token literal node, got %q", field.TokenLiteral())
	}

	method := &MethodDecl{Token: token}
	if line, col := method.Pos(); line != 7 || col != 9 {
		t.Fatalf("MethodDecl expected pos (7,9), got (%d,%d)", line, col)
	}
	if method.TokenLiteral() != "node" {
		t.Fatalf("MethodDecl expected token literal node, got %q", method.TokenLiteral())
	}

	sig := &MethodSignature{Token: token}
	if line, col := sig.Pos(); line != 7 || col != 9 {
		t.Fatalf("MethodSignature expected pos (7,9), got (%d,%d)", line, col)
	}
	if sig.TokenLiteral() != "node" {
		t.Fatalf("MethodSignature expected token literal node, got %q", sig.TokenLiteral())
	}

	selectCase := &SelectCase{Token: token}
	if line, col := selectCase.Pos(); line != 7 || col != 9 {
		t.Fatalf("SelectCase expected pos (7,9), got (%d,%d)", line, col)
	}
	if selectCase.TokenLiteral() != "node" {
		t.Fatalf("SelectCase expected token literal node, got %q", selectCase.TokenLiteral())
	}
}
