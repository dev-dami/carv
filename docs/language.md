[← Back to README](../README.md) | **Language Guide** | [API Reference](api.md) | [Architecture →](architecture.md)

---

# Carv Language Guide

Quick reference for Carv's syntax and features. This is a work in progress - things might change as I figure stuff out.

## String Interpolation

Use `f"..."` for interpolated strings:

```carv
let name = "Carv";
let version = "0.2.0";
println(f"Welcome to {name} v{version}!");

// expressions work too
let x = 10;
let y = 5;
println(f"{x} + {y} = {x + y}");

// function calls
println(f"Length of name: {len(name)}");
```

Use `{{` and `}}` to escape braces: `f"Use {{braces}} like this"`.

## Variables

```carv
let x = 10;           // immutable
mut y = 20;           // mutable
y = 30;               // ok
const PI = 3.14159;   // constant
```

### Compound Assignment

Mutable variables support compound assignment operators:

```carv
mut x = 10;
x += 5;    // 15
x -= 3;    // 12
x *= 2;    // 24
x /= 4;    // 6
x %= 4;    // 2
x &= 3;   // bitwise AND
x |= 8;   // bitwise OR
x ^= 5;   // bitwise XOR
```

## Types

Basic types: `int`, `float`, `string`, `bool`, `char`, `void`

```carv
let name: string = "hello";
let count: int = 42;
let pi: float = 3.14;
let flag: bool = true;
let c: char = 'a';
```

Type inference works most of the time so you can skip annotations.

## Ownership

Carv has a move-based ownership system. Some types are **copy types** (implicitly copied), others are **move types** (ownership is transferred).

### Copy Types

Copy types are implicitly copied on assignment: `int`, `float`, `bool`, `char`, and references (`&T`, `&mut T`).

```carv
let x = 42;
let y = x;      // x is copied, both x and y are valid
print(x);       // OK
print(y);       // OK
```

### Move Types

Move types transfer ownership on assignment: `string`, `[]T` (arrays), `{K:V}` (maps), and class instances.

```carv
let s = "hello";
let t = s;      // s is moved into t, s is now invalid
// print(s);    // ERROR: use of moved value 's'
print(t);       // OK: "hello"

mut x = "world";
x = "new";      // old value dropped, new value assigned
```

## Borrowing

References allow temporary access without transferring ownership. Carv enforces borrow rules at compile time.

### Immutable Borrows

Use `&x` to create an immutable borrow:

```carv
fn print_len(s: &string) -> int {
    return len(s);
}

let msg = "hello";
print_len(&msg);    // immutable borrow — msg still valid
print(msg);         // OK
```

### Mutable Borrows

Use `&mut x` to create a mutable borrow:

```carv
fn append_excl(s: &mut string) {
    *s = *s + "!";
}

mut greeting = "hi";
append_excl(&mut greeting);  // mutable borrow
print(greeting);             // "hi!"
```

### Borrow Rules

- **One mutable XOR many immutable**: You can have either one mutable borrow OR multiple immutable borrows, but not both.
- **No borrow escape**: References cannot be returned from functions or stored in fields.
- **Can't move while borrowed**: You cannot move or reassign a value while it is borrowed.

```carv
let s = "hello";
let r = &s;
// let t = s;   // ERROR: cannot move 's' while it is borrowed
print(len(r));  // OK: 5
```

### Dereference

Use `*x` to dereference a reference:

```carv
fn increment(x: &mut int) {
    *x = *x + 1;
}

mut n = 5;
increment(&mut n);
print(n);  // 6
```

## Interfaces

Interfaces define a set of methods that types can implement. Dispatch is dynamic via vtable.

### Defining an Interface

```carv
interface Printable {
    fn to_string(&self) -> string;
    fn display(&self);
}
```

Interface methods must declare a receiver: `&self` (immutable borrow) or `&mut self` (mutable borrow). Value receivers (`self`) are not supported in v1.

### Implementing an Interface

Use `impl Interface for Type`:

