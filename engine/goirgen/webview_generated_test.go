package goirgen

import (
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/codong-lang/codong/engine/lexer"
	"github.com/codong-lang/codong/engine/parser"
)

// TestWebviewTimerGeneratedSource verifies that the rewritten timer.cod
// (webview module) generates valid Go source with properly unescaped HTML.
func TestWebviewTimerGeneratedSource(t *testing.T) {
	src, err := os.ReadFile("../../test/gui_example/timer.cod")
	if err != nil {
		t.Skip("timer.cod not found: " + err.Error())
	}
	l := lexer.New(string(src))
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse errors: %v", p.Errors())
	}
	goSrc := Generate(prog, "../../test/gui_example")
	if strings.Contains(goSrc, `\{`) || strings.Contains(goSrc, `\}`) {
		t.Fatal("escaped braces leaked into generated source")
	}
	for _, want := range []string{"cWebviewHtml(", "body { margin: 0", "@keyframes pulse", "function toggle()", "webview.New(false)", "wv_file_dialog"} {
		if !strings.Contains(goSrc, want) {
			t.Fatalf("missing in generated source: %q", want)
		}
	}
}

// TestWebviewClipboardGeneratedSource verifies that .cod files using the
// webview module get the native clipboard bridge bindings in the generated
// Go source (window.__cClipboardRead / __cClipboardWrite on the page side).
func TestWebviewClipboardGeneratedSource(t *testing.T) {
	src, err := os.ReadFile("../../test/markdown_browser.cod")
	if err != nil {
		t.Skip("markdown_browser.cod not found: " + err.Error())
	}
	l := lexer.New(string(src))
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse errors: %v", p.Errors())
	}
	goSrc := Generate(prog, "../../test")
	wants := []string{`wv.Bind("__cClipboardRead"`, `wv.Bind("__cClipboardWrite"`}
	if runtime.GOOS == "linux" {
		wants = append(wants, "wv_clipboard_read", "wv_clipboard_write", "gtk_clipboard_wait_for_text")
	}
	for _, want := range wants {
		if !strings.Contains(goSrc, want) {
			t.Fatalf("missing in generated source: %q", want)
		}
	}
}
