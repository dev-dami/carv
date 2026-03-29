package parser

import (
	"testing"

	"github.com/dev-dami/carv/pkg/ast"
	"github.com/dev-dami/carv/pkg/lexer"
)

func TestLetStatements(t *testing.T) {
	input := `let x = 5;
let y = 10;
mut z = 15;`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 3 {
		t.Fatalf("expected 3 statements, got %d", len(program.Statements))
	}

	tests := []struct {
		expectedName string
		mutable      bool
	}{
		{"x", false},
		{"y", false},
		{"z", true},
	}

	for i, tt := range tests {
		stmt := program.Statements[i]
		letStmt, ok := stmt.(*ast.LetStatement)
		if !ok {
			t.Fatalf("stmt not *ast.LetStatement. got=%T", stmt)
		}
		if letStmt.Name.Value != tt.expectedName {
			t.Fatalf("expected name %s, got %s", tt.expectedName, letStmt.Name.Value)
		}
		if letStmt.Mutable != tt.mutable {
			t.Fatalf("expected mutable %v, got %v", tt.mutable, letStmt.Mutable)
		}
	}
}

func TestFunctionStatement(t *testing.T) {
	input := `fn add(a: int, b: int) -> int {
	return a + b;
}`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	fnStmt, ok := program.Statements[0].(*ast.FunctionStatement)
	if !ok {
		t.Fatalf("stmt not *ast.FunctionStatement. got=%T", program.Statements[0])
	}

	if fnStmt.Name.Value != "add" {
		t.Fatalf("expected function name 'add', got %s", fnStmt.Name.Value)
	}

	if len(fnStmt.Parameters) != 2 {
		t.Fatalf("expected 2 parameters, got %d", len(fnStmt.Parameters))
	}
}

func TestComparisonExpression(t *testing.T) {
	tests := []struct {
		input    string
		left     int64
		operator string
		right    int64
	}{
		{"5 == 5;", 5, "==", 5},
		{"5 != 4;", 5, "!=", 4},
		{"5 < 10;", 5, "<", 10},
		{"10 > 5;", 10, ">", 5},
		{"5 <= 5;", 5, "<=", 5},
		{"5 >= 5;", 5, ">=", 5},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		stmt := program.Statements[0].(*ast.ExpressionStatement)
		exp, ok := stmt.Expression.(*ast.InfixExpression)
		if !ok {
			t.Fatalf("expression not *ast.InfixExpression. got=%T", stmt.Expression)
		}

		if exp.Operator != tt.operator {
			t.Fatalf("expected operator %s, got %s", tt.operator, exp.Operator)
		}
	}
}

func TestCallExpression(t *testing.T) {
	input := `print("hello", 42);`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	call, ok := stmt.Expression.(*ast.CallExpression)
	if !ok {
		t.Fatalf("expression not *ast.CallExpression. got=%T", stmt.Expression)
	}

	if len(call.Arguments) != 2 {
		t.Fatalf("expected 2 arguments, got %d", len(call.Arguments))
	}
}

func TestForLoop(t *testing.T) {
	input := `for (let i = 0; i < 10; i = i + 1) {
	print(i);
}`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	forStmt, ok := program.Statements[0].(*ast.ForStatement)
	if !ok {
		t.Fatalf("stmt not *ast.ForStatement. got=%T", program.Statements[0])
	}

	if forStmt.Init == nil {
		t.Fatal("expected init statement")
	}
	if forStmt.Condition == nil {
		t.Fatal("expected condition")
	}
	if forStmt.Post == nil {
		t.Fatal("expected post statement")
	}
}

func TestIfExpression(t *testing.T) {
	input := `if x > 5 {
	print("big");
} else {
	print("small");
}`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	ifExp, ok := stmt.Expression.(*ast.IfExpression)
	if !ok {
		t.Fatalf("expression not *ast.IfExpression. got=%T", stmt.Expression)
	}

	if ifExp.Alternative == nil {
		t.Fatal("expected alternative block")
	}
}

func TestOkErrExpressions(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Ok(42);", "Ok"},
		{"Err(\"failed\");", "Err"},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		stmt := program.Statements[0].(*ast.ExpressionStatement)
		if tt.expected == "Ok" {
			_, ok := stmt.Expression.(*ast.OkExpression)
			if !ok {
				t.Fatalf("expected OkExpression, got %T", stmt.Expression)
			}
		} else {
			_, ok := stmt.Expression.(*ast.ErrExpression)
			if !ok {
				t.Fatalf("expected ErrExpression, got %T", stmt.Expression)
			}
		}
	}
}

func TestTryExpression(t *testing.T) {
	input := `read_file("test.txt")?;`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	tryExp, ok := stmt.Expression.(*ast.TryExpression)
	if !ok {
		t.Fatalf("expected TryExpression, got %T", stmt.Expression)
	}

	call, ok := tryExp.Value.(*ast.CallExpression)
	if !ok {
		t.Fatalf("expected CallExpression inside try, got %T", tryExp.Value)
	}

	fn, ok := call.Function.(*ast.Identifier)
	if !ok || fn.Value != "read_file" {
		t.Fatalf("expected read_file call")
	}
}

func TestMatchExpression(t *testing.T) {
	input := `match result {
	Ok(x) => x,
	Err(e) => 0,
};`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	matchExp, ok := stmt.Expression.(*ast.MatchExpression)
	if !ok {
		t.Fatalf("expected MatchExpression, got %T", stmt.Expression)
	}

	if len(matchExp.Arms) != 2 {
		t.Fatalf("expected 2 match arms, got %d", len(matchExp.Arms))
	}
}

func TestElseIfExpression(t *testing.T) {
	input := `if x > 10 {
	print("big");
} else if x > 5 {
	print("medium");
} else {
	print("small");
}`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	ifExp, ok := stmt.Expression.(*ast.IfExpression)
	if !ok {
		t.Fatalf("expression not *ast.IfExpression. got=%T", stmt.Expression)
	}

	if ifExp.Alternative == nil {
		t.Fatal("expected alternative block for else-if")
	}

	if len(ifExp.Alternative.Statements) != 1 {
		t.Fatalf("expected 1 statement in alternative, got %d", len(ifExp.Alternative.Statements))
	}

	altStmt, ok := ifExp.Alternative.Statements[0].(*ast.ExpressionStatement)
	if !ok {
		t.Fatalf("alternative statement not ExpressionStatement. got=%T", ifExp.Alternative.Statements[0])
	}
	nestedIf, ok := altStmt.Expression.(*ast.IfExpression)
	if !ok {
		t.Fatalf("expected nested IfExpression in else-if. got=%T", altStmt.Expression)
	}
	if nestedIf.Alternative == nil {
		t.Fatal("expected else block in nested if")
	}
}

func TestChainedElseIf(t *testing.T) {
	input := `if x > 20 {
	1;
} else if x > 10 {
	2;
} else if x > 5 {
	3;
} else {
	4;
}`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}
}

func TestCompoundAssignmentParsing(t *testing.T) {
	tests := []struct {
		input    string
		operator string
	}{
		{"x += 1;", "+="},
		{"x -= 1;", "-="},
		{"x *= 2;", "*="},
		{"x /= 2;", "/="},
		{"x %= 3;", "%="},
		{"x &= 3;", "&="},
		{"x |= 4;", "|="},
		{"x ^= 5;", "^="},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		stmt := program.Statements[0].(*ast.ExpressionStatement)
		assign, ok := stmt.Expression.(*ast.AssignExpression)
		if !ok {
			t.Fatalf("expression not *ast.AssignExpression for %s. got=%T", tt.operator, stmt.Expression)
		}
		if assign.Operator != tt.operator {
			t.Fatalf("expected operator %s, got %s", tt.operator, assign.Operator)
		}
	}
}

func TestTypeAsCallExpression(t *testing.T) {
	tests := []struct {
		input    string
		funcName string
	}{
		{"int(3.14);", "int"},
		{"float(42);", "float"},
		{"bool(1);", "bool"},
		{"string(42);", "string"},
		{"char(65);", "char"},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		stmt := program.Statements[0].(*ast.ExpressionStatement)
		call, ok := stmt.Expression.(*ast.CallExpression)
		if !ok {
			t.Fatalf("expected CallExpression for %s, got %T", tt.funcName, stmt.Expression)
		}
		ident, ok := call.Function.(*ast.Identifier)
		if !ok {
			t.Fatalf("expected Identifier for function, got %T", call.Function)
		}
		if ident.Value != tt.funcName {
			t.Fatalf("expected function name %s, got %s", tt.funcName, ident.Value)
		}
	}
}

func TestForLoopInitMutable(t *testing.T) {
	input := `for (let i = 0; i < 10; i = i + 1) { print(i); }`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	forStmt := program.Statements[0].(*ast.ForStatement)
	letStmt, ok := forStmt.Init.(*ast.LetStatement)
	if !ok {
		t.Fatalf("expected LetStatement in for init, got %T", forStmt.Init)
	}
	if !letStmt.Mutable {
		t.Fatal("for loop init 'let' should be forced mutable")
	}
}

