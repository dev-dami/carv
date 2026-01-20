package ast

import "github.com/carv-lang/carv/pkg/lexer"

type BasicType struct {
	Token lexer.Token
	Name  string
}

func (bt *BasicType) typeExprNode()        {}
func (bt *BasicType) TokenLiteral() string { return bt.Token.Literal }
func (bt *BasicType) Pos() (int, int)      { return bt.Token.Line, bt.Token.Column }

type NamedType struct {
	Token lexer.Token
	Name  *Identifier
}

func (nt *NamedType) typeExprNode()        {}
func (nt *NamedType) TokenLiteral() string { return nt.Token.Literal }
func (nt *NamedType) Pos() (int, int)      { return nt.Token.Line, nt.Token.Column }

type ArrayType struct {
	Token       lexer.Token
	ElementType TypeExpr
	Size        Expression
}

func (at *ArrayType) typeExprNode()        {}
func (at *ArrayType) TokenLiteral() string { return at.Token.Literal }
func (at *ArrayType) Pos() (int, int)      { return at.Token.Line, at.Token.Column }

type MapType struct {
	Token     lexer.Token
	KeyType   TypeExpr
	ValueType TypeExpr
}

func (mt *MapType) typeExprNode()        {}
func (mt *MapType) TokenLiteral() string { return mt.Token.Literal }
func (mt *MapType) Pos() (int, int)      { return mt.Token.Line, mt.Token.Column }

type FunctionType struct {
	Token      lexer.Token
	Parameters []TypeExpr
	ReturnType TypeExpr
}

func (ft *FunctionType) typeExprNode()        {}
func (ft *FunctionType) TokenLiteral() string { return ft.Token.Literal }
func (ft *FunctionType) Pos() (int, int)      { return ft.Token.Line, ft.Token.Column }

type ChannelType struct {
	Token       lexer.Token
	ElementType TypeExpr
	SendOnly    bool
	RecvOnly    bool
}

func (ct *ChannelType) typeExprNode()        {}
func (ct *ChannelType) TokenLiteral() string { return ct.Token.Literal }
func (ct *ChannelType) Pos() (int, int)      { return ct.Token.Line, ct.Token.Column }

type OptionalType struct {
	Token lexer.Token
	Inner TypeExpr
}

func (ot *OptionalType) typeExprNode()        {}
func (ot *OptionalType) TokenLiteral() string { return ot.Token.Literal }
func (ot *OptionalType) Pos() (int, int)      { return ot.Token.Line, ot.Token.Column }

type RefType struct {
	Token   lexer.Token
	Inner   TypeExpr
	Mutable bool
}

func (rt *RefType) typeExprNode()        {}
func (rt *RefType) TokenLiteral() string { return rt.Token.Literal }
func (rt *RefType) Pos() (int, int)      { return rt.Token.Line, rt.Token.Column }

type ResultType struct {
	Token   lexer.Token
	OkType  TypeExpr
	ErrType TypeExpr
}

func (rt *ResultType) typeExprNode()        {}
func (rt *ResultType) TokenLiteral() string { return rt.Token.Literal }
func (rt *ResultType) Pos() (int, int)      { return rt.Token.Line, rt.Token.Column }
