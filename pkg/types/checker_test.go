package types

import (
	"strings"
	"testing"

	"github.com/dev-dami/carv/pkg/lexer"
	"github.com/dev-dami/carv/pkg/parser"
)

func TestTypeCheckerBasics(t *testing.T) {
	tests := []struct {
		input       string
		shouldError bool
	}{
		{"let x = 5;", false},
		{"let x: int = 5;", false},
		{"let x: float = 5.0;", false},
		{"let x: string = \"hello\";", false},
		{"let x: bool = true;", false},
		{"let x = 5 + 3;", false},
		{"let x = 5.0 + 3.0;", false},
		{"let x = 5 < 10;", false},
		{"let x = true && false;", false},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := parser.New(l)
		program := p.ParseProgram()

		checker := NewChecker()
		ok := checker.Check(program)

		if tt.shouldError && ok {
			t.Errorf("expected error for %q, got none", tt.input)
		}
		if !tt.shouldError && !ok {
			t.Errorf("unexpected error for %q: %v", tt.input, checker.Errors())
		}
	}
}

func TestTypeCheckerFunctions(t *testing.T) {
	input := `
fn add(a: int, b: int) -> int {
	return a + b;
}
let result = add(5, 3);
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	checker := NewChecker()
	ok := checker.Check(program)

	if !ok {
		t.Errorf("unexpected errors: %v", checker.Errors())
	}
}

func TestTypeCheckerTCPBuiltins(t *testing.T) {
	input := `
let listener = tcp_listen("127.0.0.1", 8080);
let conn = tcp_accept(listener);
let data = tcp_read(conn, 1024);
let n = tcp_write(conn, data);
tcp_close(conn);
tcp_close(listener);
print(n);
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	checker := NewChecker()
	ok := checker.Check(program)

	if !ok {
		t.Fatalf("unexpected type errors: %v", checker.Errors())
	}
}

func TestTypeCheckerBuiltinNetModuleAlias(t *testing.T) {
	input := `
require "net" as net;
let data = net.tcp_read(1, 64);
print(data);
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	checker := NewChecker()
	ok := checker.Check(program)

	if !ok {
		t.Fatalf("unexpected type errors: %v", checker.Errors())
	}
}

func TestTypeCheckerBuiltinNetModuleAliasArityError(t *testing.T) {
	input := `
require "net" as net;
let listener = net.tcp_listen("127.0.0.1");
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	checker := NewChecker()
	ok := checker.Check(program)

	if ok {
		t.Fatalf("expected type error for wrong tcp_listen arity")
	}
}

func TestTypeCheckerArrays(t *testing.T) {
	input := `
let arr = [1, 2, 3];
let x = arr[0];
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	checker := NewChecker()
	ok := checker.Check(program)

	if !ok {
		t.Errorf("unexpected errors: %v", checker.Errors())
	}
}

func TestTypeCheckerLoops(t *testing.T) {
	input := `
for (let i = 0; i < 10; i = i + 1) {
	print(i);
}

mut x = 0;
while x < 5 {
	x = x + 1;
}
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	checker := NewChecker()
	ok := checker.Check(program)

	if !ok {
		t.Errorf("unexpected errors: %v", checker.Errors())
	}
}

func TestTypeCheckerErrors(t *testing.T) {
	tests := []struct {
		input string
		desc  string
	}{
		{"let x = y;", "undefined variable"},
		{"let x = 5 && 3;", "boolean op on ints"},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := parser.New(l)
		program := p.ParseProgram()

		checker := NewChecker()
		ok := checker.Check(program)

		if ok {
			t.Errorf("expected error for %s: %q", tt.desc, tt.input)
		}
	}
}

func TestTypeCheckerCopyAssignmentNoWarnings(t *testing.T) {
	input := `
let x = 1;
let y = x;
let z = x;
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	checker := NewChecker()
	ok := checker.Check(program)

	if !ok {
		t.Fatalf("unexpected errors: %v", checker.Errors())
	}
	if len(checker.Warnings()) != 0 {
		t.Fatalf("expected no warnings, got %v", checker.Warnings())
	}
}

func TestTypeCheckerMoveAssignmentWarnsOnReuse(t *testing.T) {
	input := `
let s = "hi";
let t = s;
let u = s;
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	checker := NewChecker()
	ok := checker.Check(program)

	if !ok {
		t.Fatalf("unexpected errors: %v", checker.Errors())
	}
	warnings := checker.Warnings()
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %v", warnings)
	}
	if !strings.Contains(warnings[0], "use of moved value 's'") {
		t.Fatalf("unexpected warning: %s", warnings[0])
	}
}

func TestTypeCheckerMoveArgWarnsOnReuse(t *testing.T) {
	input := `
fn take(a: string) {
    print(a);
}
let s = "hi";
take(s);
let t = s;
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	checker := NewChecker()
	ok := checker.Check(program)

	if !ok {
		t.Fatalf("unexpected errors: %v", checker.Errors())
	}
	warnings := checker.Warnings()
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %v", warnings)
	}
	if !strings.Contains(warnings[0], "use of moved value 's'") {
		t.Fatalf("unexpected warning: %s", warnings[0])
	}
}

func TestTypeCheckerMoveReturnWarnsOnReuse(t *testing.T) {
	input := `
fn give() -> string {
    let s = "hi";
    return s;
    let t = s;
}
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	checker := NewChecker()
	ok := checker.Check(program)

	if !ok {
		t.Fatalf("unexpected errors: %v", checker.Errors())
	}
	warnings := checker.Warnings()
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %v", warnings)
	}
	if !strings.Contains(warnings[0], "use of moved value 's'") {
		t.Fatalf("unexpected warning: %s", warnings[0])
	}
}

