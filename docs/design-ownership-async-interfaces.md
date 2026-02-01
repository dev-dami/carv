# Carv Design: Ownership, Interfaces, and Async/Await

This document specifies the full design for three new language systems before any implementation begins.

---

## Decisions Made

| Decision | Choice |
|----------|--------|
| Borrow checker target | C codegen only; interpreter gets warnings |
| Lifetime annotations | None — lexical scope only |
| Primitives (int/float/bool/char) | Copy types (implicit copy, no move tracking) |
| Heap types (string/array/map/class) | Move types (assignment = move, explicit `clone()`) |
| Method receivers | All three: `self`, `&self`, `&mut self` |
| Interface dispatch | Dynamic via vtable; `&Interface` / `&mut Interface` only |
| Async model | `async fn` + `await`; remove `spawn` |
| Borrows across await | Forbidden — only owned values survive suspension |
| Returning borrows | Forbidden — references cannot be returned or stored in fields |
| Interpreter divergence | Checker warns on ownership errors, interpreter ignores them |

---

## Implementation Phases

```
Phase 0: Foundation (type info plumbing)
    │
    ├── Phase 1a: Ownership Core (move + drop)
    │       │
    │       ├── Phase 1b: Borrowing (&T / &mut T)
    │       │
    │       └── Phase 1c: Interfaces (parse + check + vtable codegen)
    │               │
    │               └── Phase 2: Interface + Ownership Integration
    │                       │
    │                       ├── Phase 3a: Closures in Codegen
    │                       │       │
    │                       │       └── Phase 3b: Async Core (state machines + event loop)
    │                       │               │
    │                       │               └── Phase 4: TCP/HTTP
    │                       │
    │                       └── (1a, 1b, 1c can partially overlap)
```

---

## Phase 0: Foundation — Type Info Plumbing

**Problem**: The C codegen (`pkg/codegen/cgen.go`) has its own `inferExprType()` that is completely disconnected from the type checker (`pkg/types/checker.go`). Ownership metadata has no path from checker to codegen.

**Solution**: Make the type checker produce a `TypeInfo` map (`map[ast.Node]Type`) and make the codegen consume it instead of re-inferring types.

### Changes

**pkg/types/checker.go**:
```go
type CheckResult struct {
    NodeTypes   map[ast.Node]Type      // type of every expression
    FuncSigs    map[string]*FuncType   // function name → signature
    ClassInfo   map[string]*ClassType  // class name → field/method info
    Errors      []string
    Warnings    []string               // ownership warnings for interpreter mode
}
```

**pkg/codegen/cgen.go**:
- Accept `CheckResult` as input
- Replace all `inferExprType()` calls with lookups into `NodeTypes`
- Delete `inferExprType()` once migration is complete

**pkg/eval/eval.go**:
- Run checker before evaluation
- Print `Warnings` to stderr but continue execution

**cmd/carv/main.go**:
- `carv build`: checker errors are fatal
- `carv run`: checker ownership warnings printed, execution continues

---

## Phase 1a: Ownership Core — Move + Drop

### Type Categories

```go
// pkg/types/types.go
type TypeCategory int
const (
    CopyCat TypeCategory = iota  // int, float, bool, char, &T, &mut T
    MoveCat                       // string, []T, {K:V}, class instances
)

func (t Type) Category() TypeCategory {
    switch t.(type) {
    case *IntType, *FloatType, *BoolType, *CharType, *RefType:
        return CopyCat
    default:
        return MoveCat
    }
}
```

### Carv Syntax

```carv
let s = "hello";
let t = s;              // MOVE — s is now invalid
// print(s);            // ERROR: use of moved value 's'

let u = t.clone();      // explicit deep copy
print(t);               // OK — t still valid
print(u);               // OK

mut x = "world";
x = "new";              // OK — old value dropped, new value assigned

fn consume(s: string) { // by-value param = move
    print(s);
}
let a = "hi";
consume(a);             // a moved into function
// print(a);            // ERROR: use of moved value 'a'
```

### Checker State

```go
// pkg/types/checker.go
type OwnershipState int
const (
    Owned OwnershipState = iota
    Moved
)

type VarInfo struct {
    Type       Type
    Ownership  OwnershipState
    Mutable    bool
    ScopeDepth int
}
```

**Rules enforced**:
1. `let y = x` where x is Move type → mark x as `Moved`
2. Use of a `Moved` variable → error: `use of moved value 'x'`
3. Passing Move-type arg to by-value param → mark arg as `Moved`
4. `return x` → mark x as `Moved` (moved out of function)
5. At scope exit → any `Owned` Move-type var needs codegen to emit drop

