# CODONG — 专为 AI 设计的编程语言

**White Paper**

> *AI 需要的不是更灵活的语法，而是更确定的边界和自描述的错误响应。*

codong.org · .cod 文件格式 · MIT 开源 · Go 运行时

---

## 一、核心定位

Codong 是世界上第一个把 AI 视为第一用户（AI as First-class Citizen）来设计的编程语言。这不是对现有语言的优化，而是对编程语言底层架构的系统性重构。

| # | 摩擦点 | 现有语言的问题 | Codong 的解法 |
|---|--------|---------------|--------------|
| 1 | 语法歧义制造幻觉 | Python 有 5 种以上等价写法，AI 选择时产生幻觉 | 在 I/O、网络、并发层提供唯一的标准 API 路径 |
| 2 | 错误信息无法被机器解析 | 报错是供人类读的 Stack Trace，AI 解析浪费大量 Token | 所有错误输出结构化 JSON，含 fix 和 retry 字段 |
| 3 | 包选择消耗上下文 | AI 需要先选库、解决版本冲突，才能写业务逻辑 | 8 大模块内置零选择成本；Go Bridge 协议覆盖长尾需求 |

## 二、独立工程审计验证（三轮完整审计）

以下设计已通过独立工程审计，从编译器原理、分布式系统和大模型工程三个维度交叉验证。

| # | 设计名称 | 审计评语 |
|---|---------|---------|
| 1 | Agent 编排下沉为语言一等公民 | agent.spawn/pipeline/wait_all 直接做到标准库里，直接干掉 LangChain 和 AutoGen 这种极其臃肿的框架。用语言级原语调度 AI 模型，让多 Agent 协作像写普通多线程代码一样优雅。 |
| 2 | 结构化输出原语（llm.ask + schema） | llm.ask 中原生内置 format:json 和 schema:{}，直接返回 map。精准踩中了目前 AI 开发最大的痛点——数据清洗。 |
| 3 | 消除选择困难的终极体现 | Web/DB/Cloud 三个模块接口设计极度克制。DB 模块强制用统一 filter 语法打平 SQL 和 NoSQL，换云平台只改 cloud.connect。 |
| 4 | JSON 结构化错误 + compact 格式 | 直接给大模型喂结构化指令。compact 格式再额外节省 39% Token。在 API 计费时代是巨大的商业优势。 |
| 5 | SPEC.md 驱动的开发顺序反转 | 大模型不需要编译器就能学会这门语言。项目第一天就能做市场验证。 |
| 6 | 原生 MCP Server 支持 | Claude Desktop 可以自己写 Codong 代码、自己编译、自己本地运行测试。Codong 成为第一个拥有原生 AI 物理执行手脚的编程语言。 |
| 7 | 50 个示例作为 LLM 训练语料 | Codong 的示例是给下一代大模型准备极优质的 Few-shot Prompt 库。 |
| 8 | WASM 浏览器 Playground | 将 Go-IR 引擎编译为 WebAssembly，消除后端代码执行沙箱成本。 |
| 9 | 不可变版本 + 自动化安全门禁 | 发布后版本不可修改，切断了类似 npm left-pad 事件的隐患。 |
| 10 | OpenClaw 社区自动化闭环 | Issue 追踪、成员引导、每日摘要全部由 Agent 处理。 |

## 三、五大核心设计

### 设计一：JSON 结构化错误系统（含 compact 紧凑格式）

把 AI 从盲目重试变成带参修复，直接省掉一半以上的 Debug Token 消耗。compact 格式再额外节省 39%。

```python
# Python 报错（AI 需要解析非结构化文本）
Traceback (most recent call last):
  File app.py, line 42
    result = db.query(sql)
sqlite3.OperationalError: table users doesn't exist
```

Codong 的每一个错误都是结构化 JSON：

```json
{
  "error":   "db.find",
  "code":    "E2001_NOT_FOUND",
  "message": "table 'users' not found",
  "fix":     "run db.migrate() to create the table",
  "retry":   false,
  "docs":    "codong.org/errors#E2001_NOT_FOUND"
}
```

```
// compact 格式（节省 39% Token）：
// err_code:E2001_NOT_FOUND | src:db.find | fix:run db.migrate() | retry:false
```

### 设计二：SPEC.md 驱动的开发顺序反转