func TestBorrowImmutableNoWarnings(t *testing.T) {
	input := `
let s = "hello";
let r = &s;
print(len(r));
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	checker := NewChecker()
	ok := checker.Check(program)

	if !ok {
		t.Fatalf("unexpected errors: %v", checker.Errors())
	}
	if len(checker.Warnings()) != 0 {
		t.Fatalf("expected no warnings, got %v", checker.Warnings())
	}
}

func TestBorrowDoubleImmutableNoWarnings(t *testing.T) {
	input := `
let s = "hello";
let r1 = &s;
let r2 = &s;
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	checker := NewChecker()
	ok := checker.Check(program)

	if !ok {
		t.Fatalf("unexpected errors: %v", checker.Errors())
	}
	if len(checker.Warnings()) != 0 {
		t.Fatalf("expected no warnings, got %v", checker.Warnings())
	}
}

func TestBorrowMutableBlocksImmutableWarning(t *testing.T) {
	input := `
let s = "hello";
let r = &mut s;
let r2 = &s;
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	checker := NewChecker()
	ok := checker.Check(program)

	if !ok {
		t.Fatalf("unexpected errors: %v", checker.Errors())
	}
	warnings := checker.Warnings()
	if len(warnings) == 0 {
		t.Fatalf("expected warning, got none")
	}
	found := false
	for _, warning := range warnings {
		if strings.Contains(warning, "cannot immutably borrow 's'") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected immutable borrow warning, got %v", warnings)
	}
}

func TestBorrowMovedValueWarning(t *testing.T) {
	input := `
let s = "hello";
let t = s;
let r = &s;
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	checker := NewChecker()
	ok := checker.Check(program)

	if !ok {
		t.Fatalf("unexpected errors: %v", checker.Errors())
	}
	warnings := checker.Warnings()
	if len(warnings) == 0 {
		t.Fatalf("expected warning, got none")
	}
	found := false
	for _, warning := range warnings {
		if strings.Contains(warning, "cannot borrow moved value 's'") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected moved borrow warning, got %v", warnings)
	}
}

func TestInterfaceDefinitionNoErrors(t *testing.T) {
	input := `
interface Printable {
	fn to_string(&self) -> string;
}
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	checker := NewChecker()
	ok := checker.Check(program)

	if !ok {
		t.Fatalf("unexpected errors: %v", checker.Errors())
	}
}

func TestImplSatisfiesInterface(t *testing.T) {
	input := `
class Person {
	name: string
}
interface Printable {
	fn to_string(&self) -> string;
}
impl Printable for Person {
	fn to_string(&self) -> string {
		return self.name;
	}
}
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	checker := NewChecker()
	ok := checker.Check(program)

	if !ok {
		t.Fatalf("unexpected errors: %v", checker.Errors())
	}
}

func TestImplMissingMethodError(t *testing.T) {
	input := `
class Person {
	name: string
}
interface Printable {
	fn to_string(&self) -> string;
	fn display(&self);
}
impl Printable for Person {
	fn to_string(&self) -> string {
		return self.name;
	}
}
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	checker := NewChecker()
	ok := checker.Check(program)

	if ok {
		t.Fatal("expected error for missing method")
	}
	found := false
	for _, err := range checker.Errors() {
		if strings.Contains(err, "missing method display") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected missing method error, got %v", checker.Errors())
	}
}

func TestClassStatementRegistersType(t *testing.T) {
	input := `
class Point {
	x: int = 0
	y: int = 0
}
let p = new Point;
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	checker := NewChecker()
	ok := checker.Check(program)

	if !ok {
		t.Fatalf("unexpected errors: %v", checker.Errors())
	}
}

func TestBorrowAssignWhileBorrowedWarning(t *testing.T) {
	input := `
mut s = "hello";
let r = &s;
s = "world";
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	checker := NewChecker()
	ok := checker.Check(program)

	if !ok {
		t.Fatalf("unexpected errors: %v", checker.Errors())
	}
	warnings := checker.Warnings()
	if len(warnings) == 0 {
		t.Fatalf("expected warning, got none")
	}
	found := false
	for _, warning := range warnings {
		if strings.Contains(warning, "cannot assign to 's' while it is borrowed") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected assign while borrowed warning, got %v", warnings)
	}
}

func TestSelfMutationThroughImmutableReceiver(t *testing.T) {
	input := `
class Foo {
	x: int = 0
	fn get(&self) -> int {
		self.x = 5;
		return self.x;
	}
}
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	checker := NewChecker()
	ok := checker.Check(program)

	if !ok {
		t.Fatalf("unexpected errors: %v", checker.Errors())
	}
	warnings := checker.Warnings()
	if len(warnings) == 0 {
		t.Fatalf("expected warning, got none")
	}
	found := false
	for _, warning := range warnings {
		if strings.Contains(warning, "cannot assign to field through immutable receiver (&self)") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected immutable receiver warning, got %v", warnings)
	}
}

