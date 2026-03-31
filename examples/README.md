# Codong Examples

50 real-world programs demonstrating every Codong module and language feature.
All examples are self-contained `.cod` files. Run any example with:

```bash
codong run examples/01-web/01-hello-api.cod
```

---

## 01 · Web Server (`web` module)

| # | File | Description |
|---|------|-------------|
| 01 | [01-hello-api.cod](01-web/01-hello-api.cod) | Hello World REST API with dynamic routes |
| 02 | [02-rest-crud-api.cod](01-web/02-rest-crud-api.cod) | Full CRUD API with SQLite (Users) |
| 03 | [03-file-upload.cod](01-web/03-file-upload.cod) | File upload service with auto-thumbnail |
| 04 | [04-sse-live-feed.cod](01-web/04-sse-live-feed.cod) | Server-Sent Events real-time feed |
| 05 | [05-rate-limited-api.cod](01-web/05-rate-limited-api.cod) | 60 req/min rate limiting with Redis |
| 06 | [06-jwt-auth-api.cod](01-web/06-jwt-auth-api.cod) | JWT login flow with protected routes |
| 07 | [07-webhook-receiver.cod](01-web/07-webhook-receiver.cod) | GitHub webhook receiver with daily logs |
| 08 | [08-static-site-server.cod](01-web/08-static-site-server.cod) | Static files + API on same port |
| 09 | [09-cors-middleware.cod](01-web/09-cors-middleware.cod) | CORS + request logger + API key middleware |
| 10 | [10-realtime-dashboard.cod](01-web/10-realtime-dashboard.cod) | Live metrics dashboard via SSE + Redis |

## 02 · Database (`db` module)

| # | File | Description |
|---|------|-------------|
| 11 | [11-blog-system.cod](02-database/11-blog-system.cod) | Blog posts with tags, drafts, seeding |
| 12 | [12-analytics-aggregation.cod](02-database/12-analytics-aggregation.cod) | sum/avg/min/max aggregations on orders |
| 13 | [13-migration-system.cod](02-database/13-migration-system.cod) | Tracked database migrations |
| 14 | [14-batch-operations.cod](02-database/14-batch-operations.cod) | Bulk insert 1000 rows with batch_insert |

## 03 · LLM / AI (`llm` module)

| # | File | Description |
|---|------|-------------|
| 15 | [15-simple-chat.cod](03-llm/15-simple-chat.cod) | Single-turn Q&A with Claude |
| 16 | [16-document-summarizer.cod](03-llm/16-document-summarizer.cod) | Batch document summarization to JSON |
| 17 | [17-code-reviewer.cod](03-llm/17-code-reviewer.cod) | AI code review API with structured output |
| 18 | [18-sentiment-analysis.cod](03-llm/18-sentiment-analysis.cod) | Batch sentiment classification + DB storage |
| 19 | [19-rag-document-qa.cod](03-llm/19-rag-document-qa.cod) | RAG: upload docs, ask questions |
| 20 | [20-multi-provider-compare.cod](03-llm/20-multi-provider-compare.cod) | Same prompt, Claude vs OpenAI, compare |

## 04 · File System (`fs` module)

| # | File | Description |
|---|------|-------------|
| 21 | [21-file-organizer.cod](04-filesystem/21-file-organizer.cod) | Auto-sort files by extension into folders |
| 22 | [22-log-parser.cod](04-filesystem/22-log-parser.cod) | Access log parser: errors, IPs, slow reqs |
| 23 | [23-config-manager.cod](04-filesystem/23-config-manager.cod) | Layered config: defaults → file → env vars |
| 24 | [24-backup-tool.cod](04-filesystem/24-backup-tool.cod) | Timestamped backup with retention policy |

## 05 · HTTP Client (`http` module)

| # | File | Description |
|---|------|-------------|
| 25 | [25-weather-api-client.cod](05-http-client/25-weather-api-client.cod) | Live weather + 3-day forecast (free API) |
| 26 | [26-github-api-client.cod](05-http-client/26-github-api-client.cod) | GitHub repo info, releases, contributors |
| 27 | [27-health-checker.cod](05-http-client/27-health-checker.cod) | Monitor multiple endpoints, log to file |
| 28 | [28-http-retry-client.cod](05-http-client/28-http-retry-client.cod) | All HTTP methods with retry & error handling |

