package runner

import (
	"strings"
	"testing"

	"github.com/codong-lang/codong/engine/goirgen"
	"github.com/codong-lang/codong/engine/lexer"
	"github.com/codong-lang/codong/engine/parser"
)

func generateForTest(t *testing.T, src string) {
	t.Helper()
	p := parser.New(lexer.New(src))
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse errors: %v", p.Errors())
	}
	goirgen.Generate(prog, "")
}

// TestGoModTemplateVersion verifies the generated go.mod follows the Go
// version the last Generate call requires: mygo programs get `go 1.27.1`,
// everything else keeps the 1.22 baseline.
func TestGoModTemplateVersion(t *testing.T) {
	generateForTest(t, "view = fn(c) { mygo.text(c, \"hi\") }\nmygo.run(view)\n")
	mod := goModTemplate()
	if !strings.Contains(mod, "module codong-app") {
		t.Fatalf("missing module line:\n%s", mod)
	}
	if !strings.Contains(mod, "go 1.27.1") {
		t.Fatalf("mygo program go.mod should require go 1.27.1, got:\n%s", mod)
	}

	generateForTest(t, "print(\"hello\")\n")
	mod = goModTemplate()
	if !strings.Contains(mod, "go 1.22") {
		t.Fatalf("plain program go.mod should keep go 1.22, got:\n%s", mod)
	}
	if strings.Contains(mod, "1.27.1") {
		t.Fatalf("plain program go.mod must not be escalated, got:\n%s", mod)
	}
}

// TestLastGenerateUsesMygo verifies the flag the runner consults to decide
// between the minimal go.mod and the patched-mygo one.
func TestLastGenerateUsesMygo(t *testing.T) {
	generateForTest(t, "view = fn(c) { mygo.text(c, \"hi\") }\nmygo.run(view)\n")
	if !goirgen.LastGenerateUsesMygo() {
		t.Fatal("mygo program should set LastGenerateUsesMygo")
	}

	generateForTest(t, "print(\"hello\")\n")
	if goirgen.LastGenerateUsesMygo() {
		t.Fatal("plain program must not set LastGenerateUsesMygo")
	}
}

// TestGoModTemplateMygo verifies the mygo go.mod pins the version and
// redirects the module to the patched copy, so tidy never fetches the
// unpatched original.
func TestGoModTemplateMygo(t *testing.T) {
	mod := goModTemplateMygo("/tmp/mygo-patched")
	for _, want := range []string{
		"module codong-app",
		"go 1.27.1",
		"require github.com/egoist/mygo v0.2.16",
		"replace github.com/egoist/mygo => /tmp/mygo-patched",
	} {
		if !strings.Contains(mod, want) {
			t.Fatalf("go.mod missing %q:\n%s", want, mod)
		}
	}

	quoted := goModTemplateMygo("/tmp/dir with space")
	if !strings.Contains(quoted, `replace github.com/egoist/mygo => "/tmp/dir with space"`) {
		t.Fatalf("path with spaces should be quoted:\n%s", quoted)
	}
}
