#!/bin/sh
# Markdown 浏览器启动器（用于文件关联）
# 用法: mdview.sh [file.md ...]
DIR="$(cd "$(dirname "$0")" && pwd)"
CODONG="$DIR/../bin/codong"
[ -x "$CODONG" ] || CODONG="$(command -v codong)"
exec "$CODONG" run "$DIR/markdown_browser.cod" -- "$@"
