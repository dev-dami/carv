package ast

import "github.com/dev-dami/carv/pkg/lexer"

type Node interface {
	TokenLiteral() string
	Pos() (line, col int)
}

type Statement interface {
	Node
	statementNode()
}

type Expression interface {
	Node
	expressionNode()
}

type TypeExpr interface {
	Node
	typeExprNode()
}

type Program struct {
	Statements []Statement
}

func (p *Program) TokenLiteral() string {
	if len(p.Statements) > 0 {
		return p.Statements[0].TokenLiteral()
	}
	return ""
}

func (p *Program) Pos() (int, int) {
	if len(p.Statements) > 0 {
		return p.Statements[0].Pos()
	}
	return 0, 0
}

type Identifier struct {
	Token lexer.Token
	Value string
}

func (i *Identifier) expressionNode()      {}
func (i *Identifier) TokenLiteral() string { return i.Token.Literal }
func (i *Identifier) Pos() (int, int)      { return i.Token.Line, i.Token.Column }

type IntegerLiteral struct {
	Token lexer.Token
	Value int64
}

func (il *IntegerLiteral) expressionNode()      {}
func (il *IntegerLiteral) TokenLiteral() string { return il.Token.Literal }
func (il *IntegerLiteral) Pos() (int, int)      { return il.Token.Line, il.Token.Column }

type FloatLiteral struct {
	Token lexer.Token
	Value float64
}

func (fl *FloatLiteral) expressionNode()      {}
func (fl *FloatLiteral) TokenLiteral() string { return fl.Token.Literal }
func (fl *FloatLiteral) Pos() (int, int)      { return fl.Token.Line, fl.Token.Column }

type StringLiteral struct {
	Token lexer.Token
	Value string
}

func (sl *StringLiteral) expressionNode()      {}
func (sl *StringLiteral) TokenLiteral() string { return sl.Token.Literal }
func (sl *StringLiteral) Pos() (int, int)      { return sl.Token.Line, sl.Token.Column }

type InterpolatedString struct {
	Token lexer.Token
	Parts []Expression
}

func (is *InterpolatedString) expressionNode()      {}
func (is *InterpolatedString) TokenLiteral() string { return is.Token.Literal }
func (is *InterpolatedString) Pos() (int, int)      { return is.Token.Line, is.Token.Column }

type CharLiteral struct {
	Token lexer.Token
	Value rune
}

func (cl *CharLiteral) expressionNode()      {}
func (cl *CharLiteral) TokenLiteral() string { return cl.Token.Literal }
func (cl *CharLiteral) Pos() (int, int)      { return cl.Token.Line, cl.Token.Column }

type BoolLiteral struct {
	Token lexer.Token
	Value bool
}

func (bl *BoolLiteral) expressionNode()      {}
func (bl *BoolLiteral) TokenLiteral() string { return bl.Token.Literal }
func (bl *BoolLiteral) Pos() (int, int)      { return bl.Token.Line, bl.Token.Column }

type NilLiteral struct {
	Token lexer.Token
}

func (nl *NilLiteral) expressionNode()      {}
func (nl *NilLiteral) TokenLiteral() string { return nl.Token.Literal }
func (nl *NilLiteral) Pos() (int, int)      { return nl.Token.Line, nl.Token.Column }

type ArrayLiteral struct {
	Token    lexer.Token
	Elements []Expression
}

func (al *ArrayLiteral) expressionNode()      {}
func (al *ArrayLiteral) TokenLiteral() string { return al.Token.Literal }
func (al *ArrayLiteral) Pos() (int, int)      { return al.Token.Line, al.Token.Column }

type MapLiteral struct {
	Token lexer.Token
	Pairs map[Expression]Expression
}

func (ml *MapLiteral) expressionNode()      {}
func (ml *MapLiteral) TokenLiteral() string { return ml.Token.Literal }
func (ml *MapLiteral) Pos() (int, int)      { return ml.Token.Line, ml.Token.Column }

type PrefixExpression struct {
	Token    lexer.Token
	Operator string
	Right    Expression
}

func (pe *PrefixExpression) expressionNode()      {}
func (pe *PrefixExpression) TokenLiteral() string { return pe.Token.Literal }
func (pe *PrefixExpression) Pos() (int, int)      { return pe.Token.Line, pe.Token.Column }

type InfixExpression struct {
	Token    lexer.Token
	Left     Expression
	Operator string
	Right    Expression
}

func (ie *InfixExpression) expressionNode()      {}
func (ie *InfixExpression) TokenLiteral() string { return ie.Token.Literal }
func (ie *InfixExpression) Pos() (int, int)      { return ie.Token.Line, ie.Token.Column }

type PipeExpression struct {
	Token lexer.Token
	Left  Expression
	Right Expression
}

func (pe *PipeExpression) expressionNode()      {}
func (pe *PipeExpression) TokenLiteral() string { return pe.Token.Literal }
func (pe *PipeExpression) Pos() (int, int)      { return pe.Token.Line, pe.Token.Column }

type CallExpression struct {
	Token     lexer.Token
	Function  Expression
	Arguments []Expression
}

func (ce *CallExpression) expressionNode()      {}
func (ce *CallExpression) TokenLiteral() string { return ce.Token.Literal }
func (ce *CallExpression) Pos() (int, int)      { return ce.Token.Line, ce.Token.Column }

type IndexExpression struct {
	Token lexer.Token
	Left  Expression
	Index Expression
}

func (ie *IndexExpression) expressionNode()      {}
func (ie *IndexExpression) TokenLiteral() string { return ie.Token.Literal }
func (ie *IndexExpression) Pos() (int, int)      { return ie.Token.Line, ie.Token.Column }

