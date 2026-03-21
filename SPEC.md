# Codong Language Specification

Version 0.7.0 | codong.org | MIT License | `.cod` files

---

## 1. Basic Syntax

- Source encoding: UTF-8.
- Identifiers: start with letter or `_`, followed by letters, digits, `_`. Names use `snake_case`.
- Comments: `//` single-line, `/* */` multi-line.
- File structure: top-level statements execute sequentially. One file = one module unless `export` is used.
- Statements end at newline. No semicolons.
- Blocks use `{ }`.

String escape sequences: `\n` (newline), `\t` (tab), `\r` (carriage return), `\\` (backslash), `\"` (double quote), `\0` (null byte).

## 2. Data Types

Six primitive types + two collection types, all inferred:

| Type     | Example                        |
|----------|--------------------------------|
| `string` | `"hello"`, `"value is {x}"`    |
| `number` | `42`, `3.14`, `-1`             |
| `bool`   | `true`, `false`                |
| `null`   | `null`                         |
| `list`   | `[1, 2, 3]`                    |
| `map`    | `{name: "Ada", age: 30}`       |

String interpolation: `"text {expr}"`. Any expression is valid inside `{}`: variables, arithmetic (`{a + b}`), member access (`{user.name}`), method calls (`{items.len()}`). No other interpolation syntax allowed.

Multi-line strings: `"""..."""` (triple double quotes). Content is preserved as-is, including leading whitespace and newlines. Interpolation works inside.

Map keys: bare identifiers when valid (`{name: "Ada"}`). Keys with special characters must use double quotes (`{"Content-Type": "application/json"}`).

**Map access:** `m.key` (dot, when key is valid identifier) and `m["key"]` (bracket, any string key). Accessing a non-existent key returns `null`.

**List access:** `list[n]` (zero-indexed). Negative indices supported: `list[-1]` is last element. Out-of-bounds returns `null`.

**null rules:** `null == null` is `true`. `null == false` is `false`. `null` is falsy in `if`/`while` conditions. Only `false` and `null` are falsy; `0`, `""`, `[]`, `{}` are truthy.

Type checking: `type_of(x)` returns `"string"`, `"number"`, `"bool"`, `"null"`, `"list"`, `"map"`, `"fn"`.

## 3. Variables & Assignment

- `=` is the only assignment operator. No `var`, `let`, `:=`.
- `const` prevents rebinding only (compile error on reassignment or compound assignment). A `const` list or map can still be mutated via methods like `push` or `delete`. To prevent mutation, avoid calling mutating methods.
- Compound assignment: `+=`, `-=`, `*=`, `/=` are allowed on non-const variables.

## 4. Functions

```
fn add(a, b) {
    return a + b
}

// with type annotations (for agent.tool auto Schema inference)
fn search(query: string, limit: number) {
    return results
}

// default parameter values (= in definition)
fn create_user(name, role = "member") {
    return {name: name, role: role}
}

// arrow function (single expression)
double = fn(x) => x * 2

// anonymous function
handler = fn(req) { return {status: 200} }
```

- `fn` is the only function keyword. No `function`, `def`, `lambda`.
- Arrow form `fn(params) => expr` for single expressions only.
- Type annotations are optional, used for `agent.tool` auto Schema inference.
- Default values: `fn(a, b = 10)`. Use `=` in definition. Parameters with defaults must come after required parameters.
- Named arguments at call site: `fn(a, key: value)`. Use `:` at call site. Positional args first, named args after.
- Nested function definitions are legal (closures capture outer scope).
- Functions return a single value. Use a map or list to return multiple values.
- Block functions require explicit `return`. The last expression is not implicitly returned. A function without `return` returns `null`.

## 5. Control Flow

```
if x > 0 {
    print("positive")
} else if x == 0 {
    print("zero")
} else {
    print("negative")
}

for item in items {
    print(item)
}

for i in range(0, 10) {
    print(i)
}

while running {
    data = poll()
}

match status {
    200 => print("ok")
    404 => print("not found")
    _ => print("other: {status}")
}
```

- `break`, `continue`, `return` work as expected.
- No `switch`, `case`, ternary `?:`, or `do-while`.
- `range(start, end)` is a built-in function (not a keyword), returns a list of integers from `start` to `end-1`.
- `match` arms only support literals (`number`, `string`, `bool`, `null`) and `_` wildcard. Variable matching is not supported. String literals in match arms must be plain text — no `{}` interpolation allowed.
- `for ... in` only iterates over `list`. To iterate a map, use `m.keys()`, `m.values()`, or `m.entries()`.

## 6. Type System

```
type User = {
    name: string,
    age: number,
    email: string,
}

interface Searchable {
    fn search(query: string) => list
}
```

- Type declarations define structure shapes.
- `interface` declares required method signatures. Structural typing: any value with matching methods automatically satisfies the interface (no explicit `implements` keyword). Checked at compile time when type annotations are used.
- Type conversion: `to_string(x)`, `to_number(x)`, `to_bool(x)`.

## 7. Module System

