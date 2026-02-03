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
	lexer.TOKEN_LPAREN:       CALL,
	lexer.TOKEN_LBRACKET:     INDEX,
	lexer.TOKEN_DOT:          INDEX,
	lexer.TOKEN_QUESTION:     POSTFIX,
}

type Parser struct {
	l         *lexer.Lexer
	curToken  lexer.Token
	peekToken lexer.Token
	errors    []string

	prefixParseFns map[lexer.TokenType]prefixParseFn
	infixParseFns  map[lexer.TokenType]infixParseFn
}

type prefixParseFn func() ast.Expression
type infixParseFn func(ast.Expression) ast.Expression

func New(l *lexer.Lexer) *Parser {
	p := &Parser{l: l, errors: []string{}}

	p.prefixParseFns = make(map[lexer.TokenType]prefixParseFn)
	p.registerPrefix(lexer.TOKEN_IDENT, p.parseIdentifier)
	p.registerPrefix(lexer.TOKEN_INT, p.parseIntegerLiteral)
	p.registerPrefix(lexer.TOKEN_FLOAT, p.parseFloatLiteral)
	p.registerPrefix(lexer.TOKEN_STRING, p.parseStringLiteral)
	p.registerPrefix(lexer.TOKEN_CHAR, p.parseCharLiteral)
	p.registerPrefix(lexer.TOKEN_TRUE, p.parseBoolLiteral)
	p.registerPrefix(lexer.TOKEN_FALSE, p.parseBoolLiteral)
	p.registerPrefix(lexer.TOKEN_NIL, p.parseNilLiteral)
	p.registerPrefix(lexer.TOKEN_BANG, p.parsePrefixExpression)
	p.registerPrefix(lexer.TOKEN_MINUS, p.parsePrefixExpression)
	p.registerPrefix(lexer.TOKEN_TILDE, p.parsePrefixExpression)
	p.registerPrefix(lexer.TOKEN_AMPERSAND, p.parseBorrowExpression)
	p.registerPrefix(lexer.TOKEN_STAR, p.parseDerefExpression)
	p.registerPrefix(lexer.TOKEN_LPAREN, p.parseGroupedExpression)
	p.registerPrefix(lexer.TOKEN_LBRACKET, p.parseArrayLiteral)
	p.registerPrefix(lexer.TOKEN_IF, p.parseIfExpression)
	p.registerPrefix(lexer.TOKEN_FN, p.parseFunctionLiteral)
	p.registerPrefix(lexer.TOKEN_SPAWN, p.parseSpawnExpression)
	p.registerPrefix(lexer.TOKEN_NEW, p.parseNewExpression)
	p.registerPrefix(lexer.TOKEN_OK, p.parseOkExpression)
	p.registerPrefix(lexer.TOKEN_ERR, p.parseErrExpression)
	p.registerPrefix(lexer.TOKEN_MATCH, p.parseMatchExpression)
	p.registerPrefix(lexer.TOKEN_SELF, p.parseSelfExpression)
	p.registerPrefix(lexer.TOKEN_LBRACE, p.parseMapLiteral)
	p.registerPrefix(lexer.TOKEN_INTERP_STRING, p.parseInterpolatedString)
	p.registerPrefix(lexer.TOKEN_INT_TYPE, p.parseTypeAsIdentifier)
	p.registerPrefix(lexer.TOKEN_FLOAT_TYPE, p.parseTypeAsIdentifier)
	p.registerPrefix(lexer.TOKEN_BOOL_TYPE, p.parseTypeAsIdentifier)
	p.registerPrefix(lexer.TOKEN_STRING_TYPE, p.parseTypeAsIdentifier)
	p.registerPrefix(lexer.TOKEN_CHAR_TYPE, p.parseTypeAsIdentifier)

	p.infixParseFns = make(map[lexer.TokenType]infixParseFn)
	p.registerInfix(lexer.TOKEN_PLUS, p.parseInfixExpression)
	p.registerInfix(lexer.TOKEN_MINUS, p.parseInfixExpression)
	p.registerInfix(lexer.TOKEN_STAR, p.parseInfixExpression)
	p.registerInfix(lexer.TOKEN_SLASH, p.parseInfixExpression)
	p.registerInfix(lexer.TOKEN_PERCENT, p.parseInfixExpression)
	p.registerInfix(lexer.TOKEN_EQ, p.parseInfixExpression)
	p.registerInfix(lexer.TOKEN_NE, p.parseInfixExpression)
	p.registerInfix(lexer.TOKEN_LT, p.parseInfixExpression)
	p.registerInfix(lexer.TOKEN_LE, p.parseInfixExpression)
	p.registerInfix(lexer.TOKEN_GT, p.parseInfixExpression)
	p.registerInfix(lexer.TOKEN_GE, p.parseInfixExpression)
	p.registerInfix(lexer.TOKEN_AND, p.parseInfixExpression)
	p.registerInfix(lexer.TOKEN_OR, p.parseInfixExpression)
	p.registerInfix(lexer.TOKEN_AMPERSAND, p.parseInfixExpression)
	p.registerInfix(lexer.TOKEN_VBAR, p.parseInfixExpression)
	p.registerInfix(lexer.TOKEN_CARET, p.parseInfixExpression)
	p.registerInfix(lexer.TOKEN_LPAREN, p.parseCallExpression)
	p.registerInfix(lexer.TOKEN_LBRACKET, p.parseIndexExpression)
	p.registerInfix(lexer.TOKEN_DOT, p.parseMemberExpression)
	p.registerInfix(lexer.TOKEN_PIPE, p.parsePipeExpression)
	p.registerInfix(lexer.TOKEN_PIPE_BACK, p.parsePipeExpression)
	p.registerInfix(lexer.TOKEN_ASSIGN, p.parseAssignExpression)
	p.registerInfix(lexer.TOKEN_PLUS_EQ, p.parseAssignExpression)
	p.registerInfix(lexer.TOKEN_MINUS_EQ, p.parseAssignExpression)
	p.registerInfix(lexer.TOKEN_STAR_EQ, p.parseAssignExpression)
	p.registerInfix(lexer.TOKEN_SLASH_EQ, p.parseAssignExpression)
	p.registerInfix(lexer.TOKEN_PERCENT_EQ, p.parseAssignExpression)
	p.registerInfix(lexer.TOKEN_AMPERSAND_EQ, p.parseAssignExpression)
	p.registerInfix(lexer.TOKEN_VBAR_EQ, p.parseAssignExpression)
	p.registerInfix(lexer.TOKEN_CARET_EQ, p.parseAssignExpression)
	p.registerInfix(lexer.TOKEN_QUESTION, p.parseTryExpression)

	p.nextToken()
	p.nextToken()

	return p
}

