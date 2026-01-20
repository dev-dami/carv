# Carv

An ambitious little programming language I've been tinkering with.

---

## The Story

This project has been a long time coming. Back in September last year, I started working on something called **dyms** (Dynamic Yet Minimal Scripts) - first in Go, then rewrote it in Rust, then... gave up. Life happened, motivation faded, the usual.

Fast forward to now - I'm back at it. Ported what I had to Go, cleaned things up, and renamed it Carv. The goal is still the same: build a language that compiles to C, eventually make it self-hosted (write the Carv compiler in Carv itself).

We'll see how far I get this time.

---

## What It Does

Carv compiles to C and runs natively. It has a tree-walking interpreter too for quick testing.

Features that actually work:
- Static typing with inference
- Pipe operator (`|>`) - my favorite part
- Classes with methods
- Result types (`Ok`/`Err`) with pattern matching
- Hash maps
- Basic standard library

---

## Quick Look

```carv
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
./build/carv repl               # mess around
```

---

## Where Things Stand

- [x] Lexer, parser, type checker
- [x] Interpreter
- [x] C codegen
- [x] Result types, classes, maps
- [ ] Module system
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