### Error Messages

```
error at line 5: use of moved value 's' (moved at line 3)
error at line 8: use of moved value 'a' (moved into function 'consume' at line 7)
```

### C Codegen Patterns

**Runtime types** (emitted in C preamble):

```c
typedef struct { size_t len; char* data; } carv_string;

static carv_string carv_string_from_cstr(const char* s) {
    size_t len = strlen(s);
    char* data = malloc(len + 1);
    memcpy(data, s, len + 1);
    return (carv_string){len, data};
}

static carv_string carv_string_clone(const carv_string* s) {
    char* data = malloc(s->len + 1);
    memcpy(data, s->data, s->len + 1);
    return (carv_string){s->len, data};
}

static carv_string carv_string_move(carv_string* s) {
    carv_string out = *s;
    s->data = NULL;
    s->len = 0;
    return out;
}

static void carv_string_drop(carv_string* s) {
    if (s->data) { free(s->data); s->data = NULL; s->len = 0; }
}
```

**Single-exit functions** (all returns become `goto __carv_exit`):

```c
// Carv: fn example() { let s = "hi"; if cond { return; } print(s); }
void example(void) {
    carv_string s = carv_string_from_cstr("hi");

    if (cond) {
        goto __carv_exit;
    }
    carv_print_string(&s);

__carv_exit:
    carv_string_drop(&s);
}
```

**Move on assignment**:
```c
// let t = s;   (s is Move type)
carv_string t = carv_string_move(&s);
// s is now zeroed — safe to drop later (no-op)
```

**Drop on reassignment** (for `mut` vars):
```c
// mut x = "a"; x = "b";
carv_string x = carv_string_from_cstr("a");
carv_string_drop(&x);                        // drop old
x = carv_string_from_cstr("b");              // assign new
```

### clone() Builtin

Add `clone()` as a method-like builtin on all Move types:

```carv
let a = "hello";
let b = a.clone();  // deep copy
```

C output:
```c
carv_string b = carv_string_clone(&a);
```

---

## Phase 1b: Borrowing — &T / &mut T

### Carv Syntax

```carv
fn print_len(s: &string) -> int {
    return len(s);
}

fn append_excl(s: &mut string) {
    *s = *s + "!";
}

let msg = "hello";
print_len(&msg);           // immutable borrow — msg still valid
// append_excl(&mut msg);  // ERROR: msg is not mutable

mut greeting = "hi";
append_excl(&mut greeting); // mutable borrow
print(greeting);            // "hi!"
```

### AST Nodes

```go
// pkg/ast/ast.go — new expression nodes

type BorrowExpression struct {
    Token   lexer.Token   // the '&' token
    Mutable bool          // true for &mut
    Value   Expression    // the lvalue being borrowed
}

type DerefExpression struct {
    Token lexer.Token     // the '*' token
    Value Expression
}
```

**RefType already exists** in `pkg/ast/types.go` with a `Mutable` field — use it.

### Checker Rules

```go
type BorrowInfo struct {
    ImmutableCount int
    MutableActive  bool
    ScopeDepth     int
}
```

| Action | Requirement | Error |
|--------|-------------|-------|
| `&x` | x is Owned, not MutableActive | `cannot borrow 'x': already mutably borrowed` |
| `&mut x` | x is Owned, mutable, ImmCount==0, !MutableActive | `cannot mutably borrow 'x': already borrowed` |
| Move x | ImmCount==0 and !MutableActive | `cannot move 'x' while it is borrowed` |
| Assign x | ImmCount==0 and !MutableActive | `cannot assign to 'x' while it is borrowed` |
| Return &x | ALWAYS ERROR | `reference cannot escape function scope` |
| Store &x in class/array/map | ALWAYS ERROR | `reference cannot be stored in heap structure` |

### C Codegen

Immutable borrow → `const T*`:
```c
// fn print_len(s: &string) -> int
int64_t print_len(const carv_string* s) {
    return (int64_t)s->len;
}
// Call: print_len(&msg)
int64_t n = print_len(&msg);
```

Mutable borrow → `T*`:
```c
// fn append_excl(s: &mut string)
void append_excl(carv_string* s) {
    carv_string old = *s;
    *s = carv_string_concat(s, &(carv_string){1, "!"});
    carv_string_drop(&old);
}
// Call: append_excl(&mut greeting)
append_excl(&greeting);
```