func (p *Parser) registerPrefix(tokenType lexer.TokenType, fn prefixParseFn) {
	p.prefixParseFns[tokenType] = fn
}

func (p *Parser) registerInfix(tokenType lexer.TokenType, fn infixParseFn) {
	p.infixParseFns[tokenType] = fn
}

func (p *Parser) Errors() []string {
	return p.errors
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
	for p.peekToken.Type == lexer.TOKEN_COMMENT {
		p.peekToken = p.l.NextToken()
	}
}

func (p *Parser) curTokenIs(t lexer.TokenType) bool {
	return p.curToken.Type == t
}

func (p *Parser) peekTokenIs(t lexer.TokenType) bool {
	return p.peekToken.Type == t
}

func (p *Parser) expectPeek(t lexer.TokenType) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	}
	p.peekError(t)
	return false
}

func (p *Parser) peekError(t lexer.TokenType) {
	msg := fmt.Sprintf("line %d:%d: expected %s, got %s",
		p.peekToken.Line, p.peekToken.Column, t, p.peekToken.Type)
	p.errors = append(p.errors, msg)
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

func (p *Parser) ParseProgram() *ast.Program {
	program := &ast.Program{Statements: []ast.Statement{}}

	for !p.curTokenIs(lexer.TOKEN_EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		}
		p.nextToken()
	}

	return program
}

