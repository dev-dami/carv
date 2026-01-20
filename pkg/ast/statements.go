package ast

import "github.com/dev-dami/carv/pkg/lexer"

type LetStatement struct {
	Token   lexer.Token
	Name    *Identifier
	Type    TypeExpr
	Value   Expression
	Mutable bool
	Public  bool
}

func (ls *LetStatement) statementNode()       {}
func (ls *LetStatement) TokenLiteral() string { return ls.Token.Literal }
func (ls *LetStatement) Pos() (int, int)      { return ls.Token.Line, ls.Token.Column }

type ConstStatement struct {
	Token  lexer.Token
	Name   *Identifier
	Type   TypeExpr
	Value  Expression
	Public bool
}

func (cs *ConstStatement) statementNode()       {}
func (cs *ConstStatement) TokenLiteral() string { return cs.Token.Literal }
func (cs *ConstStatement) Pos() (int, int)      { return cs.Token.Line, cs.Token.Column }

type ReturnStatement struct {
	Token       lexer.Token
	ReturnValue Expression
}

func (rs *ReturnStatement) statementNode()       {}
func (rs *ReturnStatement) TokenLiteral() string { return rs.Token.Literal }
func (rs *ReturnStatement) Pos() (int, int)      { return rs.Token.Line, rs.Token.Column }

type ExpressionStatement struct {
	Token      lexer.Token
	Expression Expression
}

func (es *ExpressionStatement) statementNode()       {}
func (es *ExpressionStatement) TokenLiteral() string { return es.Token.Literal }
func (es *ExpressionStatement) Pos() (int, int)      { return es.Token.Line, es.Token.Column }

type BlockStatement struct {
	Token      lexer.Token
	Statements []Statement
}

func (bs *BlockStatement) statementNode()       {}
func (bs *BlockStatement) TokenLiteral() string { return bs.Token.Literal }
func (bs *BlockStatement) Pos() (int, int)      { return bs.Token.Line, bs.Token.Column }

type ForStatement struct {
	Token     lexer.Token
	Init      Statement
	Condition Expression
	Post      Statement
	Body      *BlockStatement
}

func (fs *ForStatement) statementNode()       {}
func (fs *ForStatement) TokenLiteral() string { return fs.Token.Literal }
func (fs *ForStatement) Pos() (int, int)      { return fs.Token.Line, fs.Token.Column }

type ForInStatement struct {
	Token    lexer.Token
	Key      *Identifier
	Value    *Identifier
	Iterable Expression
	Body     *BlockStatement
}

func (fis *ForInStatement) statementNode()       {}
func (fis *ForInStatement) TokenLiteral() string { return fis.Token.Literal }
func (fis *ForInStatement) Pos() (int, int)      { return fis.Token.Line, fis.Token.Column }

type WhileStatement struct {
	Token     lexer.Token
	Condition Expression
	Body      *BlockStatement
}

func (ws *WhileStatement) statementNode()       {}
func (ws *WhileStatement) TokenLiteral() string { return ws.Token.Literal }
func (ws *WhileStatement) Pos() (int, int)      { return ws.Token.Line, ws.Token.Column }

type LoopStatement struct {
	Token lexer.Token
	Body  *BlockStatement
}

func (ls *LoopStatement) statementNode()       {}
func (ls *LoopStatement) TokenLiteral() string { return ls.Token.Literal }
func (ls *LoopStatement) Pos() (int, int)      { return ls.Token.Line, ls.Token.Column }

type BreakStatement struct {
	Token lexer.Token
}

func (bs *BreakStatement) statementNode()       {}
func (bs *BreakStatement) TokenLiteral() string { return bs.Token.Literal }
func (bs *BreakStatement) Pos() (int, int)      { return bs.Token.Line, bs.Token.Column }

type ContinueStatement struct {
	Token lexer.Token
}

func (cs *ContinueStatement) statementNode()       {}
func (cs *ContinueStatement) TokenLiteral() string { return cs.Token.Literal }
func (cs *ContinueStatement) Pos() (int, int)      { return cs.Token.Line, cs.Token.Column }

