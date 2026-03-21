# Codong Language Specification — AI Edition

This document is the AI-optimized version of SPEC.md. Every rule includes a correct example and an incorrect example. Inject this into any LLM system prompt to enable correct Codong code generation.

---

## 1. Basic Syntax

**File encoding:** UTF-8. Statements end at newline. No semicolons.

```
// CORRECT
x = 10
name = "codong"

// WRONG — no semicolons
x = 10;
```

**Comments:**

```
// CORRECT — single line
// this is a comment

// CORRECT — multi line
/* this spans
   multiple lines */

// WRONG — no # comments
# this is not valid
```

**Identifiers:** letters, digits, `_`. Must start with letter or `_`.

```
// CORRECT
user_name = "Ada"
_private = 42

// WRONG — no hyphens, no starting with digit
user-name = "Ada"
2fast = true
```

**String escape sequences:**

```
// CORRECT — supported escapes
msg = "line1\nline2"       // newline
msg = "col1\tcol2"         // tab
msg = "say \"hello\""      // escaped quote
path = "C:\\Users\\brett"   // backslash
// Also supported: \r (carriage return), \0 (null byte)
```

---

## 2. Data Types

Six types: `string`, `number`, `bool`, `null`, `list`, `map`. All type-inferred.

```
// CORRECT
name = "hello"
count = 42
pi = 3.14
active = true
nothing = null
items = [1, 2, 3]
user = {name: "Ada", age: 30}

// WRONG — no type annotations on variables
string name = "hello"
int count = 42
```

**String interpolation:** `"text {expr}"` with double quotes only. Any expression is valid inside `{}`.

```
// CORRECT — simple variable
greeting = "Hello {name}"

// CORRECT — arithmetic expression
result = "Total: {a + b}"

// CORRECT — member access
info = "User: {user.name}"

// CORRECT — method call
msg = "Count: {items.len()}"

// CORRECT
greeting = "Hello {name}, you are {age} years old"

// WRONG — no backtick templates, no f-strings, no single quotes
greeting = `Hello ${name}`
greeting = f"Hello {name}"
greeting = 'hello'
```

**Multi-line strings:** use triple double quotes `"""..."""`. Content preserved as-is (including leading whitespace). Interpolation works inside.

```
// CORRECT — multi-line string
html = """
<div>
    <h1>Hello {name}</h1>
    <p>Welcome to Codong</p>
</div>
"""

// WRONG — no heredoc, no backtick multi-line
html = `
<div>hello</div>
`
```

**Map keys:** bare identifiers when valid. Keys with special characters must use double quotes.

```
// CORRECT — bare key (valid identifier)
config = {host: "localhost", port: 8080}

// CORRECT — quoted key (special characters)
headers = {"Content-Type": "application/json", "X-Request-ID": "abc123"}

// WRONG — bare key with special characters (Parser cannot parse this)
headers = {Content-Type: "application/json"}
```

**Map access:** dot notation and bracket notation both legal.

```
// CORRECT — dot access (key is valid identifier)
user = {name: "Ada", age: 30}
x = user.name        // "Ada"
x = user.age         // 30

// CORRECT — bracket access (any string key)
headers = {"Content-Type": "application/json"}
x = headers["Content-Type"]

// CORRECT — accessing non-existent key returns null (no error)
user = {name: "Ada"}
x = user.email       // null
x = user["phone"]    // null

// use m.has(key) to check existence
if user.has("email") {
    send_email(user.email)
}
```

**List access:** zero-indexed, negative indices supported.

```
// CORRECT — index access
items = [10, 20, 30]
first = items[0]     // 10
last = items[-1]     // 30 (negative = from end)
x = items[99]        // null (out-of-bounds returns null)

// WRONG — no slice syntax with ..
sub = items[1..3]
```

**null rules:**

