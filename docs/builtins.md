[← Architecture](architecture.md) | **Built-ins** | [Examples](examples.md) | [Contributing →](../CONTRIBUTING.md)

---

# Built-in Functions

Reference for all built-in functions available in Carv.

## Output

### `print(...args)`
Prints arguments to stdout, space-separated, without a trailing newline.

```carv
print("hello", "world");  // hello world
print(1, 2, 3);           // 1 2 3
```

### `println(...args)`
Prints arguments to stdout, space-separated, with a trailing newline.

## Type Conversion

### `str(value) -> string`
Convert any value to its string representation.

```carv
str(42)        // "42"
str(true)      // "true"
str([1,2,3])   // "[1, 2, 3]"
```

### `int(value) -> int`
Convert to integer. Works with floats and booleans.

```carv
int(3.14)   // 3
int(true)   // 1
int(false)  // 0
```

### `float(value) -> float`
Convert to float.

```carv
float(42)  // 42.0
```

### `type_of(value) -> string`
Get the type of a value as a string.

```carv
type_of(42)        // "INTEGER"
type_of("hello")   // "STRING"
type_of([1,2,3])   // "ARRAY"
```

## Collections

### `len(collection) -> int`
Get length of string or array.

```carv
len("hello")     // 5
len([1, 2, 3])   // 3
```

### `push(array, item) -> array`
Return new array with item appended.

```carv
let a = [1, 2];
let b = push(a, 3);  // [1, 2, 3]
// a is still [1, 2]
```

### `head(array) -> any`
Get first element of array.

```carv
head([1, 2, 3])  // 1
head([])         // nil
```

### `tail(array) -> array`
Get all elements except the first.

```carv
tail([1, 2, 3])  // [2, 3]
tail([1])        // []
```

## Maps

### `keys(map) -> array`
Get all keys from a map.

```carv
let m = {"a": 1, "b": 2};
keys(m)  // ["a", "b"]
```

### `values(map) -> array`
Get all values from a map.

```carv
let m = {"a": 1, "b": 2};
values(m)  // [1, 2]
```

### `has_key(map, key) -> bool`
Check if key exists in map.

```carv
let m = {"a": 1};
has_key(m, "a")  // true
has_key(m, "b")  // false
```

### `set(map, key, value) -> map`
Return new map with key set to value.

```carv
let m = {"a": 1};
let m2 = set(m, "b", 2);  // {"a": 1, "b": 2}
```

### `delete(map, key) -> map`
Return new map with key removed.

```carv
let m = {"a": 1, "b": 2};
let m2 = delete(m, "a");  // {"b": 2}
```

## Strings

### `split(str, separator) -> array`
Split string into array.

```carv
split("a,b,c", ",")  // ["a", "b", "c"]
```

### `join(array, separator) -> string`
Join array into string.

```carv
join(["a", "b", "c"], "-")  // "a-b-c"
```

### `trim(str) -> string`
Remove leading/trailing whitespace.

```carv
trim("  hello  ")  // "hello"
```

### `substr(str, start, end?) -> string`
Get substring. End is optional.

```carv
substr("hello", 0, 2)  // "he"
substr("hello", 2)     // "llo"
```

### `contains(str, substr) -> bool`
Check if string contains substring.

```carv
contains("hello", "ell")  // true
```

### `starts_with(str, prefix) -> bool`
Check if string starts with prefix.

```carv
starts_with("hello", "he")  // true
```

### `ends_with(str, suffix) -> bool`
Check if string ends with suffix.

```carv
ends_with("hello", "lo")  // true
```

### `replace(str, old, new) -> string`
Replace all occurrences.

```carv
replace("hello", "l", "L")  // "heLLo"
```

### `index_of(str, substr) -> int`
Find index of substring. Returns -1 if not found.

```carv
index_of("hello", "l")   // 2
index_of("hello", "x")   // -1
```

### `to_upper(str) -> string`
Convert to uppercase.

```carv
to_upper("hello")  // "HELLO"
```