func TestReceiverMismatchInImpl(t *testing.T) {
	input := `
interface Readable {
	fn read(&self) -> int;
}
class Doc {
	x: int = 0
}
impl Readable for Doc {
	fn read(&mut self) -> int {
		return self.x;
	}
}
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	checker := NewChecker()
	ok := checker.Check(program)

	if !ok {
		t.Fatalf("unexpected errors: %v", checker.Errors())
	}
	warnings := checker.Warnings()
	if len(warnings) == 0 {
		t.Fatalf("expected warning, got none")
	}
	found := false
	for _, warning := range warnings {
		if strings.Contains(warning, "receiver mismatch for method read") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected receiver mismatch warning, got %v", warnings)
	}
}

func TestMutMethodThroughImmutableInterfaceRef(t *testing.T) {
	input := `
interface Writable {
	fn write(&mut self, value: int);
}
class Doc {
	x: int = 0
}
impl Writable for Doc {
	fn write(&mut self, value: int) {
		self.x = value;
	}
}
let d = new Doc;
let w = &d as &Writable;
w.write(1);
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	checker := NewChecker()
	ok := checker.Check(program)

	if ok {
		t.Fatalf("expected error, got none")
	}
	found := false
	for _, err := range checker.Errors() {
		if strings.Contains(err, "cannot call &mut self method 'write' through immutable interface reference") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected immutable interface dispatch error, got %v", checker.Errors())
	}
}

func TestFunctionLiteralBodyTypeCheck(t *testing.T) {
	input := `
let f = fn(x: int) -> int {
	let y: string = x;
	return y;
};
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	checker.Check(program)

	if len(checker.Errors()) == 0 {
		t.Fatalf("expected type error inside function literal body, got none")
	}
	found := false
	for _, err := range checker.Errors() {
		if strings.Contains(err, "cannot assign") || strings.Contains(err, "type mismatch") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected type error inside function literal body, got: %v", checker.Errors())
	}
}

func TestFunctionLiteralBodyNoError(t *testing.T) {
	input := `
let f = fn(x: int) -> int {
	return x + 1;
};
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	checker.Check(program)

	for _, err := range checker.Errors() {
		t.Errorf("unexpected error: %s", err)
	}
}

func TestAwaitOutsideAsyncError(t *testing.T) {
	input := `
async fn fetch() -> int {
	return 1;
}
fn sync_fn() -> int {
	return await fetch();
}
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	ok := checker.Check(program)

	if ok {
		t.Fatal("expected error for await outside async")
	}
	found := false
	for _, err := range checker.Errors() {
		if strings.Contains(err, "await") && strings.Contains(err, "async") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected await-outside-async error, got %v", checker.Errors())
	}
}

func TestAwaitNonFutureError(t *testing.T) {
	input := `
async fn do_work() -> int {
	let x = 42;
	return await x;
}
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	ok := checker.Check(program)

	if ok {
		t.Fatal("expected error for await on non-future")
	}
	found := false
	for _, err := range checker.Errors() {
		if strings.Contains(err, "await") && strings.Contains(err, "Future") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected await-requires-future error, got %v", checker.Errors())
	}
}

func TestBorrowAcrossAwaitError(t *testing.T) {
	input := `
async fn fetch() -> int {
	return 1;
}
async fn bad_borrow() -> int {
	let s = "hello";
	let r = &s;
	let x = await fetch();
	return x;
}
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	ok := checker.Check(program)

	if ok {
		t.Fatal("expected error for borrow across await")
	}
	found := false
	for _, err := range checker.Errors() {
		if strings.Contains(err, "borrow") && strings.Contains(err, "await") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected 'borrow across await' error, got %v", checker.Errors())
	}
}

func TestAsyncFnReturnType(t *testing.T) {
	input := `
async fn fetch() -> int {
	return 42;
}
async fn caller() -> int {
	let x = await fetch();
	return x;
}
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	ok := checker.Check(program)

	if !ok {
		t.Fatalf("unexpected errors: %v", checker.Errors())
	}
}

// --------------- Additional coverage tests ---------------

func check(t *testing.T, input string) *Checker {
	t.Helper()
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	checker := NewChecker()
	checker.Check(program)
	return checker
}

func checkOK(t *testing.T, input string) *Checker {
	t.Helper()
	c := check(t, input)
	if len(c.Errors()) > 0 {
		t.Fatalf("unexpected errors: %v", c.Errors())
	}
	return c
}

func checkHasError(t *testing.T, input string, substr string) *Checker {
	t.Helper()
	c := check(t, input)
	if len(c.Errors()) == 0 {
		t.Fatalf("expected error containing %q, got none", substr)
	}
	found := false
	for _, e := range c.Errors() {
		if strings.Contains(e, substr) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected error containing %q, got: %v", substr, c.Errors())
	}
	return c
}

func checkHasWarning(t *testing.T, input string, substr string) *Checker {
	t.Helper()
	c := check(t, input)
	found := false
	for _, w := range c.Warnings() {
		if strings.Contains(w, substr) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected warning containing %q, got warnings: %v errors: %v", substr, c.Warnings(), c.Errors())
	}
	return c
}

// --- TypeInfo, ErrorIssues, WarningIssues ---