Built-in modules (8) are available directly — no `import` needed: `web`, `db`, `llm`, `agent`, `cloud`, `queue`, `cron`, `error`.

```
server = web.serve(port: 8080)
```

Custom modules use `import` / `export`:

```
// math_utils.cod
export fn square(x) { return x * x }
export const PI = 3.14159
export type Point = { x: number, y: number }

// main.cod
import { square, PI, Point } from "./math_utils.cod"
```

`export` can modify `fn`, `const`, and `type` declarations.

Third-party packages use `@namespace` scoped names:

```
import { verify } from "@codong/jwt"
import { hash } from "@alice/crypto"
```

- Official packages: no scope prefix (e.g., `codong-jwt`).
- Third-party packages: must use `@namespace` (e.g., `@alice/utils`), prevents name squatting.
- `codong.lock` ensures 100% reproducible builds (pinned to SHA-256 hash).

## 8. Concurrency

```
go fn() {
    data = fetch_data()
    ch <- data
}()

ch = channel()
ch <- "message"       // send (space before <-)
msg = <-ch            // receive (no space after <-)

ch = channel(size: 10) // buffered

select {
    msg = <-ch1 {
        handle(msg)
    }
    msg = <-ch2 {
        process(msg)
    }
    <-done {
        break
    }
}
```

- `go` keyword launches concurrent execution. No `async`, `await`.
- `channel()` creates channels.
- Send: `ch <- value` (space before `<-`). Receive: `<-ch` (no space, prefix operator).
- `select` multiplexes channel operations. Each arm uses `{ }` block syntax. Assignment is optional — `<-ch { ... }` discards the received value.

## 9. Error Handling

All errors are structured JSON with `code`, `message`, `fix`, `retry` fields.

```
err = error.new("E1001", "invalid type", fix: "use number instead")

try {
    result = db.find("users", {id: 1})
} catch err {
    print(err.code)    // "E2001_NOT_FOUND"
    print(err.fix)     // "check if table exists"
}

// ? operator propagates errors (postfix, no optional chaining)
data = db.find("users", {id: 1})?
```

- `error.new(code, message, opts)` is the only way to create errors.
- `error.wrap(err, context)` adds context to existing errors.
- `error.set_format("compact")` switches to compact format (saves 39% tokens).
- `?` postfix operator: if the expression evaluates to an error, immediately return that error to the caller; otherwise, evaluate to the expression's value unchanged. No optional chaining `?.` syntax.
- Error identity: an error object is created exclusively via `error.new()` or `error.wrap()` and carries an internal type tag. Regular maps with `code` or `error` fields are NOT treated as errors by `?`.

Compact format: `err_code:E2001_NOT_FOUND|src:db.find|fix:run db.migrate()|retry:false`

## 10. Built-in Functions

These are globally available without import. They are not keywords — they are functions.

| Function | Returns | Description |
|----------|---------|-------------|
| `print(value)` | null | standard output (single argument only; use interpolation for multiple: `print("{a} {b}")`) |
| `type_of(x)` | string | returns type name |
| `to_string(x)` | string | convert to string |
| `to_number(x)` | number/null | convert to number, null if invalid |
| `to_bool(x)` | bool | convert to bool |
| `range(start, end)` | list | integers from start to end-1 |

Length: use `.len()` method on each type (`s.len()`, `l.len()`, `m.len()`). No global `len()` function.

## 11. Built-in Type Methods

**Mutability rule:** strings are immutable (all methods return new strings). Lists are mutable — `push`, `pop`, `sort`, `reverse`, `shift`, `unshift` modify in place and return `self` for chaining. Maps have only one mutating method: `delete`. All other methods including `merge`, `filter`, `map_values` return new values without modifying the original.

### string (17 methods, all return new strings)

| Method | Returns | Description |
|--------|---------|-------------|
| `s.len()` | number | byte length |
| `s.upper()` | string | uppercase |
| `s.lower()` | string | lowercase |
| `s.trim()` | string | strip whitespace |
| `s.trim_start()` | string | strip leading whitespace |
| `s.trim_end()` | string | strip trailing whitespace |
| `s.split(sep)` | list | split by separator |
| `s.contains(sub)` | bool | contains substring |
| `s.starts_with(pre)` | bool | starts with prefix |
| `s.ends_with(suf)` | bool | ends with suffix |
| `s.replace(old,new)` | string | replace all occurrences |
| `s.index_of(sub)` | number | first index, -1 if absent |
| `s.slice(start,end?)` | string | substring |
| `s.repeat(n)` | string | repeat n times |
| `s.to_number()` | number | parse number, null if invalid |
| `s.to_bool()` | bool | "true"/"1" -> true |
| `s.match(pattern)` | list | regex match all |

### list (20 methods)

Mutating methods modify the original list and return `self`. Non-mutating methods return new values.