```
// null equality
null == null         // true
null == false        // false (null is NOT false)
null == 0            // false

// falsy values: ONLY null and false
// truthy values: everything else, including 0, "", [], {}
if null { }          // does NOT enter block (falsy)
if false { }         // does NOT enter block (falsy)
if 0 { }             // ENTERS block (0 is truthy)
if "" { }            // ENTERS block ("" is truthy)
if [] { }            // ENTERS block ([] is truthy)
```

---

## 3. Variables & Assignment

`=` is the only assignment. `const` prevents rebinding only. Compound assignment allowed on non-const.

```
// CORRECT
x = 10
const MAX = 100
x += 5               // compound assignment
x -= 1
x *= 2
x /= 3

// WRONG — no var, let, :=
var x = 10
let x = 10
x := 10

// CORRECT — discard return value with _
_ = some_side_effect()

// WRONG — const cannot be rebound or use compound assignment
const MAX = 100
MAX += 1              // ERROR: cannot assign to const
MAX = 200             // ERROR: cannot assign to const

// CORRECT — const list/map can still be mutated via methods
const items = [1, 2, 3]
items.push(4)         // OK: items is now [1, 2, 3, 4] (mutation, not rebinding)

// WRONG — but cannot rebind const
const items = [1, 2, 3]
items = [4, 5]        // ERROR: cannot assign to const
```

---

## 4. Functions

`fn` is the only keyword. Arrow form for single expressions.

```
// CORRECT — standard function
fn add(a, b) {
    return a + b
}

// CORRECT — with type annotations (for agent.tool auto Schema)
fn search(query: string, limit: number) {
    return db.find("results", {q: query})
}

// CORRECT — default parameter values (= in definition)
fn create_user(name, role = "member") {
    return {name: name, role: role}
}

// CORRECT — arrow function (single expression)
double = fn(x) => x * 2

// CORRECT — anonymous function
handler = fn(req) { return {status: 200} }

// CORRECT — nested function (closure, captures outer scope)
fn make_counter() {
    count = 0
    fn increment() {
        count += 1
        return count
    }
    return increment
}
```

```
// WRONG — no function keyword
function add(a, b) { return a + b }

// WRONG — no def keyword
def add(a, b):
    return a + b

// WRONG — no bare arrow without fn
double = (x) => x * 2
double = x => x * 2

// WRONG — no lambda
square = lambda x: x * x
```

**Named arguments at call site:** use `:` (not `=`). Positional args first.

```
// CORRECT — named arguments at call site use :
server = web.serve(port: 8080)
response = llm.ask(model: "gpt-4o", prompt: "hello")
user = create_user("Ada", role: "admin")

// WRONG — named args before positional
server = web.serve(port: 8080, "localhost")

// IMPORTANT DISTINCTION:
// = in function DEFINITION means default value
// : at function CALL SITE means named argument
fn greet(name, greeting = "Hello") { }   // = is default
greet("Ada", greeting: "Hi")             // : is named arg
```

**export:** can modify `fn`, `const`, and `type`.

```
// CORRECT — export functions, constants, and types
export fn square(x) { return x * x }
export const PI = 3.14159
export type Point = { x: number, y: number }

// WRONG — cannot export bare variables
export x = 10
```

**Single return value only:**

```
// CORRECT — return one value; use map for multiple
fn divide(a, b) {
    if b == 0 {
        return {result: null, error: "division by zero"}
    }
    return {result: a / b, error: null}
}

// WRONG — no multiple return values
fn divide(a, b) {
    return a / b, null
}

// WRONG — no implicit return (last expression is NOT auto-returned)
fn add(a, b) {
    a + b           // this does nothing, function returns null
}

// CORRECT — explicit return required
fn add(a, b) {
    return a + b
}
```

---

## 5. Control Flow

