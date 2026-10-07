# Codong GUI 示例程序

## 程序列表

### 1. 简单 GUI 示例 (app.cod / demo.cod)
使用 Codong 语言编写，展示基本的 GUI 功能。

```codong
gui.init("你好 Codong!", 400, 300)
gui.show_button("欢迎!", "点击我")
gui.run()
```

### 2. 倒计时程序 (countdown_timer)
功能完整的倒计时应用程序。

**功能：**
- ⏱️ 分钟/秒输入
- ▶️ 开始/暂停/重置
- 🔴 最后 10 秒红色警告
- 🎉 时间到动画窗口
- 🔔 桌面通知提醒
- ⚡ 快速预设（1/5/10/25 分钟）

**运行方式：**
```bash
# 直接运行编译好的程序（推荐，瞬间启动）
cd /home/bhzhu/source/Codong/test/gui_example/timer_go
./countdown_timer
```

**界面预览：**
```
┌─────────────────────────────────┐
│        ⏱️ 倒计时                │
├─────────────────────────────────┤
│                                 │
│        ⏱️ 01:30                 │
│                                 │
├─────────────────────────────────┤
│  分钟：[  1  ]    秒：[ 30  ]   │
│                                 │
│  [▶️ 开始]  [🔄 重置]            │
│                                 │
│  快速预设：                      │
│  [1 分][5 分][10 分][25 分]      │
└─────────────────────────────────┘
```

### 3. 终端 GUI (terminal_gui)
使用 Bubble Tea 的终端界面版本，无需图形环境。

```bash
./terminal_gui
```

## 文件结构

```
gui_example/
├── app.cod              # Codong 简单示例
├── demo.cod             # Codong 完整演示
├── timer.cod            # Codong 倒计时（计划中）
├── countdown_timer.go   # Go 源代码
├── terminal_gui.go      # 终端 GUI 源代码
├── timer_go/            # 倒计时编译目录
│   ├── countdown_timer  # 可执行文件
│   └── main.go
└── README.md
```

## 技术说明

### Codong GUI 模块

| 函数 | 说明 |
|------|------|
| `gui.init(title, w, h)` | 初始化窗口 |
| `gui.show_label(text)` | 显示标签 |
| `gui.show_button(label, btn)` | 显示按钮 |
| `gui.show_entry(label, ph)` | 显示输入框 |
| `gui.run()` | 运行主循环 |
| `gui.close()` | 关闭窗口 |

### 桌面通知支持

- **Linux**: 使用 `notify-send`
- **macOS**: 使用 `osascript`
- **Windows**: 使用 PowerShell

### 编译说明

```bash
# 首次编译（需要 2-3 分钟）
cd timer_go
go build -o countdown_timer .

# 之后瞬间启动
./countdown_timer
```

## 依赖

- Fyne v2.7.3+
- Go 1.22+
- CGO 和图形环境（X11/Wayland）
