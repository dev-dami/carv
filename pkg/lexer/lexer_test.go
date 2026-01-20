package lexer

import "testing"

func TestNextToken(t *testing.T) {
	input := `let x = 10
let y = 20.5
fn add(a: int, b: int) -> int {
	return a + b
}

x |> add(y) |> print

if x == 10 {
	spawn { print("concurrent") }
}

class Point {
	pub x: int
	pub y: int
}

interface Drawable {
	fn draw(self)
}
`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{TOKEN_LET, "let"},
		{TOKEN_IDENT, "x"},
		{TOKEN_ASSIGN, "="},
		{TOKEN_INT, "10"},
		{TOKEN_LET, "let"},
		{TOKEN_IDENT, "y"},
		{TOKEN_ASSIGN, "="},
		{TOKEN_FLOAT, "20.5"},
		{TOKEN_FN, "fn"},
		{TOKEN_IDENT, "add"},
		{TOKEN_LPAREN, "("},
		{TOKEN_IDENT, "a"},
		{TOKEN_COLON, ":"},
		{TOKEN_INT_TYPE, "int"},
		{TOKEN_COMMA, ","},
		{TOKEN_IDENT, "b"},
		{TOKEN_COLON, ":"},
		{TOKEN_INT_TYPE, "int"},
		{TOKEN_RPAREN, ")"},
		{TOKEN_ARROW, "->"},
		{TOKEN_INT_TYPE, "int"},
		{TOKEN_LBRACE, "{"},
		{TOKEN_RETURN, "return"},
		{TOKEN_IDENT, "a"},
		{TOKEN_PLUS, "+"},
		{TOKEN_IDENT, "b"},
		{TOKEN_RBRACE, "}"},
		{TOKEN_IDENT, "x"},
		{TOKEN_PIPE, "|>"},
		{TOKEN_IDENT, "add"},
		{TOKEN_LPAREN, "("},
		{TOKEN_IDENT, "y"},
		{TOKEN_RPAREN, ")"},
		{TOKEN_PIPE, "|>"},
		{TOKEN_IDENT, "print"},
		{TOKEN_IF, "if"},
		{TOKEN_IDENT, "x"},
		{TOKEN_EQ, "=="},
		{TOKEN_INT, "10"},
		{TOKEN_LBRACE, "{"},
		{TOKEN_SPAWN, "spawn"},
		{TOKEN_LBRACE, "{"},
		{TOKEN_IDENT, "print"},
		{TOKEN_LPAREN, "("},
		{TOKEN_STRING, "concurrent"},
		{TOKEN_RPAREN, ")"},
		{TOKEN_RBRACE, "}"},
		{TOKEN_RBRACE, "}"},
		{TOKEN_CLASS, "class"},
		{TOKEN_IDENT, "Point"},
		{TOKEN_LBRACE, "{"},
		{TOKEN_PUB, "pub"},
		{TOKEN_IDENT, "x"},
		{TOKEN_COLON, ":"},
		{TOKEN_INT_TYPE, "int"},
		{TOKEN_PUB, "pub"},
		{TOKEN_IDENT, "y"},
		{TOKEN_COLON, ":"},
		{TOKEN_INT_TYPE, "int"},
		{TOKEN_RBRACE, "}"},
		{TOKEN_INTERFACE, "interface"},
		{TOKEN_IDENT, "Drawable"},
		{TOKEN_LBRACE, "{"},
		{TOKEN_FN, "fn"},
		{TOKEN_IDENT, "draw"},
		{TOKEN_LPAREN, "("},
		{TOKEN_SELF, "self"},
		{TOKEN_RPAREN, ")"},
		{TOKEN_RBRACE, "}"},
		{TOKEN_EOF, ""},
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q (literal=%q)",
				i, tt.expectedType, tok.Type, tok.Literal)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestOperators(t *testing.T) {
	input := `+ - * / % ^ & | ~ ! ?
== != < <= > >= && ||
= += -= *= /= %= &= |= ^=
-> => |> <| <-`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{TOKEN_PLUS, "+"},
		{TOKEN_MINUS, "-"},
		{TOKEN_STAR, "*"},
		{TOKEN_SLASH, "/"},
		{TOKEN_PERCENT, "%"},
		{TOKEN_CARET, "^"},
		{TOKEN_AMPERSAND, "&"},
		{TOKEN_VBAR, "|"},
		{TOKEN_TILDE, "~"},
		{TOKEN_BANG, "!"},
		{TOKEN_QUESTION, "?"},
		{TOKEN_EQ, "=="},
		{TOKEN_NE, "!="},
		{TOKEN_LT, "<"},
		{TOKEN_LE, "<="},
		{TOKEN_GT, ">"},
		{TOKEN_GE, ">="},
		{TOKEN_AND, "&&"},
		{TOKEN_OR, "||"},
		{TOKEN_ASSIGN, "="},
		{TOKEN_PLUS_EQ, "+="},
		{TOKEN_MINUS_EQ, "-="},
		{TOKEN_STAR_EQ, "*="},
		{TOKEN_SLASH_EQ, "/="},
		{TOKEN_PERCENT_EQ, "%="},
		{TOKEN_AMPERSAND_EQ, "&="},
		{TOKEN_VBAR_EQ, "|="},
		{TOKEN_CARET_EQ, "^="},
		{TOKEN_ARROW, "->"},
		{TOKEN_FAT_ARROW, "=>"},
		{TOKEN_PIPE, "|>"},
		{TOKEN_PIPE_BACK, "<|"},
		{TOKEN_LARROW, "<-"},
		{TOKEN_EOF, ""},
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestConcurrencyKeywords(t *testing.T) {
	input := `spawn async await chan select send recv`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{TOKEN_SPAWN, "spawn"},
		{TOKEN_ASYNC, "async"},
		{TOKEN_AWAIT, "await"},
		{TOKEN_CHAN, "chan"},
		{TOKEN_SELECT, "select"},
		{TOKEN_SEND, "send"},
		{TOKEN_RECV, "recv"},
		{TOKEN_EOF, ""},
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}
