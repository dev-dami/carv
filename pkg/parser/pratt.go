package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/dev-dami/carv/pkg/ast"
	"github.com/dev-dami/carv/pkg/lexer"
)

type precedence int

const (
	_ precedence = iota
	LOWEST
	PIPE
	ASSIGN
	OR
	AND
	EQUALS
	LESSGREATER
	BITOR
	BITXOR
	BITAND
	SHIFT
	SUM
	PRODUCT
	CAST
	PREFIX
	CALL
	INDEX
	POSTFIX
)

var precedences = map[lexer.TokenType]precedence{
	lexer.TOKEN_PIPE:         PIPE,
	lexer.TOKEN_PIPE_BACK:    PIPE,
	lexer.TOKEN_ASSIGN:       ASSIGN,
	lexer.TOKEN_PLUS_EQ:      ASSIGN,
	lexer.TOKEN_MINUS_EQ:     ASSIGN,
	lexer.TOKEN_STAR_EQ:      ASSIGN,
	lexer.TOKEN_SLASH_EQ:     ASSIGN,
	lexer.TOKEN_PERCENT_EQ:   ASSIGN,
	lexer.TOKEN_AMPERSAND_EQ: ASSIGN,
	lexer.TOKEN_VBAR_EQ:      ASSIGN,
	lexer.TOKEN_CARET_EQ:     ASSIGN,
	lexer.TOKEN_OR:           OR,
	lexer.TOKEN_AND:          AND,
	lexer.TOKEN_EQ:           EQUALS,
	lexer.TOKEN_NE:           EQUALS,
	lexer.TOKEN_LT:           LESSGREATER,
	lexer.TOKEN_LE:           LESSGREATER,
	lexer.TOKEN_GT:           LESSGREATER,
	lexer.TOKEN_GE:           LESSGREATER,
	lexer.TOKEN_VBAR:         BITOR,
	lexer.TOKEN_CARET:        BITXOR,
	lexer.TOKEN_AMPERSAND:    BITAND,
	lexer.TOKEN_PLUS:         SUM,
	lexer.TOKEN_MINUS:        SUM,
	lexer.TOKEN_STAR:         PRODUCT,
	lexer.TOKEN_SLASH:        PRODUCT,
	lexer.TOKEN_PERCENT:      PRODUCT,
	lexer.TOKEN_AS:           CAST,
	lexer.TOKEN_LPAREN:       CALL,
	lexer.TOKEN_LBRACKET:     INDEX,
	lexer.TOKEN_DOT:          INDEX,
	lexer.TOKEN_QUESTION:     POSTFIX,
}

func (p *Parser) curPrecedence() precedence {
	if p, ok := precedences[p.curToken.Type]; ok {
		return p
	}
	return LOWEST
}

func (p *Parser) peekPrecedence() precedence {
	if p, ok := precedences[p.peekToken.Type]; ok {
		return p
	}
	return LOWEST
}

func (p *Parser) parseExpression(prec precedence) ast.Expression {
	prefix := p.prefixParseFns[p.curToken.Type]
	if prefix == nil {
		p.errors = append(p.errors, fmt.Sprintf("no prefix parse function for %s", p.curToken.Type))
		return nil
	}
	leftExp := prefix()

	for !p.peekTokenIs(lexer.TOKEN_SEMI) && prec < p.peekPrecedence() {
		infix := p.infixParseFns[p.peekToken.Type]
		if infix == nil {
			return leftExp
		}
		p.nextToken()
		leftExp = infix(leftExp)
	}

	return leftExp
}

func (p *Parser) parseIdentifier() ast.Expression {
	return &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseTypeAsIdentifier() ast.Expression {
	tok := p.curToken
	tok.Type = lexer.TOKEN_IDENT
	return &ast.Identifier{Token: tok, Value: tok.Literal}
}

func (p *Parser) parseIntegerLiteral() ast.Expression {
	lit := &ast.IntegerLiteral{Token: p.curToken}
	value, err := strconv.ParseInt(p.curToken.Literal, 0, 64)
	if err != nil {
		p.errors = append(p.errors, "line "+strconv.Itoa(p.curToken.Line)+":"+strconv.Itoa(p.curToken.Column)+": could not parse \""+p.curToken.Literal+"\" as integer")
		return nil
	}
	lit.Value = value
	return lit
}

func (p *Parser) parseFloatLiteral() ast.Expression {
	lit := &ast.FloatLiteral{Token: p.curToken}
	value, err := strconv.ParseFloat(p.curToken.Literal, 64)
	if err != nil {
		p.errors = append(p.errors, "line "+strconv.Itoa(p.curToken.Line)+":"+strconv.Itoa(p.curToken.Column)+": could not parse \""+p.curToken.Literal+"\" as float")
		return nil
	}
	lit.Value = value
	return lit
}

func (p *Parser) parseStringLiteral() ast.Expression {
	return &ast.StringLiteral{Token: p.curToken, Value: unescapeString(p.curToken.Literal)}
}

func unescapeString(s string) string {
	unquoted, err := strconv.Unquote(`"` + s + `"`)
	if err == nil {
		return unquoted
	}

	var result strings.Builder
	result.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case 'n':
				result.WriteByte('\n')
			case 't':
				result.WriteByte('\t')
			case 'r':
				result.WriteByte('\r')
			case '\\':
				result.WriteByte('\\')
			case '"':
				result.WriteByte('"')
			case '0':
				result.WriteByte('\x00')
			default:
				result.WriteByte('\\')
				result.WriteByte(s[i+1])
			}
			i++
			continue
		}
		result.WriteByte(s[i])
	}
	return result.String()
}