```
// CORRECT — if / else if / else
if x > 0 {
    print("positive")
} else if x == 0 {
    print("zero")
} else {
    print("negative")
}

// WRONG — no parentheses around condition
if (x > 0) {
    print("positive")
}

// CORRECT — for iteration
for item in items {
    print(item)
}

// CORRECT — for range (range is a built-in function, not a keyword)
for i in range(0, 10) {
    print(i)
}

// WRONG — no C-style for
for (i = 0; i < 10; i++) {
    print(i)
}

// CORRECT — while
while running {
    data = poll()
}

// WRONG — no do-while
do {
    data = poll()
} while (running)

// CORRECT — match (=> for single expressions)
match status {
    200 => print("ok")
    404 => print("not found")
    _ => print("other: {status}")
}

// WRONG — no switch/case
switch (status) {
    case 200: print("ok")
}
```

**match arms:** only literals (`number`, `string`, `bool`, `null`) and `_` wildcard. No variable matching.

```
// CORRECT — literals and _ only
match status {
    200 => print("ok")
    "error" => print("failed")
    null => print("no status")
    _ => print("other")
}

// WRONG — cannot match against variables
expected = 200
match status {
    expected => print("matched")  // ERROR: this is not valid
}

// CORRECT — use if/else for variable comparison
if status == expected {
    print("matched")
}

// WRONG — no string interpolation in match arms
match msg {
    "error: {code}" => print("caught")  // ERROR: no interpolation in match
}

// CORRECT — match with plain string literals only
match msg {
    "not found" => print("404")
    "forbidden" => print("403")
    _ => print("other")
}
```

**for...in only works on list.** To iterate a map, use `.keys()`, `.values()`, or `.entries()`.

```
// CORRECT — iterate list
for item in [1, 2, 3] {
    print(item)
}

// CORRECT — iterate map keys
m = {a: 1, b: 2}
for key in m.keys() {
    print("{key}: {m[key]}")
}

// CORRECT — iterate map entries
for entry in m.entries() {
    print("{entry[0]}: {entry[1]}")
}

// WRONG — cannot for...in a map directly
for item in m {
    print(item)
}
```

`_` in `match` is the wildcard (matches anything).

---

## 6. Type System

```
// CORRECT — type declaration
type User = {
    name: string,
    age: number,
    email: string,
}

// CORRECT — interface
interface Searchable {
    fn search(query: string) => list
}

// WRONG — no class keyword
class User {
    constructor(name) { this.name = name }
}

// WRONG — no struct keyword
struct User {
    name string
}
```

**Interface:** structural typing — any value with matching methods automatically satisfies the interface. No `implements` keyword needed.

```
// CORRECT — structural typing (duck typing)
interface Searchable {
    fn search(query: string) => list
}

fn process(s: Searchable) {
    results = s.search("test")  // works if s has a search method
}

// WRONG — no explicit implements
class MySearcher implements Searchable { }
```

**Type conversion:** use built-in functions.

```
// CORRECT
s = to_string(42)
n = to_number("42")
b = to_bool("true")

// WRONG — no casting syntax
s = (string)42
s = 42 as string
```

---

## 7. Module System

**Built-in modules (8):** `web`, `db`, `llm`, `agent`, `cloud`, `queue`, `cron`, `error` — no import needed.

```
// CORRECT — built-in modules used directly
server = web.serve(port: 8080)
conn = db.connect("postgres://localhost/mydb")
response = llm.ask(model: "gpt-4o", prompt: "hello")

// WRONG — do not import built-in modules
import web
from web import serve
```

**Custom modules:**

```
// CORRECT — export (fn, const, type all allowed)
// math_utils.cod
export fn square(x) { return x * x }
export const PI = 3.14159
export type Point = { x: number, y: number }

// CORRECT — import
import { square, PI, Point } from "./math_utils.cod"

// WRONG — no require, no default export
const m = require("./math_utils")
export default fn square(x) { return x * x }
```

**Third-party packages:** use `@namespace` scoped names.

```
// CORRECT — import from @namespace package
import { verify } from "@codong/jwt"
import { hash } from "@alice/crypto"

// WRONG — no bare package names for third-party (prevents name squatting)
import { hash } from "crypto"
```