---

## Phase 1c: Interfaces

### Carv Syntax

```carv
interface Printable {
    fn to_string(&self) -> string;
}

interface Resizable {
    fn resize(&mut self, size: int);
}

class Person {
    name: string
    age: int
}

impl Printable for Person {
    fn to_string(&self) -> string {
        return f"{self.name} ({self.age})";
    }
}

fn display(item: &Printable) {
    println(item.to_string());
}

let p = new Person;
p.name = "Ada".clone();
p.age = 36;
display(&p as &Printable);
```

### AST Nodes (already exist, need wiring)

`InterfaceStatement` and `ImplStatement` already exist in `pkg/ast/statements.go`. Add receiver info:

```go
type ReceiverKind int
const (
    RecvSelf    ReceiverKind = iota  // self (consumes)
    RecvRef                           // &self
    RecvMutRef                        // &mut self
)
```

Update `MethodSignature` and `MethodDecl` to include `Receiver ReceiverKind`.

### Checker Rules

1. `impl I for T` — verify every method in `I` is provided with matching signature
2. `fn foo(x: &Interface)` — x is a fat pointer, dynamic dispatch
3. Interface methods with `self` (by-value) are forbidden in v1:
   - `interface methods cannot take self by value; use &self or &mut self`
4. Calling `&mut self` method through `&Interface` (immutable) is an error:
   - `cannot call &mut self method through immutable interface reference`

### C Codegen — Vtable Pattern

For `interface Printable`:
```c
typedef struct {
    carv_string (*to_string)(const void* self);
} Printable_vtable;

typedef struct {
    const void* data;
    const Printable_vtable* vt;
} Printable_ref;

typedef struct {
    void* data;
    const Printable_vtable* vt;
} Printable_mut_ref;
```

For `impl Printable for Person`:
```c
static carv_string Printable__Person__to_string(const void* self) {
    const Person* p = (const Person*)self;
    return Person_to_string(p);
}

static const Printable_vtable Printable__Person__VT = {
    .to_string = Printable__Person__to_string,
};
```

Coercion (`&p as &Printable`):
```c
Printable_ref iface = { .data = &p, .vt = &Printable__Person__VT };
```

Dynamic dispatch:
```c
// item.to_string()  where item: &Printable
carv_string s = item.vt->to_string(item.data);
```

---

## Phase 2: Method Receiver Integration

Update existing class methods to declare receiver convention:

```carv
class Counter {
    value: int = 0

    fn get(&self) -> int {           // immutable borrow
        return self.value;
    }

    fn increment(&mut self) {        // mutable borrow
        self.value = self.value + 1;
    }

    fn into_value(self) -> int {     // consumes (moves) self
        return self.value;
    }
}
```

**Backward compatibility**: Existing methods without explicit receiver default to `&mut self`.

### C Codegen

```c
// &self → const ClassName*
int64_t Counter_get(const Counter* self) { return self->value; }

// &mut self → ClassName*
void Counter_increment(Counter* self) { self->value += 1; }

// self → ClassName* (caller moves, callee drops)
int64_t Counter_into_value(Counter* self) {
    int64_t __ret = self->value;
    Counter_drop(self);
    return __ret;
}
```

---

## Phase 3a: Closures in Codegen

**Prerequisite for async**. Currently `FunctionLiteral` is NOT implemented in codegen.

### C Pattern — Lambda Lifting

```carv
fn make_adder(x: int) {
    return fn(y: int) -> int { return x + y; };
}
```

C output:
```c
typedef struct {
    int64_t x;  // captured variable
} make_adder_closure_env;

static int64_t make_adder_closure_fn(make_adder_closure_env* env, int64_t y) {
    return env->x + y;
}

typedef struct {
    void* env;
    int64_t (*fn_ptr)(void*, int64_t);
} carv_closure_int_int;

carv_closure_int_int make_adder(int64_t x) {
    make_adder_closure_env* env = malloc(sizeof(make_adder_closure_env));
    env->x = x;
    return (carv_closure_int_int){ .env = env, .fn_ptr = (void*)make_adder_closure_fn };
}
```

**Ownership of captures**:
- Copy types captured by copy
- Move types captured by move (original becomes invalid)
- Borrow captures forbidden (would escape scope)

---

## Phase 3b: Async Core — State Machines + Event Loop

### Carv Syntax

