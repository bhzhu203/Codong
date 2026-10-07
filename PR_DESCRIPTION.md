# Pull Request: 编译产物体积最小化 + 图形化组件（draw / gui / webview / tui）

## PR 说明 / Description

**中文**

Codong 编译器此前把整个运行时（5920 行的单一字符串常量 `RuntimeSource`）无条件嵌入到每一个生成的 `main.go` 里。即使程序只用了一句 `print("Hello")`，生成的 Go 源码也携带全部 14 个内置模块的代码、import 和 keep-alive 引用，导致链接器无法做死代码消除，所有外部依赖（MySQL/Postgres/SQLite 驱动、go-redis、x/image、crypto 全家桶）都被链进最终二进制。

本 PR 把运行时拆分为按模块的独立分片，并在编译期通过 AST 分析按需 emit。未使用的模块在生成的 Go 源码里根本不存在，`go mod tidy` 随之剔除依赖，linker 的 DCE 将其彻底排除在二进制之外。

在按需分片架构之上，本 PR 新增四个图形化模块（同样按需 emit，不用零体积）：

- **draw** — fogleman/gg 2D 画布（纯静态）：`draw.canvas(w:, h:)` + `background/circle/rect/line/text/font` 对象方法，`save/png/svg` 导出
- **gui** — Fyne 桌面组件（cgo 动态链接）：`gui.init/show_label/show_button/run`，对象式 `gui.label/button/entry/entry_multi/add/add_hist/input_row` + 闭包回调 `widget.set_text/text/on_submit`，以及 `gui.copy`（系统剪贴板写入）。`gui.input_row(input, btn)` 生成并排输入行（输入框 + 等高按钮），并让窗口默认贴合内容高度成条状——`gui.add_hist` 首次写入后仅在底部出现 8px 可点击折叠线，点线展开到 `gui.init` 声明的高度（历史最新在上），再点收起还原条状；输入行右下角 grip 可拖拽缩放（宽 160 起、高可缩至一行，按钮高度跟随输入框）；界面字号 13.5，全部文字统一使用运行时从系统 Noto Sans CJK 集合字体（.ttc）抽取的单一 SC 字面——Latin 与中文同源同度量，彻底解决中文一大一小、间距不一的问题
- **webview** — webview_go 窗口壳（cgo，约 2MB，复用系统 WebKitGTK）：`webview.open(url, title, w, h)` / `webview.html(html, ...)`；页面脚本可调注入的 `__cResize(w, h)` 运行时调整原生窗口大小、`__cClipboardRead()`/`__cClipboardWrite(text)` 读写系统剪贴板（Linux 走 GTK CLIPBOARD，其他平台 no-op，页面可回退 navigator.clipboard）
- **tui** — Bubbletea 终端 UI（纯静态）：Elm 风格 `tui.run(update, view, init)`，按键/resize 消息以 map 传递（`msg.key`），update 返回 `{model: m, quit: true}` 退出
- **cod build -bundle** — 把动态链接程序（gui/webview）与其非 glibc、非 Mesa 的共享库闭包打包成单文件 AppImage

**English**

The Codong compiler previously embedded the entire runtime (a single 5,920-line string constant `RuntimeSource`) unconditionally into every generated `main.go`. Even a program as simple as `print("Hello")` carried the code, imports, and keep-alive references of all 14 built-in modules, which prevented the linker from doing dead-code elimination — every external dependency (MySQL/Postgres/SQLite drivers, go-redis, x/image, the full crypto stack) ended up in the final binary.

This PR splits the runtime into per-module fragments and emits them on demand based on compile-time AST analysis. Unused modules never appear in the generated Go source, so `go mod tidy` drops the dependencies and linker DCE excludes them from the binary entirely.

On top of the fragment architecture, this PR adds four graphics modules (same on-demand emission — zero cost when unused):