func (p *Parser) parseStatement() ast.Statement {
	var stmt ast.Statement
	switch p.curToken.Type {
	case lexer.TOKEN_PUB:
		stmt = p.parsePublicStatement()
	case lexer.TOKEN_LET:
		stmt = p.parseLetStatement()
	case lexer.TOKEN_MUT:
		stmt = p.parseLetStatement()
	case lexer.TOKEN_CONST:
		stmt = p.parseConstStatement()
	case lexer.TOKEN_RETURN:
		stmt = p.parseReturnStatement()
	case lexer.TOKEN_FN:
		stmt = p.parseFunctionStatement()
	case lexer.TOKEN_CLASS:
		stmt = p.parseClassStatement()
	case lexer.TOKEN_FOR:
		stmt = p.parseForStatement()
	case lexer.TOKEN_WHILE:
		stmt = p.parseWhileStatement()
	case lexer.TOKEN_BREAK:
		stmt = p.parseBreakStatement()
	case lexer.TOKEN_CONTINUE:
		stmt = p.parseContinueStatement()
	case lexer.TOKEN_REQUIRE:
		stmt = p.parseRequireStatement()
	case lexer.TOKEN_IF:
		expr := p.parseIfExpression()
		if expr != nil {
			stmt = &ast.ExpressionStatement{Token: p.curToken, Expression: expr}
			break
		}
		stmt = nil
	default:
		stmt = p.parseExpressionStatement()
	}

	if stmt == nil {
		p.synchronize()
	}

	return stmt
}

func (p *Parser) synchronize() {
	for !p.curTokenIs(lexer.TOKEN_SEMI) && !p.curTokenIs(lexer.TOKEN_RBRACE) && !p.curTokenIs(lexer.TOKEN_EOF) {
		p.nextToken()
	}
}

func (p *Parser) parsePublicStatement() ast.Statement {
	p.nextToken()
	switch p.curToken.Type {
	case lexer.TOKEN_FN:
		stmt := p.parseFunctionStatement()
		if stmt != nil {
			stmt.Public = true
		}
		return stmt
	case lexer.TOKEN_CLASS:
		stmt := p.parseClassStatement()
		if stmt != nil {
			stmt.Public = true
		}
		return stmt
	case lexer.TOKEN_CONST:
		stmt := p.parseConstStatement()
		if stmt != nil {
			stmt.Public = true
		}
		return stmt
	case lexer.TOKEN_LET:
		stmt := p.parseLetStatement()
		if stmt != nil {
			stmt.Public = true
		}
		return stmt
	default:
		p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected fn, class, const, or let after pub",
			p.curToken.Line, p.curToken.Column))
		return nil
	}
}

func (p *Parser) parseLetStatement() *ast.LetStatement {
	stmt := &ast.LetStatement{Token: p.curToken}

	if p.curTokenIs(lexer.TOKEN_MUT) {
		stmt.Mutable = true
		if !p.expectPeek(lexer.TOKEN_IDENT) {
			return nil
		}
	} else {
		if !p.expectPeek(lexer.TOKEN_IDENT) {
			return nil
		}
	}

	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if p.peekTokenIs(lexer.TOKEN_COLON) {
		p.nextToken()
		p.nextToken()
		stmt.Type = p.parseTypeExpr()
	}

	if !p.expectPeek(lexer.TOKEN_ASSIGN) {
		return nil
	}

	p.nextToken()
	stmt.Value = p.parseExpression(LOWEST)

	if !p.expectPeek(lexer.TOKEN_SEMI) {
		return nil
	}

	return stmt
}

func (p *Parser) parseConstStatement() *ast.ConstStatement {
	stmt := &ast.ConstStatement{Token: p.curToken}

	if !p.expectPeek(lexer.TOKEN_IDENT) {
		return nil
	}

	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if p.peekTokenIs(lexer.TOKEN_COLON) {
		p.nextToken()
		p.nextToken()
		stmt.Type = p.parseTypeExpr()
	}

	if !p.expectPeek(lexer.TOKEN_ASSIGN) {
		return nil
	}

	p.nextToken()
	stmt.Value = p.parseExpression(LOWEST)

	if !p.expectPeek(lexer.TOKEN_SEMI) {
		return nil
	}

	return stmt
}