```carv
class Person {
    name: string
}

impl Printable for Person {
    fn to_string(&self) -> string {
        return self.name;
    }
    fn display(&self) {
        println(self.name);
    }
}
```

The checker verifies that all interface methods are implemented with matching signatures.

### Interface References and Dynamic Dispatch

Cast a class reference to an interface reference with `as`:

```carv
let p = new Person;
p.name = "Alice";

let item: &Printable = &p as &Printable;
item.display();  // dynamic dispatch through vtable
```

Only `&Interface` (immutable) and `&mut Interface` (mutable) are supported — owned trait objects are not.

### Cast Expression

The `as` keyword is an infix operator for type casts:

```carv
let x = &p as &Printable;       // immutable interface ref
let y = &mut p as &mut Printable; // mutable interface ref
```

## Arrays

```carv
let nums = [1, 2, 3, 4, 5];
let first = nums[0];
let length = len(nums);
```

## Maps

Hash maps with curly brace syntax:

```carv
let scores = {"alice": 100, "bob": 85, "charlie": 92};

// access
print(scores["alice"]);  // 100

// check if key exists
if has_key(scores, "dave") {
    print("found dave");
}

// get keys and values
let names = keys(scores);    // ["alice", "bob", "charlie"]
let points = values(scores); // [100, 85, 92]

// immutable operations (return new maps)
let updated = set(scores, "dave", 88);
let removed = delete(scores, "bob");
```

Keys must be hashable (strings, integers, booleans).

## Functions

```carv
fn add(a: int, b: int) -> int {
    return a + b;
}

fn greet(name: string) {
    print("Hello, " + name);
}

// anonymous functions
let double = fn(x: int) -> int { return x * 2; };
```

## Async / Await

Carv supports `async fn` and `await`. Async functions are compiled into state machines in C codegen.

```carv
async fn fetch_data() -> int {
    return 42;
}

async fn carv_main() -> int {
    let v = await fetch_data();
    println(v);
    return 0;
}
```

### Rules

- `await` can only be used inside `async fn`.
- `await` expects an async/future-producing expression.
- Async locals captured across suspension points are stored in generated async frames.
- For compiled async programs (`carv build`), use `async fn carv_main() -> int` as entrypoint.

## Pipe Operator

This is my favorite feature. Pass results through a chain of functions:

```carv
// instead of: print(add(5, double(x)))
x |> double |> add(5) |> print;

// with arrays
[1, 2, 3] |> head |> print;  // 1
```

The left side becomes the first argument of the right side.

## Control Flow

### If/Else

```carv
if x > 0 {
    print("positive");
} else if x < 0 {
    print("negative");
} else {
    print("zero");
}
```

### Loops

```carv
// c-style for
for (let i = 0; i < 10; i = i + 1) {
    print(i);
}

// for-in (arrays)
for item in [1, 2, 3] {
    print(item);
}

// for-in (strings — iterates characters)
for ch in "hello" {
    print(ch);
}

// for-in (maps — iterates keys)
for key in {"a": 1, "b": 2} {
    print(key);
}

// while
while condition {
    // ...
}

// infinite loop
for {
    if done {
        break;
    }
}
```

## Classes

```carv
class Counter {
    value: int = 0

    fn increment() {
        self.value = self.value + 1;
    }

    fn get() -> int {
        return self.value;
    }
}

let c = new Counter;
c.increment();
print(c.get());  // 1
```

## Modules

Carv has a Rust-inspired module system using `require`:

### Importing

```carv
// import entire module
require "./utils";

// import with alias
require "./utils" as u;

// import specific exports
require { add, subtract } from "./math";

// import all exports
require * from "./math";

// import builtin stdlib module
require "net" as net;
```

### Exporting

Use `pub` to mark functions, classes, constants, and variables as public:

```carv
// math.carv
pub fn add(a: int, b: int) -> int {
    return a + b;
}

pub fn multiply(a: int, b: int) -> int {
    return a * b;
}

// private - not exported
fn helper() {
    // ...
}

pub const PI = 3.14159;
pub let VERSION = "0.2.0";
```