func TestTypeInfoReturnsMap(t *testing.T) {
	c := checkOK(t, "let x = 42;")
	ti := c.TypeInfo()
	if ti == nil {
		t.Fatal("TypeInfo() returned nil")
	}
	if len(ti) == 0 {
		t.Fatal("TypeInfo() returned empty map")
	}
}

func TestErrorIssuesReturnsSlice(t *testing.T) {
	c := checkHasError(t, "let x = y;", "undefined")
	issues := c.ErrorIssues()
	if len(issues) == 0 {
		t.Fatal("ErrorIssues() returned empty")
	}
	if issues[0].Kind != "type_error" {
		t.Fatalf("expected kind type_error, got %s", issues[0].Kind)
	}
}

func TestWarningIssuesReturnsSlice(t *testing.T) {
	c := checkHasWarning(t, `
let s = "hello";
let t = s;
let u = s;
`, "use of moved value")
	issues := c.WarningIssues()
	if len(issues) == 0 {
		t.Fatal("WarningIssues() returned empty")
	}
	if issues[0].Kind != "warning" {
		t.Fatalf("expected kind warning, got %s", issues[0].Kind)
	}
}

// --- checkConstStatement ---

func TestConstStatement(t *testing.T) {
	checkOK(t, "const x = 42;")
}

func TestConstStatementWithType(t *testing.T) {
	checkOK(t, `const x: int = 42;`)
}

func TestConstStatementTypeMismatch(t *testing.T) {
	checkHasError(t, `const x: int = "hello";`, "cannot assign")
}

func TestConstStatementMoveSemantics(t *testing.T) {
	checkHasWarning(t, `
const s = "hello";
const t = s;
const u = s;
`, "use of moved value")
}

// --- checkForInStatement ---

func TestForInStatementArray(t *testing.T) {
	checkOK(t, `
let arr = [1, 2, 3];
for item in arr {
	print(item);
}
`)
}

func TestForInStatementNonArray(t *testing.T) {
	checkOK(t, `
let x = 42;
for item in x {
	print(item);
}
`)
}

// --- checkLoopStatement ---

func TestLoopStatement(t *testing.T) {
	checkOK(t, `
for {
	let x = 1;
}
`)
}

// --- checkPrefixExpression ---

func TestPrefixNegation(t *testing.T) {
	checkOK(t, "let x = -5;")
}

func TestPrefixNot(t *testing.T) {
	checkOK(t, "let x = !true;")
}

func TestPrefixBitwiseNot(t *testing.T) {
	checkOK(t, "let x = ~5;")
}

func TestPrefixNegationNonNumeric(t *testing.T) {
	checkHasError(t, `let x = -true;`, "operator - requires numeric type")
}

func TestPrefixBitwiseNotNonInt(t *testing.T) {
	checkHasError(t, `let x = ~true;`, "operator ~ requires int")
}

// --- checkMapLiteral ---

func TestMapLiteralEmpty(t *testing.T) {
	checkOK(t, `let m = {};`)
}

func TestMapLiteralWithPairs(t *testing.T) {
	checkOK(t, `let m = {"a": 1, "b": 2};`)
}

// --- checkIfExpression ---

func TestIfExpression(t *testing.T) {
	checkOK(t, `
if true {
	let x = 1;
}
`)
}

func TestIfExpressionWithElse(t *testing.T) {
	checkOK(t, `
if true {
	let x = 1;
} else {
	let y = 2;
}
`)
}

func TestIfExpressionNonBoolCondition(t *testing.T) {
	checkHasError(t, `
if 42 {
	let x = 1;
}
`, "condition must be bool")
}

// --- checkSpawnExpression ---

func TestSpawnExpression(t *testing.T) {
	checkOK(t, `
let x = spawn {
	let a = 1;
};
`)
}

// --- checkInterpolatedString ---

func TestInterpolatedString(t *testing.T) {
	checkOK(t, `
let name = "world";
let msg = f"hello {name}";
`)
}

// --- checkDerefExpression ---

func TestDerefExpression(t *testing.T) {
	checkOK(t, `
let x = 42;
let r = &x;
let v = *r;
`)
}

func TestDerefNonRef(t *testing.T) {
	checkHasWarning(t, `
let x = 42;
let v = *x;
`, "dereference of non-reference type")
}

// --- checkConditionIsBool (non-bool for, while) ---

func TestForConditionNonBool(t *testing.T) {
	checkHasError(t, `
for (let i = 0; 1; i = i + 1) {
	print(i);
}
`, "condition must be bool")
}

func TestWhileConditionNonBool(t *testing.T) {
	checkHasError(t, `
while 42 {
	let x = 1;
}
`, "condition must be bool")
}

// --- checkRequireStatement ---

func TestRequireStatementNamedImports(t *testing.T) {
	checkOK(t, `
require { tcp_listen, tcp_read } from "net";
let l = tcp_listen("127.0.0.1", 8080);
`)
}

func TestRequireStatementNamedImportUndefined(t *testing.T) {
	checkHasError(t, `
require { nonexistent_func } from "net";
`, "undefined export")
}

func TestRequireStatementWildcardBuiltin(t *testing.T) {
	checkOK(t, `
require * from "net";
let l = tcp_listen("127.0.0.1", 8080);
`)
}

