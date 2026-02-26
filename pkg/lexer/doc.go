// Package lexer tokenizes Carv source text.
//
// Design decisions:
//   - Lexer output is line/column aware for precise parser and checker errors.
//   - Keywords and operators are normalized into TokenType constants in token.go.
//
// Usage pattern:
//
//	l := lexer.New(src)
//	for tok := l.NextToken(); tok.Type != lexer.TOKEN_EOF; tok = l.NextToken() {
//	    // consume token stream
//	}
package lexer