func TestForInStatement(t *testing.T) {
	input := `for x in arr { print(x); }`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	forIn, ok := program.Statements[0].(*ast.ForInStatement)
	if !ok {
		t.Fatalf("expected ForInStatement. got=%T", program.Statements[0])
	}
	if forIn.Value.Value != "x" {
		t.Fatalf("expected iterator 'x', got %s", forIn.Value.Value)
	}
}

func TestInfiniteForLoop(t *testing.T) {
	input := `for { break; }`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	_, ok := program.Statements[0].(*ast.LoopStatement)
	if !ok {
		t.Fatalf("expected LoopStatement. got=%T", program.Statements[0])
	}
}

func TestConstStatement(t *testing.T) {
	input := `const PI = 3.14;`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	constStmt, ok := program.Statements[0].(*ast.ConstStatement)
	if !ok {
		t.Fatalf("expected ConstStatement, got %T", program.Statements[0])
	}
	if constStmt.Name.Value != "PI" {
		t.Fatalf("expected name 'PI', got %s", constStmt.Name.Value)
	}
}

func TestInterpolatedString(t *testing.T) {
	input := `f"hello {name}";`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	interp, ok := stmt.Expression.(*ast.InterpolatedString)
	if !ok {
		t.Fatalf("expected InterpolatedString, got %T", stmt.Expression)
	}
	if len(interp.Parts) != 2 {
		t.Fatalf("expected 2 parts, got %d", len(interp.Parts))
	}
}

func TestMapLiteral(t *testing.T) {
	input := `{"a": 1, "b": 2};`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	mapLit, ok := stmt.Expression.(*ast.MapLiteral)
	if !ok {
		t.Fatalf("expected MapLiteral, got %T", stmt.Expression)
	}
	if len(mapLit.Pairs) != 2 {
		t.Fatalf("expected 2 pairs, got %d", len(mapLit.Pairs))
	}
}

func TestClassStatement(t *testing.T) {
	input := `class Point {
	x: int = 0
	y: int = 0
	fn getX() -> int { return self.x; }
}`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	classStmt, ok := program.Statements[0].(*ast.ClassStatement)
	if !ok {
		t.Fatalf("expected ClassStatement, got %T", program.Statements[0])
	}
	if classStmt.Name.Value != "Point" {
		t.Fatalf("expected class name 'Point', got %s", classStmt.Name.Value)
	}
	if len(classStmt.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(classStmt.Fields))
	}
	if len(classStmt.Methods) != 1 {
		t.Fatalf("expected 1 method, got %d", len(classStmt.Methods))
	}
}

func TestBorrowExpression(t *testing.T) {
	input := `&x;`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	borrow, ok := stmt.Expression.(*ast.BorrowExpression)
	if !ok {
		t.Fatalf("expected BorrowExpression, got %T", stmt.Expression)
	}
	if borrow.Mutable {
		t.Fatalf("expected immutable borrow")
	}
	ident, ok := borrow.Value.(*ast.Identifier)
	if !ok || ident.Value != "x" {
		t.Fatalf("expected identifier x, got %T", borrow.Value)
	}
}

func TestBorrowMutExpression(t *testing.T) {
	input := `&mut x;`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	borrow, ok := stmt.Expression.(*ast.BorrowExpression)
	if !ok {
		t.Fatalf("expected BorrowExpression, got %T", stmt.Expression)
	}
	if !borrow.Mutable {
		t.Fatalf("expected mutable borrow")
	}
	ident, ok := borrow.Value.(*ast.Identifier)
	if !ok || ident.Value != "x" {
		t.Fatalf("expected identifier x, got %T", borrow.Value)
	}
}

func TestDerefExpression(t *testing.T) {
	input := `*x;`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	deref, ok := stmt.Expression.(*ast.DerefExpression)
	if !ok {
		t.Fatalf("expected DerefExpression, got %T", stmt.Expression)
	}
	ident, ok := deref.Value.(*ast.Identifier)
	if !ok || ident.Value != "x" {
		t.Fatalf("expected identifier x, got %T", deref.Value)
	}
}

func TestBorrowPrecedence(t *testing.T) {
	input := `&x + 1;`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	infix, ok := stmt.Expression.(*ast.InfixExpression)
	if !ok {
		t.Fatalf("expected InfixExpression, got %T", stmt.Expression)
	}
	if infix.Operator != "+" {
		t.Fatalf("expected + operator, got %s", infix.Operator)
	}
	if _, ok := infix.Left.(*ast.BorrowExpression); !ok {
		t.Fatalf("expected BorrowExpression on left, got %T", infix.Left)
	}
}

func TestBitwiseAndInfix(t *testing.T) {
	input := `x & y;`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	infix, ok := stmt.Expression.(*ast.InfixExpression)
	if !ok {
		t.Fatalf("expected InfixExpression, got %T", stmt.Expression)
	}
	if infix.Operator != "&" {
		t.Fatalf("expected & operator, got %s", infix.Operator)
	}
}

func TestMultiplyInfix(t *testing.T) {
	input := `x * y;`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	infix, ok := stmt.Expression.(*ast.InfixExpression)
	if !ok {
		t.Fatalf("expected InfixExpression, got %T", stmt.Expression)
	}
	if infix.Operator != "*" {
		t.Fatalf("expected * operator, got %s", infix.Operator)
	}
}

func TestInterfaceStatement(t *testing.T) {
	input := `interface Printable {
	fn to_string(&self) -> string;
	fn display(&self);
}`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	iface, ok := program.Statements[0].(*ast.InterfaceStatement)
	if !ok {
		t.Fatalf("expected InterfaceStatement, got %T", program.Statements[0])
	}

	if iface.Name.Value != "Printable" {
		t.Fatalf("expected interface name 'Printable', got %s", iface.Name.Value)
	}

	if len(iface.Methods) != 2 {
		t.Fatalf("expected 2 methods, got %d", len(iface.Methods))
	}

	if iface.Methods[0].Name.Value != "to_string" {
		t.Fatalf("expected method name 'to_string', got %s", iface.Methods[0].Name.Value)
	}
	if iface.Methods[0].Receiver != ast.RecvRef {
		t.Fatalf("expected RecvRef receiver, got %d", iface.Methods[0].Receiver)
	}
	if iface.Methods[0].ReturnType == nil {
		t.Fatal("expected return type for to_string")
	}
}

func TestImplStatement(t *testing.T) {
	input := `impl Printable for Person {
	fn to_string(&self) -> string {
		return self.name;
	}
	fn display(&self) {
		println(self.name);
	}
}`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	impl, ok := program.Statements[0].(*ast.ImplStatement)
	if !ok {
		t.Fatalf("expected ImplStatement, got %T", program.Statements[0])
	}

	if impl.Interface.Value != "Printable" {
		t.Fatalf("expected interface 'Printable', got %s", impl.Interface.Value)
	}
	if impl.Type.Value != "Person" {
		t.Fatalf("expected type 'Person', got %s", impl.Type.Value)
	}
	if len(impl.Methods) != 2 {
		t.Fatalf("expected 2 methods, got %d", len(impl.Methods))
	}
	if impl.Methods[0].Receiver != ast.RecvRef {
		t.Fatalf("expected RecvRef receiver, got %d", impl.Methods[0].Receiver)
	}
}

func TestCastExpression(t *testing.T) {
	input := `&p as &Printable;`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	cast, ok := stmt.Expression.(*ast.CastExpression)
	if !ok {
		t.Fatalf("expected CastExpression, got %T", stmt.Expression)
	}

	if _, isBorrow := cast.Value.(*ast.BorrowExpression); !isBorrow {
		t.Fatalf("expected BorrowExpression as cast value, got %T", cast.Value)
	}

	ref, ok := cast.Type.(*ast.RefType)
	if !ok {
		t.Fatalf("expected RefType as cast target, got %T", cast.Type)
	}
	named, ok := ref.Inner.(*ast.NamedType)
	if !ok {
		t.Fatalf("expected NamedType inside RefType, got %T", ref.Inner)
	}
	if named.Name.Value != "Printable" {
		t.Fatalf("expected 'Printable', got %s", named.Name.Value)
	}
}

func TestMethodReceiverParsing(t *testing.T) {
	input := `class Dog {
	name: string
	fn bark(&self) -> string {
		return self.name;
	}
}`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	cls := program.Statements[0].(*ast.ClassStatement)
	if len(cls.Methods) != 1 {
		t.Fatalf("expected 1 method, got %d", len(cls.Methods))
	}
	if cls.Methods[0].Receiver != ast.RecvRef {
		t.Fatalf("expected RecvRef, got %d", cls.Methods[0].Receiver)
	}
	if len(cls.Methods[0].Parameters) != 0 {
		t.Fatalf("expected 0 params (receiver is separate), got %d", len(cls.Methods[0].Parameters))
	}
}

func TestMethodReceiverMutRefDefault(t *testing.T) {
	input := `class Counter {
	value: int = 0
	fn increment() {
		self.value = self.value + 1;
	}
}`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	cls := program.Statements[0].(*ast.ClassStatement)
	if cls.Methods[0].Receiver != ast.RecvMutRef {
		t.Fatalf("expected RecvMutRef default, got %d", cls.Methods[0].Receiver)
	}
}

