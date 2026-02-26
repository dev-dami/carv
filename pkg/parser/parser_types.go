package parser

import (
	"github.com/dev-dami/carv/pkg/ast"
	"github.com/dev-dami/carv/pkg/lexer"
)

func (p *Parser) parseTypeExpr() ast.TypeExpr {
	if p.curTokenIs(lexer.TOKEN_AMPERSAND) {
		ref := &ast.RefType{Token: p.curToken}
		p.nextToken()
		if p.curTokenIs(lexer.TOKEN_MUT) {
			ref.Mutable = true
			p.nextToken()
		}
		ref.Inner = p.parseTypeExpr()
		return ref
	}
	switch p.curToken.Type {
	case lexer.TOKEN_INT_TYPE:
		return &ast.BasicType{Token: p.curToken, Name: "int"}
	case lexer.TOKEN_FLOAT_TYPE:
		return &ast.BasicType{Token: p.curToken, Name: "float"}
	case lexer.TOKEN_BOOL_TYPE:
		return &ast.BasicType{Token: p.curToken, Name: "bool"}
	case lexer.TOKEN_STRING_TYPE:
		return &ast.BasicType{Token: p.curToken, Name: "string"}
	case lexer.TOKEN_CHAR_TYPE:
		return &ast.BasicType{Token: p.curToken, Name: "char"}
	case lexer.TOKEN_VOID_TYPE:
		return &ast.BasicType{Token: p.curToken, Name: "void"}
	case lexer.TOKEN_ANY_TYPE:
		return &ast.BasicType{Token: p.curToken, Name: "any"}
	case lexer.TOKEN_IDENT:
		return &ast.NamedType{Token: p.curToken, Name: &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}}
	case lexer.TOKEN_LBRACKET:
		return p.parseArrayType()
	default:
		return nil
	}
}

func (p *Parser) parseArrayType() ast.TypeExpr {
	arr := &ast.ArrayType{Token: p.curToken}
	p.nextToken()
	if !p.curTokenIs(lexer.TOKEN_RBRACKET) {
		arr.Size = p.parseExpression(LOWEST)
		if !p.expectPeek(lexer.TOKEN_RBRACKET) {
			return nil
		}
	}
	p.nextToken()
	arr.ElementType = p.parseTypeExpr()
	return arr
}
