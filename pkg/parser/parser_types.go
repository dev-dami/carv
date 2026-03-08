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
	case lexer.TOKEN_VOLATILE:
		vt := &ast.VolatileType{Token: p.curToken}
		if !p.expectPeek(lexer.TOKEN_LT) {
			return nil
		}
		p.nextToken()
		vt.Inner = p.parseTypeExpr()
		if !p.expectPeek(lexer.TOKEN_GT) {
			return nil
		}
		return vt
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
	case lexer.TOKEN_U8_TYPE:
		return &ast.BasicType{Token: p.curToken, Name: "u8"}
	case lexer.TOKEN_U16_TYPE:
		return &ast.BasicType{Token: p.curToken, Name: "u16"}
	case lexer.TOKEN_U32_TYPE:
		return &ast.BasicType{Token: p.curToken, Name: "u32"}
	case lexer.TOKEN_U64_TYPE:
		return &ast.BasicType{Token: p.curToken, Name: "u64"}
	case lexer.TOKEN_I8_TYPE:
		return &ast.BasicType{Token: p.curToken, Name: "i8"}
	case lexer.TOKEN_I16_TYPE:
		return &ast.BasicType{Token: p.curToken, Name: "i16"}
	case lexer.TOKEN_I32_TYPE:
		return &ast.BasicType{Token: p.curToken, Name: "i32"}
	case lexer.TOKEN_I64_TYPE:
		return &ast.BasicType{Token: p.curToken, Name: "i64"}
	case lexer.TOKEN_F32_TYPE:
		return &ast.BasicType{Token: p.curToken, Name: "f32"}
	case lexer.TOKEN_F64_TYPE:
		return &ast.BasicType{Token: p.curToken, Name: "f64"}
	case lexer.TOKEN_USIZE_TYPE:
		return &ast.BasicType{Token: p.curToken, Name: "usize"}
	case lexer.TOKEN_ISIZE_TYPE:
		return &ast.BasicType{Token: p.curToken, Name: "isize"}
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