func TestAsyncFunctionParsing(t *testing.T) {
	input := `async fn fetch_data() -> int {
	return 42;
}`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	fnStmt, ok := program.Statements[0].(*ast.FunctionStatement)
	if !ok {
		t.Fatalf("expected FunctionStatement, got %T", program.Statements[0])
	}

	if fnStmt.Name.Value != "fetch_data" {
		t.Fatalf("expected function name 'fetch_data', got %s", fnStmt.Name.Value)
	}

	if !fnStmt.Async {
		t.Fatal("expected Async to be true")
	}
}

func TestAwaitExpressionParsing(t *testing.T) {
	input := `await fetch();`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*ast.ExpressionStatement)
	if !ok {
		t.Fatalf("expected ExpressionStatement, got %T", program.Statements[0])
	}

	awaitExpr, ok := stmt.Expression.(*ast.AwaitExpression)
	if !ok {
		t.Fatalf("expected AwaitExpression, got %T", stmt.Expression)
	}

	call, ok := awaitExpr.Value.(*ast.CallExpression)
	if !ok {
		t.Fatalf("expected CallExpression inside await, got %T", awaitExpr.Value)
	}

	fn, ok := call.Function.(*ast.Identifier)
	if !ok || fn.Value != "fetch" {
		t.Fatalf("expected fetch call, got %v", call.Function)
	}
}

func TestPubAsyncFn(t *testing.T) {
	input := `pub async fn api_call() -> string {
	return "ok";
}`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	fnStmt, ok := program.Statements[0].(*ast.FunctionStatement)
	if !ok {
		t.Fatalf("expected FunctionStatement, got %T", program.Statements[0])
	}

	if !fnStmt.Public {
		t.Fatal("expected Public to be true")
	}

	if !fnStmt.Async {
		t.Fatal("expected Async to be true")
	}

	if fnStmt.Name.Value != "api_call" {
		t.Fatalf("expected function name 'api_call', got %s", fnStmt.Name.Value)
	}
}

func TestAsyncClassMethodParsing(t *testing.T) {
	input := `class Worker {
	async fn run(&self) -> int {
		return 1;
	}
}`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	cls, ok := program.Statements[0].(*ast.ClassStatement)
	if !ok {
		t.Fatalf("expected ClassStatement, got %T", program.Statements[0])
	}
	if len(cls.Methods) != 1 {
		t.Fatalf("expected 1 method, got %d", len(cls.Methods))
	}
	if cls.Methods[0].Name.Value != "run" {
		t.Fatalf("expected method name 'run', got %s", cls.Methods[0].Name.Value)
	}
	if !cls.Methods[0].Async {
		t.Fatal("expected async class method to have Async=true")
	}
}

func TestAsyncImplMethodParsing(t *testing.T) {
	input := `impl Runner for Worker {
	async fn run(&self) -> int {
		return 1;
	}
}`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	implStmt, ok := program.Statements[0].(*ast.ImplStatement)
	if !ok {
		t.Fatalf("expected ImplStatement, got %T", program.Statements[0])
	}
	if len(implStmt.Methods) != 1 {
		t.Fatalf("expected 1 method, got %d", len(implStmt.Methods))
	}
	if implStmt.Methods[0].Name.Value != "run" {
		t.Fatalf("expected method name 'run', got %s", implStmt.Methods[0].Name.Value)
	}
	if !implStmt.Methods[0].Async {
		t.Fatal("expected async impl method to have Async=true")
	}
}

func TestStringLiteralUnescaping(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`"\n\t\r\\\"";`, "\n\t\r\\\""},
		{`"\u263A\x21";`, "☺!"},
		{`"keep\qslash";`, `keep\qslash`},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		if len(program.Statements) != 1 {
			t.Fatalf("expected one statement for %q, got %d", tt.input, len(program.Statements))
		}
		stmt, ok := program.Statements[0].(*ast.ExpressionStatement)
		if !ok {
			t.Fatalf("expected ExpressionStatement, got %T", program.Statements[0])
		}
		lit, ok := stmt.Expression.(*ast.StringLiteral)
		if !ok {
			t.Fatalf("expected StringLiteral, got %T", stmt.Expression)
		}
		if lit.Value != tt.expected {
			t.Fatalf("for %q expected %q, got %q", tt.input, tt.expected, lit.Value)
		}
	}
}

// --- While statement ---

func TestWhileStatement(t *testing.T) {
	input := `while x > 0 { x = x - 1; }`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}
	ws, ok := program.Statements[0].(*ast.WhileStatement)
	if !ok {
		t.Fatalf("expected WhileStatement, got %T", program.Statements[0])
	}
	if ws.Condition == nil {
		t.Fatal("expected condition")
	}
	if ws.Body == nil || len(ws.Body.Statements) != 1 {
		t.Fatalf("expected 1 body statement, got %d", len(ws.Body.Statements))
	}
}

// --- Continue statement ---

func TestContinueStatement(t *testing.T) {
	input := `while true { continue; }`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	ws := program.Statements[0].(*ast.WhileStatement)
	_, ok := ws.Body.Statements[0].(*ast.ContinueStatement)
	if !ok {
		t.Fatalf("expected ContinueStatement, got %T", ws.Body.Statements[0])
	}
}

// --- Require statements ---

func TestRequireStatementString(t *testing.T) {
	input := `require "fmt";`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}
	req, ok := program.Statements[0].(*ast.RequireStatement)
	if !ok {
		t.Fatalf("expected RequireStatement, got %T", program.Statements[0])
	}
	if req.Path.Value != "fmt" {
		t.Fatalf("expected path 'fmt', got %s", req.Path.Value)
	}
	if req.Alias != nil {
		t.Fatal("expected no alias")
	}
}

func TestRequireStatementWithAlias(t *testing.T) {
	input := `require "fmt" as f;`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	req := program.Statements[0].(*ast.RequireStatement)
	if req.Alias == nil || req.Alias.Value != "f" {
		t.Fatalf("expected alias 'f', got %v", req.Alias)
	}
}

func TestRequireStatementNamed(t *testing.T) {
	input := `require { foo, bar } from "mylib";`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	req := program.Statements[0].(*ast.RequireStatement)
	if len(req.Names) != 2 {
		t.Fatalf("expected 2 names, got %d", len(req.Names))
	}
	if req.Names[0].Value != "foo" || req.Names[1].Value != "bar" {
		t.Fatalf("expected foo, bar got %s, %s", req.Names[0].Value, req.Names[1].Value)
	}
	if req.Path.Value != "mylib" {
		t.Fatalf("expected path 'mylib', got %s", req.Path.Value)
	}
}

func TestRequireStatementAll(t *testing.T) {
	input := `require * from "utils";`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	req := program.Statements[0].(*ast.RequireStatement)
	if !req.All {
		t.Fatal("expected All to be true")
	}
	if req.Path.Value != "utils" {
		t.Fatalf("expected path 'utils', got %s", req.Path.Value)
	}
}

// --- Array type ---

func TestArrayTypeAnnotation(t *testing.T) {
	input := `let xs: []int = arr;`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	letStmt := program.Statements[0].(*ast.LetStatement)
	arrType, ok := letStmt.Type.(*ast.ArrayType)
	if !ok {
		t.Fatalf("expected ArrayType, got %T", letStmt.Type)
	}
	basic, ok := arrType.ElementType.(*ast.BasicType)
	if !ok {
		t.Fatalf("expected BasicType element, got %T", arrType.ElementType)
	}
	if basic.Name != "int" {
		t.Fatalf("expected 'int', got %s", basic.Name)
	}
}

func TestArrayTypeWithSize(t *testing.T) {
	input := `let xs: [5]int = arr;`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	letStmt := program.Statements[0].(*ast.LetStatement)
	arrType, ok := letStmt.Type.(*ast.ArrayType)
	if !ok {
		t.Fatalf("expected ArrayType, got %T", letStmt.Type)
	}
	if arrType.Size == nil {
		t.Fatal("expected size expression")
	}
	intLit, ok := arrType.Size.(*ast.IntegerLiteral)
	if !ok {
		t.Fatalf("expected IntegerLiteral size, got %T", arrType.Size)
	}
	if intLit.Value != 5 {
		t.Fatalf("expected size 5, got %d", intLit.Value)
	}
}

// --- Char literal ---

func TestCharLiteral(t *testing.T) {
	input := `'a';`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	ch, ok := stmt.Expression.(*ast.CharLiteral)
	if !ok {
		t.Fatalf("expected CharLiteral, got %T", stmt.Expression)
	}
	if ch.Value != 'a' {
		t.Fatalf("expected 'a', got %c", ch.Value)
	}
}

func TestCharLiteralEscapes(t *testing.T) {
	tests := []struct {
		input    string
		expected rune
	}{
		{`'\n';`, '\n'},
		{`'\t';`, '\t'},
		{`'\r';`, '\r'},
		{`'\\';`, '\\'},
		{`'\'';`, '\''},
		{`'\0';`, '\x00'},
	}
	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		stmt := program.Statements[0].(*ast.ExpressionStatement)
		ch := stmt.Expression.(*ast.CharLiteral)
		if ch.Value != tt.expected {
			t.Fatalf("for %s expected %d, got %d", tt.input, tt.expected, ch.Value)
		}
	}
}