- **draw** — fogleman/gg 2D canvas (pure static): `draw.canvas(w:, h:)` plus `background/circle/rect/line/text/font` object methods, `save/png/svg` export
- **gui** — Fyne desktop widgets (cgo, dynamic): `gui.init/show_label/show_button/run`, object-style `gui.label/button/entry/entry_multi/add/add_hist/input_row` with closure callbacks (`widget.set_text/text/on_submit`), plus `gui.copy` (system clipboard write). `gui.input_row(input, btn)` builds a side-by-side input row (entry + same-height button) and makes the window hug its content as a horizontal strip — the first `gui.add_hist` adds only an 8px clickable collapse line at the bottom; clicking the line expands the window to the height declared in `gui.init` (history, newest first), clicking again collapses back to the strip. A grip at the input's bottom-right corner drag-resizes it (width from 160, height down to a single line, button height follows); UI text is sized 13.5 and rendered entirely in a single Noto Sans CJK SC face extracted at runtime from the system .ttc collection (Latin and CJK share identical metrics), eliminating mixed glyph sizes and uneven spacing
- **webview** — webview_go window shell (cgo, ~2MB, reuses system WebKitGTK): `webview.open(url, title, w, h)` / `webview.html(html, ...)`; page scripts can call the injected `__cResize(w, h)` binding to resize the native window at runtime, and `__cClipboardRead()`/`__cClipboardWrite(text)` bridge the system clipboard (GTK CLIPBOARD on Linux via cgo; no-op elsewhere so pages can fall back to navigator.clipboard)
- **tui** — Bubbletea terminal UI (pure static): Elm-style `tui.run(update, view, init)`; key/resize messages arrive as maps (`msg.key`); return `{model: m, quit: true}` from update to exit
- **cod build -bundle** — packs a dynamically linked program (gui/webview) and its non-glibc, non-Mesa shared-library closure into a single-file AppImage

## 体积收益 / Size Results

| 程序 / Program | 修改前 / Before | 修改后 / After | 节省 / Saved |
|---|---|---|---|
| `hello`（仅 `print`）| 10.43 MB | **1.36 MB** | **-87%** |
| 纯 web 服务器 / pure web server | ~10.4 MB | 5.32 MB | -49% |
| web + db + image（sqlite）| 12.7 MB | 11.0 MB | -13% |
| 纯 image 模块 / image-only | — | 2.12 MB | — |
| 纯 oauth 模块 / oauth-only | — | 2.63 MB | — |

> hello 二进制已贴近 Go 语言下限（纯 Go `fmt.Println` 程序为 1.38 MB）。
> The hello binary is already near the Go floor (a plain Go `fmt.Println` program is 1.38 MB).

### 图形模块体积矩阵 / Graphics module size matrix（新增 / new）

| 程序 / Program | 模块 / Modules | 链接 / Linking | 体积 / Size |
|---|---|---|---|
| hello 基线 / baseline | — | 纯静态 static | 1.72 MB（与基线逐字节一致 / byte-identical to base） |
| 周销售图表 / weekly chart | draw | 纯静态 static | 3.10 MB |
| 图表服务器 / chart server | draw + web | 纯静态 static | 7.25 MB |
| 桌面应用 / desktop app | gui（Fyne）| 动态 dynamic | 23.7 MB |
| HTML 窗口 / HTML window | webview | 动态 dynamic | 2.06 MB |
| 终端计数器 / terminal counter | tui | 纯静态 static | 2.89 MB |
| gui AppImage | gui + 20 libs | 单文件 single-file | 23.2 MB |
| webview AppImage | webview + 105 libs | 单文件 single-file | 113 MB |

不使用图形模块的程序体积零变化（hello 基线逐字节一致）。/ Programs not using the graphics modules are byte-identical in size to the baseline.

