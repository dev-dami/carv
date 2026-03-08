package lexer

import (
	"fmt"
	"testing"
)

func TestNextToken(t *testing.T) {
	input := `let x = 10
let y = 20.5
fn add(a: int, b: int) -> int {
	return a + b
}

add(x, y)

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
		{TOKEN_IDENT, "add"},
		{TOKEN_LPAREN, "("},
		{TOKEN_IDENT, "x"},
		{TOKEN_COMMA, ","},
		{TOKEN_IDENT, "y"},
		{TOKEN_RPAREN, ")"},
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
-> => <-`

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

func TestCharLiteral(t *testing.T) {
	tests := []struct {
		input   string
		literal string
	}{
		{`'a'`, "a"},
		{`'z'`, "z"},
		{`'\n'`, `\n`},
		{`'\t'`, `\t`},
	}
	for _, tt := range tests {
		l := New(tt.input)
		tok := l.NextToken()
		if tok.Type != TOKEN_CHAR {
			t.Fatalf("expected TOKEN_CHAR, got %q for input %q", tok.Type, tt.input)
		}
		if tok.Literal != tt.literal {
			t.Fatalf("expected literal %q, got %q for input %q", tt.literal, tok.Literal, tt.input)
		}
	}
}

func TestLineComment(t *testing.T) {
	input := "// this is a comment\nlet x = 5"
	l := New(input)
	tok := l.NextToken()
	if tok.Type != TOKEN_COMMENT {
		t.Fatalf("expected TOKEN_COMMENT, got %q", tok.Type)
	}
	if tok.Literal != "// this is a comment" {
		t.Fatalf("expected comment literal, got %q", tok.Literal)
	}
	tok = l.NextToken()
	if tok.Type != TOKEN_LET {
		t.Fatalf("expected TOKEN_LET after comment, got %q", tok.Type)
	}
}

func TestLineCommentAtEOF(t *testing.T) {
	input := "// comment at eof"
	l := New(input)
	tok := l.NextToken()
	if tok.Type != TOKEN_COMMENT {
		t.Fatalf("expected TOKEN_COMMENT, got %q", tok.Type)
	}
	if tok.Literal != "// comment at eof" {
		t.Fatalf("expected comment literal, got %q", tok.Literal)
	}
}

func TestBlockComment(t *testing.T) {
	input := "/* block comment */ let x = 5"
	l := New(input)
	tok := l.NextToken()
	if tok.Type != TOKEN_COMMENT {
		t.Fatalf("expected TOKEN_COMMENT, got %q", tok.Type)
	}
	if tok.Literal != "/* block comment */" {
		t.Fatalf("expected block comment literal, got %q", tok.Literal)
	}
	tok = l.NextToken()
	if tok.Type != TOKEN_LET {
		t.Fatalf("expected TOKEN_LET after block comment, got %q", tok.Type)
	}
}

func TestBlockCommentUnterminated(t *testing.T) {
	input := "/* unterminated"
	l := New(input)
	tok := l.NextToken()
	if tok.Type != TOKEN_COMMENT {
		t.Fatalf("expected TOKEN_COMMENT, got %q", tok.Type)
	}
}

func TestBlockCommentMultiline(t *testing.T) {
	input := "/* line1\nline2\nline3 */"
	l := New(input)
	tok := l.NextToken()
	if tok.Type != TOKEN_COMMENT {
		t.Fatalf("expected TOKEN_COMMENT, got %q", tok.Type)
	}
	if tok.Literal != "/* line1\nline2\nline3 */" {
		t.Fatalf("expected multiline block comment, got %q", tok.Literal)
	}
}

func TestTokenTypeString(t *testing.T) {
	tests := []struct {
		tt   TokenType
		want string
	}{
		{TOKEN_PLUS, "+"},
		{TOKEN_EOF, "EOF"},
		{TOKEN_IDENT, "IDENT"},
		{TokenType(9999), "UNKNOWN"},
	}
	for _, tt := range tests {
		got := tt.tt.String()
		if got != tt.want {
			t.Fatalf("TokenType(%d).String() = %q, want %q", tt.tt, got, tt.want)
		}
	}
}

func TestTokenPos(t *testing.T) {
	tok := Token{Type: TOKEN_IDENT, Literal: "x", Line: 3, Column: 7}
	pos := tok.Pos()
	expected := fmt.Sprintf("%d:%d", 3, 7)
	if pos != expected {
		t.Fatalf("Pos() = %q, want %q", pos, expected)
	}
}

func TestIsKeyword(t *testing.T) {
	tests := []struct {
		tt   TokenType
		want bool
	}{
		{TOKEN_FN, true},
		{TOKEN_RETURN, true},
		{TOKEN_PACKED, true},
		{TOKEN_IDENT, false},
		{TOKEN_INT, false},
		{TOKEN_PLUS, false},
	}
	for _, tt := range tests {
		tok := Token{Type: tt.tt}
		if tok.IsKeyword() != tt.want {
			t.Fatalf("Token{Type: %d}.IsKeyword() = %v, want %v", tt.tt, !tt.want, tt.want)
		}
	}
}

func TestReadNumberFloat(t *testing.T) {
	tests := []struct {
		input       string
		expectedTyp TokenType
		expectedLit string
	}{
		{"3.14", TOKEN_FLOAT, "3.14"},
		{"0.001", TOKEN_FLOAT, "0.001"},
		{"1e10", TOKEN_FLOAT, "1e10"},
		{"2E5", TOKEN_FLOAT, "2E5"},
		{"1e+2", TOKEN_FLOAT, "1e+2"},
		{"1e-3", TOKEN_FLOAT, "1e-3"},
		{"3.14e10", TOKEN_FLOAT, "3.14e10"},
		{"42", TOKEN_INT, "42"},
	}
	for _, tt := range tests {
		l := New(tt.input)
		tok := l.NextToken()
		if tok.Type != tt.expectedTyp {
			t.Fatalf("input %q: expected type %q, got %q", tt.input, tt.expectedTyp, tok.Type)
		}
		if tok.Literal != tt.expectedLit {
			t.Fatalf("input %q: expected literal %q, got %q", tt.input, tt.expectedLit, tok.Literal)
		}
	}
}

func TestReadStringEscapeSequences(t *testing.T) {
	tests := []struct {
		input   string
		literal string
	}{
		{`"hello"`, "hello"},
		{`"he\"llo"`, `he\"llo`},
		{`"line\nbreak"`, `line\nbreak`},
		{`"tab\there"`, `tab\there`},
		{`"back\\slash"`, `back\\slash`},
	}
	for _, tt := range tests {
		l := New(tt.input)
		tok := l.NextToken()
		if tok.Type != TOKEN_STRING {
			t.Fatalf("input %q: expected TOKEN_STRING, got %q", tt.input, tok.Type)
		}
		if tok.Literal != tt.literal {
			t.Fatalf("input %q: expected literal %q, got %q", tt.input, tt.literal, tok.Literal)
		}
	}
}

func TestUnterminatedString(t *testing.T) {
	input := `"unterminated`
	l := New(input)
	tok := l.NextToken()
	if tok.Type != TOKEN_STRING {
		t.Fatalf("expected TOKEN_STRING, got %q", tok.Type)
	}
}

func TestInterpString(t *testing.T) {
	input := `f"hello {name}"`
	l := New(input)
	tok := l.NextToken()
	if tok.Type != TOKEN_INTERP_STRING {
		t.Fatalf("expected TOKEN_INTERP_STRING, got %q", tok.Type)
	}
	if tok.Literal != "hello {name}" {
		t.Fatalf("expected literal %q, got %q", "hello {name}", tok.Literal)
	}
}

func TestIllegalToken(t *testing.T) {
	input := "@"
	l := New(input)
	tok := l.NextToken()
	if tok.Type != TOKEN_ILLEGAL {
		t.Fatalf("expected TOKEN_ILLEGAL, got %q", tok.Type)
	}
}

func TestPeekCharAtEOF(t *testing.T) {
	l := New("a")
	tok := l.NextToken()
	if tok.Type != TOKEN_IDENT {
		t.Fatalf("expected TOKEN_IDENT, got %q", tok.Type)
	}
	tok = l.NextToken()
	if tok.Type != TOKEN_EOF {
		t.Fatalf("expected TOKEN_EOF, got %q", tok.Type)
	}
}

func TestEmptyInput(t *testing.T) {
	l := New("")
	tok := l.NextToken()
	if tok.Type != TOKEN_EOF {
		t.Fatalf("expected TOKEN_EOF, got %q", tok.Type)
	}
}

func TestWhitespaceHandling(t *testing.T) {
	input := "  \t\r\n  42"
	l := New(input)
	tok := l.NextToken()
	if tok.Type != TOKEN_INT {
		t.Fatalf("expected TOKEN_INT, got %q", tok.Type)
	}
	if tok.Literal != "42" {
		t.Fatalf("expected literal '42', got %q", tok.Literal)
	}
}

func TestNumberFollowedByDotNonDigit(t *testing.T) {
	input := "5.a"
	l := New(input)
	tok := l.NextToken()
	if tok.Type != TOKEN_INT {
		t.Fatalf("expected TOKEN_INT, got %q", tok.Type)
	}
	if tok.Literal != "5" {
		t.Fatalf("expected '5', got %q", tok.Literal)
	}
	tok = l.NextToken()
	if tok.Type != TOKEN_DOT {
		t.Fatalf("expected TOKEN_DOT, got %q", tok.Type)
	}
}

func TestMiscKeywords(t *testing.T) {
	input := "true false nil Ok Err volatile packed mut const else match for while loop break continue impl priv static super new type as in is"
	expected := []TokenType{
		TOKEN_TRUE, TOKEN_FALSE, TOKEN_NIL, TOKEN_OK, TOKEN_ERR,
		TOKEN_VOLATILE, TOKEN_PACKED, TOKEN_MUT, TOKEN_CONST,
		TOKEN_ELSE, TOKEN_MATCH, TOKEN_FOR, TOKEN_WHILE, TOKEN_LOOP,
		TOKEN_BREAK, TOKEN_CONTINUE, TOKEN_IMPL, TOKEN_PRIV,
		TOKEN_STATIC, TOKEN_SUPER, TOKEN_NEW, TOKEN_TYPE,
		TOKEN_AS, TOKEN_IN, TOKEN_IS,
	}
	l := New(input)
	for i, exp := range expected {
		tok := l.NextToken()
		if tok.Type != exp {
			t.Fatalf("misc keywords[%d]: expected %q, got %q (literal=%q)", i, exp, tok.Type, tok.Literal)
		}
	}
}