// --- Bool literal ---

func TestBoolLiteral(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"true;", true},
		{"false;", false},
	}
	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		stmt := program.Statements[0].(*ast.ExpressionStatement)
		bl, ok := stmt.Expression.(*ast.BoolLiteral)
		if !ok {
			t.Fatalf("expected BoolLiteral, got %T", stmt.Expression)
		}
		if bl.Value != tt.expected {
			t.Fatalf("expected %v, got %v", tt.expected, bl.Value)
		}
	}
}

// --- Nil literal ---

func TestNilLiteral(t *testing.T) {
	input := `nil;`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	_, ok := stmt.Expression.(*ast.NilLiteral)
	if !ok {
		t.Fatalf("expected NilLiteral, got %T", stmt.Expression)
	}
}

// --- Prefix expression ---

func TestPrefixExpressions(t *testing.T) {
	tests := []struct {
		input    string
		operator string
	}{
		{"!true;", "!"},
		{"-5;", "-"},
		{"~x;", "~"},
	}
	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		stmt := program.Statements[0].(*ast.ExpressionStatement)
		prefix, ok := stmt.Expression.(*ast.PrefixExpression)
		if !ok {
			t.Fatalf("expected PrefixExpression for %s, got %T", tt.input, stmt.Expression)
		}
		if prefix.Operator != tt.operator {
			t.Fatalf("expected operator %s, got %s", tt.operator, prefix.Operator)
		}
		if prefix.Right == nil {
			t.Fatalf("expected right operand for %s", tt.input)
		}
	}
}

// --- Grouped expression ---

func TestGroupedExpression(t *testing.T) {
	input := `(1 + 2) * 3;`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	infix, ok := stmt.Expression.(*ast.InfixExpression)
	if !ok {
		t.Fatalf("expected InfixExpression, got %T", stmt.Expression)
	}
	if infix.Operator != "*" {
		t.Fatalf("expected * operator, got %s", infix.Operator)
	}
	// left should be (1+2) -> InfixExpression
	leftInfix, ok := infix.Left.(*ast.InfixExpression)
	if !ok {
		t.Fatalf("expected InfixExpression on left, got %T", infix.Left)
	}
	if leftInfix.Operator != "+" {
		t.Fatalf("expected + operator on left, got %s", leftInfix.Operator)
	}
}

// --- Array literal ---

func TestArrayLiteral(t *testing.T) {
	input := `[1, 2, 3];`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	arr, ok := stmt.Expression.(*ast.ArrayLiteral)
	if !ok {
		t.Fatalf("expected ArrayLiteral, got %T", stmt.Expression)
	}
	if len(arr.Elements) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(arr.Elements))
	}
}

func TestEmptyArrayLiteral(t *testing.T) {
	input := `[];`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	arr, ok := stmt.Expression.(*ast.ArrayLiteral)
	if !ok {
		t.Fatalf("expected ArrayLiteral, got %T", stmt.Expression)
	}
	if len(arr.Elements) != 0 {
		t.Fatalf("expected 0 elements, got %d", len(arr.Elements))
	}
}

// --- Function literal ---

func TestFunctionLiteral(t *testing.T) {
	input := `let f = fn(x: int) -> int { return x + 1; };`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	letStmt := program.Statements[0].(*ast.LetStatement)
	fnLit, ok := letStmt.Value.(*ast.FunctionLiteral)
	if !ok {
		t.Fatalf("expected FunctionLiteral, got %T", letStmt.Value)
	}
	if len(fnLit.Parameters) != 1 {
		t.Fatalf("expected 1 parameter, got %d", len(fnLit.Parameters))
	}
	if fnLit.ReturnType == nil {
		t.Fatal("expected return type")
	}
	if fnLit.Body == nil {
		t.Fatal("expected body")
	}
}

func TestNamedFunctionLiteral(t *testing.T) {
	input := `let f = fn adder(a: int) -> int { return a + 1; };`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	letStmt := program.Statements[0].(*ast.LetStatement)
	fnLit, ok := letStmt.Value.(*ast.FunctionLiteral)
	if !ok {
		t.Fatalf("expected FunctionLiteral, got %T", letStmt.Value)
	}
	if fnLit.Name == nil || fnLit.Name.Value != "adder" {
		t.Fatalf("expected name 'adder', got %v", fnLit.Name)
	}
}

// --- Index expression ---

func TestIndexExpression(t *testing.T) {
	input := `arr[0];`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	idx, ok := stmt.Expression.(*ast.IndexExpression)
	if !ok {
		t.Fatalf("expected IndexExpression, got %T", stmt.Expression)
	}
	ident, ok := idx.Left.(*ast.Identifier)
	if !ok || ident.Value != "arr" {
		t.Fatalf("expected 'arr' on left, got %T", idx.Left)
	}
	intLit, ok := idx.Index.(*ast.IntegerLiteral)
	if !ok || intLit.Value != 0 {
		t.Fatalf("expected index 0, got %v", idx.Index)
	}
}

// --- Spawn expression ---

func TestSpawnExpression(t *testing.T) {
	input := `spawn { do_work(); };`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	sp, ok := stmt.Expression.(*ast.SpawnExpression)
	if !ok {
		t.Fatalf("expected SpawnExpression, got %T", stmt.Expression)
	}
	if sp.Body == nil || len(sp.Body.Statements) != 1 {
		t.Fatalf("expected 1 body statement")
	}
}

// --- New expression ---

func TestNewExpression(t *testing.T) {
	input := `new Point;`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	ne, ok := stmt.Expression.(*ast.NewExpression)
	if !ok {
		t.Fatalf("expected NewExpression, got %T", stmt.Expression)
	}
	named, ok := ne.Type.(*ast.NamedType)
	if !ok {
		t.Fatalf("expected NamedType, got %T", ne.Type)
	}
	if named.Name.Value != "Point" {
		t.Fatalf("expected 'Point', got %s", named.Name.Value)
	}
}

// --- peekError / synchronize ---

func TestPeekErrorAndSynchronize(t *testing.T) {
	// Missing semicolon triggers peekError, then synchronize skips to next statement
	input := `let x = 5
let y = 10;`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()

	if len(p.Errors()) == 0 {
		t.Fatal("expected parser errors for missing semicolon")
	}
	// synchronize should recover; we should get at least the second statement
	if len(program.Statements) == 0 {
		t.Fatal("expected at least one recovered statement")
	}
}

func TestSynchronizeOnUnknownToken(t *testing.T) {
	// Invalid token in statement position triggers synchronize
	input := `+ ;
let x = 1;`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()

	// Should have errors but still recover
	if len(p.Errors()) == 0 {
		t.Fatal("expected parser errors")
	}
	// Should recover the let statement
	found := false
	for _, s := range program.Statements {
		if _, ok := s.(*ast.LetStatement); ok {
			found = true
		}
	}
	if !found {
		t.Fatal("expected to recover LetStatement after sync")
	}
}

// --- parseStatement branches ---

func TestParseStatementPackedClass(t *testing.T) {
	input := `packed class Vec2 {
	x: float = 0.0
	y: float = 0.0
}`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	cls, ok := program.Statements[0].(*ast.ClassStatement)
	if !ok {
		t.Fatalf("expected ClassStatement, got %T", program.Statements[0])
	}
	if !cls.Packed {
		t.Fatal("expected Packed to be true")
	}
	if cls.Name.Value != "Vec2" {
		t.Fatalf("expected name Vec2, got %s", cls.Name.Value)
	}
}

func TestParseStatementPackedInvalid(t *testing.T) {
	input := `packed fn foo() {}`
	l := lexer.New(input)
	p := New(l)
	_ = p.ParseProgram()
	if len(p.Errors()) == 0 {
		t.Fatal("expected error for packed non-class")
	}
}

func TestStaticLetStatement(t *testing.T) {
	input := `static let counter = 0;`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	letStmt, ok := program.Statements[0].(*ast.LetStatement)
	if !ok {
		t.Fatalf("expected LetStatement, got %T", program.Statements[0])
	}
	if !letStmt.Static {
		t.Fatal("expected Static to be true")
	}
}

func TestStaticMutStatement(t *testing.T) {
	input := `static mut counter = 0;`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	letStmt, ok := program.Statements[0].(*ast.LetStatement)
	if !ok {
		t.Fatalf("expected LetStatement, got %T", program.Statements[0])
	}
	if !letStmt.Static {
		t.Fatal("expected Static to be true")
	}
	if !letStmt.Mutable {
		t.Fatal("expected Mutable to be true")
	}
}

func TestStaticConstStatement(t *testing.T) {
	input := `static const MAX = 100;`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	cs, ok := program.Statements[0].(*ast.ConstStatement)
	if !ok {
		t.Fatalf("expected ConstStatement, got %T", program.Statements[0])
	}
	if !cs.Static {
		t.Fatal("expected Static to be true")
	}
}

func TestStaticInvalid(t *testing.T) {
	input := `static fn foo() {}`
	l := lexer.New(input)
	p := New(l)
	_ = p.ParseProgram()
	if len(p.Errors()) == 0 {
		t.Fatal("expected error for static fn")
	}
}