### `to_lower(str) -> string`
Convert to lowercase.

```carv
to_lower("HELLO")  // "hello"
```

### `ord(char) -> int`
Get ASCII code of character.

```carv
ord('A')     // 65
ord("A")     // 65 (takes first char)
```

### `chr(int) -> char`
Get character from ASCII code.

```carv
chr(65)  // 'A'
```

### `char_at(str, index) -> char`
Get character at index.

```carv
char_at("hello", 1)  // 'e'
```

## Parsing

### `parse_int(str) -> int`
Parse a string as an integer.

```carv
parse_int("42")    // 42
parse_int("-10")   // -10
```

### `parse_float(str) -> float`
Parse a string as a float.

```carv
parse_float("3.14")   // 3.14
parse_float("2.0")    // 2
```

## Process & Environment

### `args() -> array`
Get command-line arguments passed to the script.

```carv
let a = args();
print(a);   // ["arg1", "arg2", ...]
```

### `exec(command, ...args) -> int`
Run an external command. Returns the exit code.

```carv
let code = exec("echo", "hello");  // prints "hello", returns 0
```

### `exec_output(command, ...args) -> Result`
Run an external command and capture output. Returns `Ok(stdout)` or `Err(stderr)`.

```carv
let result = exec_output("echo", "hello");
match result {
    Ok(out) => print(trim(out)),
    Err(e) => print("failed: " + e),
}
```

### `getenv(key) -> string`
Get an environment variable. Returns empty string if not set.

```carv
let home = getenv("HOME");
```

### `setenv(key, value)`
Set an environment variable.

```carv
setenv("MY_VAR", "hello");
```

## File I/O

### `read_file(path) -> string`
Read entire file contents.

```carv
let content = read_file("data.txt");
```

### `write_file(path, content)`
Write string to file.

```carv
write_file("out.txt", "hello");
```

### `append_file(path, content)`
Append string to file. Creates the file if it doesn't exist.

```carv
append_file("log.txt", "new line\n");
```

### `file_exists(path) -> bool`
Check if file exists.

```carv
if file_exists("config.txt") {
    // ...
}
```

### `mkdir(path)`
Create a directory (and parent directories).

```carv
mkdir("build/output");
```

### `remove_file(path)`
Delete a file.

```carv
remove_file("old.txt");
```

### `rename_file(old_path, new_path)`
Rename or move a file.

```carv
rename_file("old.txt", "new.txt");
```

### `read_dir(path) -> array`
Read directory entries and return sorted file/directory names.

```carv
let entries = read_dir(".");
println(entries);
```

### `cwd() -> string`
Get current working directory.

```carv
println(cwd());
```

## Networking (TCP)

Use these via the built-in `net` or `web` module:

```carv
require "net" as net;
```

`net` and `web` currently expose the same low-level TCP primitives.

### `tcp_listen(host, port) -> int`
Create a TCP listener and return a listener handle.

```carv
let listener = net.tcp_listen("127.0.0.1", 8080);
```

### `tcp_accept(listener) -> int`
Accept one incoming TCP connection and return a connection handle.

```carv
let conn = net.tcp_accept(listener);
```

### `tcp_read(conn, max_bytes) -> string`
Read up to `max_bytes` bytes from a TCP connection.

```carv
let request = net.tcp_read(conn, 4096);
```

### `tcp_write(conn, data) -> int`
Write string data to a TCP connection. Returns number of bytes written.

```carv
let wrote = net.tcp_write(conn, "hello");
```

### `tcp_close(handle) -> bool`
Close a listener or connection handle.

```carv
net.tcp_close(conn);
net.tcp_close(listener);
```

## Control Flow

### `exit(code?)`
Exit program with optional status code.

```carv
exit();    // exit with 0
exit(1);   // exit with 1
```

### `panic(message)`
Crash with error message.

```carv
panic("something went wrong");
```

---

[← Architecture](architecture.md) | **Built-ins** | [Examples](examples.md) | [Contributing →](../CONTRIBUTING.md)
