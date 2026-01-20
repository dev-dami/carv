[← Back to README](../README.md) | **Language Guide** | [Architecture →](architecture.md)

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

// for-in
for item in items {
    print(item);
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
```

### Exporting

Use `pub` to mark functions and classes as public:

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

// constants don't support visibility modifiers (no `pub` keyword)
const PI = 3.14159;
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
- `print(...)` - print to stdout
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
- `to_upper(str)` - uppercase
- `to_lower(str)` - lowercase
- `ord(char)` - character to ASCII code
- `chr(int)` - ASCII code to character
- `char_at(str, idx)` - get character at index

### Maps
- `keys(map)` - get all keys as array
- `values(map)` - get all values as array
- `has_key(map, key)` - check if key exists
- `set(map, key, value)` - return new map with key set
- `delete(map, key)` - return new map with key removed

### Files
- `read_file(path)` - read file contents
- `write_file(path, content)` - write to file
- `file_exists(path)` - check if file exists

### Other
- `exit(code?)` - exit program
- `panic(msg)` - crash with message

## Notes

- Semicolons are required at the end of statements
- The language is statically typed but has decent inference
- Maps, arrays, and strings are immutable (operations return new values)
- Error handling uses Result types instead of exceptions

---

[← Back to README](../README.md) | **Language Guide** | [Architecture →](architecture.md)