func (p *Parser) parseReturnStatement() *ast.ReturnStatement {
	stmt := &ast.ReturnStatement{Token: p.curToken}

	if p.peekTokenIs(lexer.TOKEN_SEMI) {
		p.nextToken()
		return stmt
	}

	p.nextToken()
	stmt.ReturnValue = p.parseExpression(LOWEST)

	if !p.expectPeek(lexer.TOKEN_SEMI) {
		return nil
	}

	return stmt
}

func (p *Parser) parseExpressionStatement() *ast.ExpressionStatement {
	stmt := &ast.ExpressionStatement{Token: p.curToken}
	stmt.Expression = p.parseExpression(LOWEST)

	if !p.expectPeek(lexer.TOKEN_SEMI) {
		return nil
	}

	return stmt
}

func (p *Parser) parseFunctionStatement() *ast.FunctionStatement {
	stmt := &ast.FunctionStatement{Token: p.curToken}

	if !p.expectPeek(lexer.TOKEN_IDENT) {
		return nil
	}
	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(lexer.TOKEN_LPAREN) {
		return nil
	}
	stmt.Parameters = p.parseFunctionParameters()

	if p.peekTokenIs(lexer.TOKEN_ARROW) {
		p.nextToken()
		p.nextToken()
		stmt.ReturnType = p.parseTypeExpr()
	}

	if !p.expectPeek(lexer.TOKEN_LBRACE) {
		return nil
	}
	stmt.Body = p.parseBlockStatement()

	return stmt
}

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
	method.Parameters = p.parseFunctionParameters()

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

func (p *Parser) parseForStatement() ast.Statement {
	token := p.curToken
	p.nextToken()

	if p.curTokenIs(lexer.TOKEN_LBRACE) {
		body := p.parseBlockStatement()
		return &ast.LoopStatement{Token: token, Body: body}
	}

	if p.curTokenIs(lexer.TOKEN_IDENT) && p.peekTokenIs(lexer.TOKEN_IN) {
		return p.parseForInStatement(token)
	}

	return p.parseCStyleFor(token)
}

func (p *Parser) parseForInStatement(token lexer.Token) *ast.ForInStatement {
	stmt := &ast.ForInStatement{Token: token}
	stmt.Value = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(lexer.TOKEN_IN) {
		return nil
	}
	p.nextToken()

	stmt.Iterable = p.parseExpression(LOWEST)

	if !p.expectPeek(lexer.TOKEN_LBRACE) {
		return nil
	}
	stmt.Body = p.parseBlockStatement()

	return stmt
}