type MemberExpression struct {
	Token  lexer.Token
	Object Expression
	Member *Identifier
}

func (me *MemberExpression) expressionNode()      {}
func (me *MemberExpression) TokenLiteral() string { return me.Token.Literal }
func (me *MemberExpression) Pos() (int, int)      { return me.Token.Line, me.Token.Column }

type AssignExpression struct {
	Token    lexer.Token
	Left     Expression
	Operator string
	Right    Expression
}

func (ae *AssignExpression) expressionNode()      {}
func (ae *AssignExpression) TokenLiteral() string { return ae.Token.Literal }
func (ae *AssignExpression) Pos() (int, int)      { return ae.Token.Line, ae.Token.Column }

type IfExpression struct {
	Token       lexer.Token
	Condition   Expression
	Consequence *BlockStatement
	Alternative *BlockStatement
}

func (ie *IfExpression) expressionNode()      {}
func (ie *IfExpression) TokenLiteral() string { return ie.Token.Literal }
func (ie *IfExpression) Pos() (int, int)      { return ie.Token.Line, ie.Token.Column }

type MatchExpression struct {
	Token lexer.Token
	Value Expression
	Arms  []*MatchArm
}

func (me *MatchExpression) expressionNode()      {}
func (me *MatchExpression) TokenLiteral() string { return me.Token.Literal }
func (me *MatchExpression) Pos() (int, int)      { return me.Token.Line, me.Token.Column }

type MatchArm struct {
	Token   lexer.Token
	Pattern Expression
	Body    Expression
}

type FunctionLiteral struct {
	Token      lexer.Token
	Name       *Identifier
	Parameters []*Parameter
	ReturnType TypeExpr
	Body       *BlockStatement
}

func (fl *FunctionLiteral) expressionNode()      {}
func (fl *FunctionLiteral) TokenLiteral() string { return fl.Token.Literal }
func (fl *FunctionLiteral) Pos() (int, int)      { return fl.Token.Line, fl.Token.Column }

type Parameter struct {
	Name    *Identifier
	Type    TypeExpr
	Mutable bool
}

type SpawnExpression struct {
	Token lexer.Token
	Body  *BlockStatement
}

func (se *SpawnExpression) expressionNode()      {}
func (se *SpawnExpression) TokenLiteral() string { return se.Token.Literal }
func (se *SpawnExpression) Pos() (int, int)      { return se.Token.Line, se.Token.Column }

type AwaitExpression struct {
	Token lexer.Token
	Value Expression
}

func (ae *AwaitExpression) expressionNode()      {}
func (ae *AwaitExpression) TokenLiteral() string { return ae.Token.Literal }
func (ae *AwaitExpression) Pos() (int, int)      { return ae.Token.Line, ae.Token.Column }

type SendExpression struct {
	Token   lexer.Token
	Channel Expression
	Value   Expression
}

func (se *SendExpression) expressionNode()      {}
func (se *SendExpression) TokenLiteral() string { return se.Token.Literal }
func (se *SendExpression) Pos() (int, int)      { return se.Token.Line, se.Token.Column }

type RecvExpression struct {
	Token   lexer.Token
	Channel Expression
}

func (re *RecvExpression) expressionNode()      {}
func (re *RecvExpression) TokenLiteral() string { return re.Token.Literal }
func (re *RecvExpression) Pos() (int, int)      { return re.Token.Line, re.Token.Column }

type NewExpression struct {
	Token     lexer.Token
	Type      TypeExpr
	Arguments []Expression
}

func (ne *NewExpression) expressionNode()      {}
func (ne *NewExpression) TokenLiteral() string { return ne.Token.Literal }
func (ne *NewExpression) Pos() (int, int)      { return ne.Token.Line, ne.Token.Column }

type CastExpression struct {
	Token lexer.Token
	Value Expression
	Type  TypeExpr
}

func (ce *CastExpression) expressionNode()      {}
func (ce *CastExpression) TokenLiteral() string { return ce.Token.Literal }
func (ce *CastExpression) Pos() (int, int)      { return ce.Token.Line, ce.Token.Column }

type IsExpression struct {
	Token lexer.Token
	Value Expression
	Type  TypeExpr
}

func (ie *IsExpression) expressionNode()      {}
func (ie *IsExpression) TokenLiteral() string { return ie.Token.Literal }
func (ie *IsExpression) Pos() (int, int)      { return ie.Token.Line, ie.Token.Column }

type OkExpression struct {
	Token lexer.Token
	Value Expression
}

func (oe *OkExpression) expressionNode()      {}
func (oe *OkExpression) TokenLiteral() string { return oe.Token.Literal }
func (oe *OkExpression) Pos() (int, int)      { return oe.Token.Line, oe.Token.Column }

type ErrExpression struct {
	Token lexer.Token
	Value Expression
}

func (ee *ErrExpression) expressionNode()      {}
func (ee *ErrExpression) TokenLiteral() string { return ee.Token.Literal }
func (ee *ErrExpression) Pos() (int, int)      { return ee.Token.Line, ee.Token.Column }

type TryExpression struct {
	Token lexer.Token
	Value Expression
}

func (te *TryExpression) expressionNode()      {}
func (te *TryExpression) TokenLiteral() string { return te.Token.Literal }
func (te *TryExpression) Pos() (int, int)      { return te.Token.Line, te.Token.Column }

type BlockExpression struct {
	Token lexer.Token
	Block *BlockStatement
}

func (be *BlockExpression) expressionNode()      {}
func (be *BlockExpression) TokenLiteral() string { return be.Token.Literal }
func (be *BlockExpression) Pos() (int, int)      { return be.Token.Line, be.Token.Column }
