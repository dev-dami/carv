# Carv

An ambitious little programming language I've been tinkering with.

---

## The Story

This project has been a long time coming. Back in September last year, I started working on something called **dyms** (Dynamic Yet Minimal Script) - first in Go, then rewrote it in Rust, then... i gave up. Life happened, motivation faded, School, yk the usual.

Fast forward to now right now and i'm back at it. Ported what I had to Go, cleaned things up, and renamed it Carv. The goal is still the same: build a language that compiles to C, eventually make it self-hosted (write the Carv compiler in Carv itself).

We'll see how far I get this time.

---

## What It Does

Carv compiles to C and runs natively. It has a tree-walking interpreter too for quick testing.

Features that actually work:
- Static typing with inference
- Pipe operator (`|>`) - my favorite part and why not
- `let` / `mut` / `const` with proper immutability enforcement
- Compound assignment (`+=`, `-=`, `*=`, `/=`, `%=`, `&=`, `|=`, `^=`)
- Classes with methods
- Result types (`Ok`/`Err`) with pattern matching cause **RUST**
- Hash maps
- `for-in` loops over arrays, strings, and maps
- **Module system** with `require` (Rust-inspired, package manager ready)
- **String interpolation** with `f"hello {name}"`
- Project config via `carv.toml`
- 40+ built-in functions (strings, files, process, environment, etc.)

---

## Quick Look

```carv
// string interpolation
let name = "World";
println(f"Hello, {name}!");
println(f"2 + 2 = {2 + 2}");

// pipes make everything nicer
10 |> double |> add(5) |> print;

// error handling without exceptions
fn divide(a: int, b: int) {
    if b == 0 {
        return Err("nope");
    }
    return Ok(a / b);
}

let x = divide(10, 2)?;

// hash maps
let scores = {"alice": 100, "bob": 85};

// classes
class Counter {
    value: int = 0
    fn increment() {
        self.value = self.value + 1;
    }
}
```

### Modules

```carv
// math.carv
pub fn add(a: int, b: int) -> int {
    return a + b;
}

// main.carv
require { add } from "./math";
println(f"1 + 2 = {add(1, 2)}");
```

---

## Building

```bash
git clone https://github.com/dev-dami/carv
cd carv
make build
```

Then:
```bash
./build/carv run file.carv      # interpret
./build/carv build file.carv    # compile to binary
./build/carv emit-c file.carv   # emit generated C source
./build/carv init               # create new project with carv.toml
./build/carv repl               # mess around
```

---

## Where Things Stand

- [x] Lexer, parser, type checker
- [x] Interpreter
- [x] C codegen
- [x] Result types, classes, maps
- [x] Module system (`require`)
- [x] String interpolation (`f"..."`)
- [x] Project config (`carv.toml`)
- [ ] Package manager
- [ ] Self-hosting

---

## Docs

- [Language Guide](docs/language.md)
- [Architecture](docs/architecture.md)
- [Built-ins](docs/builtins.md)
- [Contributing](CONTRIBUTING.md)

---

## License

MIT

---

*This is a hobby project. I work on it when I have the energy. No promises, no timelines.*
