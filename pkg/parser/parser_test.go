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

func TestPipeExpression(t *testing.T) {
	input := `x |> double |> print;`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*ast.ExpressionStatement)
	if !ok {
		t.Fatalf("stmt not *ast.ExpressionStatement. got=%T", program.Statements[0])
	}

	pipe, ok := stmt.Expression.(*ast.PipeExpression)
	if !ok {
		t.Fatalf("expression not *ast.PipeExpression. got=%T", stmt.Expression)
	}

	rightIdent, ok := pipe.Right.(*ast.Identifier)
	if !ok {
		t.Fatalf("right not identifier. got=%T", pipe.Right)
	}
	if rightIdent.Value != "print" {
		t.Fatalf("expected 'print', got %s", rightIdent.Value)
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
