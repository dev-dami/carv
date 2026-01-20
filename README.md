# Carv

An ambitious little programming language I've been tinkering with. It compiles to C, has a pipe operator I really like, and I'm slowly working toward making it self-hosted (writing the Carv compiler in Carv itself).

This is very much a hobby project so expect some rough edges, half-baked features, and occasional breaking changes as I figure things out.

## What's Working

- Static typing with type inference
- Pipe operator (`|>`) for chaining function calls
- Classes with methods
- Result types (`Ok`/`Err`) with pattern matching
- Hash maps
- Compiles to C, runs natively
- Tree-walking interpreter for quick iteration

## Quick Look at the Syntax

```carv
// pipe operator - probably my favorite feature
let result = [1, 2, 3, 4, 5]
    |> map(fn(x) { x * 2; })
    |> filter(fn(x) { x > 4; })
    |> print;

// result types for error handling
fn divide(a: int, b: int) -> Result {
    if b == 0 {
        return Err("division by zero");
    }
    return Ok(a / b);
}

let x = divide(10, 2)?;  // unwraps Ok or returns Err

// hash maps
let scores = {"alice": 100, "bob": 85};
print(scores["alice"]);

// classes
class Counter {
    value: int = 0
    fn increment() {
        self.value = self.value + 1;
    }
}
```

## Building

You'll need Go installed.

```bash
git clone https://github.com/dev-dami/carv
cd carv
make build
```

## Usage

```bash
# run with interpreter
./build/carv run yourfile.carv

# compile to native binary
./build/carv build yourfile.carv

# see the generated C
./build/carv emit-c yourfile.carv

# repl for messing around
./build/carv repl
```

## Project Status

This is a learning project. I started it to understand how compilers work, and I keep adding features as I learn more. The end goal is to make it self-hosted - writing the Carv compiler in Carv.

Current focus:
- [x] Core language (lexer, parser, type checker)
- [x] Tree-walking interpreter
- [x] C code generation
- [x] Result types and pattern matching
- [x] Classes
- [x] Hash maps
- [ ] Module/import system
- [ ] Self-hosting (the fun part)

## Documentation

- [Language Guide](docs/language.md) - syntax and features
- [Architecture](docs/architecture.md) - how the compiler works
- [Built-in Functions](docs/builtins.md) - standard library reference

## Syntax Notes

Semicolons are required (I know, I know). The syntax is roughly C-like with some Rust-isms for error handling.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines. This is mainly a personal project but feel free to open issues if you find bugs or have ideas.

## License

MIT