func (p *Parser) parseCStyleFor(token lexer.Token) *ast.ForStatement {
	stmt := &ast.ForStatement{Token: token}

	if !p.curTokenIs(lexer.TOKEN_LPAREN) {
		p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected ( in for loop", p.curToken.Line, p.curToken.Column))
		return nil
	}
	p.nextToken()

	if !p.curTokenIs(lexer.TOKEN_SEMI) {
		if p.curTokenIs(lexer.TOKEN_LET) || p.curTokenIs(lexer.TOKEN_MUT) {
			letStmt := p.parseLetStatement()
			if letStmt != nil {
				letStmt.Mutable = true
			}
			stmt.Init = letStmt
		} else {
			stmt.Init = p.parseExpressionStatement()
		}
	}
	if p.curTokenIs(lexer.TOKEN_SEMI) {
		p.nextToken()
	}

	if !p.curTokenIs(lexer.TOKEN_SEMI) {
		stmt.Condition = p.parseExpression(LOWEST)
	}
	if !p.expectPeek(lexer.TOKEN_SEMI) {
		return nil
	}
	p.nextToken()

	if !p.curTokenIs(lexer.TOKEN_RPAREN) {
		postExpr := p.parseExpression(LOWEST)
		stmt.Post = &ast.ExpressionStatement{Token: p.curToken, Expression: postExpr}
	}
	if !p.expectPeek(lexer.TOKEN_RPAREN) {
		return nil
	}

	if !p.expectPeek(lexer.TOKEN_LBRACE) {
		return nil
	}
	stmt.Body = p.parseBlockStatement()

	return stmt
}

func (p *Parser) parseWhileStatement() *ast.WhileStatement {
	stmt := &ast.WhileStatement{Token: p.curToken}

	p.nextToken()
	stmt.Condition = p.parseExpression(LOWEST)

	if !p.expectPeek(lexer.TOKEN_LBRACE) {
		return nil
	}
	stmt.Body = p.parseBlockStatement()

	return stmt
}

func (p *Parser) parseBreakStatement() *ast.BreakStatement {
	stmt := &ast.BreakStatement{Token: p.curToken}
	if !p.expectPeek(lexer.TOKEN_SEMI) {
		return nil
	}
	return stmt
}

func (p *Parser) parseContinueStatement() *ast.ContinueStatement {
	stmt := &ast.ContinueStatement{Token: p.curToken}
	if !p.expectPeek(lexer.TOKEN_SEMI) {
		return nil
	}
	return stmt
}

func (p *Parser) parseRequireStatement() *ast.RequireStatement {
	stmt := &ast.RequireStatement{Token: p.curToken}

	if p.peekTokenIs(lexer.TOKEN_STRING) {
		p.nextToken()
		stmt.Path = &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}

		if p.peekTokenIs(lexer.TOKEN_AS) {
			p.nextToken()
			if !p.expectPeek(lexer.TOKEN_IDENT) {
				return nil
			}
			stmt.Alias = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
		}

		if !p.expectPeek(lexer.TOKEN_SEMI) {
			return nil
		}
		return stmt
	}

	if p.peekTokenIs(lexer.TOKEN_LBRACE) {
		p.nextToken()
		stmt.Names = []*ast.Identifier{}

		if !p.peekTokenIs(lexer.TOKEN_RBRACE) {
			if !p.expectPeek(lexer.TOKEN_IDENT) {
				return nil
			}
			stmt.Names = append(stmt.Names, &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal})

			for p.peekTokenIs(lexer.TOKEN_COMMA) {
				p.nextToken()
				if !p.expectPeek(lexer.TOKEN_IDENT) {
					return nil
				}
				stmt.Names = append(stmt.Names, &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal})
			}
		}

		if !p.expectPeek(lexer.TOKEN_RBRACE) {
			return nil
		}

		if !p.expectPeek(lexer.TOKEN_FROM) {
			return nil
		}

		if !p.expectPeek(lexer.TOKEN_STRING) {
			return nil
		}
		stmt.Path = &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}

		if !p.expectPeek(lexer.TOKEN_SEMI) {
			return nil
		}
		return stmt
	}

	if p.peekTokenIs(lexer.TOKEN_STAR) {
		p.nextToken()
		stmt.All = true

		if !p.expectPeek(lexer.TOKEN_FROM) {
			return nil
		}

		if !p.expectPeek(lexer.TOKEN_STRING) {
			return nil
		}
		stmt.Path = &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}

		if !p.expectPeek(lexer.TOKEN_SEMI) {
			return nil
		}
		return stmt
	}

	p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected string, { or * after require",
		p.peekToken.Line, p.peekToken.Column))
	return nil
}

func (p *Parser) parseBlockStatement() *ast.BlockStatement {
	block := &ast.BlockStatement{Token: p.curToken, Statements: []ast.Statement{}}

	p.nextToken()

	for !p.curTokenIs(lexer.TOKEN_RBRACE) && !p.curTokenIs(lexer.TOKEN_EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}
		p.nextToken()
	}

	return block
}

func (p *Parser) parseExpression(prec precedence) ast.Expression {
	prefix := p.prefixParseFns[p.curToken.Type]
	if prefix == nil {
		p.errors = append(p.errors, fmt.Sprintf("line %d:%d: no prefix parse function for %s",
			p.curToken.Line, p.curToken.Column, p.curToken.Type))
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
		p.errors = append(p.errors, fmt.Sprintf("line %d:%d: could not parse %q as integer",
			p.curToken.Line, p.curToken.Column, p.curToken.Literal))
		return nil
	}
	lit.Value = value
	return lit
}

