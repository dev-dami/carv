package lexer

type Lexer struct {
	input        string
	position     int
	readPosition int
	ch           byte
	line         int
	column       int
}

func New(input string) *Lexer {
	l := &Lexer{input: input, line: 1, column: 0}
	l.readChar()
	return l
}

func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPosition]
	}
	l.position = l.readPosition
	l.readPosition++
	l.column++
	if l.ch == '\n' {
		l.line++
		l.column = 0
	}
}

func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

func (l *Lexer) NextToken() Token {
	var tok Token

	l.skipWhitespace()

	tok.Line = l.line
	tok.Column = l.column

	switch l.ch {
	case '+':
		if l.peekChar() == '=' {
			l.readChar()
			tok = Token{Type: TOKEN_PLUS_EQ, Literal: "+=", Line: tok.Line, Column: tok.Column}
		} else {
			tok = l.newToken(TOKEN_PLUS, l.ch)
		}
	case '-':
		if l.peekChar() == '>' {
			l.readChar()
			tok = Token{Type: TOKEN_ARROW, Literal: "->", Line: tok.Line, Column: tok.Column}
		} else if l.peekChar() == '=' {
			l.readChar()
			tok = Token{Type: TOKEN_MINUS_EQ, Literal: "-=", Line: tok.Line, Column: tok.Column}
		} else {
			tok = l.newToken(TOKEN_MINUS, l.ch)
		}
	case '*':
		if l.peekChar() == '=' {
			l.readChar()
			tok = Token{Type: TOKEN_STAR_EQ, Literal: "*=", Line: tok.Line, Column: tok.Column}
		} else {
			tok = l.newToken(TOKEN_STAR, l.ch)
		}
	case '/':
		if l.peekChar() == '/' {
			tok.Type = TOKEN_COMMENT
			tok.Literal = l.readLineComment()
		} else if l.peekChar() == '*' {
			tok.Type = TOKEN_COMMENT
			tok.Literal = l.readBlockComment()
		} else if l.peekChar() == '=' {
			l.readChar()
			tok = Token{Type: TOKEN_SLASH_EQ, Literal: "/=", Line: tok.Line, Column: tok.Column}
		} else {
			tok = l.newToken(TOKEN_SLASH, l.ch)
		}
	case '%':
		if l.peekChar() == '=' {
			l.readChar()
			tok = Token{Type: TOKEN_PERCENT_EQ, Literal: "%=", Line: tok.Line, Column: tok.Column}
		} else {
			tok = l.newToken(TOKEN_PERCENT, l.ch)
		}
	case '^':
		if l.peekChar() == '=' {
			l.readChar()
			tok = Token{Type: TOKEN_CARET_EQ, Literal: "^=", Line: tok.Line, Column: tok.Column}
		} else {
			tok = l.newToken(TOKEN_CARET, l.ch)
		}
	case '&':
		if l.peekChar() == '&' {
			l.readChar()
			tok = Token{Type: TOKEN_AND, Literal: "&&", Line: tok.Line, Column: tok.Column}
		} else if l.peekChar() == '=' {
			l.readChar()
			tok = Token{Type: TOKEN_AMPERSAND_EQ, Literal: "&=", Line: tok.Line, Column: tok.Column}
		} else {
			tok = l.newToken(TOKEN_AMPERSAND, l.ch)
		}
	case '|':
		if l.peekChar() == '|' {
			l.readChar()
			tok = Token{Type: TOKEN_OR, Literal: "||", Line: tok.Line, Column: tok.Column}
		} else if l.peekChar() == '>' {
			l.readChar()
			tok = Token{Type: TOKEN_PIPE, Literal: "|>", Line: tok.Line, Column: tok.Column}
		} else if l.peekChar() == '=' {
			l.readChar()
			tok = Token{Type: TOKEN_VBAR_EQ, Literal: "|=", Line: tok.Line, Column: tok.Column}
		} else {
			tok = l.newToken(TOKEN_VBAR, l.ch)
		}
	case '~':
		tok = l.newToken(TOKEN_TILDE, l.ch)
	case '!':
		if l.peekChar() == '=' {
			l.readChar()
			tok = Token{Type: TOKEN_NE, Literal: "!=", Line: tok.Line, Column: tok.Column}
		} else {
			tok = l.newToken(TOKEN_BANG, l.ch)
		}
	case '?':
		tok = l.newToken(TOKEN_QUESTION, l.ch)
	case '=':
		if l.peekChar() == '=' {
			l.readChar()
			tok = Token{Type: TOKEN_EQ, Literal: "==", Line: tok.Line, Column: tok.Column}
		} else if l.peekChar() == '>' {
			l.readChar()
			tok = Token{Type: TOKEN_FAT_ARROW, Literal: "=>", Line: tok.Line, Column: tok.Column}
		} else {
			tok = l.newToken(TOKEN_ASSIGN, l.ch)
		}
	case '<':
		if l.peekChar() == '=' {
			l.readChar()
			tok = Token{Type: TOKEN_LE, Literal: "<=", Line: tok.Line, Column: tok.Column}
		} else if l.peekChar() == '-' {
			l.readChar()
			tok = Token{Type: TOKEN_LARROW, Literal: "<-", Line: tok.Line, Column: tok.Column}
		} else if l.peekChar() == '|' {
			l.readChar()
			tok = Token{Type: TOKEN_PIPE_BACK, Literal: "<|", Line: tok.Line, Column: tok.Column}
		} else {
			tok = l.newToken(TOKEN_LT, l.ch)
		}
	case '>':
		if l.peekChar() == '=' {
			l.readChar()
			tok = Token{Type: TOKEN_GE, Literal: ">=", Line: tok.Line, Column: tok.Column}
		} else {
			tok = l.newToken(TOKEN_GT, l.ch)
		}
	case '(':
		tok = l.newToken(TOKEN_LPAREN, l.ch)
	case ')':
		tok = l.newToken(TOKEN_RPAREN, l.ch)
	case '{':
		tok = l.newToken(TOKEN_LBRACE, l.ch)
	case '}':
		tok = l.newToken(TOKEN_RBRACE, l.ch)
	case '[':
		tok = l.newToken(TOKEN_LBRACKET, l.ch)
	case ']':
		tok = l.newToken(TOKEN_RBRACKET, l.ch)
	case ',':
		tok = l.newToken(TOKEN_COMMA, l.ch)
	case '.':
		tok = l.newToken(TOKEN_DOT, l.ch)
	case ':':
		tok = l.newToken(TOKEN_COLON, l.ch)
	case ';':
		tok = l.newToken(TOKEN_SEMI, l.ch)
	case '"':
		tok.Type = TOKEN_STRING
		tok.Literal = l.readString()
	case '\'':
		tok.Type = TOKEN_CHAR
		tok.Literal = l.readCharLiteral()
	case 0:
		tok.Type = TOKEN_EOF
		tok.Literal = ""
	default:
		if isLetter(l.ch) {
			tok.Literal = l.readIdentifier()
			tok.Type = LookupIdent(tok.Literal)
			return tok
		} else if isDigit(l.ch) {
			tok.Type, tok.Literal = l.readNumber()
			return tok
		} else {
			tok = l.newToken(TOKEN_ILLEGAL, l.ch)
		}
	}

	l.readChar()
	return tok
}