## 06 · Redis (`redis` module)

| # | File | Description |
|---|------|-------------|
| 29 | [29-session-manager.cod](06-redis/29-session-manager.cod) | Create/read/refresh/destroy sessions |
| 30 | [30-leaderboard.cod](06-redis/30-leaderboard.cod) | Real-time leaderboard with sorted sets |
| 31 | [31-cache-aside.cod](06-redis/31-cache-aside.cod) | Cache-aside pattern with singleflight |
| 32 | [32-distributed-lock.cod](06-redis/32-distributed-lock.cod) | Distributed lock preventing double-processing |

## 07 · Image Processing (`image` module)

| # | File | Description |
|---|------|-------------|
| 33 | [33-image-resizer-service.cod](07-image/33-image-resizer-service.cod) | API: upload → resize to 4 standard sizes |
| 34 | [34-watermark-batch.cod](07-image/34-watermark-batch.cod) | Batch watermark all images in a directory |
| 35 | [35-image-pipeline.cod](07-image/35-image-pipeline.cod) | Resize → sharpen → watermark → optimize |

## 08 · OAuth & Auth (`oauth` module)

| # | File | Description |
|---|------|-------------|
| 36 | [36-github-oauth-login.cod](08-oauth/36-github-oauth-login.cod) | Complete GitHub SSO login flow |
| 37 | [37-rbac-permission-system.cod](08-oauth/37-rbac-permission-system.cod) | Role-based access control with permission matrix |

## 09 · Algorithms

| # | File | Description |
|---|------|-------------|
| 38 | [38-fibonacci-memoized.cod](09-algorithms/38-fibonacci-memoized.cod) | Naive vs memoized vs iterative Fibonacci |
| 39 | [39-sorting-algorithms.cod](09-algorithms/39-sorting-algorithms.cod) | Bubble sort, merge sort, quicksort |
| 40 | [40-graph-traversal.cod](09-algorithms/40-graph-traversal.cod) | BFS + DFS path finding on adjacency list |
| 41 | [41-binary-search-tree.cod](09-algorithms/41-binary-search-tree.cod) | BST insert, search, in-order traversal |
| 42 | [42-functional-patterns.cod](09-algorithms/42-functional-patterns.cod) | compose, pipe, curry, partial, group-by |

## 10 · Data Processing

| # | File | Description |
|---|------|-------------|
| 43 | [43-csv-processor.cod](10-data-processing/43-csv-processor.cod) | Parse CSV, aggregate, export filtered JSON |
| 44 | [44-json-transformer.cod](10-data-processing/44-json-transformer.cod) | Reshape nested API response, flatten, merge |
| 45 | [45-report-generator.cod](10-data-processing/45-report-generator.cod) | Sales report to JSON + HTML from SQLite |
| 46 | [46-data-validator.cod](10-data-processing/46-data-validator.cod) | Schema validation with detailed errors |
| 47 | [47-event-system.cod](10-data-processing/47-event-system.cod) | In-process pub/sub event bus with closures |
| 48 | [48-state-machine.cod](10-data-processing/48-state-machine.cod) | Finite state machine: order lifecycle |
| 49 | [49-pipeline-etl.cod](10-data-processing/49-pipeline-etl.cod) | ETL: API → transform → SQLite with dedup |
| 50 | [50-full-stack-app.cod](10-data-processing/50-full-stack-app.cod) | **Complete app**: Todo API + auth + Redis cache + DB |

---

## Requirements

- Codong v0.1.3+: `curl -fsSL https://codong.org/install.sh | sh`
- Examples 05, 15–20: require internet access
- Examples 29–32: require Redis (`redis-server`)
- Examples 15–20: require `ANTHROPIC_API_KEY` or `OPENAI_API_KEY`
- Example 36: requires `GITHUB_CLIENT_ID` and `GITHUB_CLIENT_SECRET`
