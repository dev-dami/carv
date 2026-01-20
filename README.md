# Carv Programming Language

Carv is a memory-safe, concurrent scripting language that compiles to C and then to native binaries. It features a clean, Lua-inspired syntax with curly braces and a powerful pipe operator for functional data flow.

## Features

- **Static Typing**: Strong type system with type inference.
- **Pipe Operator**: Functional data flow using the `|>` operator.
- **Multi-paradigm**: Supports both functional and imperative programming styles.
- **Compiled**: Compiles to C for high performance and portability.
- **Memory Safe**: Designed with safety in mind.

## Installation

### From Source

Ensure you have Go installed on your system.

```bash
git clone https://github.com/carv-lang/carv
cd carv
go build -o carv ./cmd/carv
```

### Via Go Install

```bash
go install github.com/carv-lang/carv/cmd/carv@latest
```

## Usage

The `carv` CLI provides several commands:

- **run**: Execute a Carv source file using the interpreter.
  ```bash
  carv run hello.carv
  ```
- **build**: Compile a Carv file to a native binary via C.
  ```bash
  carv build hello.carv
  ```
- **emit-c**: Output the generated C code for a Carv file.
  ```bash
  carv emit-c hello.carv
  ```
- **repl**: Start an interactive Read-Eval-Print Loop.
  ```bash
  carv repl
  ```
- **version**: Show the current version.
  ```bash
  carv version
  ```

## Quick Example

```carv
// Define a function with static types
fn double(n: int) -> int {
    return n * 2
}

fn add(a: int, b: int) -> int {
    return a + b
}

let x = 10

// Use the pipe operator for clean data flow
x |> double |> add(5) |> print
```

## Documentation

- [Language Specification](docs/language.md)
- [Example Code](examples/)

## License

This project is licensed under the MIT License.