### Project Structure

Initialize a project with `carv init`:

```text
myproject/
├── carv.toml          # project config
├── src/
│   └── main.carv      # entry point
└── carv_modules/      # dependencies (future)
```

### carv.toml

```toml
[package]
name = "myproject"
version = "0.1.0"
entry = "src/main.carv"

[dependencies]
# future: external packages

[build]
output = "build"
optimize = true
```

## Result Types

For error handling without exceptions:

```carv
fn divide(a: int, b: int) -> Result {
    if b == 0 {
        return Err("cannot divide by zero");
    }
    return Ok(a / b);
}

// pattern matching
let result = divide(10, 2);
match result {
    Ok(v) => print(v),
    Err(e) => print("error: " + e),
}

// try operator (early return on error)
fn calculate() -> Result {
    let x = divide(10, 2)?;  // unwraps Ok or returns Err
    let y = divide(x, 3)?;
    return Ok(y);
}
```

## Built-in Functions

### General
- `print(...)` / `println(...)` - print to stdout
- `len(x)` - length of string or array
- `str(x)` - convert to string
- `int(x)` - convert to int
- `float(x)` - convert to float
- `type_of(x)` - get type as string

### Arrays
- `push(arr, item)` - return new array with item appended
- `head(arr)` - first element
- `tail(arr)` - all elements except first

### Strings
- `split(str, sep)` - split string into array
- `join(arr, sep)` - join array into string
- `trim(str)` - remove whitespace
- `substr(str, start, end?)` - substring
- `contains(str, substr)` - check if contains
- `starts_with(str, prefix)` - check prefix
- `ends_with(str, suffix)` - check suffix
- `replace(str, old, new)` - replace all occurrences
- `index_of(str, substr)` - find index (-1 if not found)
- `to_upper(str)` / `to_lower(str)` - case conversion
- `ord(char)` - character to ASCII code
- `chr(int)` - ASCII code to character
- `char_at(str, idx)` - get character at index

### Parsing
- `parse_int(str)` - parse string as integer
- `parse_float(str)` - parse string as float

### Maps
- `keys(map)` - get all keys as array
- `values(map)` - get all values as array
- `has_key(map, key)` - check if key exists
- `set(map, key, value)` - return new map with key set
- `delete(map, key)` - return new map with key removed

### File I/O
- `read_file(path)` - read file contents
- `write_file(path, content)` - write to file
- `append_file(path, content)` - append to file
- `file_exists(path)` - check if file exists
- `mkdir(path)` - create directory
- `remove_file(path)` - delete file
- `rename_file(old_path, new_path)` - rename/move file
- `read_dir(path)` - list directory entries
- `cwd()` - current working directory

### Networking (TCP)
- Import built-in module: `require "net" as net;` (or `require "web" as web;`)
- `net.tcp_listen(host, port)` - create listener, return handle
- `net.tcp_accept(listener)` - accept connection, return handle
- `net.tcp_read(conn, max_bytes)` - read bytes, return string
- `net.tcp_write(conn, data)` - write string, return bytes written
- `net.tcp_close(handle)` - close listener/connection handle

### Process & Environment
- `args()` - get CLI arguments
- `exec(cmd, ...args)` - run command, return exit code
- `exec_output(cmd, ...args)` - run command, return `Ok(stdout)` / `Err(stderr)`
- `getenv(key)` - get environment variable
- `setenv(key, value)` - set environment variable

### Control Flow
- `exit(code?)` - exit program
- `panic(msg)` - crash with message

## Notes

- Semicolons are required at the end of statements
- The language is statically typed but has decent inference
- Map and array builtins (`push`, `set`, `delete`) return new values (originals unchanged)
- Mutable variables (`mut`) support direct mutation via index/field assignment
- Error handling uses Result types instead of exceptions

---

[← Back to README](../README.md) | **Language Guide** | [API Reference](api.md) | [Architecture →](architecture.md)