所有二进制保持 `CGO_ENABLED=0` 纯静态、`-ldflags "-s -w"` 原生编译，**未使用 UPX**；使用 gui/webview 的程序自动切换 `CGO_ENABLED=1` 动态链接（用户已接受桌面程序动态链接），并在缺失 gcc 时给出友好错误。`-bundle` 打包排除 glibc 核心与 Mesa/GL 驱动栈（目标机器必有），其余库全部内置。/ All non-graphics binaries remain pure-static via `CGO_ENABLED=0` with `-ldflags "-s -w"`, **no UPX**; gui/webview programs automatically switch to `CGO_ENABLED=1` dynamic linking (accepted trade-off for desktop apps) with a friendly error when gcc is missing. `-bundle` excludes the glibc core and Mesa/GL driver stack (guaranteed on graphical targets) and bundles everything else.

## 实现方式 / How It Works

**中文**

1. **运行时分片**：`runtime.go` 拆成 14 个文件（`runtime_core.go`、`runtime_web.go`、`runtime_db.go` 等），每个导出三样东西：Go 源码字符串常量（`Runtime{Name}`）、import 列表（`{Name}Imports`）、keep-alive 符号列表（`{Name}KeepAlive`）。

2. **三层模块识别**：
   - AST 预扫描（`scanUsedModules`）：递归遍历整棵语法树，识别 `xxx.yyy(...)` 形式中 `xxx` 为内置模块名的调用，记入 `usedModules`。覆盖 if/match/try 分支、循环体、闭包、字符串插值等所有嵌套位置。
   - codegen 二次埋点（`genMethodCall` / `genMemberAccess`）：代码生成阶段再次记录。这一层不是冗余——`import` 语句引入的文件是在 codegen 阶段才展开的，里面的模块引用只有这层能发现。
   - 事后组装（`emitRuntime`）：放在 codegen **结束之后**执行（body 先缓存再前置拼接），此时模块集合已完整，逐模块判断 emit 与否。

3. **模块分档**：
   - 永远 emit（always-on）：core、builtin、json、env、time、fs、errorextended —— 全部只依赖标准库，体积代价可忽略，且被核心调度代码交叉引用。
   - 按需 emit（条件式）：web、image、http、llm、db、redis、oauth —— 任何一个都会引入 MB 级外部依赖。

4. **跨模块引用桩（stub）**：core 的 `cCall` 调度直接引用 web/image 的符号（`cWebRoute`、`cImageObj` 等）。当 web/image 不被 emit 时，core 里的桩声明保证编译通过；当被 emit 时，通过 `__GIRGEN_STRIP_{WEB,IMAGE}_STUBS` 标记精确剥离桩，避免重复声明。

5. **DB 驱动细粒度拆分**：`db.connect("sqlite:///app.db")` 的 DSN 是字符串字面量时静态解析 scheme，只 emit 对应驱动（如 `_ "modernc.org/sqlite"`），单驱动场景再省约 880 KB；DSN 是变量/插值时保守回退到全链三个驱动，行为与修改前一致。

6. **图形模块（新增 / new）**：`draw`/`gui`/`webview`/`tui` 各为一个分片（`runtime_draw.go`、`runtime_gui.go`、`runtime_webview.go`、`runtime_tui.go`），遵循同一套 检测 → 条件 emit → stub 剥离 机制；`gui`/`webview` 额外通过包级 `LastGenerateNeedsCGO()` 通知构建器切换 CGO 与链接模式。gui 的事件循环无需 goroutine 包装——用户代码本就运行在 main goroutine，`gui.run()`/`webview.open()` 阻塞主线程正好满足 Fyne/GTK 的约束。**7. AppImage 打包（新增 / new）**：`cod build -bundle`（`engine/runner/bundle.go`）用 `ldd` 收集共享库闭包（排除 glibc 核心与 Mesa/GL），组装 AppDir（AppRun + desktop + 自动生成图标），调用 appimagetool（含 type2-runtime，首次自动下载到 `~/.codong/tools/`，`APPIMAGE_EXTRACT_AND_RUN=1` 免 FUSE 运行）。

**English**