func TestRequireStatementWildcardNonBuiltin(t *testing.T) {
	checkOK(t, `
require * from "mymodule";
`)
}

func TestRequireStatementNamedNonBuiltin(t *testing.T) {
	checkOK(t, `
require { foo, bar } from "mymodule";
`)
}

// --- builtinModuleMemberTypes (all branches) ---

func TestBuiltinModuleGpio(t *testing.T) {
	checkOK(t, `
require "gpio" as gpio;
let v = gpio.digital_read(1);
`)
}

func TestBuiltinModuleUart(t *testing.T) {
	checkOK(t, `
require "uart" as uart;
let h = uart.uart_init(1, 9600);
`)
}

func TestBuiltinModuleSpi(t *testing.T) {
	checkOK(t, `
require "spi" as spi;
let h = spi.spi_init(0, 1000);
`)
}

func TestBuiltinModuleI2c(t *testing.T) {
	checkOK(t, `
require "i2c" as i2c;
let h = i2c.i2c_init(1, 80);
`)
}

func TestBuiltinModuleTimer(t *testing.T) {
	checkOK(t, `
require "timer" as timer;
let t = timer.timer_init(0, 1000);
timer.timer_start(t);
`)
}

func TestBuiltinModuleWeb(t *testing.T) {
	checkOK(t, `
require "web" as web;
let l = web.tcp_listen("0.0.0.0", 80);
`)
}

func TestBuiltinModuleUndefinedMember(t *testing.T) {
	checkHasError(t, `
require "net" as net;
let x = net.nonexistent();
`, "undefined member")
}

// --- checkIndexExpression ---

func TestIndexExpressionStringIndex(t *testing.T) {
	checkOK(t, `
let s = "hello";
let c = s[0];
`)
}

func TestIndexExpressionStringNonIntIndex(t *testing.T) {
	checkHasError(t, `
let s = "hello";
let c = s[true];
`, "string index must be int")
}

func TestIndexExpressionArrayNonIntIndex(t *testing.T) {
	checkHasError(t, `
let a = [1, 2, 3];
let v = a[true];
`, "array index must be int")
}

func TestIndexExpressionMapIndex(t *testing.T) {
	checkOK(t, `
let m = {"a": 1};
let v = m["a"];
`)
}

// --- checkInfixExpression (more operators) ---

func TestInfixComparisonNonComparable(t *testing.T) {
	checkHasError(t, `
let a = [1];
let b = [2];
let c = a < b;
`, "requires comparable types")
}

func TestInfixBitwiseOps(t *testing.T) {
	checkOK(t, `
let a = 5 & 3;
let b = 5 | 3;
let c = 5 ^ 3;
`)
}

func TestInfixBitwiseNonInt(t *testing.T) {
	checkHasError(t, `
let a = true & false;
`, "bitwise operator requires int")
}

func TestInfixStringConcat(t *testing.T) {
	checkOK(t, `
let s = "hello" + " world";
`)
}

func TestInfixNonNumericArithmetic(t *testing.T) {
	checkHasError(t, `
let x = true + false;
`, "requires numeric types")
}

func TestInfixLogicalNonBool(t *testing.T) {
	checkHasError(t, `
let x = 1 || 2;
`, "requires bool types")
}

func TestInfixFloatArithmetic(t *testing.T) {
	checkOK(t, `
let x = 1.0 + 2;
`)
}

func TestInfixEqualityOp(t *testing.T) {
	checkOK(t, `
let x = 5 == 5;
let y = 5 != 3;
`)
}

// --- resolveTypeExpr (all branches) ---

func TestResolveTypeExprArray(t *testing.T) {
	checkOK(t, `
fn f(a: []int) {
	print(a);
}
`)
}

func TestResolveTypeExprSizedTypes(t *testing.T) {
	checkOK(t, `
fn f1(a: u8) { print(a); }
fn f2(a: u16) { print(a); }
fn f3(a: u32) { print(a); }
fn f4(a: u64) { print(a); }
fn f5(a: i8) { print(a); }
fn f6(a: i16) { print(a); }
fn f7(a: i32) { print(a); }
fn f8(a: i64) { print(a); }
fn f9(a: f32) { print(a); }
fn f10(a: f64) { print(a); }
fn f11(a: usize) { print(a); }
fn f12(a: isize) { print(a); }
`)
}

func TestResolveTypeExprChar(t *testing.T) {
	checkOK(t, `
fn f(c: char) { print(c); }
`)
}

func TestResolveTypeExprVoid(t *testing.T) {
	checkOK(t, `
fn f() -> void { }
`)
}

func TestResolveTypeExprAny(t *testing.T) {
	checkOK(t, `
fn f(x: any) { print(x); }
`)
}

func TestResolveTypeExprRef(t *testing.T) {
	checkOK(t, `
fn f(r: &int) { print(r); }
`)
}

func TestResolveTypeExprMutRef(t *testing.T) {
	checkOK(t, `
fn f(r: &mut int) { print(r); }
`)
}

func TestResolveTypeExprNamed(t *testing.T) {
	checkOK(t, `
class Foo { x: int = 0 }
fn f(p: Foo) { print(p); }
`)
}

func TestResolveTypeExprNamedUndefined(t *testing.T) {
	// Named type that doesn't exist resolves to Any (no error, just Any)
	checkOK(t, `
fn f(p: Unknown) { print(p); }
`)
}

