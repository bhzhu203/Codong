package goirgen

import (
	"os"
	"strings"
	"testing"

	"github.com/codong-lang/codong/engine/lexer"
	"github.com/codong-lang/codong/engine/parser"
)

// TestMygoGeneratedSource verifies that test/mygo_demo.cod emits the mygo
// runtime fragment on demand, with the wrapper calls and imports the module
// needs, and that the generated go.mod version requirement is raised.
func TestMygoGeneratedSource(t *testing.T) {
	src, err := os.ReadFile("../../test/mygo_demo.cod")
	if err != nil {
		t.Skip("mygo_demo.cod not found: " + err.Error())
	}
	l := lexer.New(string(src))
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse errors: %v", p.Errors())
	}
	goSrc := Generate(prog, "../../test")
	wants := []string{
		"// --- Mygo Module",
		"mygo \"github.com/egoist/mygo\"",
		"\"github.com/egoist/mygo/ui\"",
		"cMygoRun(",
		"cMygoButton(",
		"cMygoCheckbox(",
		"cMygoInput(",
		"cMygoTheme(",
		`placeholder: toString("输入名字后回车")`,
		"mygo.App.WhenReady",
		"var _ = mygo.App.Run",
	}
	for _, want := range wants {
		if !strings.Contains(goSrc, want) {
			t.Fatalf("missing in generated source: %q", want)
		}
	}
	// On-demand emit: a mygo-only program must not pull in other UI modules.
	for _, unwanted := range []string{"fyne.io/fyne", "webview_go", "bubbletea"} {
		if strings.Contains(goSrc, unwanted) {
			t.Fatalf("unexpected fragment in generated source: %q", unwanted)
		}
	}
	if v := LastGenerateGoVersion(); v != MygoGoVersion {
		t.Fatalf("go version = %q, want %q", v, MygoGoVersion)
	}
}

// TestMygoBaselineKeepsGoVersion verifies that programs not using mygo keep
// the Go 1.22 go.mod baseline (no toolchain escalation, no fragment).
func TestMygoBaselineKeepsGoVersion(t *testing.T) {
	l := lexer.New(`print("hello")`)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse errors: %v", p.Errors())
	}
	goSrc := Generate(prog, "")
	if strings.Contains(goSrc, "// --- Mygo Module") {
		t.Fatal("mygo fragment emitted for a program not using it")
	}
	if strings.Contains(goSrc, "github.com/egoist/mygo") {
		t.Fatal("mygo import emitted for a program not using it")
	}
	if v := LastGenerateGoVersion(); v != DefaultGoVersion {
		t.Fatalf("go version = %q, want %q", v, DefaultGoVersion)
	}
}