```carv
async fn fetch(url: string) -> string {
    let conn = await tcp_connect(url, 80);
    await tcp_write(&mut conn, "GET / HTTP/1.1\r\n\r\n");
    let response = await tcp_read(&mut conn, 4096);
    return response;
}

async fn main() {
    let body = await fetch("example.com");
    println(body);
}
```

### AST Changes

```go
// FunctionStatement — add Async field
type FunctionStatement struct {
    // ... existing fields ...
    Async bool
}

// AwaitExpression already exists in ast.go
```

### Checker Rules

| Rule | Error |
|------|-------|
| `await` outside `async fn` | `await can only be used inside async functions` |
| `await x` where x is not Future | `cannot await non-future value of type X` |
| Borrow alive across await | `borrow of 'x' cannot live across await point` |
| `async fn` return type | Implicitly `Future<T>` where T is the declared return |

### C Codegen — State Machine Transformation

Each `async fn` becomes:

1. A **frame struct** holding state + locals that survive await points
2. A **poll function** implementing the state machine
3. The original function becomes a **constructor** returning the frame

```c
// Frame for: async fn fetch(url: string) -> string
typedef struct fetch_frame {
    int state;
    // Parameters (owned)
    carv_string url;
    // Locals that live across awaits
    TcpStream conn;
    carv_string response;
    // Sub-futures
    tcp_connect_frame* sub_connect;
    tcp_write_frame*   sub_write;
    tcp_read_frame*    sub_read;
    // Return slot
    carv_string ret;
} fetch_frame;

// Poll function — returns true when done
static bool fetch_poll(fetch_frame* f, carv_loop* loop) {
    switch (f->state) {
    case 0:
        f->sub_connect = tcp_connect_start(loop, &f->url, 80);
        f->state = 1;
        return false;
    case 1:
        if (!tcp_connect_poll(f->sub_connect, loop, &f->conn)) return false;
        free(f->sub_connect); f->sub_connect = NULL;
        f->sub_write = tcp_write_start(loop, &f->conn, "GET / HTTP/1.1\r\n\r\n");
        f->state = 2;
        return false;
    case 2:
        if (!tcp_write_poll(f->sub_write, loop)) return false;
        free(f->sub_write); f->sub_write = NULL;
        f->sub_read = tcp_read_start(loop, &f->conn, 4096);
        f->state = 3;
        return false;
    case 3:
        if (!tcp_read_poll(f->sub_read, loop, &f->response)) return false;
        free(f->sub_read); f->sub_read = NULL;
        f->ret = carv_string_move(&f->response);
        f->state = 4;
        return true;
    }
    return false;
}

// Drop — cleanup owned resources if cancelled
static void fetch_drop(fetch_frame* f) {
    carv_string_drop(&f->url);
    TcpStream_drop(&f->conn);
    carv_string_drop(&f->response);
    carv_string_drop(&f->ret);
    if (f->sub_connect) free(f->sub_connect);
    if (f->sub_write) free(f->sub_write);
    if (f->sub_read) free(f->sub_read);
    free(f);
}
```

### C Runtime — Event Loop

```c
typedef struct carv_task {
    bool (*poll)(void* frame, carv_loop* loop);
    void (*drop)(void* frame);
    void* frame;
} carv_task;

typedef struct carv_fd_interest {
    int fd;
    int events;          // POLLIN, POLLOUT
    carv_task* task;     // task waiting on this fd
} carv_fd_interest;

typedef struct carv_loop {
    carv_task** ready;       // tasks ready to poll
    int ready_count;
    int ready_cap;
    carv_fd_interest* fds;   // fd wait registrations
    int fd_count;
    int fd_cap;
} carv_loop;

void carv_loop_run(carv_loop* loop) {
    while (loop->ready_count > 0 || loop->fd_count > 0) {
        // 1. Poll all ready tasks
        for (int i = 0; i < loop->ready_count; i++) {
            carv_task* t = loop->ready[i];
            if (t->poll(t->frame, loop)) {
                t->drop(t->frame);    // task done, cleanup
                // remove from ready list
            }
        }
        // 2. Wait for fd events via poll()
        struct pollfd* pfds = /* build from loop->fds */;
        int n = poll(pfds, loop->fd_count, -1);
        // 3. Move tasks with ready fds to ready list
    }
}
```

---

## Phase 4: TCP/HTTP Builtins

### TCP Primitives (all async, return Futures)