先写一份 <2000 字的 SPEC.md 语法规范，大模型不需要编译器就能学会这门语言。SPEC.md 写完第一天即可对外演示，还能给下一代 LLM 准备训练数据。

### 设计三：8 大内置模块 + Go Bridge 扩展协议

8 大模块覆盖 AI 编程 90% 场景，零包管理成本。Go Bridge 协议作为逃生舱，人类架构师封装任意 Go 库，AI 无需关心内部实现直接调用。

| 模块 | 核心能力 | 对 AI 的价值 |
|------|---------|-------------|
| web | HTTP serve · 路由 · 中间件 · WebSocket | 三行代码起服务，无需选框架 |
| db | SQL/NoSQL/向量（单表 CRUD）；复杂查询用 db.query | 统一 filter 语法；多表 Join 明确使用原生 SQL |
| llm | GPT/Claude/Gemini 统一接口；format:json 直接返回 map | 换模型只改 model 参数；结构化输出无需 JSON.parse |
| agent | spawn/wait_all/pipeline/race · 工具自动 Schema 推断 | Agent 编排是语言一等公民；彻底干掉 LangChain |
| cloud | AWS/GCP/Azure 统一接口 | 换云平台只改 connect 配置 |
| queue | Kafka/RabbitMQ/SQS 统一接口 | 消息队列无需学各家 SDK |
| cron | 声明式定时任务 | 无需 crontab，代码即配置 |
| error | 结构化 JSON + compact 错误系统 | AI 可直接解析和修复 |
| Go Bridge | 任意 Go 生态库（人类封装、AI 调用） | 生态无限延伸；Bridge 函数须声明权限 |

### 设计四：AI DevOps 物理隔离架构

Claude Code 负责所有精确编码工作，OpenClaw 部署在隔离服务器只做监控和自动化，绝不直接修改代码仓库。

| Claude Code（主力编码） | OpenClaw（自动化监控） |
|------------------------|---------------------|
| 所有代码编写（编译器/标准库/CLI/插件） | CI/CD 监控（测试失败报警） |
| 单元测试和集成测试 | Issue 追踪、成员引导、日常运营 |
| → 有 GitHub 写权限 | → 无 GitHub 写权限（只读） |

### 设计五：原生 MCP Server + @namespace 包生态

官方直接提供 MCP Server，Claude Desktop 可自己写 Codong、自己运行。@namespace 作用域机制防止包名抢注，codong.lock 确保构建 100% 可复现。

## 四、Token 经济学

| Token 消耗场景 | Python/JS | Codong | 说明 |
|---------------|-----------|--------|------|
| 选择合适的 HTTP 框架 | ~300 token | 0 | 内置 web 模块 |
| 选择数据库 ORM 并配置 | ~400 token | 0 | 内置 db 模块 |
| 解析并理解报错信息 | ~500 token | ~50 token | 结构化 JSON vs 非结构化文本 |
| 解决包版本冲突 | ~800 token | 0 | 无包管理器 |
| 实际业务逻辑代码 | ~800 token | ~800 token | 相同 |
| **合计** | **~2800 token** | **~850 token** | **节省约 70%+** |

## 五、技术架构

### 5.0 与 Go 的关系：站在巨人肩上

Codong 是一门全新的编程语言，但它不是从零造轮子。Codong 的编译器将 .cod 源码翻译为等价的 Go 代码（Go IR），再由 Go 工具链完成执行或编译。

| Codong 负责的层 | Go 负责的层 |
|----------------|-----------|
| 全新语法设计（面向 AI） | 内存管理 · 垃圾回收 |
| 高约束性领域 API | goroutine 并发调度 |
| 结构化 JSON 错误系统 | 跨平台编译（linux/mac/windows） |
| 8 大内置模块抽象层 | 十年工业验证的运行时 |
| Go Bridge 扩展协议 | 数十万 Go 生态库 |
| AI 专属语言特性 | 高性能执行（无解释器损耗） |

这与 TypeScript 编译到 JavaScript、Kotlin 运行在 JVM 上的模式相同。Codong 的创新在语言层——为 AI 这个新型用户设计了更确定的语法边界和自描述的错误系统；执行层则完全依托 Go 这个经过十年工业验证的运行时，性能与手写 Go 程序完全一致。