func (p *Parser) parseFloatLiteral() ast.Expression {
	lit := &ast.FloatLiteral{Token: p.curToken}
	value, err := strconv.ParseFloat(p.curToken.Literal, 64)
	if err != nil {
		p.errors = append(p.errors, fmt.Sprintf("line %d:%d: could not parse %q as float",
			p.curToken.Line, p.curToken.Column, p.curToken.Literal))
		return nil
	}
	lit.Value = value
	return lit
}

func (p *Parser) parseStringLiteral() ast.Expression {
	return &ast.StringLiteral{Token: p.curToken, Value: unescapeString(p.curToken.Literal)}
}

func unescapeString(s string) string {
	var result []byte
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case 'n':
				result = append(result, '\n')
			case 't':
				result = append(result, '\t')
			case 'r':
				result = append(result, '\r')
			case '\\':
				result = append(result, '\\')
			case '"':
				result = append(result, '"')
			case '0':
				result = append(result, '\x00')
			default:
				result = append(result, s[i+1])
			}
			i++
		} else {
			result = append(result, s[i])
		}
	}
	return string(result)
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
	expr := &ast.AssignExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
		Left:     left,
	}
	p.nextToken()
	expr.Right = p.parseExpression(LOWEST)
	return expr
}

func (p *Parser) parseGroupedExpression() ast.Expression {
	p.nextToken()
	exp := p.parseExpression(LOWEST)
	if !p.expectPeek(lexer.TOKEN_RPAREN) {
		return nil
	}
	return exp
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

		// Support "else if" chains
		if p.peekTokenIs(lexer.TOKEN_IF) {
			p.nextToken()
			nestedIf := p.parseIfExpression()
			if nestedIf == nil {
				return nil
			}
			expr.Alternative = &ast.BlockStatement{
				Token:      p.curToken,
				Statements: []ast.Statement{&ast.ExpressionStatement{Token: p.curToken, Expression: nestedIf}},
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
	param := &ast.Parameter{}

	if p.curTokenIs(lexer.TOKEN_MUT) {
		param.Mutable = true
		p.nextToken()
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

func (p *Parser) parseNewExpression() ast.Expression {
	exp := &ast.NewExpression{Token: p.curToken}
	p.nextToken()
	exp.Type = p.parseTypeExpr()
	return exp
}

func (p *Parser) parseBorrowExpression() ast.Expression {
	expr := &ast.BorrowExpression{Token: p.curToken}
	if p.peekTokenIs(lexer.TOKEN_MUT) {
		expr.Mutable = true
		p.nextToken()
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
	p.nextToken()

	for !p.curTokenIs(lexer.TOKEN_RBRACE) && !p.curTokenIs(lexer.TOKEN_EOF) {
		arm := p.parseMatchArm()
		if arm != nil {
			expr.Arms = append(expr.Arms, arm)
		}
		p.nextToken()
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

func (p *Parser) parseMapLiteral() ast.Expression {
	mapLit := &ast.MapLiteral{Token: p.curToken}
	mapLit.Pairs = make(map[ast.Expression]ast.Expression)

	if p.peekTokenIs(lexer.TOKEN_RBRACE) {
		p.nextToken()
		return mapLit
	}

	p.nextToken()
	key := p.parseExpression(LOWEST)

	if !p.expectPeek(lexer.TOKEN_COLON) {
		return nil
	}

	p.nextToken()
	value := p.parseExpression(LOWEST)
	mapLit.Pairs[key] = value

	for p.peekTokenIs(lexer.TOKEN_COMMA) {
		p.nextToken()
		p.nextToken()

		key := p.parseExpression(LOWEST)

		if !p.expectPeek(lexer.TOKEN_COLON) {
			return nil
		}

		p.nextToken()
		value := p.parseExpression(LOWEST)
		mapLit.Pairs[key] = value
	}

	if !p.expectPeek(lexer.TOKEN_RBRACE) {
		return nil
	}

	return mapLit
}
