# Codong

**A programming language designed for AI.**

```
name = "Ada"
result = llm.ask(model: "gpt-4o", prompt: "Say hello to {name}")
print(result.text)
```

Most programming languages were designed for humans to write and machines to execute. Codong is designed for AI to write, humans to review, and machines to execute.

---

## Why Codong

When an AI agent writes code, three things slow it down:

**1. Choosing between equivalent options**
Python has 5+ ways to make an HTTP request. Every choice burns tokens and creates unpredictable output. Codong has one way.

**2. Unparseable error messages**
Stack traces are designed for humans. AI has to spend hundreds of tokens parsing them before it can fix anything. Every Codong error is structured JSON with a `fix` field.

**3. Package selection**
Before writing a single line of business logic, an AI has to pick an HTTP library, a database driver, a JSON parser. Codong ships 8 built-in modules that cover 90% of AI workloads — no package manager required.

---

## Quick Start

Download the binary for your platform and run your first `.cod` file:

```bash
# Hello World
echo 'print("Hello, Codong!")' > hello.cod
codong eval hello.cod
```

```bash
# A simple web API (coming in Stage 2)
codong run server.cod
```

---

## The Language

Codong is deliberately small. 23 keywords. 6 primitive types. One way to do each thing.

### Variables

```
name = "Ada"
age = 30
active = true
nothing = null
const MAX_RETRIES = 3
```

No `var`, `let`, or `:=`. Assignment is `=`, always.

### Functions

```
fn greet(name, greeting = "Hello") {
    return "{greeting}, {name}!"
}

print(greet("Ada"))                    // Hello, Ada!
print(greet("Bob", greeting: "Hi"))    // Hi, Bob!

double = fn(x) => x * 2               // arrow function
```

### Collections

```
items = [1, 2, 3, 4, 5]
doubled = items.map(fn(x) => x * 2)
evens = items.filter(fn(x) => x % 2 == 0)
total = items.reduce(fn(acc, x) => acc + x, 0)

user = {name: "Ada", age: 30}
user.email = "ada@example.com"
print(user.get("phone", "N/A"))        // N/A
```

### Control Flow

```
// if / else if / else
if score >= 90 {
    print("A")
} else if score >= 80 {
    print("B")
} else {
    print("C")
}

// for...in (lists only — use .keys()/.entries() for maps)
for item in items {
    print(item)
}

// match
match status {
    200 => print("ok")
    404 => print("not found")
    _   => print("error: {status}")
}
```

### Error Handling

Every error is structured JSON with `code`, `message`, `fix`, and `retry` fields.

```
fn load_user(id) {
    if id <= 0 {
        return error.new("E_INVALID_ID", "id must be positive",
            fix: "pass a positive integer for id")
    }
    return db.find("users", {id: id})
}

try {
    user = load_user(-1)?
} catch err {
    print(err.code)      // E_INVALID_ID
    print(err.fix)       // id must be positive
}
```

The `?` operator propagates errors up the call stack — no nested `if err != nil` chains.

AI agents can parse this directly. No stack trace interpretation required.

### Compact Error Format

Switch to compact format to save ~39% tokens in AI pipelines:

```
error.set_format("compact")
// output: err_code:E_INVALID_ID|src:syntax|fix:pass a positive integer|retry:false
```

---

## Built-in Modules

Eight modules ship with Codong. No installation, no version conflicts, no choices.

| Module | What it does |
|--------|-------------|
| `web` | HTTP server, routing, middleware, WebSocket |
| `db` | PostgreSQL, MySQL, MongoDB, Redis, SQLite, vector DBs |
| `llm` | GPT, Claude, Gemini — unified interface, structured output |
| `agent` | spawn, pipeline, wait_all, race — multi-agent orchestration |
| `cloud` | AWS, GCP, Azure — unified storage, functions, secrets |
| `queue` | Kafka, RabbitMQ, SQS, NATS |
| `cron` | Scheduled jobs, timezone support |
| `error` | Structured error creation, wrapping, formatting |

