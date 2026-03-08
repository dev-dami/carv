package ast

import (
	"testing"

	"github.com/dev-dami/carv/pkg/lexer"
)

// tok is a helper to create a token with the given literal and position.
func tok(literal string, line, col int) lexer.Token {
	return lexer.Token{Literal: literal, Line: line, Column: col}
}

// ---------- Program ----------

func TestProgram_Empty(t *testing.T) {
	p := &Program{Statements: []Statement{}}
	if p.TokenLiteral() != "" {
		t.Errorf("expected empty token literal, got %q", p.TokenLiteral())
	}
	line, col := p.Pos()
	if line != 0 || col != 0 {
		t.Errorf("expected position (0,0), got (%d,%d)", line, col)
	}
}

func TestProgram_WithStatements(t *testing.T) {
	letStmt := &LetStatement{Token: tok("let", 1, 1)}
	p := &Program{Statements: []Statement{letStmt}}

	if p.TokenLiteral() != "let" {
		t.Errorf("expected token literal 'let', got %q", p.TokenLiteral())
	}
	line, col := p.Pos()
	if line != 1 || col != 1 {
		t.Errorf("expected position (1,1), got (%d,%d)", line, col)
	}
}

// ---------- Expression nodes ----------

func TestIdentifier(t *testing.T) {
	n := &Identifier{Token: tok("foo", 5, 10), Value: "foo"}
	n.expressionNode()
	assertNodeLiteralPos(t, n, "foo", 5, 10)
}

func TestIntegerLiteral(t *testing.T) {
	n := &IntegerLiteral{Token: tok("42", 1, 5), Value: 42}
	n.expressionNode()
	assertNodeLiteralPos(t, n, "42", 1, 5)
}

func TestFloatLiteral(t *testing.T) {
	n := &FloatLiteral{Token: tok("3.14", 2, 3), Value: 3.14}
	n.expressionNode()
	assertNodeLiteralPos(t, n, "3.14", 2, 3)
}

func TestStringLiteral(t *testing.T) {
	n := &StringLiteral{Token: tok("hello", 1, 1), Value: "hello"}
	n.expressionNode()
	assertNodeLiteralPos(t, n, "hello", 1, 1)
}

func TestInterpolatedString(t *testing.T) {
	n := &InterpolatedString{Token: tok("f\"hello\"", 3, 2), Parts: []Expression{}}
	n.expressionNode()
	assertNodeLiteralPos(t, n, "f\"hello\"", 3, 2)
}

func TestCharLiteral(t *testing.T) {
	n := &CharLiteral{Token: tok("a", 1, 1), Value: 'a'}
	n.expressionNode()
	assertNodeLiteralPos(t, n, "a", 1, 1)
}

func TestBoolLiteral(t *testing.T) {
	n := &BoolLiteral{Token: tok("true", 1, 1), Value: true}
	n.expressionNode()
	assertNodeLiteralPos(t, n, "true", 1, 1)
}

func TestNilLiteral(t *testing.T) {
	n := &NilLiteral{Token: tok("nil", 4, 8)}
	n.expressionNode()
	assertNodeLiteralPos(t, n, "nil", 4, 8)
}

func TestArrayLiteral(t *testing.T) {
	n := &ArrayLiteral{Token: tok("[", 1, 1), Elements: []Expression{}}
	n.expressionNode()
	assertNodeLiteralPos(t, n, "[", 1, 1)
}

func TestMapLiteral(t *testing.T) {
	n := &MapLiteral{Token: tok("{", 2, 5), Pairs: map[Expression]Expression{}}
	n.expressionNode()
	assertNodeLiteralPos(t, n, "{", 2, 5)
}

func TestPrefixExpression(t *testing.T) {
	n := &PrefixExpression{Token: tok("!", 1, 1), Operator: "!"}
	n.expressionNode()
	assertNodeLiteralPos(t, n, "!", 1, 1)
}

func TestInfixExpression(t *testing.T) {
	n := &InfixExpression{Token: tok("+", 1, 5), Operator: "+"}
	n.expressionNode()
	assertNodeLiteralPos(t, n, "+", 1, 5)
}

func TestCallExpression(t *testing.T) {
	n := &CallExpression{Token: tok("(", 1, 5), Arguments: []Expression{}}
	n.expressionNode()
	assertNodeLiteralPos(t, n, "(", 1, 5)
}