func TestAsyncInvalid(t *testing.T) {
	input := `async let x = 1;`
	l := lexer.New(input)
	p := New(l)
	_ = p.ParseProgram()
	if len(p.Errors()) == 0 {
		t.Fatal("expected error for async non-fn")
	}
}

// --- parsePublicStatement branches ---

func TestPublicClass(t *testing.T) {
	input := `pub class Foo { x: int }`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	cls, ok := program.Statements[0].(*ast.ClassStatement)
	if !ok {
		t.Fatalf("expected ClassStatement, got %T", program.Statements[0])
	}
	if !cls.Public {
		t.Fatal("expected Public to be true")
	}
}

func TestPublicInterface(t *testing.T) {
	input := `pub interface Greeter {
	fn greet(&self) -> string;
}`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	iface, ok := program.Statements[0].(*ast.InterfaceStatement)
	if !ok {
		t.Fatalf("expected InterfaceStatement, got %T", program.Statements[0])
	}
	if !iface.Public {
		t.Fatal("expected Public to be true")
	}
}

func TestPublicConst(t *testing.T) {
	input := `pub const MAX = 100;`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	cs, ok := program.Statements[0].(*ast.ConstStatement)
	if !ok {
		t.Fatalf("expected ConstStatement, got %T", program.Statements[0])
	}
	if !cs.Public {
		t.Fatal("expected Public to be true")
	}
}

func TestPublicLet(t *testing.T) {
	input := `pub let x = 1;`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	ls, ok := program.Statements[0].(*ast.LetStatement)
	if !ok {
		t.Fatalf("expected LetStatement, got %T", program.Statements[0])
	}
	if !ls.Public {
		t.Fatal("expected Public to be true")
	}
}

func TestPublicFn(t *testing.T) {
	input := `pub fn foo() { return; }`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	fs, ok := program.Statements[0].(*ast.FunctionStatement)
	if !ok {
		t.Fatalf("expected FunctionStatement, got %T", program.Statements[0])
	}
	if !fs.Public {
		t.Fatal("expected Public to be true")
	}
}

func TestPublicInvalid(t *testing.T) {
	input := `pub while true {}`
	l := lexer.New(input)
	p := New(l)
	_ = p.ParseProgram()
	if len(p.Errors()) == 0 {
		t.Fatal("expected error for pub before invalid keyword")
	}
}

// --- parseTypeExpr additional branches ---

func TestTypeExprVoid(t *testing.T) {
	input := `fn foo() -> void { return; }`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	fs := program.Statements[0].(*ast.FunctionStatement)
	bt, ok := fs.ReturnType.(*ast.BasicType)
	if !ok {
		t.Fatalf("expected BasicType, got %T", fs.ReturnType)
	}
	if bt.Name != "void" {
		t.Fatalf("expected void, got %s", bt.Name)
	}
}

func TestTypeExprAny(t *testing.T) {
	input := `fn foo(x: any) { return; }`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	fs := program.Statements[0].(*ast.FunctionStatement)
	bt, ok := fs.Parameters[0].Type.(*ast.BasicType)
	if !ok {
		t.Fatalf("expected BasicType, got %T", fs.Parameters[0].Type)
	}
	if bt.Name != "any" {
		t.Fatalf("expected any, got %s", bt.Name)
	}
}

func TestTypeExprFixedWidthTypes(t *testing.T) {
	types := []string{"u8", "u16", "u32", "u64", "i8", "i16", "i32", "i64", "f32", "f64", "usize", "isize"}
	for _, ty := range types {
		input := `fn foo(x: ` + ty + `) { return; }`
		l := lexer.New(input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		fs := program.Statements[0].(*ast.FunctionStatement)
		bt, ok := fs.Parameters[0].Type.(*ast.BasicType)
		if !ok {
			t.Fatalf("for type %s: expected BasicType, got %T", ty, fs.Parameters[0].Type)
		}
		if bt.Name != ty {
			t.Fatalf("expected %s, got %s", ty, bt.Name)
		}
	}
}

func TestTypeExprNamedType(t *testing.T) {
	input := `let x: MyType = val;`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	ls := program.Statements[0].(*ast.LetStatement)
	nt, ok := ls.Type.(*ast.NamedType)
	if !ok {
		t.Fatalf("expected NamedType, got %T", ls.Type)
	}
	if nt.Name.Value != "MyType" {
		t.Fatalf("expected MyType, got %s", nt.Name.Value)
	}
}

func TestTypeExprRefType(t *testing.T) {
	input := `fn foo(x: &int) { return; }`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	fs := program.Statements[0].(*ast.FunctionStatement)
	rt, ok := fs.Parameters[0].Type.(*ast.RefType)
	if !ok {
		t.Fatalf("expected RefType, got %T", fs.Parameters[0].Type)
	}
	if rt.Mutable {
		t.Fatal("expected immutable ref")
	}
	bt, ok := rt.Inner.(*ast.BasicType)
	if !ok || bt.Name != "int" {
		t.Fatalf("expected inner BasicType int, got %v", rt.Inner)
	}
}

func TestTypeExprMutRefType(t *testing.T) {
	input := `fn foo(x: &mut int) { return; }`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	fs := program.Statements[0].(*ast.FunctionStatement)
	rt, ok := fs.Parameters[0].Type.(*ast.RefType)
	if !ok {
		t.Fatalf("expected RefType, got %T", fs.Parameters[0].Type)
	}
	if !rt.Mutable {
		t.Fatal("expected mutable ref")
	}
}

func TestTypeExprVolatile(t *testing.T) {
	input := `let x: volatile<int> = v;`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	ls := program.Statements[0].(*ast.LetStatement)
	vt, ok := ls.Type.(*ast.VolatileType)
	if !ok {
		t.Fatalf("expected VolatileType, got %T", ls.Type)
	}
	bt, ok := vt.Inner.(*ast.BasicType)
	if !ok || bt.Name != "int" {
		t.Fatalf("expected inner BasicType int, got %v", vt.Inner)
	}
}

// --- parseReceiverAndParams branches ---

func TestReceiverMutRefWithParams(t *testing.T) {
	input := `class Foo {
	fn bar(&mut self, x: int) -> int { return x; }
}`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	cls := program.Statements[0].(*ast.ClassStatement)
	m := cls.Methods[0]
	if m.Receiver != ast.RecvMutRef {
		t.Fatalf("expected RecvMutRef, got %d", m.Receiver)
	}
	if len(m.Parameters) != 1 {
		t.Fatalf("expected 1 param, got %d", len(m.Parameters))
	}
}

func TestReceiverValue(t *testing.T) {
	input := `class Foo {
	fn consume(self) { return; }
}`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	cls := program.Statements[0].(*ast.ClassStatement)
	m := cls.Methods[0]
	if m.Receiver != ast.RecvValue {
		t.Fatalf("expected RecvValue, got %d", m.Receiver)
	}
}

func TestReceiverValueWithParams(t *testing.T) {
	input := `class Foo {
	fn consume(self, x: int) { return; }
}`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	cls := program.Statements[0].(*ast.ClassStatement)
	m := cls.Methods[0]
	if m.Receiver != ast.RecvValue {
		t.Fatalf("expected RecvValue, got %d", m.Receiver)
	}
	if len(m.Parameters) != 1 {
		t.Fatalf("expected 1 param, got %d", len(m.Parameters))
	}
}

func TestReceiverRefWithParams(t *testing.T) {
	input := `class Foo {
	fn read(&self, idx: int) -> int { return idx; }
}`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	cls := program.Statements[0].(*ast.ClassStatement)
	m := cls.Methods[0]
	if m.Receiver != ast.RecvRef {
		t.Fatalf("expected RecvRef, got %d", m.Receiver)
	}
	if len(m.Parameters) != 1 {
		t.Fatalf("expected 1 param, got %d", len(m.Parameters))
	}
}

// --- parseParameter mutable branch ---

func TestMutableParameter(t *testing.T) {
	input := `fn foo(mut x: int) { return; }`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	fs := program.Statements[0].(*ast.FunctionStatement)
	if len(fs.Parameters) != 1 {
		t.Fatalf("expected 1 param, got %d", len(fs.Parameters))
	}
	if !fs.Parameters[0].Mutable {
		t.Fatal("expected mutable parameter")
	}
	if fs.Parameters[0].Name.Value != "x" {
		t.Fatalf("expected name 'x', got %s", fs.Parameters[0].Name.Value)
	}
}

func TestParameterWithoutType(t *testing.T) {
	input := `fn foo(x) { return; }`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	fs := program.Statements[0].(*ast.FunctionStatement)
	if fs.Parameters[0].Type != nil {
		t.Fatal("expected no type annotation")
	}
}

// --- Return statement with no value ---

func TestReturnStatementEmpty(t *testing.T) {
	input := `return;`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	rs, ok := program.Statements[0].(*ast.ReturnStatement)
	if !ok {
		t.Fatalf("expected ReturnStatement, got %T", program.Statements[0])
	}
	if rs.ReturnValue != nil {
		t.Fatal("expected nil return value")
	}
}

// --- Const statement with type annotation ---

func TestConstStatementWithType(t *testing.T) {
	input := `const MAX: int = 100;`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	cs := program.Statements[0].(*ast.ConstStatement)
	if cs.Type == nil {
		t.Fatal("expected type annotation")
	}
	bt, ok := cs.Type.(*ast.BasicType)
	if !ok || bt.Name != "int" {
		t.Fatalf("expected BasicType int, got %v", cs.Type)
	}
}

// --- Let statement with type annotation ---

func TestLetStatementWithType(t *testing.T) {
	input := `let x: int = 5;`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	ls := program.Statements[0].(*ast.LetStatement)
	if ls.Type == nil {
		t.Fatal("expected type annotation")
	}
}

// --- Break inside loop ---

func TestBreakInsideWhile(t *testing.T) {
	input := `while true { break; }`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	ws := program.Statements[0].(*ast.WhileStatement)
	_, ok := ws.Body.Statements[0].(*ast.BreakStatement)
	if !ok {
		t.Fatalf("expected BreakStatement, got %T", ws.Body.Statements[0])
	}
}

// --- MatchExpression with block body arm ---

func TestMatchExpressionBlockArm(t *testing.T) {
	input := `match x {
	1 => { let y = 2; },
	2 => 3,
};`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	me := stmt.Expression.(*ast.MatchExpression)
	if len(me.Arms) != 2 {
		t.Fatalf("expected 2 arms, got %d", len(me.Arms))
	}
	_, ok := me.Arms[0].Body.(*ast.BlockExpression)
	if !ok {
		t.Fatalf("expected BlockExpression for first arm, got %T", me.Arms[0].Body)
	}
}

// --- Empty map literal ---

func TestEmptyMapLiteral(t *testing.T) {
	input := `{};`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	ml, ok := stmt.Expression.(*ast.MapLiteral)
	if !ok {
		t.Fatalf("expected MapLiteral, got %T", stmt.Expression)
	}
	if len(ml.Pairs) != 0 {
		t.Fatalf("expected 0 pairs, got %d", len(ml.Pairs))
	}
}

// --- Member expression ---

func TestMemberExpression(t *testing.T) {
	input := `obj.field;`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	me, ok := stmt.Expression.(*ast.MemberExpression)
	if !ok {
		t.Fatalf("expected MemberExpression, got %T", stmt.Expression)
	}
	if me.Member.Value != "field" {
		t.Fatalf("expected 'field', got %s", me.Member.Value)
	}
}

// --- Self expression ---

func TestSelfExpression(t *testing.T) {
	input := `class Foo {
	fn me(&self) -> string { return self.name; }
}`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	cls := program.Statements[0].(*ast.ClassStatement)
	body := cls.Methods[0].Body
	retStmt := body.Statements[0].(*ast.ReturnStatement)
	me, ok := retStmt.ReturnValue.(*ast.MemberExpression)
	if !ok {
		t.Fatalf("expected MemberExpression, got %T", retStmt.ReturnValue)
	}
	self, ok := me.Object.(*ast.Identifier)
	if !ok || self.Value != "self" {
		t.Fatalf("expected self identifier, got %v", me.Object)
	}
}

// --- pub async fn ---

func TestPublicAsyncFnInvalid(t *testing.T) {
	input := `pub async let x = 1;`
	l := lexer.New(input)
	p := New(l)
	_ = p.ParseProgram()
	if len(p.Errors()) == 0 {
		t.Fatal("expected error for pub async non-fn")
	}
}

// --- Require error cases ---

func TestRequireInvalidToken(t *testing.T) {
	input := `require 42;`
	l := lexer.New(input)
	p := New(l)
	_ = p.ParseProgram()
	if len(p.Errors()) == 0 {
		t.Fatal("expected error for require with invalid token")
	}
}

// --- Float literal ---

func TestFloatLiteral(t *testing.T) {
	input := `3.14;`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	fl, ok := stmt.Expression.(*ast.FloatLiteral)
	if !ok {
		t.Fatalf("expected FloatLiteral, got %T", stmt.Expression)
	}
	if fl.Value != 3.14 {
		t.Fatalf("expected 3.14, got %f", fl.Value)
	}
}

// --- If statement as top-level statement ---

func TestIfAsStatement(t *testing.T) {
	input := `if true { let x = 1; }`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	es, ok := program.Statements[0].(*ast.ExpressionStatement)
	if !ok {
		t.Fatalf("expected ExpressionStatement, got %T", program.Statements[0])
	}
	_, ok = es.Expression.(*ast.IfExpression)
	if !ok {
		t.Fatalf("expected IfExpression, got %T", es.Expression)
	}
}

// --- Boolean/logical operators ---

func TestLogicalOperators(t *testing.T) {
	tests := []struct {
		input string
		op    string
	}{
		{"true && false;", "&&"},
		{"true || false;", "||"},
	}
	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		stmt := program.Statements[0].(*ast.ExpressionStatement)
		infix, ok := stmt.Expression.(*ast.InfixExpression)
		if !ok {
			t.Fatalf("expected InfixExpression for %s, got %T", tt.op, stmt.Expression)
		}
		if infix.Operator != tt.op {
			t.Fatalf("expected %s, got %s", tt.op, infix.Operator)
		}
	}
}

