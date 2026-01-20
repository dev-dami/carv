# Carv Language Specification

Carv is a statically-typed, concurrent scripting language designed for safety and performance. This document provides a comprehensive overview of the language syntax and features.

## Variable Declarations

Carv supports three types of variable declarations:

- `let`: Immutable variable (by default, but can be shadowed).
- `mut`: Mutable variable.
- `const`: Constant value that cannot be changed.

```carv
let x = 10
mut y = 20
y = 30 // OK

const PI = 3.14
```

## Data Types

### Basic Types

- `int`: 64-bit integer.
- `float`: 64-bit floating point.
- `string`: UTF-8 string.
- `bool`: Boolean (`true` or `false`).
- `char`: Single character.
- `void`: No value.

### Arrays

Arrays are ordered collections of elements of the same type.

```carv
let arr = [1, 2, 3, 4, 5]
let first = arr[0]
```

## Operators

### Arithmetic

- `+`, `-`, `*`, `/`, `%`

### Comparison

- `==`, `!=`, `<`, `<=`, `>`, `>=`

### Logical

- `&&` (AND)
- `||` (OR)
- `!` (NOT)

### Pipe Operator

The pipe operator `|>` allows for clean functional data flow. It passes the result of the left expression as the first argument to the right function.

```carv
x |> double |> add(5) |> print
```

## Functions

Functions are defined using the `fn` keyword.

```carv
fn add(a: int, b: int) -> int {
    return a + b
}

// Higher order functions are supported
fn apply(f: fn(int) -> int, x: int) -> int {
    return f(x)
}
```

## Control Flow

### If/Else

```carv
if x > 0 {
    print("Positive")
} else if x < 0 {
    print("Negative")
} else {
    print("Zero")
}
```

### Loops

#### For loop (C-style)

```carv
for (let i = 0; i < 10; i = i + 1) {
    print(i)
}
```

#### For-in loop

```carv
for item in arr {
    print(item)
}
```

#### While loop

```carv
mut i = 0
while i < 10 {
    print(i)
    i = i + 1
}
```

#### Loop keyword (Infinite loop)

```carv
loop {
    print("Forever")
    break // Use break to exit
}
```

## Built-in Functions

- `print(...)`: Print values to stdout.
- `println(...)`: Print values followed by a newline.
- `len(iterable)`: Return the length of a string or array.
- `str(value)`: Convert value to string.
- `int(value)`: Convert to integer.
- `float(value)`: Convert to float.
- `push(array, element)`: Return a new array with element appended.
- `head(array)`: Return the first element.
- `tail(array)`: Return all elements except the first.

## Concurrency (Planned)

Carv has reserved keywords for concurrency features that are currently under development:

- `spawn { ... }`: Launch a concurrent task.
- `await expression`: Wait for a task to complete.
- `chan`: Communication channels.
- `send` / `recv`: Channel operations.
- `select`: Multiplexed channel operations.