func TestIndexExpression(t *testing.T) {
	n := &IndexExpression{Token: tok("[", 3, 7)}
	n.expressionNode()
	assertNodeLiteralPos(t, n, "[", 3, 7)
}

func TestMemberExpression(t *testing.T) {
	n := &MemberExpression{Token: tok(".", 2, 4)}
	n.expressionNode()
	assertNodeLiteralPos(t, n, ".", 2, 4)
}

func TestAssignExpression(t *testing.T) {
	n := &AssignExpression{Token: tok("=", 1, 10), Operator: "="}
	n.expressionNode()
	assertNodeLiteralPos(t, n, "=", 1, 10)
}

func TestIfExpression(t *testing.T) {
	n := &IfExpression{Token: tok("if", 1, 1)}
	n.expressionNode()
	assertNodeLiteralPos(t, n, "if", 1, 1)
}

func TestMatchExpression(t *testing.T) {
	n := &MatchExpression{Token: tok("match", 1, 1), Arms: []*MatchArm{}}
	n.expressionNode()
	assertNodeLiteralPos(t, n, "match", 1, 1)
}

func TestFunctionLiteral(t *testing.T) {
	n := &FunctionLiteral{Token: tok("fn", 1, 1), Parameters: []*Parameter{}}
	n.expressionNode()
	assertNodeLiteralPos(t, n, "fn", 1, 1)
}

func TestSpawnExpression(t *testing.T) {
	n := &SpawnExpression{Token: tok("spawn", 5, 3)}
	n.expressionNode()
	assertNodeLiteralPos(t, n, "spawn", 5, 3)
}

func TestAwaitExpression(t *testing.T) {
	n := &AwaitExpression{Token: tok("await", 6, 1)}
	n.expressionNode()
	assertNodeLiteralPos(t, n, "await", 6, 1)
}

func TestSendExpression(t *testing.T) {
	n := &SendExpression{Token: tok("<-", 7, 2)}
	n.expressionNode()
	assertNodeLiteralPos(t, n, "<-", 7, 2)
}

func TestRecvExpression(t *testing.T) {
	n := &RecvExpression{Token: tok("<-", 8, 3)}
	n.expressionNode()
	assertNodeLiteralPos(t, n, "<-", 8, 3)
}

func TestNewExpression(t *testing.T) {
	n := &NewExpression{Token: tok("new", 1, 1)}
	n.expressionNode()
	assertNodeLiteralPos(t, n, "new", 1, 1)
}

func TestCastExpression(t *testing.T) {
	n := &CastExpression{Token: tok("as", 2, 6)}
	n.expressionNode()
	assertNodeLiteralPos(t, n, "as", 2, 6)
}

func TestIsExpression(t *testing.T) {
	n := &IsExpression{Token: tok("is", 3, 4)}
	n.expressionNode()
	assertNodeLiteralPos(t, n, "is", 3, 4)
}

func TestOkExpression(t *testing.T) {
	n := &OkExpression{Token: tok("Ok", 1, 1)}
	n.expressionNode()
	assertNodeLiteralPos(t, n, "Ok", 1, 1)
}

func TestErrExpression(t *testing.T) {
	n := &ErrExpression{Token: tok("Err", 1, 1)}
	n.expressionNode()
	assertNodeLiteralPos(t, n, "Err", 1, 1)
}

func TestTryExpression(t *testing.T) {
	n := &TryExpression{Token: tok("?", 1, 10)}
	n.expressionNode()
	assertNodeLiteralPos(t, n, "?", 1, 10)
}

func TestBlockExpression(t *testing.T) {
	n := &BlockExpression{Token: tok("{", 4, 1)}
	n.expressionNode()
	assertNodeLiteralPos(t, n, "{", 4, 1)
}

func TestBorrowExpression(t *testing.T) {
	n := &BorrowExpression{Token: tok("&", 2, 3), Mutable: true}
	n.expressionNode()
	assertNodeLiteralPos(t, n, "&", 2, 3)
}

func TestDerefExpression(t *testing.T) {
	n := &DerefExpression{Token: tok("*", 1, 5)}
	n.expressionNode()
	assertNodeLiteralPos(t, n, "*", 1, 5)
}

// ---------- Statement nodes ----------

func TestLetStatement(t *testing.T) {
	n := &LetStatement{Token: tok("let", 1, 1)}
	n.statementNode()
	assertNodeLiteralPos(t, n, "let", 1, 1)
}

