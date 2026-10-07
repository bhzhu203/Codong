package goirgen

// RuntimeGui contains the Go source for the GUI module fragment (Fyne).
// Concatenated by the generator into the final main.go. Requires CGO.
// All widget creation happens on the main goroutine before cGuiApp.Run(),
// which Fyne requires.
const RuntimeGui = `
// --- GUI Module (Fyne) ---

var cGuiApp fyne.App

var cGuiWin fyne.Window

var cGuiBox *fyne.Container

var cGuiHistBox *fyne.Container

var cGuiHistScroll fyne.CanvasObject

var cGuiHistOpen bool

var cGuiWinW float32

var cGuiWinH float32

var cGuiFit bool

var cGuiEntryW float32

var cGuiEntryH float32

var cGuiEntryMinH float32

var cGuiEntryBox *fyne.Container

var cGuiBtnBox *fyne.Container

var cGuiGripObj *cGuiGrip

// Shared row geometry used by cGuiInputRow and the grip drag clamp.

const cGuiBtnW = 96

const cGuiMinEntryW = 160

const cGuiRowPad = 16

type cGuiTheme struct {

	fyne.Theme

}

func (t *cGuiTheme) Size(s fyne.ThemeSizeName) float32 {

	if s == theme.SizeNameText {

		return 13.5

	}

	return t.Theme.Size(s)

}

// All gui text uses one font face (system Noto Sans CJK SC) so Latin and CJK
// glyphs share the same metrics; the per-rune fallback otherwise mixes faces
// with different sizes and spacing.

var cGuiFontRes fyne.Resource

var cGuiFontResBold fyne.Resource

func (t *cGuiTheme) Font(s fyne.TextStyle) fyne.Resource {

	if s.Monospace {

		return t.Theme.Font(s)

	}

	if s.Bold && cGuiFontResBold != nil {

		return cGuiFontResBold

	}

	if cGuiFontRes != nil {

		return cGuiFontRes

	}

	return t.Theme.Font(s)

}

func cGuiFaceName(data []byte, off uint32) string {

	if int(off)+6 > len(data) {

		return ""

	}

	numTables := int(binary.BigEndian.Uint16(data[off+4 : off+6]))

	var nameOff, nameLen uint32

	for i := 0; i < numTables; i++ {

		rec := off + 12 + uint32(i*16)

		if int(rec)+16 > len(data) {

			return ""

		}

		if string(data[rec:rec+4]) == "name" {

			nameOff = binary.BigEndian.Uint32(data[rec+8 : rec+12])

			nameLen = binary.BigEndian.Uint32(data[rec+12 : rec+16])

		}

	}

	if nameOff == 0 || int(nameOff+nameLen) > len(data) {

		return ""

	}

	nt := data[nameOff : nameOff+nameLen]

	if len(nt) < 6 {

		return ""

	}

	count := int(binary.BigEndian.Uint16(nt[2:4]))

	strOff := int(binary.BigEndian.Uint16(nt[4:6]))

	for i := 0; i < count; i++ {

		rec := 6 + i*12

		if rec+12 > len(nt) {

			return ""

		}

		plat := binary.BigEndian.Uint16(nt[rec : rec+2])

		nameID := binary.BigEndian.Uint16(nt[rec+6 : rec+8])

		if nameID != 1 {

			continue

		}

		l := int(binary.BigEndian.Uint16(nt[rec+10 : rec+12]))

		o := int(binary.BigEndian.Uint16(nt[rec+8 : rec+10]))

		if l <= 0 || strOff+o+l > len(nt) {

			continue

		}

		s := nt[strOff+o : strOff+o+l]

		if plat == 3 {

			var b strings.Builder

			for j := 0; j+1 < l; j += 2 {

				b.WriteRune(rune(binary.BigEndian.Uint16(s[j : j+2])))

			}

			return b.String()

		}

		return string(s)

	}

	return ""

}

// cGuiExtractFace rebuilds a standalone sfnt font from one face of a .ttc
// collection: Fyne's font parser cannot read collections directly.

func cGuiExtractFace(data []byte, want uint32) []byte {

	if int(want)+6 > len(data) {

		return nil

	}

	numTables := int(binary.BigEndian.Uint16(data[want+4 : want+6]))

	if numTables <= 0 || numTables > 4096 || 12+numTables*16 > len(data) {

		return nil

	}

	type cGuiTbl struct {

		tag string

		off uint32

		ln  uint32

	}

	recs := make([]cGuiTbl, numTables)

	for i := 0; i < numTables; i++ {

		rec := want + 12 + uint32(i*16)

		if int(rec)+16 > len(data) {

			return nil

		}

		recs[i] = cGuiTbl{

			tag: string(data[rec : rec+4]),

			off: binary.BigEndian.Uint32(data[rec+8 : rec+12]),

			ln:  binary.BigEndian.Uint32(data[rec+12 : rec+16]),

		}

		if int(recs[i].off)+int(recs[i].ln) > len(data) {

			return nil

		}

	}

	p, es := 1, 0

	for p*2 <= numTables {

		p *= 2

		es++

	}

	out := make([]byte, 12+numTables*16)

	binary.BigEndian.PutUint32(out[0:4], binary.BigEndian.Uint32(data[want:want+4]))

	binary.BigEndian.PutUint16(out[4:6], uint16(numTables))

	binary.BigEndian.PutUint16(out[6:8], uint16(p*16))

	binary.BigEndian.PutUint16(out[8:10], uint16(es))

	binary.BigEndian.PutUint16(out[10:12], uint16(numTables*16-p*16))

	cursor := uint32(12 + numTables*16)

	for i, r := range recs {

		rec := uint32(12 + i*16)

		copy(out[rec:rec+4], r.tag)

		pad := (4 - r.ln%4) % 4

		binary.BigEndian.PutUint32(out[rec+8:rec+12], cursor)

		binary.BigEndian.PutUint32(out[rec+12:rec+16], r.ln)

		out = append(out, data[r.off:r.off+r.ln]...)

		out = append(out, make([]byte, pad)...)

		cursor += r.ln + pad

	}

	return out

}

func cGuiLoadFont(path string) fyne.Resource {

	data, err := os.ReadFile(path)

	if err != nil || len(data) < 16 {

		return nil

	}

	if string(data[0:4]) != "ttcf" {

		return fyne.NewStaticResource("font.bin", data)

	}

	numFonts := binary.BigEndian.Uint32(data[8:12])

	for i := uint32(0); i < numFonts; i++ {

		if int(16+i*4) > len(data) {

			break

		}

		off := binary.BigEndian.Uint32(data[12+i*4 : 16+i*4])

		if strings.HasSuffix(cGuiFaceName(data, off), "Noto Sans CJK SC") {

			ex := cGuiExtractFace(data, off)

			if ex != nil {

				return fyne.NewStaticResource("font.bin", ex)

			}

		}

	}

	return nil

}

func cGuiPickFont(bold bool) fyne.Resource {

	style := "Regular"

	if bold {

		style = "Bold"

	}

	dirs := []string{

		"/usr/share/fonts/opentype/noto",

		"/usr/share/fonts/noto-cjk",

		"/usr/share/fonts/truetype/noto",

		"/usr/share/fonts/google-noto-sans-cjk-fonts",

	}

	for _, d := range dirs {

		for _, n := range []string{"NotoSansCJK-" + style + ".ttc", "NotoSansCJKsc-" + style + ".otf"} {

			if r := cGuiLoadFont(filepath.Join(d, n)); r != nil {

				return r

			}

		}

	}

	return nil

}

type cGuiFixedSize struct {

	w *float32

	h *float32

}

func (l cGuiFixedSize) Layout(objects []fyne.CanvasObject, size fyne.Size) {

	for _, o := range objects {

		o.Resize(fyne.NewSize(*l.w, *l.h))

		o.Move(fyne.NewPos(0, 0))

	}

}

func (l cGuiFixedSize) MinSize(_ []fyne.CanvasObject) fyne.Size {

	return fyne.NewSize(*l.w, *l.h)

}

type cGuiGrip struct {

	widget.BaseWidget

}

func (g *cGuiGrip) CreateRenderer() fyne.WidgetRenderer {

	r := canvas.NewRectangle(color.NRGBA{R: 0x9a, G: 0xa3, B: 0xb2, A: 0x99})

	r.CornerRadius = 3

	r.StrokeWidth = 1

	r.StrokeColor = color.NRGBA{R: 0x6b, G: 0x72, B: 0x80, A: 0xff}

	return widget.NewSimpleRenderer(r)

}

func (g *cGuiGrip) Dragged(e *fyne.DragEvent) {

	cGuiEntryW += e.Dragged.DX

	cGuiEntryH += e.Dragged.DY

	maxW := cGuiWinW - cGuiBtnW - cGuiRowPad

	if maxW < cGuiMinEntryW {

		maxW = cGuiMinEntryW

	}

	if cGuiEntryW < cGuiMinEntryW {

		cGuiEntryW = cGuiMinEntryW

	}

	if cGuiEntryW > maxW {

		cGuiEntryW = maxW

	}

	if cGuiEntryH < cGuiEntryMinH {

		cGuiEntryH = cGuiEntryMinH

	}

	if cGuiEntryH > 420 {

		cGuiEntryH = 420

	}

	if cGuiEntryBox != nil {

		cGuiEntryBox.Refresh()

	}

	if cGuiBtnBox != nil {

		cGuiBtnBox.Refresh()

	}

	cGuiPlaceGrip()

	if cGuiFit && !cGuiHistOpen {

		cGuiFitWindow()

	}

}

func (g *cGuiGrip) DragEnd() {}

func (g *cGuiGrip) Cursor() desktop.Cursor {

	return desktop.VResizeCursor

}

func cGuiPlaceGrip() {

	if cGuiGripObj == nil {

		return

	}

	cGuiGripObj.Move(fyne.NewPos(cGuiEntryW-11, cGuiEntryH-11))

	cGuiGripObj.Resize(fyne.NewSize(14, 14))

}

type cGuiTapBar struct {

	widget.BaseWidget

	bar *canvas.Rectangle

	onTap func()

}

func (c *cGuiTapBar) CreateRenderer() fyne.WidgetRenderer {

	return widget.NewSimpleRenderer(c.bar)

}

func (c *cGuiTapBar) Tapped(*fyne.PointEvent) {

	if c.onTap != nil {

		c.onTap()

	}

}

func (c *cGuiTapBar) MinSize() fyne.Size {

	return fyne.NewSize(0, 8)

}

func cGuiInit(title string, w, h int) Value {

	cGuiFontRes = cGuiPickFont(false)

	cGuiFontResBold = cGuiPickFont(true)

	cGuiApp = app.New()

	cGuiApp.Settings().SetTheme(&cGuiTheme{Theme: theme.DefaultTheme()})

	cGuiWin = cGuiApp.NewWindow(title)

	cGuiWinW, cGuiWinH = float32(w), float32(h)

	cGuiWin.Resize(fyne.NewSize(float32(w), float32(h)))

	cGuiBox = container.NewVBox()

	return nil

}

func cGuiShowLabel(text string) Value {

	cGuiBox.Add(widget.NewLabel(text))

	return nil

}

func cGuiShowButton(msg, text string) Value {

	lbl := widget.NewLabel(msg)

	btn := widget.NewButton(text, func() { lbl.SetText("按钮被点击了！") })

	cGuiBox.Add(lbl)

	cGuiBox.Add(btn)

	return nil

}

func cGuiRun() Value {

	if cGuiWin == nil { cGuiInit("Codong", 400, 300) }

	if cGuiHistBox != nil {

		if !cGuiHistOpen {

			cGuiHistScroll.Hide()

		}

		cGuiWin.SetContent(container.NewBorder(cGuiBox, nil, nil, nil, cGuiHistScroll))

	} else {

		cGuiWin.SetContent(cGuiBox)

	}

	if cGuiFit {

		cGuiFitWindow()

	}

	cGuiWin.Show()

	cGuiApp.Run()

	return nil

}

// cGuiWidget wraps one Fyne canvas object for the object-style API:
// b = gui.button("点我", fn() { ... }); gui.add(b); b.set_text("已点")

type cGuiWidget struct {

	obj   fyne.CanvasObject

	label *widget.Label

	btn   *widget.Button

	entry *widget.Entry

}

func cGuiNewLabel(text string) *cGuiWidget {

	l := widget.NewLabel(text)

	return &cGuiWidget{obj: l, label: l}

}

func cGuiNewButton(text string, on_click Value) *cGuiWidget {

	w := &cGuiWidget{}

	btn := widget.NewButton(text, func() {

		if on_click != nil { cCallFn(on_click) }

	})

	w.obj = btn

	w.btn = btn

	return w

}

func cGuiNewEntry(placeholder string) *cGuiWidget {

	e := widget.NewEntry()

	e.SetPlaceHolder(placeholder)

	return &cGuiWidget{obj: e, entry: e}

}

func cGuiNewEntryMulti(placeholder string) *cGuiWidget {

	e := widget.NewMultiLineEntry()

	e.SetPlaceHolder(placeholder)

	return &cGuiWidget{obj: e, entry: e}

}

func cGuiClipboardSet(text string) Value {

	if cGuiWin != nil {

		cGuiWin.Clipboard().SetContent(text)

	}

	return nil

}

func cGuiAdd(w Value) Value {

	if wg, ok := w.(*cGuiWidget); ok && cGuiBox != nil {

		cGuiBox.Add(wg.obj)

	}

	return nil

}

func cGuiHistEnsure() {

	if cGuiHistBox != nil {

		return

	}

	cGuiHistBox = container.NewVBox()

	cGuiHistScroll = container.NewScroll(cGuiHistBox)

	cGuiHistScroll.Hide()

	bar := &cGuiTapBar{bar: canvas.NewRectangle(color.NRGBA{R: 0x4b, G: 0x55, B: 0x6b, A: 0xff})}

	bar.ExtendBaseWidget(bar)

	bar.onTap = cGuiHistFlip

	cGuiBox.Add(bar)

	if cGuiWin != nil && cGuiWin.Content() != cGuiHistScroll {

		cGuiWin.SetContent(container.NewBorder(cGuiBox, nil, nil, nil, cGuiHistScroll))

	}

	if cGuiFit && !cGuiHistOpen {

		cGuiFitWindow()

	}

}

func cGuiFitWindow() {

	if cGuiWin == nil {

		return

	}

	// Resize to a tiny height: the driver clamps up to the content minimum,
	// so the window hugs the input strip (hidden history contributes nothing).

	cGuiWin.Resize(fyne.NewSize(cGuiWinW, 1))

}

func cGuiHistFlip() {

	cGuiHistOpen = !cGuiHistOpen

	if cGuiHistOpen {

		cGuiHistScroll.Show()

		if cGuiWin != nil {

			cGuiWin.Resize(fyne.NewSize(cGuiWinW, cGuiWinH))

		}

	} else {

		cGuiHistScroll.Hide()

		if cGuiFit {

			cGuiFitWindow()

		}

	}

}

func cGuiAddScrolled(w Value) Value {

	wg, ok := w.(*cGuiWidget)

	if !ok {

		return nil

	}

	cGuiHistEnsure()

	cGuiHistBox.Objects = append([]fyne.CanvasObject{wg.obj}, cGuiHistBox.Objects...)

	cGuiHistBox.Refresh()

	return nil

}

func cGuiInputRow(input, btn Value) Value {

	iv, iok := input.(*cGuiWidget)

	bv, bok := btn.(*cGuiWidget)

	if !iok || !bok || cGuiBox == nil {

		return nil

	}

	cGuiFit = true

	side := float32(cGuiBtnW)

	// Entry.MinSize() can report a multi-line wrapped height when the
	// placeholder is long, so measure a single CJK line explicitly.

	cGuiEntryMinH = fyne.MeasureText("中", theme.Current().Size(theme.SizeNameText), fyne.TextStyle{}).Height + 18

	cGuiEntryW = cGuiWinW - side - cGuiRowPad

	if cGuiEntryW < cGuiMinEntryW {

		cGuiEntryW = cGuiMinEntryW

	}

	cGuiEntryH = side

	cGuiEntryBox = container.New(cGuiFixedSize{w: &cGuiEntryW, h: &cGuiEntryH}, iv.obj)

	// Button width stays 96; its height follows the entry so the whole row
	// can collapse to a single text line when the entry is dragged down.

	cGuiBtnBox = container.New(cGuiFixedSize{w: &side, h: &cGuiEntryH}, bv.obj)

	grip := &cGuiGrip{}

	grip.ExtendBaseWidget(grip)

	cGuiGripObj = grip

	wrap := container.NewStack(cGuiEntryBox, container.NewWithoutLayout(grip))

	cGuiPlaceGrip()

	cGuiBox.Add(container.NewBorder(nil, nil, nil, cGuiBtnBox, wrap))

	return nil

}

func cGuiWidgetCall(o *cGuiWidget, method string, args ...Value) Value {

	switch method {

	case "set_text":

		s := ""

		if len(args) > 0 { s = toString(args[0]) }

		if o.label != nil { o.label.SetText(s) }

		if o.btn != nil { o.btn.SetText(s) }

		if o.entry != nil { o.entry.SetText(s) }

		return nil

	case "text":

		if o.entry != nil { return o.entry.Text }

		if o.label != nil { return o.label.Text }

		if o.btn != nil { return o.btn.Text }

		return nil

	case "on_submit":

		if o.entry != nil && len(args) > 0 {

			fn := args[0]

			o.entry.OnSubmitted = func(string) { cCallFn(fn) }

		}

		return nil

	}

	return nil

}
`

// GuiImports lists the Go imports required by RuntimeGui.
const GuiImports = `
	"encoding/binary"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"image/color"
	"os"
	"path/filepath"
	"strings"
`

// GuiKeepAlive lists the Go symbols that must survive linking.
const GuiKeepAlive = `
	app.New
	widget.NewLabel
	container.NewVBox
`