- Official packages: no scope (e.g., `codong-jwt`)
- Third-party: must use `@namespace` (e.g., `@alice/utils`)
- `codong.lock` pins all dependencies to SHA-256 hash for 100% reproducible builds

---

## 8. Concurrency

```
// CORRECT — goroutine
go fn() {
    data = fetch_data()
    ch <- data
}()

// WRONG — no async/await
async fn fetch() {
    data = await get_data()
}

// CORRECT — channel
ch = channel()
ch <- "message"       // send: space before <-
msg = <-ch            // receive: no space, <-ch is prefix operator

ch = channel(size: 10) // buffered

// WRONG — no channel methods
ch.send("message")
msg = ch.receive()

// CORRECT — select uses { } blocks (NOT => arrows)
select {
    msg = <-ch1 {
        handle(msg)
    }
    msg = <-ch2 {
        process(msg)
    }
}

// CORRECT — select without assignment (discard received value)
select {
    <-done {
        break
    }
}

// WRONG — select with => is ambiguous, do NOT use
select {
    msg = <- ch1 => handle(msg)
}
```

**Channel syntax summary:**
- Send: `ch <- value` (space before `<-`, binary operator)
- Receive: `<-ch` (no space, prefix operator)
- This distinction is unambiguous for the Parser.

---

## 9. Error Handling

All errors are structured JSON: `code`, `message`, `fix`, `retry`.

```
// CORRECT — create error
err = error.new("E1001", "invalid type", fix: "use number instead")

// WRONG — no new Error() or raise
err = new Error("invalid type")
raise Exception("invalid type")

// CORRECT — try/catch
try {
    result = db.find("users", {id: 1})
} catch err {
    print(err.code)    // "E2001_NOT_FOUND"
    print(err.fix)     // "check if table exists"
}

// CORRECT — ? operator propagates errors (postfix)
data = db.find("users", {id: 1})?

// CORRECT — wrap error with context
wrapped = error.wrap(err, "while loading user profile")

// CORRECT — compact format (saves 39% tokens)
error.set_format("compact")
// output: err_code:E2001_NOT_FOUND|src:db.find|fix:run db.migrate()|retry:false
```

**`?` postfix operator semantics:** if the expression evaluates to an error, immediately return that error to the caller. Otherwise, evaluate to the expression's value unchanged. No optional chaining `?.`.

**Error identity:** an error object is created exclusively via `error.new()` or `error.wrap()` and carries an internal type tag. Regular maps with `code` fields are NOT errors.

```
// CORRECT — only error.new() creates real errors
err = error.new("E1001", "bad input")
data = might_fail()?   // ? only triggers on real error objects

// This is just a normal map, ? will NOT propagate it
result = {code: "SUCCESS", message: "ok"}
x = result?            // x = {code: "SUCCESS", message: "ok"} (not an error)

// WRONG — Codong does NOT have optional chaining
x = user?.address?.city

// CORRECT — check null explicitly
if user != null {
    if user.address != null {
        city = user.address.city
    }
}
```

---

## 10. Built-in Functions

Globally available without import. These are functions, NOT keywords.

| Function | Returns | Description |
|----------|---------|-------------|
| `print(value)` | null | standard output (the ONLY output function) |
| `type_of(x)` | string | returns type name |
| `to_string(x)` | string | convert to string |
| `to_number(x)` | number/null | convert to number, null if invalid |
| `to_bool(x)` | bool | convert to bool |
| `range(start, end)` | list | integers from start to end-1 |