**代价与补偿**：运行 Codong 程序的环境需要安装 Go。`codong run` 的冷启动时间（0.3s~2s）包含了 Go 的编译过程。对于需要真正瞬时响应的 REPL 场景，`codong eval` 提供了绕过 Go 编译的轻量解释器，实现真正亚秒级启动。`codong build` 输出的二进制文件则无需 Go 环境即可运行。

### 5.1 双模式运行时

**codong run（动态执行）**
- 路径：.cod → AST → Go IR → go run
- 冷启动：亚秒级（视复杂度 0.3s~2s）
- 适合：AI Agent 动态执行、开发调试

**codong eval（REPL 模式）**
- 路径：.cod → AST → 轻量解释器
- 冷启动：真正亚秒（无 Go 编译步骤）
- 适合：简单脚本/REPL/极速验证

**codong build（生产部署）**
- 路径：.cod → AST → Go IR → go build
- 产物：单一静态链接二进制，无外部依赖
- 基础服务（web+db）：< 20MB；全量 8 大标准库：< 50MB
- 平台：Linux / macOS / Windows × amd64/arm64

### 5.2 高约束性领域 API 设计

| 约束层次 | 覆盖范围 | 效果 |
|---------|---------|------|
| 强制唯一写法 | HTTP 服务启动、数据库连接、LLM 调用、错误抛出 | AI 不会产生多种等价写法，100% 可预期 |
| 推荐标准路径 | 单表 CRUD、JSON 处理、环境变量 | SPEC.md 定义标准路径，AI 优先选择 |
| 完全自由 | 业务逻辑、算法、多表 Join（db.query） | 用户完全自由，Codong 不干预 |
| Go Bridge 扩展 | 任意 Go 生态库（须声明 I/O 权限） | 生态无限延伸，AI 只调用注册函数名 |

## 六、生态兼容性

| 类别 | 支持 |
|------|------|
| AI 模型 | GPT-4o · Claude 3.5 · Gemini 1.5 Pro · Llama 3 · 任何 OpenAI 兼容 API |
| 数据库 | PostgreSQL · MySQL · MongoDB · Redis · SQLite · Pinecone · Qdrant · Supabase |
| 云平台 | AWS · GCP · Azure · Cloudflare R2 · Vercel |
| 消息队列 | Kafka · RabbitMQ · AWS SQS · NATS |
| 容器/编排 | Docker · Kubernetes · Helm · Terraform |
| Go Bridge | 任意 Go 生态库，通过 Go Bridge 协议封装后供 AI 调用 |
| 包生态 | @namespace/pkg 作用域机制；codong.lock 确保构建可复现 |

## 七、AI 接入方式

**方式一：SPEC.md 注入（Day 1 可用）**
将 SPEC.md（<2000字）注入任何 LLM 的 System Prompt，LLM 立即能写出正确的 Codong 代码，无需任何安装。

**方式二：官方 MCP Server（Claude Desktop 原生支持）**
官方直接提供 MCP Server。Claude Desktop 可自己写 Codong 代码、自己在本地编译运行。Codong 是第一个拥有原生 AI 物理执行手脚的编程语言。

**方式三：OpenAI Function Calling**
将 Codong 执行器注册为 GPT 的 Function，GPT 对话中直接编写并运行 Codong 代码。

## 八、里程碑

| # | 里程碑 | 可对外展示内容 | 验收标准 |
|---|-------|-------------|---------|
| M0 | SPEC.md 发布 | AI 无需编译器即写 Codong 代码 | 3 个 LLM 独立生成结果语法一致 |
| M1 | Go-IR 引擎 + codong eval | AI 动态执行；eval 命令真正亚秒级 REPL | 50 个示例全部通过 |
| M2 | 核心标准库完成 | 完整增删改查 Web 应用 + LLM 调用 | 50 行代码完成带 DB 的 AI API |
| M3 | 全部标准库完成 | multi-agent 并行工作流 | 3 Agent 并行聚合结果 |
| M4 | 二进制编译完成 | 可分发单一可执行文件 | 基础版 < 20MB，全量 < 50MB |
| M5 | 文档 + 示例完整 | 15 分钟完成第一个项目 | 新用户测试通过 |
| M6 | 官网 + Playground | WASM Playground | codong.org 上线 |
| M7 | IDE + MCP | VS Code 支持 + Claude Desktop 原生执行 | 插件发布 Marketplace |
| M8 | 生态建立 | @namespace 包注册表 | GitHub 100+ star |

---

*CODONG · codong.org · MIT 开源 · White Paper*
