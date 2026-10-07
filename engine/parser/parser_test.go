package parser

import (
	"testing"

	"github.com/codong-lang/codong/engine/lexer"
)

func parseAll(t *testing.T, src string) *Program {
	t.Helper()
	p := New(lexer.New(src))
	prog := p.ParseProgram()
	if errs := p.Errors(); len(errs) > 0 {
		t.Fatalf("parse errors: %v", errs)
	}
	return prog
}

// Regression: an inline bare `if c { return }` used to leave the parser on the
// block's closing brace; the block loop then stepped over it, swallowing every
// following top-level statement into the function being defined.
func TestInlineBareReturnDoesNotSwallowFollowingStatements(t *testing.T) {
	prog := parseAll(t, `f = fn() {
    if true { return }
    print("f")
}
g = fn() {
    return;
}
print("top")
`)

	if got := len(prog.Statements); got != 3 {
		t.Fatalf("top-level statements = %d, want 3", got)
	}

	assign, ok := prog.Statements[0].(*AssignStatement)
	if !ok {
		t.Fatalf("statement 0 is %T, want *AssignStatement", prog.Statements[0])
	}
	fn, ok := assign.Value.(*FunctionLiteral)
	if !ok {
		t.Fatalf("f's value is %T, want *FunctionLiteral", assign.Value)
	}
	if got := len(fn.Body.Statements); got != 2 {
		t.Fatalf("f body statements = %d, want 2 (if + print)", got)
	}
	if _, ok := fn.Body.Statements[0].(*IfStatement); !ok {
		t.Fatalf("f body statement 0 is %T, want *IfStatement", fn.Body.Statements[0])
	}

	gAssign, ok := prog.Statements[1].(*AssignStatement)
	if !ok {
		t.Fatalf("statement 1 is %T, want *AssignStatement", prog.Statements[1])
	}
	gFn, ok := gAssign.Value.(*FunctionLiteral)
	if !ok {
		t.Fatalf("g's value is %T, want *FunctionLiteral", gAssign.Value)
	}
	if got := len(gFn.Body.Statements); got != 1 {
		t.Fatalf("g body statements = %d, want 1", got)
	}
	if _, ok := gFn.Body.Statements[0].(*ReturnStatement); !ok {
		t.Fatalf("g body statement 0 is %T, want *ReturnStatement", gFn.Body.Statements[0])
	}
}

func TestBareReturnVariantsKeepTopLevelCount(t *testing.T) {
	cases := []struct {
		name    string
		src     string
		wantTop int
	}{
		{
			name: "multiline return in nested block",
			src: `f = fn() {
    if true {
        return
    }
    print("f")
}
print("top")`,
			wantTop: 2,
		},
		{
			name: "semicolon after return",
			src: `f = fn() { if true { return; } }
g = 1`,
			wantTop: 2,
		},
		{
			name: "bare return last in body",
			src: `f = fn() {
    print("a")
    return
}
print("top")`,
			wantTop: 2,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			prog := parseAll(t, tc.src)
			if got := len(prog.Statements); got != tc.wantTop {
				t.Fatalf("top-level statements = %d, want %d", got, tc.wantTop)
			}
		})
	}
}