1. **Runtime fragmentation**: `runtime.go` is split into 14 files (`runtime_core.go`, `runtime_web.go`, `runtime_db.go`, ...), each exporting three things: a Go source string constant (`Runtime{Name}`), an import list (`{Name}Imports`), and a keep-alive symbol list (`{Name}KeepAlive`).

2. **Three layers of module detection**:
   - AST pre-pass (`scanUsedModules`): recursively walks the whole syntax tree, identifying calls of the form `xxx.yyy(...)` where `xxx` is a built-in module name, recording them in `usedModules`. Covers all nesting positions — if/match/try branches, loop bodies, closures, string interpolations.
   - Codegen instrumentation (`genMethodCall` / `genMemberAccess`): records usage again during code generation. This layer is not redundant — files brought in by `import` statements are only expanded during codegen, so module references inside them are only visible here.
   - Deferred assembly (`emitRuntime`): runs *after* codegen completes (the body is buffered, then the runtime prefix is prepended), when the module set is complete, deciding per module whether to emit.

3. **Module tiers**:
   - Always-on: core, builtin, json, env, time, fs, errorextended — all stdlib-only, negligible size cost, and cross-referenced by the core dispatch code.
   - Conditional: web, image, http, llm, db, redis, oauth — each pulls in MB-scale external dependencies.

4. **Cross-module stubs**: core's `cCall` dispatch references web/image symbols directly (`cWebRoute`, `cImageObj`, etc.). When web/image are not emitted, stub declarations in core keep it compiling; when they are emitted, the stubs are precisely stripped via `__GIRGEN_STRIP_{WEB,IMAGE}_STUBS` markers to avoid duplicate declarations.

5. **Fine-grained DB driver splitting**: when the DSN of `db.connect("sqlite:///app.db")` is a plain string literal, the scheme is statically resolved and only the matching driver (e.g. `_ "modernc.org/sqlite"`) is emitted — saving another ~880 KB for single-driver programs; when the DSN is a variable or interpolation, it conservatively falls back to linking all three drivers, matching the pre-change behavior.

6. **Graphics modules (new)**: `draw`/`gui`/`webview`/`tui` are each one fragment (`runtime_draw.go`, `runtime_gui.go`, `runtime_webview.go`, `runtime_tui.go`) following the same detect → conditional-emit → stub-strip mechanism; `gui`/`webview` additionally notify the builder via package-level `LastGenerateNeedsCGO()` to switch CGO and linking mode. The gui event loop needs no goroutine wrapping — user code already runs on the main goroutine, and `gui.run()`/`webview.open()` blocking the main thread exactly satisfies Fyne/GTK constraints. **7. AppImage bundling (new)**: `cod build -bundle` (`engine/runner/bundle.go`) collects the shared-library closure via `ldd` (excluding glibc core and Mesa/GL), assembles an AppDir (AppRun + desktop entry + generated icon), and invokes appimagetool (with type2-runtime, both auto-downloaded to `~/.codong/tools/` on first use; runs without FUSE via `APPIMAGE_EXTRACT_AND_RUN=1`).

## 顺带修复的 Bug / Bugs Fixed Along the Way

这些 bug 部分是拆分引入、部分是原有问题，都被矩阵测试（7 个条件模块各自单独编译）暴露：/ These bugs were partly introduced by the split and partly pre-existing, all exposed by matrix testing (each of the 7 conditional modules compiled standalone):