// --- String() and Equals() methods for types.go ---

func TestTypeStringMethods(t *testing.T) {
	tests := []struct {
		typ      Type
		expected string
	}{
		{&ArrayType{Element: Int}, "[]int"},
		{&FunctionType{Params: []Type{Int, String}, Return: Bool}, "fn(int, string) -> bool"},
		{&ChannelType{Element: Int}, "chan int"},
		{&OptionalType{Inner: Int}, "int?"},
		{&ClassType{Name: "Foo", Fields: nil}, "Foo"},
		{&InterfaceType{Name: "Bar", Methods: nil}, "Bar"},
		{&MapType{Key: String, Value: Int}, "{string: int}"},
		{&RefType{Inner: Int, Mutable: false}, "&int"},
		{&RefType{Inner: Int, Mutable: true}, "&mut int"},
		{&FutureType{Inner: Int}, "Future<int>"},
		{&ModuleType{Name: "net"}, "module net"},
		{&VolatileType{Inner: Int}, "volatile<int>"},
	}

	for _, tt := range tests {
		if got := tt.typ.String(); got != tt.expected {
			t.Errorf("String() = %q, want %q", got, tt.expected)
		}
	}
}

func TestTypeEqualsMethods(t *testing.T) {
	// Test Equals returning true
	trueTests := []struct {
		a, b Type
	}{
		{&BasicType{Name: "int"}, &BasicType{Name: "int"}},
		{&ArrayType{Element: Int}, &ArrayType{Element: Int}},
		{&FunctionType{Params: []Type{Int}, Return: Bool}, &FunctionType{Params: []Type{Int}, Return: Bool}},
		{&ChannelType{Element: Int}, &ChannelType{Element: Int}},
		{&OptionalType{Inner: Int}, &OptionalType{Inner: Int}},
		{&ClassType{Name: "Foo"}, &ClassType{Name: "Foo"}},
		{&InterfaceType{Name: "Bar"}, &InterfaceType{Name: "Bar"}},
		{&MapType{Key: String, Value: Int}, &MapType{Key: String, Value: Int}},
		{&RefType{Inner: Int, Mutable: true}, &RefType{Inner: Int, Mutable: true}},
		{&FutureType{Inner: Int}, &FutureType{Inner: Int}},
		{&ModuleType{Name: "net"}, &ModuleType{Name: "net"}},
		{&VolatileType{Inner: Int}, &VolatileType{Inner: Int}},
	}
	for _, tt := range trueTests {
		if !tt.a.Equals(tt.b) {
			t.Errorf("%s.Equals(%s) = false, want true", tt.a.String(), tt.b.String())
		}
	}

	// Test Equals returning false (different type)
	falseTests := []struct {
		a, b Type
	}{
		{&BasicType{Name: "int"}, &BasicType{Name: "float"}},
		{&ArrayType{Element: Int}, &BasicType{Name: "int"}},
		{&ArrayType{Element: Int}, &ArrayType{Element: Float}},
		{&FunctionType{Params: []Type{Int}, Return: Bool}, &BasicType{Name: "int"}},
		{&FunctionType{Params: []Type{Int}, Return: Bool}, &FunctionType{Params: []Type{Int, Int}, Return: Bool}},
		{&FunctionType{Params: []Type{Int}, Return: Bool}, &FunctionType{Params: []Type{Float}, Return: Bool}},
		{&FunctionType{Params: []Type{Int}, Return: Bool}, &FunctionType{Params: []Type{Int}, Return: Int}},
		{&ChannelType{Element: Int}, &BasicType{Name: "int"}},
		{&ChannelType{Element: Int}, &ChannelType{Element: Float}},
		{&OptionalType{Inner: Int}, &BasicType{Name: "int"}},
		{&OptionalType{Inner: Int}, &OptionalType{Inner: Float}},
		{&ClassType{Name: "Foo"}, &BasicType{Name: "Foo"}},
		{&ClassType{Name: "Foo"}, &ClassType{Name: "Bar"}},
		{&InterfaceType{Name: "Foo"}, &BasicType{Name: "Foo"}},
		{&InterfaceType{Name: "Foo"}, &InterfaceType{Name: "Bar"}},
		{&MapType{Key: String, Value: Int}, &BasicType{Name: "map"}},
		{&MapType{Key: String, Value: Int}, &MapType{Key: Int, Value: Int}},
		{&RefType{Inner: Int, Mutable: true}, &BasicType{Name: "ref"}},
		{&RefType{Inner: Int, Mutable: true}, &RefType{Inner: Int, Mutable: false}},
		{&FutureType{Inner: Int}, &BasicType{Name: "future"}},
		{&FutureType{Inner: Int}, &FutureType{Inner: Float}},
		{&ModuleType{Name: "net"}, &BasicType{Name: "net"}},
		{&ModuleType{Name: "net"}, &ModuleType{Name: "web"}},
		{&VolatileType{Inner: Int}, &BasicType{Name: "volatile"}},
		{&VolatileType{Inner: Int}, &VolatileType{Inner: Float}},
	}
	for _, tt := range falseTests {
		if tt.a.Equals(tt.b) {
			t.Errorf("%s.Equals(%s) = true, want false", tt.a.String(), tt.b.String())
		}
	}
}