| Method | Mutates? | Returns | Description |
|--------|----------|---------|-------------|
| `l.len()` | no | number | element count |
| `l.push(item)` | **yes** | self | append to end |
| `l.pop()` | **yes** | item | remove and return last |
| `l.shift()` | **yes** | item | remove and return first |
| `l.unshift(item)` | **yes** | self | prepend |
| `l.sort(fn?)` | **yes** | self | sort in place |
| `l.reverse()` | **yes** | self | reverse in place |
| `l.slice(start,end?)` | no | list | new sub-list |
| `l.map(fn)` | no | list | new transformed list |
| `l.filter(fn)` | no | list | new filtered list |
| `l.reduce(fn,init)` | no | any | accumulate |
| `l.find(fn)` | no | item/null | first match |
| `l.find_index(fn)` | no | number | first match index |
| `l.contains(item)` | no | bool | membership test |
| `l.index_of(item)` | no | number | first index |
| `l.flat(depth?)` | no | list | new flattened list |
| `l.unique()` | no | list | new deduplicated list |
| `l.join(sep)` | no | string | join as string |
| `l.first()` | no | item/null | first element |
| `l.last()` | no | item/null | last element |

### map (10 methods)

| Method | Mutates? | Returns | Description |
|--------|----------|---------|-------------|
| `m.len()` | no | number | key count |
| `m.keys()` | no | list | all keys |
| `m.values()` | no | list | all values |
| `m.entries()` | no | list | [[key,value],...] |
| `m.has(key)` | no | bool | key exists |
| `m.get(key,default?)` | no | any | get with default |
| `m.delete(key)` | **yes** | self | remove key in place |
| `m.merge(other)` | no | map | new merged map, other wins |
| `m.map_values(fn)` | no | map | new map with transformed values |
| `m.filter(fn)` | no | map | new filtered map |

## 12. Operator Precedence

From highest to lowest:

| Precedence | Operators | Description |
|------------|-----------|-------------|
| 1 | `()` `[]` `.` `?` | grouping, index, member, error propagation |
| 2 | `!` `-` (unary) | logical not, negate |
| 3 | `*` `/` `%` | multiply, divide, modulo |
| 4 | `+` `-` | add, subtract |
| 5 | `<` `>` `<=` `>=` | comparison |
| 6 | `==` `!=` | equality |
| 7 | `&&` | logical and |
| 8 | `\|\|` | logical or |
| 9 | `<-` | channel send/receive |
| 10 | `=` `+=` `-=` `*=` `/=` | assignment |

## 13. Keywords (23 total)

```
fn       return   if       else     for      while    match
break    continue const    import   export   try      catch
go       select   interface type    null     true     false
in       _
```

`_` is a keyword: match wildcard and discard marker (`_ = side_effect()`).

Not keywords (built-in modules or functions, can appear in expressions):
- `error` — built-in module (`error.new()`, `error.wrap()`), not a keyword
- `channel` — built-in function (`channel()`, `channel(size: 10)`), not a keyword
- `range`, `print`, `type_of`, `to_string`, `to_number`, `to_bool` — built-in functions (not keywords)
- `bridge` — `codong.toml` config section
- `use` — valid identifier (e.g., `server.use(middleware)`)

## 14. Mandatory Code Style

| Rule | Standard |
|------|----------|
| Indentation | 4 spaces (no tabs) |
| Naming | `snake_case` for variables, functions, modules |
| Type names | `PascalCase` |
| Line length | max 120 characters |
| Braces | opening `{` on same line |
| Strings | double quotes `"` only (no single quotes) |
| Trailing comma | required in multi-line list/map |

`codong fmt` enforces all style rules automatically.

## 15. Go Bridge Extension Protocol

Go Bridge allows human architects to wrap any Go library for AI consumption.

### Workflow

1. **Write** Go wrapper in `bridge/` directory
2. **Register** in `codong.toml` with permissions
3. **Call** registered function name in `.cod` files

### Registration (codong.toml)

```toml
[bridge]
wechat_pay = { fn = "bridge.WechatPay", permissions = ["net:outbound"] }
pdf_render = { fn = "bridge.RenderPDF", permissions = ["fs:write:/tmp/codong-sandbox"] }
hash_md5   = { fn = "bridge.HashMD5", permissions = [] }
```

### Permission Types

| Permission | Format | Scope |
|------------|--------|-------|
| None | `[]` | pure computation, no I/O |
| Network | `["net:outbound"]` | outbound HTTP only |
| File read | `["fs:read:<path>"]` | read specified directory |
| File write | `["fs:write:<path>"]` | write specified directory |

**Prohibited operations:** `os.Exit`, `syscall`, `os/exec`, `net.Listen`, host root access.

### Calling from .cod

```
result = wechat_pay(amount: 99.9, order_id: order.id)
if result.error {
    print("payment failed: {result.error}")
}
```

AI calls registered function names directly. Bridge functions accept and return Codong basic types only (`string`, `number`, `bool`, `map`, `list`). Bridge functions must not panic.

**Bridge error convention:** on failure, return `{error: "description"}`. On success, the `error` field must be `null` or absent (accessing non-existent key returns `null`). Never use `{error: ""}` for success — empty string `""` is truthy in Codong and would incorrectly trigger error handling.

---

CODONG | codong.org | MIT License