| # | Bug | 修复 / Fix |
|---|-----|------|
| 1 | `oauth` 模块永不 emit：映射表里写的是 `"Oauth"`，模块名是 `"OAuth"` | 大小写对齐 |
| 2 | `import` 文件内使用模块编译失败：`emitRuntime` 在 codegen 之前执行，看不到 import 展开后的引用 | 移到 codegen 之后 |
| 3 | Http/Llm 单独使用报 `undefined: bytes`：用了 `bytes.NewReader` 却没声明 import（此前被 Image 的 import 掩盖） | 补齐 `"bytes"` |
| 4 | Redis 单独使用报 `undefined: cGenerateRandomHex`：函数定义在 OAuth 分片，却被 Redis 分布式锁引用 | 移到 always-on 的 core |
| 5 | `c.png().len()` 返回 null：`cCall` 对 `[]byte` 无方法分发（图形模块测试暴露） | core 增加 `[]byte` 的 `len` case |
| 6 | `web.serve(8091)` 编译失败：位置参数端口按 float64 传入 `cWebMakeServer(int)` | 生成 `int(toFloat(...))` 包裹 |
| 7 | `web.bytes(...)` 无对应响应分支：写 PNG 等二进制响应无法返回 | 响应写入器增加 `"bytes"` case + `web.bytes()` 生成器 |
| 8 | 用户变量与内置函数重名被静默吞掉：`input = ...` 声明用原名、引用却被映射成内置 `cInput`（剪贴板管理器 demo 暴露） | `genIdentifier` 先查 `g.declared`，用户声明优先（真正的遮蔽语义） |

## 测试 / Testing