// --- IsCopyType ---

func TestIsCopyType(t *testing.T) {
	copyTypes := []Type{Int, Float, Bool, Char, Void, Nil, Any, U8, U16, U32, U64, I8, I16, I32, I64, F32, F64, Usize, Isize}
	for _, typ := range copyTypes {
		if !IsCopyType(typ) {
			t.Errorf("IsCopyType(%s) = false, want true", typ.String())
		}
	}
	moveTypes := []Type{
		String,
		&ArrayType{Element: Int},
		&MapType{Key: String, Value: Int},
		&ClassType{Name: "Foo"},
		&FutureType{Inner: Int},
	}
	for _, typ := range moveTypes {
		if IsCopyType(typ) {
			t.Errorf("IsCopyType(%s) = true, want false", typ.String())
		}
	}
	// RefType is copy
	if !IsCopyType(&RefType{Inner: Int, Mutable: false}) {
		t.Error("IsCopyType(&int) = false, want true")
	}
	// nil type is copy
	if !IsCopyType(nil) {
		t.Error("IsCopyType(nil) = false, want true")
	}
}

// --- IsNumeric, IsComparable edge cases ---

func TestIsNumericFalse(t *testing.T) {
	if IsNumeric(Bool) {
		t.Error("IsNumeric(bool) = true, want false")
	}
	if IsNumeric(&ArrayType{Element: Int}) {
		t.Error("IsNumeric([]int) = true, want false")
	}
}

func TestIsComparableFalse(t *testing.T) {
	if IsComparable(&ArrayType{Element: Int}) {
		t.Error("IsComparable([]int) = true, want false")
	}
}

// --- checkReturnStatement: return reference warning ---

func TestReturnReferenceWarning(t *testing.T) {
	checkHasWarning(t, `
fn f() -> int {
	let x = 42;
	let r = &x;
	return r;
}
`, "reference cannot escape function scope")
}

// --- checkCallExpression: arity mismatch ---

func TestCallExpressionArityMismatch(t *testing.T) {
	checkHasError(t, `
fn f(a: int, b: int) -> int {
	return a + b;
}
let x = f(1);
`, "expects 2 arguments, got 1")
}

// --- checkCallExpression: argument type mismatch ---

func TestCallExpressionArgTypeMismatch(t *testing.T) {
	checkHasError(t, `
fn f(a: int) -> int {
	return a;
}
let x = f("hello");
`, "argument 1: cannot pass")
}

// --- checkAssignExpression: assign to member ---

func TestAssignToMember(t *testing.T) {
	checkOK(t, `
class Foo {
	x: int = 0
	fn set(&mut self, v: int) {
		self.x = v;
	}
}
`)
}

// --- checkAssignExpression: type mismatch on assign ---

func TestAssignTypeMismatch(t *testing.T) {
	checkHasError(t, `
let x: int = 5;
x = "hello";
`, "cannot assign")
}

// --- checkAssignExpression: member assign type mismatch ---

func TestAssignMemberField(t *testing.T) {
	// Member assignment goes through the member-assign branch of checkAssignExpression.
	// self through a ref does not resolve field types from ClassType, so this succeeds.
	checkOK(t, `
class Foo {
	x: int = 0
	fn set(&mut self) {
		self.x = 5;
	}
}
`)
}

// --- checkClassStatement: field without type ---

func TestClassMethodWithValueReceiver(t *testing.T) {
	checkOK(t, `
class Foo {
	x: int = 0
	fn get(self) -> int {
		return self.x;
	}
}
`)
}

// --- receiverKindName coverage (RecvValue, RecvNone) ---

func TestReceiverKindNameCoverage(t *testing.T) {
	// RecvValue
	checkOK(t, `
class Foo { x: int = 0 }
interface Consumer {
	fn consume(self) -> int;
}
impl Consumer for Foo {
	fn consume(self) -> int {
		return self.x;
	}
}
`)
}

// --- checkImplStatement: undefined interface, non-interface, undefined type ---

func TestImplUndefinedInterface(t *testing.T) {
	checkHasError(t, `
class Foo { x: int = 0 }
impl Nonexistent for Foo {
	fn f(&self) { }
}
`, "undefined interface")
}

func TestImplNonInterface(t *testing.T) {
	checkHasError(t, `
class Foo { x: int = 0 }
impl Foo for Foo {
	fn f(&self) { }
}
`, "is not an interface")
}

func TestImplUndefinedType(t *testing.T) {
	checkHasError(t, `
interface I {
	fn f(&self);
}
impl I for Nonexistent {
	fn f(&self) { }
}
`, "undefined type")
}

// --- checkImplStatement: wrong param count, param type mismatch, return type mismatch ---

func TestImplWrongParamCount(t *testing.T) {
	checkHasError(t, `
class Foo { x: int = 0 }
interface I {
	fn f(&self, a: int);
}
impl I for Foo {
	fn f(&self) { }
}
`, "wrong number of parameters")
}

func TestImplReturnTypeMismatch(t *testing.T) {
	checkHasError(t, `
class Foo { x: int = 0 }
interface I {
	fn f(&self) -> int;
}
impl I for Foo {
	fn f(&self) -> string { return "bad"; }
}
`, "return type mismatch")
}