func TestConstStatement(t *testing.T) {
	n := &ConstStatement{Token: tok("const", 2, 1)}
	n.statementNode()
	assertNodeLiteralPos(t, n, "const", 2, 1)
}

func TestReturnStatement(t *testing.T) {
	n := &ReturnStatement{Token: tok("return", 3, 1)}
	n.statementNode()
	assertNodeLiteralPos(t, n, "return", 3, 1)
}

func TestExpressionStatement(t *testing.T) {
	n := &ExpressionStatement{Token: tok("x", 4, 1)}
	n.statementNode()
	assertNodeLiteralPos(t, n, "x", 4, 1)
}

func TestBlockStatement(t *testing.T) {
	n := &BlockStatement{Token: tok("{", 5, 1), Statements: []Statement{}}
	n.statementNode()
	assertNodeLiteralPos(t, n, "{", 5, 1)
}

func TestForStatement(t *testing.T) {
	n := &ForStatement{Token: tok("for", 6, 1)}
	n.statementNode()
	assertNodeLiteralPos(t, n, "for", 6, 1)
}

func TestForInStatement(t *testing.T) {
	n := &ForInStatement{Token: tok("for", 7, 1)}
	n.statementNode()
	assertNodeLiteralPos(t, n, "for", 7, 1)
}

func TestWhileStatement(t *testing.T) {
	n := &WhileStatement{Token: tok("while", 8, 1)}
	n.statementNode()
	assertNodeLiteralPos(t, n, "while", 8, 1)
}

func TestLoopStatement(t *testing.T) {
	n := &LoopStatement{Token: tok("loop", 9, 1)}
	n.statementNode()
	assertNodeLiteralPos(t, n, "loop", 9, 1)
}

func TestBreakStatement(t *testing.T) {
	n := &BreakStatement{Token: tok("break", 10, 5)}
	n.statementNode()
	assertNodeLiteralPos(t, n, "break", 10, 5)
}

func TestContinueStatement(t *testing.T) {
	n := &ContinueStatement{Token: tok("continue", 11, 5)}
	n.statementNode()
	assertNodeLiteralPos(t, n, "continue", 11, 5)
}

func TestFunctionStatement(t *testing.T) {
	n := &FunctionStatement{Token: tok("fn", 12, 1)}
	n.statementNode()
	assertNodeLiteralPos(t, n, "fn", 12, 1)
}

func TestClassStatement(t *testing.T) {
	n := &ClassStatement{Token: tok("class", 13, 1)}
	n.statementNode()
	assertNodeLiteralPos(t, n, "class", 13, 1)
}

func TestInterfaceStatement(t *testing.T) {
	n := &InterfaceStatement{Token: tok("interface", 14, 1)}
	n.statementNode()
	assertNodeLiteralPos(t, n, "interface", 14, 1)
}

func TestImplStatement(t *testing.T) {
	n := &ImplStatement{Token: tok("impl", 15, 1)}
	n.statementNode()
	assertNodeLiteralPos(t, n, "impl", 15, 1)
}

func TestTypeAliasStatement(t *testing.T) {
	n := &TypeAliasStatement{Token: tok("type", 16, 1)}
	n.statementNode()
	assertNodeLiteralPos(t, n, "type", 16, 1)
}

func TestImportStatement(t *testing.T) {
	n := &ImportStatement{Token: tok("import", 17, 1)}
	n.statementNode()
	assertNodeLiteralPos(t, n, "import", 17, 1)
}

func TestRequireStatement(t *testing.T) {
	n := &RequireStatement{Token: tok("require", 18, 1)}
	n.statementNode()
	assertNodeLiteralPos(t, n, "require", 18, 1)
}

func TestModuleStatement(t *testing.T) {
	n := &ModuleStatement{Token: tok("module", 19, 1)}
	n.statementNode()
	assertNodeLiteralPos(t, n, "module", 19, 1)
}

func TestSelectStatement(t *testing.T) {
	n := &SelectStatement{Token: tok("select", 20, 1), Cases: []*SelectCase{}}
	n.statementNode()
	assertNodeLiteralPos(t, n, "select", 20, 1)
}

// ---------- Helper / non-interface nodes ----------

func TestMatchArm(t *testing.T) {
	n := &MatchArm{Token: tok("=>", 7, 9)}
	assertLiteralPos(t, "MatchArm", n.TokenLiteral(), n.Pos, "=>", 7, 9)
}