// --- Bitwise operators ---

func TestBitwiseOperators(t *testing.T) {
	tests := []struct {
		input string
		op    string
	}{
		{"x | y;", "|"},
		{"x ^ y;", "^"},
	}
	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		stmt := program.Statements[0].(*ast.ExpressionStatement)
		infix, ok := stmt.Expression.(*ast.InfixExpression)
		if !ok {
			t.Fatalf("expected InfixExpression for %s, got %T", tt.op, stmt.Expression)
		}
		if infix.Operator != tt.op {
			t.Fatalf("expected %s, got %s", tt.op, infix.Operator)
		}
	}
}

// --- Numeric type as identifiers used in calls ---

func TestNumericTypeAsIdentifiers(t *testing.T) {
	numTypes := []string{"u8", "u16", "u32", "u64", "i8", "i16", "i32", "i64", "f32", "f64", "usize", "isize"}
	for _, ty := range numTypes {
		input := ty + `(42);`
		l := lexer.New(input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		stmt := program.Statements[0].(*ast.ExpressionStatement)
		call, ok := stmt.Expression.(*ast.CallExpression)
		if !ok {
			t.Fatalf("for %s: expected CallExpression, got %T", ty, stmt.Expression)
		}
		ident, ok := call.Function.(*ast.Identifier)
		if !ok || ident.Value != ty {
			t.Fatalf("for %s: expected ident %s, got %v", ty, ty, call.Function)
		}
	}
}

// --- Chained member and index ---

func TestChainedMemberAndIndex(t *testing.T) {
	input := `a.b[0].c;`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	me, ok := stmt.Expression.(*ast.MemberExpression)
	if !ok {
		t.Fatalf("expected MemberExpression, got %T", stmt.Expression)
	}
	if me.Member.Value != "c" {
		t.Fatalf("expected 'c', got %s", me.Member.Value)
	}
}

// --- Interface method with &mut self ---

func TestInterfaceMethodMutRef(t *testing.T) {
	input := `interface Mutable {
	fn mutate(&mut self);
}`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	iface := program.Statements[0].(*ast.InterfaceStatement)
	if iface.Methods[0].Receiver != ast.RecvMutRef {
		t.Fatalf("expected RecvMutRef, got %d", iface.Methods[0].Receiver)
	}
}

// --- Interface method with value self ---

func TestInterfaceMethodValueSelf(t *testing.T) {
	input := `interface Consumable {
	fn consume(self) -> int;
}`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	iface := program.Statements[0].(*ast.InterfaceStatement)
	if iface.Methods[0].Receiver != ast.RecvValue {
		t.Fatalf("expected RecvValue, got %d", iface.Methods[0].Receiver)
	}
}

// --- Function with multiple params ---

func TestFunctionMultipleParams(t *testing.T) {
	input := `fn add(a: int, b: int, c: int) -> int { return a; }`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	fs := program.Statements[0].(*ast.FunctionStatement)
	if len(fs.Parameters) != 3 {
		t.Fatalf("expected 3 params, got %d", len(fs.Parameters))
	}
}

// --- Impl with &mut self method ---

func TestImplMutRefMethod(t *testing.T) {
	input := `impl Setter for Obj {
	fn set(&mut self, v: int) { self.v = v; }
}`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	impl := program.Statements[0].(*ast.ImplStatement)
	if impl.Methods[0].Receiver != ast.RecvMutRef {
		t.Fatalf("expected RecvMutRef, got %d", impl.Methods[0].Receiver)
	}
	if len(impl.Methods[0].Parameters) != 1 {
		t.Fatalf("expected 1 param, got %d", len(impl.Methods[0].Parameters))
	}
}

// --- Require with empty braces ---

func TestRequireEmptyBraces(t *testing.T) {
	input := `require {} from "mod";`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	req := program.Statements[0].(*ast.RequireStatement)
	if len(req.Names) != 0 {
		t.Fatalf("expected 0 names, got %d", len(req.Names))
	}
}

// --- Error paths for better coverage ---

// parseLetStatement error: missing assign
func TestLetStatementMissingAssign(t *testing.T) {
	input := `let x: int;`
	l := lexer.New(input)
	p := New(l)
	_ = p.ParseProgram()
	if len(p.Errors()) == 0 {
		t.Fatal("expected error for missing assign in let")
	}
}

// parseLetStatement error: missing ident
func TestLetStatementMissingIdent(t *testing.T) {
	input := `let = 5;`
	l := lexer.New(input)
	p := New(l)
	_ = p.ParseProgram()
	if len(p.Errors()) == 0 {
		t.Fatal("expected error for missing ident in let")
	}
}

// parseConstStatement error: missing ident
func TestConstStatementMissingIdent(t *testing.T) {
	input := `const = 5;`
	l := lexer.New(input)
	p := New(l)
	_ = p.ParseProgram()
	if len(p.Errors()) == 0 {
		t.Fatal("expected error for missing ident in const")
	}
}

// parseConstStatement error: missing assign
func TestConstStatementMissingAssign(t *testing.T) {
	input := `const X: int;`
	l := lexer.New(input)
	p := New(l)
	_ = p.ParseProgram()
	if len(p.Errors()) == 0 {
		t.Fatal("expected error for missing assign in const")
	}
}

// parseFunctionStatement error paths
func TestFunctionStatementMissingIdent(t *testing.T) {
	input := `fn () {}`
	l := lexer.New(input)
	p := New(l)
	_ = p.ParseProgram()
	if len(p.Errors()) == 0 {
		t.Fatal("expected error for fn missing ident")
	}
}

func TestFunctionStatementMissingLParen(t *testing.T) {
	input := `fn foo {}`
	l := lexer.New(input)
	p := New(l)
	_ = p.ParseProgram()
	if len(p.Errors()) == 0 {
		t.Fatal("expected error for fn missing lparen")
	}
}

func TestFunctionStatementMissingLBrace(t *testing.T) {
	input := `fn foo() -> int`
	l := lexer.New(input)
	p := New(l)
	_ = p.ParseProgram()
	if len(p.Errors()) == 0 {
		t.Fatal("expected error for fn missing lbrace")
	}
}

// parseWhileStatement error: missing lbrace
func TestWhileStatementMissingLBrace(t *testing.T) {
	input := `while true`
	l := lexer.New(input)
	p := New(l)
	_ = p.ParseProgram()
	if len(p.Errors()) == 0 {
		t.Fatal("expected error for while missing lbrace")
	}
}

// parseBreakStatement error: missing semi
func TestBreakStatementMissingSemi(t *testing.T) {
	input := `for { break }`
	l := lexer.New(input)
	p := New(l)
	_ = p.ParseProgram()
	if len(p.Errors()) == 0 {
		t.Fatal("expected error for break missing semi")
	}
}

// parseContinueStatement error: missing semi
func TestContinueStatementMissingSemi(t *testing.T) {
	input := `for { continue }`
	l := lexer.New(input)
	p := New(l)
	_ = p.ParseProgram()
	if len(p.Errors()) == 0 {
		t.Fatal("expected error for continue missing semi")
	}
}

// parseReturnStatement error: missing semi
func TestReturnStatementMissingSemi(t *testing.T) {
	input := `fn foo() { return 5 }`
	l := lexer.New(input)
	p := New(l)
	_ = p.ParseProgram()
	if len(p.Errors()) == 0 {
		t.Fatal("expected error for return missing semi")
	}
}

// parseGroupedExpression error: missing rparen
func TestGroupedExpressionMissingRParen(t *testing.T) {
	input := `(1 + 2;`
	l := lexer.New(input)
	p := New(l)
	_ = p.ParseProgram()
	if len(p.Errors()) == 0 {
		t.Fatal("expected error for missing rparen")
	}
}

// parseIndexExpression error: missing rbracket
func TestIndexExpressionMissingRBracket(t *testing.T) {
	input := `arr[0;`
	l := lexer.New(input)
	p := New(l)
	_ = p.ParseProgram()
	if len(p.Errors()) == 0 {
		t.Fatal("expected error for missing rbracket in index")
	}
}

// parseMemberExpression error: missing ident
func TestMemberExpressionMissingIdent(t *testing.T) {
	input := `obj.42;`
	l := lexer.New(input)
	p := New(l)
	_ = p.ParseProgram()
	if len(p.Errors()) == 0 {
		t.Fatal("expected error for member missing ident")
	}
}

// parseSpawnExpression error: missing lbrace
func TestSpawnExpressionMissingLBrace(t *testing.T) {
	input := `spawn 42;`
	l := lexer.New(input)
	p := New(l)
	_ = p.ParseProgram()
	if len(p.Errors()) == 0 {
		t.Fatal("expected error for spawn missing lbrace")
	}
}

// parseOkExpression error: missing lparen
func TestOkExpressionMissingLParen(t *testing.T) {
	input := `Ok 42;`
	l := lexer.New(input)
	p := New(l)
	_ = p.ParseProgram()
	if len(p.Errors()) == 0 {
		t.Fatal("expected error for Ok missing lparen")
	}
}

// parseErrExpression error: missing lparen
func TestErrExpressionMissingLParen(t *testing.T) {
	input := `Err 42;`
	l := lexer.New(input)
	p := New(l)
	_ = p.ParseProgram()
	if len(p.Errors()) == 0 {
		t.Fatal("expected error for Err missing lparen")
	}
}

// parseMatchExpression error: missing lbrace
func TestMatchExpressionMissingLBrace(t *testing.T) {
	input := `match x 1 => 2;`
	l := lexer.New(input)
	p := New(l)
	_ = p.ParseProgram()
	if len(p.Errors()) == 0 {
		t.Fatal("expected error for match missing lbrace")
	}
}

// parseMapLiteral error: missing colon
func TestMapLiteralMissingColon(t *testing.T) {
	input := `{"a" 1};`
	l := lexer.New(input)
	p := New(l)
	_ = p.ParseProgram()
	if len(p.Errors()) == 0 {
		t.Fatal("expected error for map missing colon")
	}
}

// parseForInStatement error: missing lbrace
func TestForInStatementMissingLBrace(t *testing.T) {
	input := `for x in arr`
	l := lexer.New(input)
	p := New(l)
	_ = p.ParseProgram()
	if len(p.Errors()) == 0 {
		t.Fatal("expected error for for-in missing lbrace")
	}
}

// parseCStyleFor error: missing lparen
func TestCStyleForMissingLParen(t *testing.T) {
	input := `for let i = 0; i < 10; i = i + 1 {}`
	l := lexer.New(input)
	p := New(l)
	_ = p.ParseProgram()
	if len(p.Errors()) == 0 {
		t.Fatal("expected error for c-style for missing lparen")
	}
}

// parseClassStatement error: missing ident
func TestClassStatementMissingIdent(t *testing.T) {
	input := `class {}`
	l := lexer.New(input)
	p := New(l)
	_ = p.ParseProgram()
	if len(p.Errors()) == 0 {
		t.Fatal("expected error for class missing ident")
	}
}

// parseInterfaceStatement error: missing ident
func TestInterfaceStatementMissingIdent(t *testing.T) {
	input := `interface {}`
	l := lexer.New(input)
	p := New(l)
	_ = p.ParseProgram()
	if len(p.Errors()) == 0 {
		t.Fatal("expected error for interface missing ident")
	}
}

// parseImplStatement error: missing FOR keyword
func TestImplStatementMissingFor(t *testing.T) {
	input := `impl Foo Bar {}`
	l := lexer.New(input)
	p := New(l)
	_ = p.ParseProgram()
	if len(p.Errors()) == 0 {
		t.Fatal("expected error for impl missing for")
	}
}

// parseFunctionLiteral error: missing lbrace
func TestFunctionLiteralMissingLBrace(t *testing.T) {
	input := `let f = fn(x: int) -> int;`
	l := lexer.New(input)
	p := New(l)
	_ = p.ParseProgram()
	if len(p.Errors()) == 0 {
		t.Fatal("expected error for fn literal missing lbrace")
	}
}

// parseFunctionLiteral without return type
func TestFunctionLiteralNoReturnType(t *testing.T) {
	input := `let f = fn(x: int) { return; };`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	ls := program.Statements[0].(*ast.LetStatement)
	fnLit, ok := ls.Value.(*ast.FunctionLiteral)
	if !ok {
		t.Fatalf("expected FunctionLiteral, got %T", ls.Value)
	}
	if fnLit.ReturnType != nil {
		t.Fatal("expected no return type")
	}
}

// For loop with mut init
func TestForLoopMutInit(t *testing.T) {
	input := `for (mut i = 0; i < 10; i = i + 1) { print(i); }`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	forStmt := program.Statements[0].(*ast.ForStatement)
	letStmt, ok := forStmt.Init.(*ast.LetStatement)
	if !ok {
		t.Fatalf("expected LetStatement init, got %T", forStmt.Init)
	}
	if !letStmt.Mutable {
		t.Fatal("expected mutable init")
	}
}

// For loop with expression init (not let)
func TestCStyleForExprInit(t *testing.T) {
	input := `for (i = 0; i < 10; i = i + 1) { print(i); }`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	forStmt, ok := program.Statements[0].(*ast.ForStatement)
	if !ok {
		t.Fatalf("expected ForStatement, got %T", program.Statements[0])
	}
	_, ok = forStmt.Init.(*ast.ExpressionStatement)
	if !ok {
		t.Fatalf("expected ExpressionStatement init, got %T", forStmt.Init)
	}
}

// parseReceiverAndParams error: missing rparen
func TestReceiverMissingRParen(t *testing.T) {
	input := `class Foo { fn bar(&self { return; } }`
	l := lexer.New(input)
	p := New(l)
	_ = p.ParseProgram()
	if len(p.Errors()) == 0 {
		t.Fatal("expected error for missing rparen in receiver")
	}
}

// parseMethodDecl error: missing lparen
func TestMethodDeclMissingLParen(t *testing.T) {
	input := `class Foo { fn bar { return; } }`
	l := lexer.New(input)
	p := New(l)
	_ = p.ParseProgram()
	if len(p.Errors()) == 0 {
		t.Fatal("expected error for method missing lparen")
	}
}

// parseMethodDecl error: missing lbrace
func TestMethodDeclMissingLBrace(t *testing.T) {
	input := `class Foo { fn bar() -> int }`
	l := lexer.New(input)
	p := New(l)
	_ = p.ParseProgram()
	// The "}" closes the class, not the method, so lbrace error expected
	if len(p.Errors()) == 0 {
		t.Fatal("expected error for method missing lbrace")
	}
}

// parseIfExpression error: missing lbrace
func TestIfExpressionMissingLBrace(t *testing.T) {
	input := `if true return;`
	l := lexer.New(input)
	p := New(l)
	_ = p.ParseProgram()
	if len(p.Errors()) == 0 {
		t.Fatal("expected error for if missing lbrace")
	}
}

// Interpolated string with escaped braces
func TestInterpolatedStringEscapedBraces(t *testing.T) {
	input := `f"{{literal}} and {x}";`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	interp, ok := stmt.Expression.(*ast.InterpolatedString)
	if !ok {
		t.Fatalf("expected InterpolatedString, got %T", stmt.Expression)
	}
	// Should have: "{literal}" as string, "} and " as string, x as ident
	if len(interp.Parts) < 2 {
		t.Fatalf("expected at least 2 parts, got %d", len(interp.Parts))
	}
}

// Interpolated string with closing escaped brace
func TestInterpolatedStringEscapedClosingBrace(t *testing.T) {
	input := `f"a}}b";`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	interp := stmt.Expression.(*ast.InterpolatedString)
	// Should have single string part "a}b"
	if len(interp.Parts) != 1 {
		t.Fatalf("expected 1 part, got %d", len(interp.Parts))
	}
	sl, ok := interp.Parts[0].(*ast.StringLiteral)
	if !ok {
		t.Fatalf("expected StringLiteral, got %T", interp.Parts[0])
	}
	if sl.Value != "a}b" {
		t.Fatalf("expected 'a}b', got %q", sl.Value)
	}
}

// parseRemainingParams with multiple params
func TestRemainingParamsMultiple(t *testing.T) {
	input := `class Foo {
	fn bar(&self, a: int, b: string, c: float) { return; }
}`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	cls := program.Statements[0].(*ast.ClassStatement)
	if len(cls.Methods[0].Parameters) != 3 {
		t.Fatalf("expected 3 params, got %d", len(cls.Methods[0].Parameters))
	}
}

// parseFieldDecl error: missing colon
func TestFieldDeclMissingColon(t *testing.T) {
	input := `class Foo { x int }`
	l := lexer.New(input)
	p := New(l)
	_ = p.ParseProgram()
	if len(p.Errors()) == 0 {
		t.Fatal("expected error for field missing colon")
	}
}

// parseImplMethodDecl error: missing ident
func TestImplMethodDeclMissingIdent(t *testing.T) {
	input := `impl Foo for Bar { fn () {} }`
	l := lexer.New(input)
	p := New(l)
	_ = p.ParseProgram()
	if len(p.Errors()) == 0 {
		t.Fatal("expected error for impl method missing ident")
	}
}

// parseMethodSignature error: missing ident
func TestMethodSignatureMissingIdent(t *testing.T) {
	input := `interface Foo { fn (); }`
	l := lexer.New(input)
	p := New(l)
	_ = p.ParseProgram()
	if len(p.Errors()) == 0 {
		t.Fatal("expected error for method sig missing ident")
	}
}

// Comment skipping in nextToken
func TestCommentSkipping(t *testing.T) {
	input := `let x = 5; // this is a comment
let y = 10;`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 2 {
		t.Fatalf("expected 2 statements, got %d", len(program.Statements))
	}
}

// parseReceiverAndParams: regular params (no receiver)
func TestInterfaceMethodWithParams(t *testing.T) {
	input := `interface Calc {
	fn add(a: int, b: int) -> int;
}`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	iface := program.Statements[0].(*ast.InterfaceStatement)
	if iface.Methods[0].Receiver != ast.RecvNone {
		t.Fatalf("expected RecvNone, got %d", iface.Methods[0].Receiver)
	}
	if len(iface.Methods[0].Parameters) != 2 {
		t.Fatalf("expected 2 params, got %d", len(iface.Methods[0].Parameters))
	}
}

// For c-style for loop: empty init section
func TestCStyleForEmptyInit(t *testing.T) {
	input := `for (; i < 10; i = i + 1) { print(i); }`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	forStmt := program.Statements[0].(*ast.ForStatement)
	if forStmt.Init != nil {
		t.Fatal("expected nil init")
	}
}

// curPrecedence: test with a token that has no precedence entry
func TestNoPrefixParseFunction(t *testing.T) {
	input := `);`
	l := lexer.New(input)
	p := New(l)
	_ = p.ParseProgram()
	if len(p.Errors()) == 0 {
		t.Fatal("expected error for no prefix parse function")
	}
}

// Require: named with missing FROM keyword
func TestRequireNamedMissingFrom(t *testing.T) {
	input := `require { foo } "lib";`
	l := lexer.New(input)
	p := New(l)
	_ = p.ParseProgram()
	if len(p.Errors()) == 0 {
		t.Fatal("expected error for require missing from")
	}
}

// Require: all missing FROM keyword
func TestRequireAllMissingFrom(t *testing.T) {
	input := `require * "lib";`
	l := lexer.New(input)
	p := New(l)
	_ = p.ParseProgram()
	if len(p.Errors()) == 0 {
		t.Fatal("expected error for require * missing from")
	}
}

func checkParserErrors(t *testing.T, p *Parser) {
	errors := p.Errors()
	if len(errors) == 0 {
		return
	}

	t.Errorf("parser has %d errors", len(errors))
	for _, msg := range errors {
		t.Errorf("parser error: %s", msg)
	}
	t.FailNow()
}

func TestFunctionKeywordAlias(t *testing.T) {
	input := `function add(a: int, b: int) -> int {
	return a + b;
}`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	fnStmt, ok := program.Statements[0].(*ast.FunctionStatement)
	if !ok {
		t.Fatalf("stmt not *ast.FunctionStatement. got=%T", program.Statements[0])
	}

	if fnStmt.Name.Value != "add" {
		t.Fatalf("expected function name 'add', got %s", fnStmt.Name.Value)
	}

	if len(fnStmt.Parameters) != 2 {
		t.Fatalf("expected 2 parameters, got %d", len(fnStmt.Parameters))
	}
}

func TestUnsafeFunctionStatement(t *testing.T) {
	input := `unsafe fn set_sp(sp: usize) {
	asm("mov rsp, %0");
}`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	fnStmt, ok := program.Statements[0].(*ast.FunctionStatement)
	if !ok {
		t.Fatalf("stmt not *ast.FunctionStatement. got=%T", program.Statements[0])
	}

	if fnStmt.Name.Value != "set_sp" {
		t.Fatalf("expected function name 'set_sp', got %s", fnStmt.Name.Value)
	}

	if !fnStmt.Unsafe {
		t.Fatal("expected function to be marked unsafe")
	}
}

func TestUnsafeBlock(t *testing.T) {
	input := `unsafe {
	asm("nop");
}`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	unsafeStmt, ok := program.Statements[0].(*ast.UnsafeStatement)
	if !ok {
		t.Fatalf("stmt not *ast.UnsafeStatement. got=%T", program.Statements[0])
	}

	if len(unsafeStmt.Body.Statements) != 1 {
		t.Fatalf("expected 1 statement in unsafe body, got %d", len(unsafeStmt.Body.Statements))
	}
}

func TestAsmExpression(t *testing.T) {
	input := `unsafe {
	asm("nop");
}`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	unsafeStmt := program.Statements[0].(*ast.UnsafeStatement)
	exprStmt, ok := unsafeStmt.Body.Statements[0].(*ast.ExpressionStatement)
	if !ok {
		t.Fatalf("expected ExpressionStatement inside unsafe, got %T", unsafeStmt.Body.Statements[0])
	}

	asmExpr, ok := exprStmt.Expression.(*ast.AsmExpression)
	if !ok {
		t.Fatalf("expected AsmExpression, got %T", exprStmt.Expression)
	}

	if asmExpr.Template.Value != "nop" {
		t.Fatalf("expected asm template 'nop', got %q", asmExpr.Template.Value)
	}
}
