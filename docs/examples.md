[← Back to README](../README.md) | **Examples** | [Built-ins →](builtins.md)

---

# Carv Examples

The old `/examples` tree was consolidated into this document and `docs/samples/` so language usage is documented in one place.

## Hello World + Pipes

```carv
fn double(n: int) -> int {
    return n * 2;
}

print("Hello, Carv!");
let x = 10;
x |> double |> print;
```

## String Interpolation

```carv
let name = "Carv";
let version = "0.4.0";
println(f"Welcome to {name} v{version}!");
println(f"2 + 3 = {2 + 3}");
```

## Ownership and Borrowing

```carv
let s = "hello";
let moved = s;

fn print_len(v: &string) {
    println(len(v));
}

let msg = "world";
print_len(&msg);
```

## Result + Match

```carv
fn divide(a: int, b: int) {
    if b == 0 {
        return Err("division by zero");
    }
    return Ok(a / b);
}

match divide(10, 2) {
    Ok(v) => println(v),
    Err(e) => println(e),
};
```

## Modules

```carv
// math.carv
pub fn add(a: int, b: int) -> int {
    return a + b;
}

// main.carv
require { add } from "./math";
println(add(1, 2));
```

## Native TCP Server

```carv
require "net" as net;

let listener = net.tcp_listen("127.0.0.1", 8080);
let conn = net.tcp_accept(listener);
let req = net.tcp_read(conn, 4096);
println(req);

let body = "Hello from Carv TCP server!\n";
let response =
    "HTTP/1.1 200 OK\r\n" +
    "Content-Type: text/plain\r\n" +
    "Content-Length: 28\r\n" +
    "Connection: close\r\n\r\n" +
    body;

net.tcp_write(conn, response);
net.tcp_close(conn);
net.tcp_close(listener);
```

## Runnable Samples

Use these maintained sample files:
- `docs/samples/hello.carv`
- `docs/samples/showcase.carv`

Run with:

```bash
./build/carv run docs/samples/hello.carv
./build/carv run docs/samples/showcase.carv
```

---

[← Back to README](../README.md) | **Examples** | [Built-ins →](builtins.md)