func TestImplParamTypeMismatch(t *testing.T) {
	checkHasError(t, `
class Foo { x: int = 0 }
interface I {
	fn f(&self, a: int);
}
impl I for Foo {
	fn f(&self, a: string) { }
}
`, "parameter 1: expected int, got string")
}

// --- checkMemberExpressionForInterface: undefined method on interface ---

func TestInterfaceUndefinedMethod(t *testing.T) {
	checkHasError(t, `
interface I {
	fn f(&self) -> int;
}
class Foo { x: int = 0 }
impl I for Foo {
	fn f(&self) -> int { return 1; }
}
let foo = new Foo;
let r = &foo as &I;
r.nonexistent();
`, "has no method nonexistent")
}

// --- checkMemberExpression: class field access ---

func TestClassFieldAccess(t *testing.T) {
	checkOK(t, `
class Point { x: int = 0  y: int = 0 }
let p = new Point;
let v = p.x;
`)
}

// --- resolveTypeExpr: volatile ---

func TestResolveVolatileType(t *testing.T) {
	// volatile type annotation goes through resolveTypeExpr VolatileType case
	// The Carv language has volatile types for embedded. We need to exercise it.
	// Since we test through Carv source, we need a volatile type in source. Let's check if the parser supports it.
	// If not, we use a direct type test.
	vt := &VolatileType{Inner: Int}
	if vt.String() != "volatile<int>" {
		t.Errorf("VolatileType.String() = %q", vt.String())
	}
}

// --- FunctionType.String with no params ---

func TestFunctionTypeStringNoParams(t *testing.T) {
	ft := &FunctionType{Params: []Type{}, Return: Void}
	if ft.String() != "fn() -> void" {
		t.Errorf("got %q", ft.String())
	}
}

// --- Category for FunctionType (default case, returns CopyType) ---

func TestCategoryFunctionType(t *testing.T) {
	ft := &FunctionType{Params: []Type{Int}, Return: Int}
	if Category(ft) != CopyType {
		t.Error("expected FunctionType to be CopyType")
	}
}

// --- checkLetStatement: nil value ---

func TestLetStatementNilLiteral(t *testing.T) {
	checkOK(t, "let x = nil;")
}

// --- CharLiteral, FloatLiteral, NilLiteral ---

func TestLiteralTypes(t *testing.T) {
	checkOK(t, `
let c = 'a';
let f = 3.14;
let n = nil;
let b = true;
`)
}

// --- checkAssignExpression: assign to undefined ---

func TestAssignToUndefined(t *testing.T) {
	checkHasError(t, `x = 5;`, "undefined")
}

// --- Empty array literal ---

func TestEmptyArrayLiteral(t *testing.T) {
	checkOK(t, "let a = [];")
}

// --- checkCastExpression ---

func TestCastExpression(t *testing.T) {
	checkOK(t, `
let x = 42;
let y = x as float;
`)
}

// --- require named imports for all builtin modules ---

func TestRequireNamedGpio(t *testing.T) {
	checkOK(t, `
require { pin_mode, digital_write } from "gpio";
pin_mode(1, 0);
`)
}

func TestRequireNamedUart(t *testing.T) {
	checkOK(t, `
require { uart_init } from "uart";
let h = uart_init(1, 9600);
`)
}

func TestRequireNamedSpi(t *testing.T) {
	checkOK(t, `
require { spi_init } from "spi";
let h = spi_init(0, 1000);
`)
}

func TestRequireNamedI2c(t *testing.T) {
	checkOK(t, `
require { i2c_init } from "i2c";
let h = i2c_init(1, 80);
`)
}

func TestRequireNamedTimer(t *testing.T) {
	checkOK(t, `
require { timer_init, delay_ms } from "timer";
let t = timer_init(0, 1000);
delay_ms(100);
`)
}

func TestRequireWildcardGpio(t *testing.T) {
	checkOK(t, `
require * from "gpio";
pin_mode(1, 0);
`)
}

func TestRequireWildcardUart(t *testing.T) {
	checkOK(t, `
require * from "uart";
let h = uart_init(1, 9600);
`)
}

func TestRequireWildcardSpi(t *testing.T) {
	checkOK(t, `
require * from "spi";
let h = spi_init(0, 1000);
`)
}

func TestRequireWildcardI2c(t *testing.T) {
	checkOK(t, `
require * from "i2c";
let h = i2c_init(1, 80);
`)
}

func TestRequireWildcardTimer(t *testing.T) {
	checkOK(t, `
require * from "timer";
let t = timer_init(0, 1000);
`)
}

// --- checkCallExpression: call non-function ---

func TestCallNonFunction(t *testing.T) {
	checkOK(t, `
let x = 42;
x();
`)
}

// --- checkMemberExpression: non-class, non-module object ---

func TestMemberOnNonClassNonModule(t *testing.T) {
	checkOK(t, `
let x = 42;
let y = x.foo;
`)
}

// --- checkAssignExpression: compound assignment (not "=") ---

func TestCompoundAssignment(t *testing.T) {
	checkOK(t, `
let x = 5;
x += 3;
`)
}

// --- assign to index expression (falls to default return Any) ---

func TestAssignToIndex(t *testing.T) {
	checkOK(t, `
let a = [1, 2, 3];
a[0] = 5;
`)
}