func (p *Parser) parseCharLiteral() ast.Expression {
	lit := p.curToken.Literal
	var r rune
	if len(lit) > 0 {
		if lit[0] == '\\' && len(lit) > 1 {
			switch lit[1] {
			case 'n':
				r = '\n'
			case 't':
				r = '\t'
			case 'r':
				r = '\r'
			case '\\':
				r = '\\'
			case '\'':
				r = '\''
			case '0':
				r = '\x00'
			default:
				r = rune(lit[1])
			}
		} else {
			r = rune(lit[0])
		}
	}
	return &ast.CharLiteral{Token: p.curToken, Value: r}
}

func (p *Parser) parseBoolLiteral() ast.Expression {
	return &ast.BoolLiteral{Token: p.curToken, Value: p.curTokenIs(lexer.TOKEN_TRUE)}
}

func (p *Parser) parseNilLiteral() ast.Expression {
	return &ast.NilLiteral{Token: p.curToken}
}

func (p *Parser) parsePrefixExpression() ast.Expression {
	expr := &ast.PrefixExpression{Token: p.curToken, Operator: p.curToken.Literal}
	p.nextToken()
	expr.Right = p.parseExpression(PREFIX)
	return expr
}

func (p *Parser) parseInfixExpression(left ast.Expression) ast.Expression {
	expr := &ast.InfixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
		Left:     left,
	}
	prec := p.curPrecedence()
	p.nextToken()
	expr.Right = p.parseExpression(prec)
	return expr
}

func (p *Parser) parsePipeExpression(left ast.Expression) ast.Expression {
	expr := &ast.PipeExpression{Token: p.curToken, Left: left}
	prec := p.curPrecedence()
	p.nextToken()
	expr.Right = p.parseExpression(prec)
	return expr
}

func (p *Parser) parseAssignExpression(left ast.Expression) ast.Expression {
	expr := &ast.AssignExpression{Token: p.curToken, Left: left, Operator: p.curToken.Literal}
	prec := p.curPrecedence() - 1
	p.nextToken()
	expr.Right = p.parseExpression(prec)
	return expr
}

func (p *Parser) parseGroupedExpression() ast.Expression {
	p.nextToken()
	expr := p.parseExpression(LOWEST)
	if !p.expectPeek(lexer.TOKEN_RPAREN) {
		return nil
	}
	return expr
}

func (p *Parser) parseArrayLiteral() ast.Expression {
	array := &ast.ArrayLiteral{Token: p.curToken}
	array.Elements = p.parseExpressionList(lexer.TOKEN_RBRACKET)
	return array
}

func (p *Parser) parseExpressionList(end lexer.TokenType) []ast.Expression {
	list := []ast.Expression{}

	if p.peekTokenIs(end) {
		p.nextToken()
		return list
	}

	p.nextToken()
	list = append(list, p.parseExpression(LOWEST))

	for p.peekTokenIs(lexer.TOKEN_COMMA) {
		p.nextToken()
		p.nextToken()
		list = append(list, p.parseExpression(LOWEST))
	}

	if !p.expectPeek(end) {
		return nil
	}

	return list
}

func (p *Parser) parseIfExpression() ast.Expression {
	expr := &ast.IfExpression{Token: p.curToken}

	p.nextToken()
	expr.Condition = p.parseExpression(LOWEST)

	if !p.expectPeek(lexer.TOKEN_LBRACE) {
		return nil
	}
	expr.Consequence = p.parseBlockStatement()

	if p.peekTokenIs(lexer.TOKEN_ELSE) {
		p.nextToken()
		if p.peekTokenIs(lexer.TOKEN_IF) {
			p.nextToken()
			nested := p.parseIfExpression()
			if nested != nil {
				nestedStmt := &ast.ExpressionStatement{Token: p.curToken, Expression: nested}
				expr.Alternative = &ast.BlockStatement{Token: p.curToken, Statements: []ast.Statement{nestedStmt}}
			}
		} else {
			if !p.expectPeek(lexer.TOKEN_LBRACE) {
				return nil
			}
			expr.Alternative = p.parseBlockStatement()
		}
	}

	return expr
}