```carv
// Built-in async functions
async fn tcp_listen(addr: string, port: int) -> TcpListener;
async fn tcp_accept(listener: &mut TcpListener) -> TcpStream;
async fn tcp_connect(host: string, port: int) -> TcpStream;
async fn tcp_read(stream: &mut TcpStream, max_bytes: int) -> string;
async fn tcp_write(stream: &mut TcpStream, data: string) -> int;
fn tcp_close(stream: TcpStream);  // sync, consumes (drops) the stream
```

### HTTP Built on TCP

```carv
class HttpRequest {
    method: string
    path: string
    headers: {string: string}
    body: string

    fn method(&self) -> string { return self.method.clone(); }
    fn path(&self) -> string { return self.path.clone(); }
}

class HttpResponse {
    stream: TcpStream

    async fn send(&mut self, status: int, body: string) {
        let header = f"HTTP/1.1 {status} OK\r\nContent-Length: {len(body)}\r\n\r\n";
        await tcp_write(&mut self.stream, header);
        await tcp_write(&mut self.stream, body);
    }
}

async fn http_listen(addr: string, port: int, handler: fn(HttpRequest, &mut HttpResponse)) {
    let listener = await tcp_listen(addr, port);
    for {
        let stream = await tcp_accept(&mut listener);
        let req = parse_http_request(&mut stream);
        mut res = HttpResponse { stream: stream };
        handler(req, &mut res);
    }
}
```

### Usage

```carv
async fn handle(req: HttpRequest, res: &mut HttpResponse) {
    if req.path() == "/" {
        await res.send(200, "Hello from Carv!");
    } else {
        await res.send(404, "Not Found");
    }
}

async fn main() {
    println("Server starting on :8080");
    await http_listen("0.0.0.0", 8080, handle);
}
```

---

## Known Risks and Mitigations

| Risk | Severity | Mitigation |
|------|----------|------------|
| Codegen ignores checker output | CRITICAL | Phase 0 MUST be done first — bridge checker→codegen |
| `Any` type bypasses ownership | CRITICAL | Define: `Any` treated as Move type, ownership rules still apply |
| Drop on all control flow paths | HIGH | Single-exit pattern (`goto __carv_exit`) for all functions |
| `self` receiver backward compat | HIGH | Default existing methods to `&mut self`; no breaking change |
| Arena allocator vs malloc/free | HIGH | Dual strategy: arena for temporaries, malloc for owned heap values |
| `?` operator + drops | MEDIUM | Emit drop code before the early-return goto |
| Pipe operator + moves | MEDIUM | Pipes borrow by default (pass `&` implicitly) |
| Closures capture ownership | MEDIUM | Copy types by copy, Move types by move, borrows forbidden |

---

## What This Design Does NOT Include (v1 Scope)

- Lifetime annotations
- Owned trait objects (`Printable` by value)
- Self-by-value in interface methods
- Multi-threaded async (single-threaded event loop only)
- Generic types / type parameters
- `Drop` trait (automatic drop only, no custom destructors yet)
- HTTP keep-alive, chunked encoding, TLS
- Channel-based concurrency (`chan`/`select`)

---

## Acceptance Tests (one per phase)

**Phase 0**: `carv build` and `carv run` produce identical output for existing test programs.

**Phase 1a** (moves):
```carv
let s = "hello";
let t = s;
print(t);       // OK: "hello"
// print(s);    // ERROR: use of moved value 's'
```

**Phase 1b** (borrows):
```carv
let s = "hello";
let r = &s;
print(len(r));  // OK: 5
// let t = s;   // ERROR: cannot move 's' while it is borrowed
```

**Phase 1c** (interfaces):
```carv
interface Greetable {
    fn greet(&self) -> string;
}
class Dog { name: string }
impl Greetable for Dog {
    fn greet(&self) -> string { return f"Woof from {self.name}!"; }
}
fn say_hi(g: &Greetable) { println(g.greet()); }
let d = new Dog;
d.name = "Rex".clone();
say_hi(&d as &Greetable);  // "Woof from Rex!"
```

**Phase 3b** (async):
```carv
async fn delayed_hello() -> string {
    return "hello async!";
}
async fn main() {
    let msg = await delayed_hello();
    println(msg);  // "hello async!"
}
```

**Phase 4** (HTTP):
```carv
async fn handle(req: HttpRequest, res: &mut HttpResponse) {
    await res.send(200, "Hello from Carv!");
}
async fn main() {
    await http_listen("0.0.0.0", 8080, handle);
}
// curl http://localhost:8080 → "Hello from Carv!"
```
