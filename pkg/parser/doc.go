// Package parser turns lexer tokens into AST nodes.
//
// Design decisions:
//   - Pratt parsing is used for expressions (operator precedence and associativity).
//   - Recursive-descent parsing is used for declarations and statements for clarity.
//
// Usage pattern:
//
//	l := lexer.New(src)
//	p := parser.New(l)
//	prog := p.ParseProgram()
//	if len(p.Errors()) > 0 {
//	    // handle syntax errors
//	}
package parser