func (p *Parser) parseFunctionLiteral() ast.Expression {
	lit := &ast.FunctionLiteral{Token: p.curToken}

	if p.peekTokenIs(lexer.TOKEN_IDENT) {
		p.nextToken()
		lit.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	}

	if !p.expectPeek(lexer.TOKEN_LPAREN) {
		return nil
	}
	lit.Parameters = p.parseFunctionParameters()

	if p.peekTokenIs(lexer.TOKEN_ARROW) {
		p.nextToken()
		p.nextToken()
		lit.ReturnType = p.parseTypeExpr()
	}

	if !p.expectPeek(lexer.TOKEN_LBRACE) {
		return nil
	}
	lit.Body = p.parseBlockStatement()

	return lit
}

func (p *Parser) parseFunctionParameters() []*ast.Parameter {
	params := []*ast.Parameter{}

	if p.peekTokenIs(lexer.TOKEN_RPAREN) {
		p.nextToken()
		return params
	}

	p.nextToken()
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

	if !p.expectPeek(lexer.TOKEN_RPAREN) {
		return nil
	}

	return params
}

func (p *Parser) parseParameter() *ast.Parameter {
	param := &ast.Parameter{Token: p.curToken}

	if p.curTokenIs(lexer.TOKEN_MUT) {
		param.Mutable = true
		p.nextToken()
		param.Token = p.curToken
	}

	if !p.curTokenIs(lexer.TOKEN_IDENT) {
		return nil
	}
	param.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if p.peekTokenIs(lexer.TOKEN_COLON) {
		p.nextToken()
		p.nextToken()
		param.Type = p.parseTypeExpr()
	}

	return param
}

func (p *Parser) parseCallExpression(function ast.Expression) ast.Expression {
	exp := &ast.CallExpression{Token: p.curToken, Function: function}
	exp.Arguments = p.parseExpressionList(lexer.TOKEN_RPAREN)
	return exp
}

func (p *Parser) parseIndexExpression(left ast.Expression) ast.Expression {
	exp := &ast.IndexExpression{Token: p.curToken, Left: left}
	p.nextToken()
	exp.Index = p.parseExpression(LOWEST)
	if !p.expectPeek(lexer.TOKEN_RBRACKET) {
		return nil
	}
	return exp
}

func (p *Parser) parseMemberExpression(left ast.Expression) ast.Expression {
	exp := &ast.MemberExpression{Token: p.curToken, Object: left}
	if !p.expectPeek(lexer.TOKEN_IDENT) {
		return nil
	}
	exp.Member = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	return exp
}

func (p *Parser) parseSpawnExpression() ast.Expression {
	exp := &ast.SpawnExpression{Token: p.curToken}
	if !p.expectPeek(lexer.TOKEN_LBRACE) {
		return nil
	}
	exp.Body = p.parseBlockStatement()
	return exp
}

func (p *Parser) parseAwaitExpression() ast.Expression {
	expr := &ast.AwaitExpression{Token: p.curToken}
	p.nextToken()
	expr.Value = p.parseExpression(PREFIX)
	return expr
}

func (p *Parser) parseNewExpression() ast.Expression {
	expr := &ast.NewExpression{Token: p.curToken}
	p.nextToken()
	expr.Type = p.parseTypeExpr()
	return expr
}

func (p *Parser) parseBorrowExpression() ast.Expression {
	expr := &ast.BorrowExpression{Token: p.curToken}
	if p.peekTokenIs(lexer.TOKEN_MUT) {
		p.nextToken()
		expr.Mutable = true
	}
	p.nextToken()
	expr.Value = p.parseExpression(PREFIX)
	return expr
}

func (p *Parser) parseDerefExpression() ast.Expression {
	expr := &ast.DerefExpression{Token: p.curToken}
	p.nextToken()
	expr.Value = p.parseExpression(PREFIX)
	return expr
}

func (p *Parser) parseOkExpression() ast.Expression {
	expr := &ast.OkExpression{Token: p.curToken}
	if !p.expectPeek(lexer.TOKEN_LPAREN) {
		return nil
	}
	p.nextToken()
	expr.Value = p.parseExpression(LOWEST)
	if !p.expectPeek(lexer.TOKEN_RPAREN) {
		return nil
	}
	return expr
}

