package parser

import (
	"github.com/dev-dami/carv/pkg/ast"
	"github.com/dev-dami/carv/pkg/lexer"
)

func (p *Parser) parseClassStatement() *ast.ClassStatement {
	stmt := &ast.ClassStatement{Token: p.curToken}

	if !p.expectPeek(lexer.TOKEN_IDENT) {
		return nil
	}
	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(lexer.TOKEN_LBRACE) {
		return nil
	}

	stmt.Fields = []*ast.FieldDecl{}
	stmt.Methods = []*ast.MethodDecl{}

	p.nextToken()
	for !p.curTokenIs(lexer.TOKEN_RBRACE) && !p.curTokenIs(lexer.TOKEN_EOF) {
		if p.curTokenIs(lexer.TOKEN_FN) {
			method := p.parseMethodDecl()
			if method != nil {
				stmt.Methods = append(stmt.Methods, method)
			}
		} else if p.curTokenIs(lexer.TOKEN_ASYNC) && p.peekTokenIs(lexer.TOKEN_FN) {
			p.nextToken()
			method := p.parseMethodDecl()
			if method != nil {
				method.Async = true
				stmt.Methods = append(stmt.Methods, method)
			}
		} else if p.curTokenIs(lexer.TOKEN_IDENT) {
			field := p.parseFieldDecl()
			if field != nil {
				stmt.Fields = append(stmt.Fields, field)
			}
		}
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseFieldDecl() *ast.FieldDecl {
	field := &ast.FieldDecl{Token: p.curToken}
	field.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(lexer.TOKEN_COLON) {
		return nil
	}
	p.nextToken()
	field.Type = p.parseTypeExpr()

	if p.peekTokenIs(lexer.TOKEN_ASSIGN) {
		p.nextToken()
		p.nextToken()
		field.Default = p.parseExpression(LOWEST)
	}

	return field
}

func (p *Parser) parseMethodDecl() *ast.MethodDecl {
	method := &ast.MethodDecl{Token: p.curToken}

	if !p.expectPeek(lexer.TOKEN_IDENT) {
		return nil
	}
	method.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(lexer.TOKEN_LPAREN) {
		return nil
	}

	method.Receiver, method.Parameters = p.parseReceiverAndParams()
	if method.Receiver == ast.RecvNone {
		method.Receiver = ast.RecvMutRef
	}

	if p.peekTokenIs(lexer.TOKEN_ARROW) {
		p.nextToken()
		p.nextToken()
		method.ReturnType = p.parseTypeExpr()
	}

	if !p.expectPeek(lexer.TOKEN_LBRACE) {
		return nil
	}
	method.Body = p.parseBlockStatement()

	return method
}

func (p *Parser) parseInterfaceStatement() *ast.InterfaceStatement {
	stmt := &ast.InterfaceStatement{Token: p.curToken}

	if !p.expectPeek(lexer.TOKEN_IDENT) {
		return nil
	}
	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(lexer.TOKEN_LBRACE) {
		return nil
	}

	stmt.Methods = []*ast.MethodSignature{}
	p.nextToken()

	for !p.curTokenIs(lexer.TOKEN_RBRACE) && !p.curTokenIs(lexer.TOKEN_EOF) {
		if p.curTokenIs(lexer.TOKEN_FN) {
			sig := p.parseMethodSignature()
			if sig != nil {
				stmt.Methods = append(stmt.Methods, sig)
			}
		}
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseMethodSignature() *ast.MethodSignature {
	sig := &ast.MethodSignature{Token: p.curToken}

	if !p.expectPeek(lexer.TOKEN_IDENT) {
		return nil
	}
	sig.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(lexer.TOKEN_LPAREN) {
		return nil
	}

	sig.Receiver, sig.Parameters = p.parseReceiverAndParams()

	if p.peekTokenIs(lexer.TOKEN_ARROW) {
		p.nextToken()
		p.nextToken()
		sig.ReturnType = p.parseTypeExpr()
	}

	if p.peekTokenIs(lexer.TOKEN_SEMI) {
		p.nextToken()
	}

	return sig
}

func (p *Parser) parseReceiverAndParams() (ast.ReceiverKind, []*ast.Parameter) {
	recv := ast.RecvNone
	params := []*ast.Parameter{}

	if p.peekTokenIs(lexer.TOKEN_RPAREN) {
		p.nextToken()
		return recv, params
	}

	p.nextToken()

	if p.curTokenIs(lexer.TOKEN_AMPERSAND) {
		if p.peekTokenIs(lexer.TOKEN_MUT) {
			p.nextToken()
			if p.peekTokenIs(lexer.TOKEN_SELF) {
				p.nextToken()
				recv = ast.RecvMutRef
				if p.peekTokenIs(lexer.TOKEN_COMMA) {
					p.nextToken()
					p.nextToken()
					params = p.parseRemainingParams()
				}
				if !p.expectPeek(lexer.TOKEN_RPAREN) {
					return recv, nil
				}
				return recv, params
			}
		} else if p.peekTokenIs(lexer.TOKEN_SELF) {
			p.nextToken()
			recv = ast.RecvRef
			if p.peekTokenIs(lexer.TOKEN_COMMA) {
				p.nextToken()
				p.nextToken()
				params = p.parseRemainingParams()
			}
			if !p.expectPeek(lexer.TOKEN_RPAREN) {
				return recv, nil
			}
			return recv, params
		}
	} else if p.curTokenIs(lexer.TOKEN_SELF) {
		recv = ast.RecvValue
		if p.peekTokenIs(lexer.TOKEN_COMMA) {
			p.nextToken()
			p.nextToken()
			params = p.parseRemainingParams()
		}
		if !p.expectPeek(lexer.TOKEN_RPAREN) {
			return recv, nil
		}
		return recv, params
	}

	param := p.parseParameter()
	if param != nil {
		params = append(params, param)
	}
	params = append(params, p.parseRemainingParams()...)

	if !p.expectPeek(lexer.TOKEN_RPAREN) {
		return recv, nil
	}
	return recv, params
}

func (p *Parser) parseRemainingParams() []*ast.Parameter {
	params := []*ast.Parameter{}
	param := p.parseParameter()
	if param != nil {
		params = append(params, param)
	}
	for p.peekTokenIs(lexer.TOKEN_COMMA) {
		p.nextToken()
		p.nextToken()
		param := p.parseParameter()
		if param != nil {
			params = append(params, param)
		}
	}
	return params
}

func (p *Parser) parseImplStatement() *ast.ImplStatement {
	stmt := &ast.ImplStatement{Token: p.curToken}

	if !p.expectPeek(lexer.TOKEN_IDENT) {
		return nil
	}
	ifaceName := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(lexer.TOKEN_FOR) {
		return nil
	}

	if !p.expectPeek(lexer.TOKEN_IDENT) {
		return nil
	}
	stmt.Type = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	stmt.Interface = ifaceName

	if !p.expectPeek(lexer.TOKEN_LBRACE) {
		return nil
	}

	stmt.Methods = []*ast.MethodDecl{}
	p.nextToken()

	for !p.curTokenIs(lexer.TOKEN_RBRACE) && !p.curTokenIs(lexer.TOKEN_EOF) {
		if p.curTokenIs(lexer.TOKEN_FN) {
			method := p.parseImplMethodDecl()
			if method != nil {
				stmt.Methods = append(stmt.Methods, method)
			}
		} else if p.curTokenIs(lexer.TOKEN_ASYNC) && p.peekTokenIs(lexer.TOKEN_FN) {
			p.nextToken()
			method := p.parseImplMethodDecl()
			if method != nil {
				method.Async = true
				stmt.Methods = append(stmt.Methods, method)
			}
		}
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseImplMethodDecl() *ast.MethodDecl {
	method := &ast.MethodDecl{Token: p.curToken}

	if !p.expectPeek(lexer.TOKEN_IDENT) {
		return nil
	}
	method.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(lexer.TOKEN_LPAREN) {
		return nil
	}

	method.Receiver, method.Parameters = p.parseReceiverAndParams()

	if p.peekTokenIs(lexer.TOKEN_ARROW) {
		p.nextToken()
		p.nextToken()
		method.ReturnType = p.parseTypeExpr()
	}

	if !p.expectPeek(lexer.TOKEN_LBRACE) {
		return nil
	}
	method.Body = p.parseBlockStatement()

	return method
}
