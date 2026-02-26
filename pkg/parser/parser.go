package parser

import (
	"fmt"

	"github.com/dev-dami/carv/pkg/ast"
	"github.com/dev-dami/carv/pkg/lexer"
)

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
	p.registerPrefix(lexer.TOKEN_AWAIT, p.parseAwaitExpression)
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
	p.registerInfix(lexer.TOKEN_AS, p.parseCastExpression)

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
	case lexer.TOKEN_ASYNC:
		p.nextToken()
		if !p.curTokenIs(lexer.TOKEN_FN) {
			p.peekError(lexer.TOKEN_FN)
			return nil
		}
		fnStmt := p.parseFunctionStatement()
		if fnStmt != nil {
			fnStmt.Async = true
		}
		stmt = fnStmt
	case lexer.TOKEN_CLASS:
		stmt = p.parseClassStatement()
	case lexer.TOKEN_INTERFACE:
		stmt = p.parseInterfaceStatement()
	case lexer.TOKEN_IMPL:
		stmt = p.parseImplStatement()
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
	case lexer.TOKEN_ASYNC:
		p.nextToken()
		if !p.curTokenIs(lexer.TOKEN_FN) {
			p.peekError(lexer.TOKEN_FN)
			return nil
		}
		stmt := p.parseFunctionStatement()
		if stmt != nil {
			stmt.Public = true
			stmt.Async = true
		}
		return stmt
	case lexer.TOKEN_CLASS:
		stmt := p.parseClassStatement()
		if stmt != nil {
			stmt.Public = true
		}
		return stmt
	case lexer.TOKEN_INTERFACE:
		stmt := p.parseInterfaceStatement()
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