func (p *Parser) parseErrExpression() ast.Expression {
	expr := &ast.ErrExpression{Token: p.curToken}
	if !p.expectPeek(lexer.TOKEN_LPAREN) {
		return nil
	}
	p.nextToken()
	expr.Value = p.parseExpression(LOWEST)
	if !p.expectPeek(lexer.TOKEN_RPAREN) {
		return nil
	}
	return expr
}

func (p *Parser) parseTryExpression(left ast.Expression) ast.Expression {
	return &ast.TryExpression{Token: p.curToken, Value: left}
}

func (p *Parser) parseMatchExpression() ast.Expression {
	expr := &ast.MatchExpression{Token: p.curToken}
	p.nextToken()
	expr.Value = p.parseExpression(LOWEST)

	if !p.expectPeek(lexer.TOKEN_LBRACE) {
		return nil
	}

	expr.Arms = []*ast.MatchArm{}
	for !p.peekTokenIs(lexer.TOKEN_RBRACE) {
		p.nextToken()
		arm := p.parseMatchArm()
		if arm != nil {
			expr.Arms = append(expr.Arms, arm)
		}
	}

	if !p.expectPeek(lexer.TOKEN_RBRACE) {
		return nil
	}

	return expr
}

func (p *Parser) parseMatchArm() *ast.MatchArm {
	arm := &ast.MatchArm{Token: p.curToken}
	arm.Pattern = p.parseExpression(LOWEST)

	if !p.expectPeek(lexer.TOKEN_FAT_ARROW) {
		return nil
	}

	p.nextToken()
	if p.curTokenIs(lexer.TOKEN_LBRACE) {
		block := p.parseBlockStatement()
		arm.Body = &ast.BlockExpression{Token: p.curToken, Block: block}
	} else {
		arm.Body = p.parseExpression(LOWEST)
	}

	if p.peekTokenIs(lexer.TOKEN_COMMA) {
		p.nextToken()
	}

	return arm
}

func (p *Parser) parseSelfExpression() ast.Expression {
	return &ast.Identifier{Token: p.curToken, Value: "self"}
}

func (p *Parser) parseInterpolatedString() ast.Expression {
	expr := &ast.InterpolatedString{Token: p.curToken}
	expr.Parts = []ast.Expression{}

	str := p.curToken.Literal
	var current strings.Builder
	i := 0

	for i < len(str) {
		if str[i] == '{' && i+1 < len(str) && str[i+1] != '{' {
			if current.Len() > 0 {
				expr.Parts = append(expr.Parts, &ast.StringLiteral{
					Token: p.curToken,
					Value: current.String(),
				})
				current.Reset()
			}

			end := i + 1
			braceCount := 1
			for end < len(str) && braceCount > 0 {
				if str[end] == '{' {
					braceCount++
				} else if str[end] == '}' {
					braceCount--
				}
				end++
			}

			exprStr := str[i+1 : end-1]
			exprLexer := lexer.New(exprStr)
			exprParser := New(exprLexer)
			parsedExpr := exprParser.parseExpression(LOWEST)

			if len(exprParser.Errors()) > 0 {
				p.errors = append(p.errors, exprParser.Errors()...)
			}

			if parsedExpr != nil {
				expr.Parts = append(expr.Parts, parsedExpr)
			}

			i = end
		} else if str[i] == '{' && i+1 < len(str) && str[i+1] == '{' {
			current.WriteByte('{')
			i += 2
		} else if str[i] == '}' && i+1 < len(str) && str[i+1] == '}' {
			current.WriteByte('}')
			i += 2
		} else {
			current.WriteByte(str[i])
			i++
		}
	}

	if current.Len() > 0 {
		expr.Parts = append(expr.Parts, &ast.StringLiteral{
			Token: p.curToken,
			Value: current.String(),
		})
	}

	return expr
}

func (p *Parser) parseCastExpression(left ast.Expression) ast.Expression {
	expr := &ast.CastExpression{Token: p.curToken, Value: left}
	p.nextToken()
	expr.Type = p.parseTypeExpr()
	return expr
}

func (p *Parser) parseMapLiteral() ast.Expression {
	mapLit := &ast.MapLiteral{Token: p.curToken}
	mapLit.Pairs = make(map[ast.Expression]ast.Expression)

	for !p.peekTokenIs(lexer.TOKEN_RBRACE) {
		p.nextToken()
		key := p.parseExpression(LOWEST)

		if !p.expectPeek(lexer.TOKEN_COLON) {
			return nil
		}

		p.nextToken()
		value := p.parseExpression(LOWEST)
		mapLit.Pairs[key] = value

		if !p.peekTokenIs(lexer.TOKEN_RBRACE) && !p.expectPeek(lexer.TOKEN_COMMA) {
			return nil
		}
	}

	if !p.expectPeek(lexer.TOKEN_RBRACE) {
		return nil
	}

	return mapLit
}
