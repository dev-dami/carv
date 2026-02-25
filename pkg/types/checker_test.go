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

func TestTypeCheckerPipes(t *testing.T) {
	input := `
fn double(x: int) -> int {
	return x * 2;
}
5 |> double |> print;
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