```
// CORRECT — print is the standard output function
print("Hello World")
print("count: {count}")

// WRONG — log() is NOT a Codong function
log("Hello World")
console.log("Hello World")
fmt.Println("Hello World")

// print() takes a single argument. Use interpolation for multiple values:
// CORRECT
print("{a} {b} {c}")

// WRONG — no multiple arguments
print(a, b, c)

// CORRECT — use .len() method for length (no global len() function)
items = [1, 2, 3]
n = items.len()       // 3
s = "hello"
n = s.len()           // 5
m = {a: 1, b: 2}
n = m.len()           // 2

// WRONG — no global len() function
n = len(items)

// CORRECT — range is a built-in function (not a keyword)
nums = range(0, 5)    // [0, 1, 2, 3, 4]
for i in range(1, 4) {
    print(i)           // 1, 2, 3
}
```

---

## 11. Built-in Type Methods

**Mutability rule:**
- **Strings** are immutable — all methods return new strings, original unchanged.
- **Lists** are mutable — `push`, `pop`, `sort`, `reverse`, `shift`, `unshift` modify in place and return `self` for chaining. **Maps:** only `delete` mutates in place; `merge`, `filter`, `map_values` all return new maps.

```
// CORRECT — list mutation (modifies original)
items = [1, 2, 3]
items.push(4)          // items is now [1, 2, 3, 4], no reassignment needed
last = items.pop()     // last = 4, items is now [1, 2, 3]

// CORRECT — chaining mutating methods (returns self)
items.push(4).push(5)  // items is now [1, 2, 3, 4, 5]

// PITFALL — sort() mutates in place (unlike filter/map)
items = [3, 1, 2]
sorted = items.sort()  // items is now [1, 2, 3], sorted is same reference
// To preserve original, copy first:
original = [3, 1, 2]
copy = original.slice(0)
copy.sort()            // copy is [1, 2, 3], original still [3, 1, 2]

// CORRECT — non-mutating methods return NEW values (original unchanged)
items = [1, 2, 3, 4]
evens = items.filter(fn(x) => x % 2 == 0)  // evens = [2, 4], items unchanged

// CORRECT — string is immutable
s = "hello"
upper = s.upper()      // upper = "HELLO", s is still "hello"
```

### string (17 methods, all return new strings)

```
s = "Hello World"
s.len()              // 11
s.upper()            // "HELLO WORLD" (s unchanged)
s.lower()            // "hello world" (s unchanged)
s.trim()             // removes whitespace
s.trim_start()       // removes leading whitespace
s.trim_end()         // removes trailing whitespace
s.split(" ")         // ["Hello", "World"]
s.contains("World")  // true
s.starts_with("He")  // true
s.ends_with("ld")    // true
s.replace("World", "Codong")  // "Hello Codong" (s unchanged)
s.index_of("World")  // 6
s.slice(0, 5)        // "Hello"
s.repeat(2)          // "Hello WorldHello World"
"42".to_number()     // 42
"true".to_bool()     // true
"abc123".match("[0-9]+")  // ["123"]
```

### list (20 methods)

Mutating: `push`, `pop`, `shift`, `unshift`, `sort`, `reverse` — modify original, return self (or removed item for pop/shift).
Non-mutating: all others — return new values.

```
// Mutating methods (modify original)
l = [3, 1, 2]
l.push(4)            // l is now [3, 1, 2, 4], returns l
l.pop()              // returns 4, l is now [3, 1, 2]
l.shift()            // returns 3, l is now [1, 2]
l.unshift(0)         // l is now [0, 1, 2], returns l
l.sort()             // l is now [0, 1, 2] (sorted in place), returns l
l.reverse()          // l is now [2, 1, 0] (reversed in place), returns l

// Non-mutating methods (return new values, original unchanged)
l = [3, 1, 2]
l.len()              // 3
l.slice(0, 2)        // [3, 1] (new list, l unchanged)
l.map(fn(x) => x * 2)     // [6, 2, 4] (new list)
l.filter(fn(x) => x > 1)  // [3, 2] (new list)
l.reduce(fn(a, b) => a + b, 0)  // 6
l.find(fn(x) => x > 2)    // 3
l.find_index(fn(x) => x > 2)  // 0
l.contains(1)        // true
l.index_of(1)        // 1
l.flat()             // flattens nested lists (new list)
l.unique()           // deduplicates (new list)
l.join("-")          // "3-1-2" (returns string)
l.first()            // 3
l.last()             // 2
```