func TestParameter(t *testing.T) {
	n := &Parameter{Token: tok("x", 7, 9)}
	assertLiteralPos(t, "Parameter", n.TokenLiteral(), n.Pos, "x", 7, 9)
}

func TestFieldDecl(t *testing.T) {
	n := &FieldDecl{Token: tok("name", 7, 9)}
	assertLiteralPos(t, "FieldDecl", n.TokenLiteral(), n.Pos, "name", 7, 9)
}

func TestMethodDecl(t *testing.T) {
	n := &MethodDecl{Token: tok("run", 7, 9)}
	assertLiteralPos(t, "MethodDecl", n.TokenLiteral(), n.Pos, "run", 7, 9)
}

func TestMethodSignature(t *testing.T) {
	n := &MethodSignature{Token: tok("do", 7, 9)}
	assertLiteralPos(t, "MethodSignature", n.TokenLiteral(), n.Pos, "do", 7, 9)
}

func TestSelectCase(t *testing.T) {
	n := &SelectCase{Token: tok("case", 7, 9)}
	assertLiteralPos(t, "SelectCase", n.TokenLiteral(), n.Pos, "case", 7, 9)
}

// ---------- Type expression nodes ----------

func TestBasicType(t *testing.T) {
	n := &BasicType{Token: tok("int", 1, 5), Name: "int"}
	n.typeExprNode()
	assertNodeLiteralPos(t, n, "int", 1, 5)
}

func TestNamedType(t *testing.T) {
	n := &NamedType{Token: tok("MyType", 2, 3)}
	n.typeExprNode()
	assertNodeLiteralPos(t, n, "MyType", 2, 3)
}

func TestArrayType(t *testing.T) {
	n := &ArrayType{Token: tok("[]", 3, 1)}
	n.typeExprNode()
	assertNodeLiteralPos(t, n, "[]", 3, 1)
}

func TestMapType(t *testing.T) {
	n := &MapType{Token: tok("map", 4, 1)}
	n.typeExprNode()
	assertNodeLiteralPos(t, n, "map", 4, 1)
}

func TestFunctionType(t *testing.T) {
	n := &FunctionType{Token: tok("fn", 5, 1)}
	n.typeExprNode()
	assertNodeLiteralPos(t, n, "fn", 5, 1)
}

func TestChannelType(t *testing.T) {
	n := &ChannelType{Token: tok("chan", 6, 1)}
	n.typeExprNode()
	assertNodeLiteralPos(t, n, "chan", 6, 1)
}

func TestOptionalType(t *testing.T) {
	n := &OptionalType{Token: tok("?", 7, 1)}
	n.typeExprNode()
	assertNodeLiteralPos(t, n, "?", 7, 1)
}

func TestRefType(t *testing.T) {
	n := &RefType{Token: tok("&", 8, 1)}
	n.typeExprNode()
	assertNodeLiteralPos(t, n, "&", 8, 1)
}

func TestResultType(t *testing.T) {
	n := &ResultType{Token: tok("Result", 9, 1)}
	n.typeExprNode()
	assertNodeLiteralPos(t, n, "Result", 9, 1)
}

func TestVolatileType(t *testing.T) {
	n := &VolatileType{Token: tok("volatile", 10, 1)}
	n.typeExprNode()
	assertNodeLiteralPos(t, n, "volatile", 10, 1)
}

// ---------- ReceiverKind constants ----------

func TestReceiverKindValues(t *testing.T) {
	if RecvNone != 0 {
		t.Errorf("expected RecvNone=0, got %d", RecvNone)
	}
	if RecvValue != 1 {
		t.Errorf("expected RecvValue=1, got %d", RecvValue)
	}
	if RecvRef != 2 {
		t.Errorf("expected RecvRef=2, got %d", RecvRef)
	}
	if RecvMutRef != 3 {
		t.Errorf("expected RecvMutRef=3, got %d", RecvMutRef)
	}
}

// ---------- Interface satisfaction compile-time checks ----------