func (l *Lexer) newToken(tokenType TokenType, ch byte) Token {
	return Token{Type: tokenType, Literal: string(ch), Line: l.line, Column: l.column}
}

func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.readChar()
	}
}

func (l *Lexer) readIdentifier() string {
	position := l.position
	for isLetter(l.ch) || isDigit(l.ch) {
		l.readChar()
	}
	return l.input[position:l.position]
}

func (l *Lexer) readNumber() (TokenType, string) {
	position := l.position
	tokenType := TOKEN_INT

	for isDigit(l.ch) {
		l.readChar()
	}

	if l.ch == '.' && isDigit(l.peekChar()) {
		tokenType = TOKEN_FLOAT
		l.readChar()
		for isDigit(l.ch) {
			l.readChar()
		}
	}

	if l.ch == 'e' || l.ch == 'E' {
		tokenType = TOKEN_FLOAT
		l.readChar()
		if l.ch == '+' || l.ch == '-' {
			l.readChar()
		}
		for isDigit(l.ch) {
			l.readChar()
		}
	}

	return tokenType, l.input[position:l.position]
}

func (l *Lexer) readString() string {
	position := l.position + 1
	for {
		l.readChar()
		if l.ch == '"' || l.ch == 0 {
			break
		}
		if l.ch == '\\' {
			l.readChar()
		}
	}
	return l.input[position:l.position]
}

func (l *Lexer) readCharLiteral() string {
	position := l.position + 1
	l.readChar()
	if l.ch == '\\' {
		l.readChar()
	}
	l.readChar()
	return l.input[position:l.position]
}

func (l *Lexer) readLineComment() string {
	position := l.position
	for l.ch != '\n' && l.ch != 0 {
		l.readChar()
	}
	return l.input[position:l.position]
}

func (l *Lexer) readBlockComment() string {
	position := l.position
	l.readChar()
	l.readChar()
	for {
		if l.ch == 0 {
			break
		}
		if l.ch == '*' && l.peekChar() == '/' {
			l.readChar()
			l.readChar()
			break
		}
		l.readChar()
	}
	return l.input[position:l.position]
}

func isLetter(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_'
}

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}