### map (10 methods)

Mutating: `delete` only — modifies original, returns self.
Non-mutating: all others — return new values.

```
m = {a: 1, b: 2, c: 3}

// Non-mutating
m.len()              // 3
m.keys()             // ["a", "b", "c"]
m.values()           // [1, 2, 3]
m.entries()          // [["a",1], ["b",2], ["c",3]]
m.has("a")           // true
m.get("x", 0)        // 0 (default)
m.map_values(fn(v) => v * 10)   // {a: 10, b: 20, c: 30} (new map)
m.filter(fn(v) => v > 1)       // {b: 2, c: 3} (new map)
m.merge({d: 4})      // {a: 1, b: 2, c: 3, d: 4} (new map, m unchanged)

// Mutating
m.delete("a")        // m is now {b: 2, c: 3}, returns m

// CORRECT — safe merge pattern (creates new map, originals untouched)
config = defaults.merge(user_config)   // defaults is NOT modified
```

Note: use `m.map_values(fn)` to transform values (not `m.map(fn)`, avoids confusion with type name).

---

## 12. Operator Precedence (highest to lowest)

| # | Operators | Description |
|---|-----------|-------------|
| 1 | `()` `[]` `.` `?` | grouping, index, member, error propagation (postfix) |
| 2 | `!` `-`(unary) | not, negate |
| 3 | `*` `/` `%` | multiply, divide, modulo |
| 4 | `+` `-` | add, subtract |
| 5 | `<` `>` `<=` `>=` | comparison |
| 6 | `==` `!=` | equality |
| 7 | `&&` | logical and |
| 8 | `\|\|` | logical or |
| 9 | `<-` | channel send/receive |
| 10 | `=` `+=` `-=` `*=` `/=` | assignment |

```
// WRONG — no === or !== or ternary
if x === 10 { }
result = x > 0 ? "yes" : "no"
```

---

## 13. Keywords (23 total)

```
fn       return   if       else     for      while    match
break    continue const    import   export   try      catch
go       select   interface type    null     true     false
in       _
```

`_` is a keyword: match wildcard and discard marker (`_ = side_effect()`).

**NOT keywords** (built-in modules/functions, can appear in expressions):

```
// These are NOT keywords — they are built-in modules or functions:
error      // built-in module: error.new(), error.wrap()
channel    // built-in function: channel(), channel(size: 10)
range      // built-in function: range(0, 10)
print      // built-in function: print("hello")
type_of    // built-in function: type_of(x)
to_string  // built-in function: to_string(42)
to_number  // built-in function: to_number("42")
to_bool    // built-in function: to_bool("true")

// CORRECT — error and channel used as expressions
err = error.new("E1001", "msg")   // error is a module, not keyword
ch = channel()                     // channel is a function, not keyword

// This is why they CANNOT be keywords:
// Keywords cannot appear as the left side of . access or be called as functions
// error.new() requires error to be an identifier, not a keyword
// channel() requires channel to be a callable, not a keyword
```

```
// WRONG — these are NOT Codong keywords either
var  let  class  struct  async  await  def  function  switch
case throw new   this    self   lambda yield require  bridge  use
```

---

## 14. Mandatory Code Style

```
// CORRECT — 4 space indent, snake_case, double quotes, same-line brace
fn calculate_total(items) {
    total = 0
    for item in items {
        total = total + item.price
    }
    return total
}

// WRONG — tabs
fn calculate_total(items) {
	total = 0
}

// WRONG — camelCase
fn calculateTotal(items) {
    myVar = 10
}

// WRONG — single quotes
name = 'hello'

// WRONG — brace on new line
fn add(a, b)
{
    return a + b
}

// CORRECT — trailing comma in multi-line
config = {
    host: "localhost",
    port: 8080,
    debug: true,
}

// WRONG — no trailing comma in multi-line
config = {
    host: "localhost",
    port: 8080,
    debug: true
}
```

