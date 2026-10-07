package goirgen

// RuntimeTui contains the Go source for the TUI module fragment
// (charmbracelet/bubbletea, pure Go — static binaries). Concatenated by the
// generator into the final main.go.
//
// Elm-style API:
//
//	update = fn(model, msg) { ... return newModel }        // msg.key / msg.type
//	view    = fn(model) { return "rendered text" }
//	final   = tui.run(update, view, initialModel)
//
// Messages arrive as maps: {type: "key", key: "up"}, {type: "resize", ...}.
// Returning {model: m, quit: true} from update exits the program.
const RuntimeTui = `
// --- TUI Module (bubbletea) ---

type cTuiProgram struct {

	update Value

	view Value

	model Value
}

func (p *cTuiProgram) Init() tea.Cmd { return nil }

func (p *cTuiProgram) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	var in Value

	switch t := msg.(type) {

	case tea.KeyMsg:

		in = cMap("type", "key", "key", string(t.String()))

	case tea.WindowSizeMsg:

		in = cMap("type", "resize", "width", float64(t.Width), "height", float64(t.Height))

	default:

		return p, nil

	}

	res := cCallFn(p.update, p.model, in)

	quit := false

	if m, ok := res.(*CodongMap); ok {

		if q, ok2 := m.Entries["quit"]; ok2 && q != nil && toFloat(q) != 0 { quit = true }

		if mm, ok3 := m.Entries["model"]; ok3 { res = mm }

	}

	p.model = res

	if quit { return p, tea.Quit }

	return p, nil

}

func (p *cTuiProgram) View() string {

	return toString(cCallFn(p.view, p.model))

}

func cTuiRun(update Value, view Value, init Value) Value {

	if update == nil || view == nil { return nil }

	p := &cTuiProgram{update: update, view: view, model: init}

	if _, err := tea.NewProgram(p).Run(); err != nil {

		fmt.Fprintln(os.Stderr, "tui error:", err)

	}

	return p.model

}
`

// TuiImports lists the Go imports required by RuntimeTui.
const TuiImports = `
	"fmt"
	"os"
	tea "github.com/charmbracelet/bubbletea"
`

// TuiKeepAlive lists the Go symbols that must survive linking.
const TuiKeepAlive = `
	tea.NewProgram
`