func TestExpressionInterfaceSatisfaction(t *testing.T) {
	// Verify all expression types satisfy the Expression interface.
	var _ Expression = (*Identifier)(nil)
	var _ Expression = (*IntegerLiteral)(nil)
	var _ Expression = (*FloatLiteral)(nil)
	var _ Expression = (*StringLiteral)(nil)
	var _ Expression = (*InterpolatedString)(nil)
	var _ Expression = (*CharLiteral)(nil)
	var _ Expression = (*BoolLiteral)(nil)
	var _ Expression = (*NilLiteral)(nil)
	var _ Expression = (*ArrayLiteral)(nil)
	var _ Expression = (*MapLiteral)(nil)
	var _ Expression = (*PrefixExpression)(nil)
	var _ Expression = (*InfixExpression)(nil)
	var _ Expression = (*CallExpression)(nil)
	var _ Expression = (*IndexExpression)(nil)
	var _ Expression = (*MemberExpression)(nil)
	var _ Expression = (*AssignExpression)(nil)
	var _ Expression = (*IfExpression)(nil)
	var _ Expression = (*MatchExpression)(nil)
	var _ Expression = (*FunctionLiteral)(nil)
	var _ Expression = (*SpawnExpression)(nil)
	var _ Expression = (*AwaitExpression)(nil)
	var _ Expression = (*SendExpression)(nil)
	var _ Expression = (*RecvExpression)(nil)
	var _ Expression = (*NewExpression)(nil)
	var _ Expression = (*CastExpression)(nil)
	var _ Expression = (*IsExpression)(nil)
	var _ Expression = (*OkExpression)(nil)
	var _ Expression = (*ErrExpression)(nil)
	var _ Expression = (*TryExpression)(nil)
	var _ Expression = (*BlockExpression)(nil)
	var _ Expression = (*BorrowExpression)(nil)
	var _ Expression = (*DerefExpression)(nil)
}

func TestStatementInterfaceSatisfaction(t *testing.T) {
	var _ Statement = (*LetStatement)(nil)
	var _ Statement = (*ConstStatement)(nil)
	var _ Statement = (*ReturnStatement)(nil)
	var _ Statement = (*ExpressionStatement)(nil)
	var _ Statement = (*BlockStatement)(nil)
	var _ Statement = (*ForStatement)(nil)
	var _ Statement = (*ForInStatement)(nil)
	var _ Statement = (*WhileStatement)(nil)
	var _ Statement = (*LoopStatement)(nil)
	var _ Statement = (*BreakStatement)(nil)
	var _ Statement = (*ContinueStatement)(nil)
	var _ Statement = (*FunctionStatement)(nil)
	var _ Statement = (*ClassStatement)(nil)
	var _ Statement = (*InterfaceStatement)(nil)
	var _ Statement = (*ImplStatement)(nil)
	var _ Statement = (*TypeAliasStatement)(nil)
	var _ Statement = (*ImportStatement)(nil)
	var _ Statement = (*RequireStatement)(nil)
	var _ Statement = (*ModuleStatement)(nil)
	var _ Statement = (*SelectStatement)(nil)
}

func TestTypeExprInterfaceSatisfaction(t *testing.T) {
	var _ TypeExpr = (*BasicType)(nil)
	var _ TypeExpr = (*NamedType)(nil)
	var _ TypeExpr = (*ArrayType)(nil)
	var _ TypeExpr = (*MapType)(nil)
	var _ TypeExpr = (*FunctionType)(nil)
	var _ TypeExpr = (*ChannelType)(nil)
	var _ TypeExpr = (*OptionalType)(nil)
	var _ TypeExpr = (*RefType)(nil)
	var _ TypeExpr = (*ResultType)(nil)
	var _ TypeExpr = (*VolatileType)(nil)
}

// ---------- Helpers ----------

// assertNodeLiteralPos checks TokenLiteral and Pos on any Node.
func assertNodeLiteralPos(t *testing.T, n Node, wantLit string, wantLine, wantCol int) {
	t.Helper()
	if got := n.TokenLiteral(); got != wantLit {
		t.Errorf("TokenLiteral: want %q, got %q", wantLit, got)
	}
	line, col := n.Pos()
	if line != wantLine || col != wantCol {
		t.Errorf("Pos: want (%d,%d), got (%d,%d)", wantLine, wantCol, line, col)
	}
}

// assertLiteralPos is for types that are not full Node implementors but have
// TokenLiteral() and Pos() methods (e.g. MatchArm, Parameter, etc.).
func assertLiteralPos(t *testing.T, name, gotLit string, posFn func() (int, int), wantLit string, wantLine, wantCol int) {
	t.Helper()
	if gotLit != wantLit {
		t.Errorf("%s TokenLiteral: want %q, got %q", name, wantLit, gotLit)
	}
	line, col := posFn()
	if line != wantLine || col != wantCol {
		t.Errorf("%s Pos: want (%d,%d), got (%d,%d)", name, wantLine, wantCol, line, col)
	}
}
