package goirgen

// RuntimeMygo contains the Go source for the mygo module fragment
// (github.com/egoist/mygo native UI — pure Go, no cgo; the toolchain must
// be Go >= 1.27.1, which the runner writes into the generated go.mod).
// Concatenated by the generator into the final main.go.
//
// Declarative, per-frame view model:
//
//	view = fn(c) { ... }                  // builds the window's interface
//	mygo.run(view, title:, width:, height:)
//
// mygo.run blocks on the mygo event loop (main goroutine, like gui.run) and
// calls view(c) again after every input. State lives in the closures around
// view; stateful widgets return their current value so the caller keeps it:
//
//	map.checked = mygo.checkbox(c, map.checked, "Enable")
const RuntimeMygo = `
// --- Mygo Module (native UI) ---

var cMygoView Value

var cMygoWin *mygo.Window

var cMygoImages = map[string]*ui.Bitmap{}

// cMygoOpts carries the named options shared by the mygo widgets. Each
// widget reads the fields it understands and ignores the rest.
type cMygoOpts struct {

	fill bool

	center bool

	gap float64

	width float64

	height float64

	grow float64

	padding float64

	size float64

	bold bool

	italic bool

	color string

	muted bool

	primary bool

	disabled bool

	placeholder string

	align string

	selectable bool

	lo float64

	hi float64
}

func cMygoCtx(v Value) *ui.Context {

	if c, ok := v.(*ui.Context); ok { return c }

	return nil

}

func cMygoRun(view Value, title string, w int, h int) Value {

	cMygoView = view

	mygo.App.WhenReady(func() {

		cMygoWin = mygo.NewWindow(mygo.WindowOptions{

			Title: title,

			Width: w,

			Height: h,

			Content: ui.View(cMygoFrame),

		})

	})

	if err := mygo.App.Run(); err != nil {

		fmt.Fprintln(os.Stderr, "mygo error:", err)

	}

	return nil

}

func cMygoFrame(c *ui.Context) {

	if cMygoView == nil { return }

	cCallFn(cMygoView, c)

}

func cMygoQuit() Value {

	mygo.App.Quit()

	return nil

}

func cMygoSetTitle(title string) Value {

	if cMygoWin != nil { cMygoWin.SetTitle(title) }

	return nil

}

// cMygoTheme returns the current theme as a map of hex colors, so views can
// style text with the system's appearance (dark/light).
func cMygoTheme(ctxV Value) Value {

	c := cMygoCtx(ctxV)

	if c == nil { return nil }

	t := c.Theme()

	hex := func(col ui.Color) string { return fmt.Sprintf("#%02x%02x%02x", col.R, col.G, col.B) }

	return cMap(

		"dark", t.Dark,

		"text", hex(t.Text),

		"muted", hex(t.TextMuted),

		"surface", hex(t.Surface),

		"background", hex(t.Background),

		"accent", hex(t.Accent),

	)

}

// cMygoShortcut reports whether the named key was pressed in the frame
// being built. Each key is claimed both on the window and on the view root:
// the window claim serves when nothing has the focus, and the root claim
// shields the key from a focused button, which otherwise takes Enter and
// Space for itself.
func cMygoShortcut(c *ui.Context, name string) bool {

	root := c.Root()

	chk := func(mods ui.Modifiers, key ui.Key) bool {

		win := c.Shortcut(mods, key)

		el := false

		if root != nil { el = root.Shortcut(mods, key) }

		return win || el

	}

	switch name {

	case "0":

		return chk(0, ui.Key0)

	case "1":

		return chk(0, ui.Key1)

	case "2":

		return chk(0, ui.Key2)

	case "3":

		return chk(0, ui.Key3)

	case "4":

		return chk(0, ui.Key4)

	case "5":

		return chk(0, ui.Key5)

	case "6":

		return chk(0, ui.Key6)

	case "7":

		return chk(0, ui.Key7)

	case "8":

		return chk(0, ui.Key8)

	case "9":

		return chk(0, ui.Key9)

	case ".":

		return chk(0, ui.KeyPeriod)

	case "+":

		a := chk(ui.Shift, ui.KeyPlus)

		b := chk(0, ui.KeyPlus)

		return a || b

	case "-":

		return chk(0, ui.KeyMinus)

	case "*":

		a := chk(ui.Shift, ui.KeyAsterisk)

		b := chk(0, ui.KeyAsterisk)

		return a || b

	case "/":

		return chk(0, ui.KeySlash)

	case "=":

		return chk(0, ui.KeyEqual)

	case "enter":

		return chk(0, ui.KeyEnter)

	case "backspace":

		return chk(0, ui.KeyBackspace)

	case "escape":

		return chk(0, ui.KeyEscape)

	case "delete":

		return chk(0, ui.KeyDelete)

	case "c":

		a := chk(0, ui.KeyC)

		b := chk(ui.Shift, ui.KeyC)

		return a || b

	case "up":

		return chk(0, ui.KeyUp)

	case "down":

		return chk(0, ui.KeyDown)

	case "left":

		return chk(0, ui.KeyLeft)

	case "right":

		return chk(0, ui.KeyRight)

	case "space":

		return chk(0, ui.KeySpace)

	case "tab":

		return chk(0, ui.KeyTab)

	}

	return false

}

// cMygoKey reports whether the named key was pressed in the frame being
// built: "0" to "9", ".", "+", "-", "*", "/", "=", "enter", "backspace",
// "escape", "delete", "c", "up", "down", "left", "right", "space", "tab".
// On the main keyboard "+" and "*" sit behind Shift on most layouts while
// the numpad sends them bare: both shapes answer, and the numpad digits,
// "." and Enter work too (with NumLock on, as in any application).
func cMygoKey(ctxV Value, name string) Value {

	c := cMygoCtx(ctxV)

	if c == nil { return false }

	return cMygoShortcut(c, name)

}

// cMygoKeyNames are the keys cMygoKeys reports on, checked in order.
var cMygoKeyNames = []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", ".", "+", "-", "*", "/", "=", "enter", "backspace", "escape", "delete", "c"}

// cMygoKeys returns the name of the first key pressed in the frame being
// built, "" when none was. One call consumes one key; drain the queue:
//
//	k = mygo.key(c)
//	while k != "" { handle(k)
//		k = mygo.key(c) }
func cMygoKeys(ctxV Value) Value {

	c := cMygoCtx(ctxV)

	if c == nil { return "" }

	for _, name := range cMygoKeyNames {

		if cMygoShortcut(c, name) { return name }

	}

	return ""

}

func cMygoBox(el *ui.Element, o cMygoOpts) *ui.Element {

	if o.fill { el = el.Fill() }

	if o.center { el = el.Center() }

	if o.gap > 0 { el = el.Gap(float32(o.gap)) }

	if o.width > 0 { el = el.Width(float32(o.width)) }

	if o.height > 0 { el = el.Height(float32(o.height)) }

	if o.grow > 0 { el = el.Grow(float32(o.grow)) }

	if o.padding > 0 { el = el.Padding(float32(o.padding)) }

	return el

}

func cMygoColumn(ctxV Value, children Value, o cMygoOpts) Value {

	c := cMygoCtx(ctxV)

	if c == nil || children == nil { return nil }

	cMygoBox(ui.Column(c), o).Children(func() { cCallFn(children) })

	return nil

}

func cMygoRow(ctxV Value, children Value, o cMygoOpts) Value {

	c := cMygoCtx(ctxV)

	if c == nil || children == nil { return nil }

	cMygoBox(ui.Row(c), o).Children(func() { cCallFn(children) })

	return nil

}

func cMygoScroll(ctxV Value, children Value, o cMygoOpts) Value {

	c := cMygoCtx(ctxV)

	if c == nil || children == nil { return nil }

	cMygoBox(ui.Scroll(c), o).Children(func() { cCallFn(children) })

	return nil

}

func cMygoSpacer(ctxV Value) Value {

	c := cMygoCtx(ctxV)

	if c == nil { return nil }

	ui.Spacer(c)

	return nil

}

func cMygoDivider(ctxV Value) Value {

	c := cMygoCtx(ctxV)

	if c == nil { return nil }

	ui.Divider(c)

	return nil

}

func cMygoText(ctxV Value, s string, o cMygoOpts) Value {

	c := cMygoCtx(ctxV)

	if c == nil { return nil }

	el := ui.Text(c, s)

	if o.size > 0 { el = el.FontSize(float32(o.size)) }

	if o.bold { el = el.Bold() }

	if o.italic { el = el.Italic() }

	if o.color != "" {

		el = el.TextColor(ui.Hex(o.color))

	} else if o.muted {

		el = el.TextColor(c.Theme().TextMuted)

	}

	switch o.align {

	case "start":

		el = el.TextAlign(ui.Start)

	case "center":

		el = el.TextAlign(ui.Center)

	case "end":

		el = el.TextAlign(ui.End)

	}

	if o.fill { el = el.FillWidth() }

	if o.width > 0 { el = el.Width(float32(o.width)) }

	if o.selectable { el = el.Selectable() }

	return nil

}

// cMygoButton returns whether the button was clicked in this frame:
//
//	if mygo.button(c, "+") { n = n + 1 }
func cMygoButton(ctxV Value, label string, o cMygoOpts) Value {

	c := cMygoCtx(ctxV)

	if c == nil { return false }

	var el *ui.Element

	if o.primary { el = ui.PrimaryButton(c, label) } else { el = ui.Button(c, label) }

	if o.width > 0 { el = el.Width(float32(o.width)) }

	if o.disabled { el = el.Disabled(true) }

	clicked := el.Clicked()

	if clicked { c.Invalidate() }

	return clicked

}

func cMygoLink(ctxV Value, label string, url string) Value {

	c := cMygoCtx(ctxV)

	if c == nil { return nil }

	ui.Link(c, label, url)

	return nil

}

// cMygoProgress shows value (0 to 1) as a progress bar.
func cMygoProgress(ctxV Value, v float64, o cMygoOpts) Value {

	c := cMygoCtx(ctxV)

	if c == nil { return nil }

	el := ui.Progress(c, v)

	if o.width > 0 { el = el.Width(float32(o.width)) }

	return nil

}

// cMygoCheckbox returns the checkbox's value after this frame's input:
//
//	checked = mygo.checkbox(c, checked, "Enable")
func cMygoCheckbox(ctxV Value, checkedV Value, label string, o cMygoOpts) Value {

	c := cMygoCtx(ctxV)

	if c == nil { return checkedV }

	checked := isTruthy(checkedV)

	el := ui.Checkbox(c, &checked, label)

	if o.disabled { el = el.Disabled(true) }

	if el.Changed() { c.Invalidate() }

	return checked

}

// cMygoSwitch returns the switch's value after this frame's input:

func cMygoSwitch(ctxV Value, onV Value, o cMygoOpts) Value {

	c := cMygoCtx(ctxV)

	if c == nil { return onV }

	on := isTruthy(onV)

	el := ui.Switch(c, &on)

	if o.disabled { el = el.Disabled(true) }

	if el.Changed() { c.Invalidate() }

	return on

}

// cMygoSlider returns the slider's value after this frame's input:
//
//	v = mygo.slider(c, v, lo: 0, hi: 100)

func cMygoSlider(ctxV Value, vV Value, o cMygoOpts) Value {

	c := cMygoCtx(ctxV)

	if c == nil { return vV }

	v := toFloat(vV)

	el := ui.Slider(c, &v, o.lo, o.hi)

	if o.width > 0 { el = el.Width(float32(o.width)) }

	if el.Changed() { c.Invalidate() }

	return v

}

// cMygoInput returns the text input's value after this frame's input. The
// on_submit callback (optional) runs with the text when Enter is pressed:
//
//	text = mygo.input(c, text, on_submit: fn(t) { send(t) })

func cMygoInput(ctxV Value, textV Value, submitFn Value, o cMygoOpts) Value {

	c := cMygoCtx(ctxV)

	if c == nil { return textV }

	text := toString(textV)

	el := ui.TextInput(c, &text)

	if o.placeholder != "" { el = el.Placeholder(o.placeholder) }

	if o.width > 0 { el = el.Width(float32(o.width)) }

	submitted := el.Submitted()

	if submitted && submitFn != nil { cCallFn(submitFn, text) }

	if el.Changed() || submitted { c.Invalidate() }

	return text

}

// cMygoTextArea returns the multi-line text input's value after this
// frame's input.

func cMygoTextArea(ctxV Value, textV Value, o cMygoOpts) Value {

	c := cMygoCtx(ctxV)

	if c == nil { return textV }

	text := toString(textV)

	el := ui.TextArea(c, &text)

	if o.placeholder != "" { el = el.Placeholder(o.placeholder) }

	if o.width > 0 { el = el.Width(float32(o.width)) }

	if o.height > 0 { el = el.Height(float32(o.height)) }

	if el.Changed() { c.Invalidate() }

	return text

}

// cMygoImage shows the image file at path (decoded once and kept).

func cMygoImage(ctxV Value, path string, o cMygoOpts) Value {

	c := cMygoCtx(ctxV)

	if c == nil { return nil }

	bmp, ok := cMygoImages[path]

	if !ok {

		data, err := os.ReadFile(path)

		if err != nil {

			fmt.Fprintln(os.Stderr, "mygo image:", err)

			return nil

		}

		b, err := ui.DecodeBitmap(data)

		if err != nil {

			fmt.Fprintln(os.Stderr, "mygo image:", err)

			return nil

		}

		bmp = b

		cMygoImages[path] = bmp

	}

	el := ui.Image(c, bmp)

	if o.width > 0 { el = el.Width(float32(o.width)) }

	if o.height > 0 { el = el.Height(float32(o.height)) }

	return nil

}
`

// MygoImports lists the Go imports required by RuntimeMygo. fmt/os are
// shared with the core fragment and deduplicated by emitRuntime.
const MygoImports = `
	"os"
	mygo "github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
`

// MygoKeepAlive lists the Go symbols that must survive linking.
const MygoKeepAlive = `
	mygo.App.Run
	ui.View
`