Type names use `PascalCase`:

```
// CORRECT
type UserProfile = { name: string, age: number }

// WRONG
type user_profile = { name: string, age: number }
```

---

## 15. Go Bridge Extension Protocol

Go Bridge lets human architects wrap Go libraries for AI to call. AI only calls the registered function name.

```
// CORRECT — call registered bridge function
result = wechat_pay(amount: 99.9, order_id: "ORD001")

// WRONG — AI must NOT implement Go internals
import "github.com/wechatpay-apiv3/wechatpay-go"
```

### Registration in codong.toml

```toml
[bridge]
wechat_pay = { fn = "bridge.WechatPay", permissions = ["net:outbound"] }
pdf_render = { fn = "bridge.RenderPDF", permissions = ["fs:write:/tmp/codong-sandbox"] }
hash_md5   = { fn = "bridge.HashMD5", permissions = [] }
```

### Permission Rules

| Permission | Format | Meaning |
|------------|--------|---------|
| No I/O | `[]` | pure computation |
| Network | `["net:outbound"]` | HTTP requests allowed |
| Read files | `["fs:read:<path>"]` | read specific directory |
| Write files | `["fs:write:<path>"]` | write specific directory |

**Prohibited:** `os.Exit`, `syscall`, `os/exec`, `net.Listen`, host root filesystem access.

```
// CORRECT — handle bridge errors via map
result = pdf_render(html: content, output: "report.pdf")
if result.error {
    print("render failed: {result.error}")
}

// Bridge error convention:
// On failure:  return {error: "description", ...}
// On success:  error field must be null or absent
//              (accessing absent key returns null, which is falsy)
// WRONG — never use empty string for success ("" is truthy in Codong!)
//   return {error: ""}   // BAD: "" is truthy, triggers error branch
```

---

## Quick Reference: Unique Syntax Rules

| Scenario | Codong Way | NOT Allowed |
|----------|-----------|-------------|
| Variable | `x = 10` | `var x`, `let x`, `x := 10` |
| Function | `fn add(a,b) {}` | `function`, `def` |
| Default param | `fn(a, b = 10)` | `fn(a, b: 10)` in definition |
| Named arg | `f(a, key: val)` | `f(a, key=val)` at call site |
| String | `"hello {x}"` | `'hello'`, `` `hello ${x}` ``, `f"hello {x}"` |
| Multi-line | `"""..."""` | backtick blocks, heredoc |
| Output | `print(x)` | `log(x)`, `console.log(x)` |
| Map access | `m.key`, `m["key"]` | only `m.get()` |
| List access | `l[0]`, `l[-1]` | `l.at(0)` |
| HTTP server | `web.serve(port: 8080)` | `express()`, `http.ListenAndServe()` |
| DB connect | `db.connect(url)` | `new Pool()`, `createConnection()` |
| LLM call | `llm.ask(model, prompt)` | `openai.chat.completions.create()` |
| Error | `error.new(code, msg)` | `new Error()`, `raise Exception()` |
| Goroutine | `go fn() {}()` | `async`, `go func(){}()` |
| Channel send | `ch <- value` | `ch.send(value)` |
| Channel recv | `msg = <-ch` | `msg = <- ch`, `ch.receive()` |
| Arrow fn | `fn(x) => x * 2` | `x => x*2`, `lambda x: x*2` |
| Error prop | `expr?` | `expr?.field` (no optional chain) |
| Select arm | `<-ch { body }` | `<-ch => body` |
| Match arm | literals + `_` only | variable matching |
| Iterate map | `for k in m.keys()` | `for k in m` |
| Multiple returns | `return {a: 1, b: 2}` | `return a, b` |
| Length | `x.len()` | `len(x)` |

---

CODONG | codong.org | MIT License | AI Edition