type FunctionStatement struct {
	Token      lexer.Token
	Name       *Identifier
	Parameters []*Parameter
	ReturnType TypeExpr
	Body       *BlockStatement
	Public     bool
}

func (fs *FunctionStatement) statementNode()       {}
func (fs *FunctionStatement) TokenLiteral() string { return fs.Token.Literal }
func (fs *FunctionStatement) Pos() (int, int)      { return fs.Token.Line, fs.Token.Column }

type ClassStatement struct {
	Token      lexer.Token
	Name       *Identifier
	Fields     []*FieldDecl
	Methods    []*MethodDecl
	Implements []*Identifier
	Public     bool
}

func (cs *ClassStatement) statementNode()       {}
func (cs *ClassStatement) TokenLiteral() string { return cs.Token.Literal }
func (cs *ClassStatement) Pos() (int, int)      { return cs.Token.Line, cs.Token.Column }

type FieldDecl struct {
	Token   lexer.Token
	Name    *Identifier
	Type    TypeExpr
	Default Expression
	Public  bool
	Static  bool
}

type MethodDecl struct {
	Token      lexer.Token
	Name       *Identifier
	Parameters []*Parameter
	ReturnType TypeExpr
	Body       *BlockStatement
	Public     bool
	Static     bool
}

type InterfaceStatement struct {
	Token   lexer.Token
	Name    *Identifier
	Methods []*MethodSignature
	Public  bool
}

func (is *InterfaceStatement) statementNode()       {}
func (is *InterfaceStatement) TokenLiteral() string { return is.Token.Literal }
func (is *InterfaceStatement) Pos() (int, int)      { return is.Token.Line, is.Token.Column }

type MethodSignature struct {
	Token      lexer.Token
	Name       *Identifier
	Parameters []*Parameter
	ReturnType TypeExpr
}

type ImplStatement struct {
	Token     lexer.Token
	Type      *Identifier
	Interface *Identifier
	Methods   []*MethodDecl
}

func (is *ImplStatement) statementNode()       {}
func (is *ImplStatement) TokenLiteral() string { return is.Token.Literal }
func (is *ImplStatement) Pos() (int, int)      { return is.Token.Line, is.Token.Column }

type TypeAliasStatement struct {
	Token  lexer.Token
	Name   *Identifier
	Type   TypeExpr
	Public bool
}

func (tas *TypeAliasStatement) statementNode()       {}
func (tas *TypeAliasStatement) TokenLiteral() string { return tas.Token.Literal }
func (tas *TypeAliasStatement) Pos() (int, int)      { return tas.Token.Line, tas.Token.Column }

type ImportStatement struct {
	Token lexer.Token
	Path  *StringLiteral
	Alias *Identifier
	Names []*Identifier
}

func (is *ImportStatement) statementNode()       {}
func (is *ImportStatement) TokenLiteral() string { return is.Token.Literal }
func (is *ImportStatement) Pos() (int, int)      { return is.Token.Line, is.Token.Column }

type RequireStatement struct {
	Token lexer.Token
	Path  *StringLiteral
	Alias *Identifier
	Names []*Identifier
	All   bool
}

func (rs *RequireStatement) statementNode()       {}
func (rs *RequireStatement) TokenLiteral() string { return rs.Token.Literal }
func (rs *RequireStatement) Pos() (int, int)      { return rs.Token.Line, rs.Token.Column }

type ModuleStatement struct {
	Token lexer.Token
	Name  *Identifier
}

func (ms *ModuleStatement) statementNode()       {}
func (ms *ModuleStatement) TokenLiteral() string { return ms.Token.Literal }
func (ms *ModuleStatement) Pos() (int, int)      { return ms.Token.Line, ms.Token.Column }

type SelectStatement struct {
	Token lexer.Token
	Cases []*SelectCase
}

func (ss *SelectStatement) statementNode()       {}
func (ss *SelectStatement) TokenLiteral() string { return ss.Token.Literal }
func (ss *SelectStatement) Pos() (int, int)      { return ss.Token.Line, ss.Token.Column }

type SelectCase struct {
	Token   lexer.Token
	Comm    Expression
	Body    *BlockStatement
	Default bool
}