```
// A complete AI API in ~10 lines
server = web.serve(port: 8080)
conn = db.connect("postgres://localhost/mydb")

server.post("/ask", fn(req) {
    question = req.body.question
    context = db.find("docs", {relevant: true})?
    answer = llm.ask(model: "gpt-4o",
        prompt: "Answer using context: {context}\n\nQuestion: {question}",
        format: "json")?
    return {status: 200, body: answer}
})
```

> **Note:** The `web`, `db`, `llm`, `agent`, `cloud`, `queue`, and `cron` modules are coming in Stage 2 (in progress). The core language and `error` module are fully available today.

---

## Go Bridge

Need a library that isn't in the 8 built-in modules? Go Bridge lets human architects wrap any Go package for AI consumption.

```toml
# codong.toml
[bridge]
pdf_render = { fn = "bridge.RenderPDF", permissions = ["fs:write:/tmp/output"] }
wechat_pay = { fn = "bridge.WechatPay", permissions = ["net:outbound"] }
```

```
# .cod file — AI calls the registered name directly
result = pdf_render(html: content, output: "report.pdf")
if result.error {
    print("render failed: {result.error}")
}
```

AI only sees function names and return values. Permissions are declared explicitly. No `os.Exit`, no `syscall`, no host filesystem access.

---

## Two Execution Modes

```bash
codong eval script.cod   # AST interpreter — instant startup, no stdlib
codong run app.cod       # Go IR → go run — full stdlib, production behavior
codong build app.cod     # Compiles to a single static binary
```

`eval` is for quick scripts, REPL, and the browser Playground. `run` and `build` use the Go IR path and support all 8 modules.

---

## String Interpolation

```
name = "Ada"
print("Hello, {name}!")                      // variable
print("Total: {items.len()} items")          // method call
print("Sum: {a + b}")                        // expression
print("{user.name} joined on {user.date}")   // member access
```

Any expression is valid inside `{}`. No backticks, no `f"..."`, no `${}`.

---

## Type Annotations (for agent.tool)

Type annotations are optional everywhere — except when you want `agent.tool` to auto-generate JSON Schema for LLM function calling:

```
fn search(query: string, limit: number) {
    return db.find("docs", {q: query}, limit: limit)
}

// agent.tool reads the type annotations and generates the schema automatically
agent.tool("search", search, "Search the knowledge base")
```

No hand-written JSON Schema. The engine reads the annotations.

---

## Roadmap

| Stage | Status | Deliverable |
|-------|--------|-------------|
| 0 | ✅ Done | SPEC.md — AI can write Codong without a compiler |
| 1 | ✅ Done | `codong eval` — core language, error module, CLI |
| 2 | 🔨 In progress | `web`, `db`, `http`, `llm` modules |
| 3 | Planned | `agent`, `cloud`, `queue`, `cron` modules |
| 4 | Planned | `codong build` — single static binary |
| 5 | Planned | 50 examples + full documentation |
| 6 | Planned | codong.org + browser Playground |
| 7 | Planned | VS Code extension + MCP Server for Claude Desktop |
| 8 | Planned | Package registry + `codong.lock` |

---

## Language Reference

For the complete language specification, see [`SPEC.md`](./SPEC.md).

For the AI-optimized version with correct/incorrect examples for every rule, see [`SPEC_FOR_AI.md`](./SPEC_FOR_AI.md). Inject this into any LLM system prompt to enable correct Codong code generation without installing anything.

For the full design rationale, architecture decisions, and project vision, see [`WHITEPAPER.md`](./WHITEPAPER.md).

---

## Contributing

Codong is MIT licensed and open to contributions.

- Found a bug? Open an issue.
- Want to add a standard library module? Read [`SPEC.md §15`](./SPEC.md) on Go Bridge first.
- Want to help with the Go IR generator? That's the highest-leverage contribution right now.

---

## License

MIT — see [LICENSE](./LICENSE)

---

*Codong — codong.org*