- 完整示例套件：**55/56 通过**（`tests/run_examples.sh`）。唯一失败的 `51_web_hello.cod` 是预存在的语言 bug——`"/greet/{user}"` 中的 `{user}` 被字符串插值解析而非路由参数（正确写法是 `:user`），与本次改动无关。
- Full example suite: **55/56 passing** (`tests/run_examples.sh`). The single failure, `51_web_hello.cod`, is a pre-existing language bug — `{user}` in `"/greet/{user}"` is parsed as string interpolation instead of a route parameter (`:user` is the correct syntax) — unrelated to this change.
- 编译路径回归：56 个示例 `cod build` 编译 **54/56 通过**；2 个失败（`30_discard_return` 的 `_` 赋值、`51_web_hello`）在基线提交（50f04d6）上同样失败，均为预存在问题。
- Compiler-path regression: all 56 examples `cod build` — **54/56 pass**; the 2 failures (`30_discard_return`'s `_` assignment, `51_web_hello`) fail identically on the base commit (50f04d6) — both pre-existing.
- 图形模块端到端 / Graphics modules end-to-end:
  - `draw`：`test/draw_demo.cod` 生成 PNG（柱状图+圆点，视觉核验）+ SVG；`test/draw_web_server.cod` 通过 `web.bytes(c.png(), "image/png")` 以 HTTP 200 + `image/png` 返回 640×360 PNG（curl + `file` 核验）
  - `gui`：`test/gui_example/app.cod`（用户原 API 兼容）+ `test/gui_widget_demo.cod`（对象式 API）在 X11 实测——窗口渲染正确，xdotool 真实点击按钮后闭包回调把标签更新为「按钮被点击了！/ 点击成功！」
  - `webview`：`test/webview_demo.cod` 打开 560×380 HTML/CSS 窗口，2.06MB 二进制
  - `tui`：`test/tui_counter.cod` 在 xterm 实测——↑/↓ 增减计数、q 退出并打印 `final count: 1`（expect/sandbox PTY 下 bubbletea 的 epoll 输入不唤醒，属环境限制；真实终端正常）
  - `-bundle`：gui 程序打包 23.2MB AppImage（20 libs）、webview 程序打包 113MB AppImage（105 libs），两者 `APPIMAGE_EXTRACT_AND_RUN=1` 实测均能启动并渲染窗口
  - 剪贴板管理器双实现对比 / clipboard-manager comparison: `test/gui_clipboard.cod`（gui 版，23.3MB/31 libs）与 `test/webview_clipboard.cod`（webview 版，1.76MB/117 libs）在 Xvfb 虚拟显示器（xdotool 真实指针 + ffmpeg 截帧）全流程实测——初始窗口贴合成条状（gui 560×145 / webview 560×140，无下方空白）、Shift+Enter 提交复制、首条提交后底部仅出现折叠线且不自动展开、点线展开到全高（历史最新在上）、点历史条目再复制（状态栏反馈 + tkinter 独立进程读回剪贴板核对）、再点线收起还原条状；输入框 grip 拖拽缩放两端实测（gui 自绘 grip / webview JS grip + `__cResize` 窗口跟随）：拖大窗口跟随变高、拖小到一行时按钮同步变一行高、提交/展开/收起全回归通过；gui 文字统一单一 SC 字面（中文无一大一小、间距一致），状态栏字号降为 13.5；gui 新增 `gui.copy/entry_multi/add_hist/input_row` 与 `widget.on_submit`，webview 新增页面可调的 `__cResize` 与 `__cClipboardRead/__cClipboardWrite` 剪贴板绑定
- 模块矩阵：7 个条件模块（web/db/http/llm/redis/oauth/image）各自单独编译、组合编译全部通过，运行时行为验证正常（oauth 生成 state、db 连接 sqlite、http 请求返回 200 等）。
- Module matrix: each of the 7 conditional modules (web/db/http/llm/redis/oauth/image) compiles standalone and in combination; runtime behavior verified (oauth generates state, db connects to sqlite, http requests return 200, etc.).
- 端到端验证：用本编译器写了一个目录列出 web server（`web.get` + `fs.list` + 路由参数 + JSON/HTML 响应），编译为 5.65 MB 零外部依赖二进制，curl 实测全部路由正确。
- End-to-end: wrote a directory-listing web server with this compiler (`web.get` + `fs.list` + route params + JSON/HTML responses), compiled to a 5.65 MB zero-external-dependency binary, all routes verified with curl.

## 兼容性 / Compatibility

- 语言语义零变化：所有程序行为与修改前一致，仅二进制更小。
- 图形模块完全按需：不用 `draw`/`gui`/`webview`/`tui` 的程序，生成的 Go 源码与基线逐字节一致（hello 基线验证）。
- 新 CLI 参数仅 `cod build -bundle`（打包 AppImage）；gui/webview 程序自动切换 CGO 构建，无需用户配置；`test/gui_example/*.cod` 的既有 API（`gui.init/show_label/show_button/run`）保持原样可用。
- 动态 DSN 等无法静态分析的场景自动回退到原有行为（全量链接）。

- Zero language-semantics change: every program behaves identically, only smaller.
- Graphics modules are fully on-demand: programs not using `draw`/`gui`/`webview`/`tui` generate Go source byte-identical to the baseline (verified on the hello baseline).
- The only new CLI flag is `cod build -bundle` (AppImage packaging); gui/webview programs switch to CGO builds automatically with no user configuration; the existing `test/gui_example/*.cod` API (`gui.init/show_label/show_button/run`) works unchanged.
- Cases that cannot be statically analyzed (e.g. dynamic DSNs) automatically fall back to the previous behavior (link everything).

## 后续可做 / Follow-ups (Optional)

- `51_web_hello.cod` 的字符串插值 vs 路由参数冲突值得在语言层面澄清；`cod build` 对 `_ =` 丢弃赋值的支持（`30_discard_return`）同理。
- The string-interpolation vs route-parameter conflict in `51_web_hello.cod` deserves clarification at the language level; likewise `cod build` support for `_ =` discard assignment (`30_discard_return`).
- core 中 `cStrMethod` 的 `match` 分支引入 `regexp`；若程序未用 `match`，可进一步拆出 regex 分片。
- The `match` branch in core's `cStrMethod` pulls in `regexp`; if the program never uses `match`, a regex fragment could be split out for further savings.
- `-bundle` 的 glibc 下限取决于构建机：面向老发行版分发时应在旧 glibc 容器中构建（本 PR 仅实现打包机制）。
- The `-bundle` glibc floor follows the build machine: for targeting older distros, build inside an old-glibc container (this PR implements the packaging mechanism only).
- gui/webview 可继续扩充组件面（Fyne 表单/菜单、webview Bind 双向 JS 绑定）；tui 可补充 text/input 等组件。
- gui/webview can grow more surface (Fyne forms/menus, webview Bind for two-way JS calls); tui can add text/input components.
