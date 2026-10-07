package goirgen

// RuntimeDraw contains the Go source for the Draw module fragment.
// Concatenated by the generator into the final main.go.
const RuntimeDraw = `
// --- Draw Module ---

// cDrawCanvas wraps a gg.Context for 2D drawing. Methods accept both
// positional and named arguments; the generator compiles named arguments
// into a trailing CodongMap which cDrawSplitArgs separates.

type cDrawCanvas struct {

	dc *gg.Context

	w int

	h int

	fontSet bool

	log []string

}

func cDrawCanvasNew(w, h int) *cDrawCanvas {

	if w <= 0 { w = 1 }

	if h <= 0 { h = 1 }

	if w*h > 50000000 { w, h = 800, 600 }

	dc := gg.NewContext(w, h)

	dc.SetHexColor("#ffffff")

	dc.Clear()

	return &cDrawCanvas{dc: dc, w: w, h: h}

}

// cDrawSplitArgs separates a trailing named-argument map from positional args.

func cDrawSplitArgs(args []Value) ([]Value, *CodongMap) {

	if len(args) > 0 {

		if m, ok := args[len(args)-1].(*CodongMap); ok { return args[:len(args)-1], m }

	}

	return args, nil

}

// cDrawKw resolves one argument: named key first, then positional index.

func cDrawKw(pos []Value, named *CodongMap, idx int, key string) Value {

	if named != nil {

		if v := cGet(named, key); v != nil { return v }

	}

	if idx >= 0 && idx < len(pos) { return pos[idx] }

	return nil

}

func cDrawStr(v Value, def string) string {

	if v == nil { return def }

	if s, ok := v.(string); ok { return s }

	return def

}

func cDrawCall(o *cDrawCanvas, method string, args ...Value) Value {

	pos, named := cDrawSplitArgs(args)

	switch method {

	case "background":

		bg := cDrawStr(cDrawKw(pos, named, 0, "color"), "#ffffff")

		o.dc.SetHexColor(bg)

		o.dc.Clear()

		o.log = append(o.log, fmt.Sprintf("<rect width=\"%d\" height=\"%d\" fill=%q/>", o.w, o.h, bg))

		return o

	case "circle":

		x := toFloat(cDrawKw(pos, named, 0, "x"))

		y := toFloat(cDrawKw(pos, named, 1, "y"))

		r := toFloat(cDrawKw(pos, named, 2, "r"))

		fill := cDrawStr(cDrawKw(pos, named, 3, "fill"), "")

		stroke := cDrawStr(cDrawKw(pos, named, 4, "stroke"), "")

		lw := toFloat(cDrawKw(pos, named, 5, "line_width"))

		if lw <= 0 { lw = 1 }

		if fill != "" {

			o.dc.SetHexColor(fill)

			o.dc.DrawCircle(x, y, r)

			o.dc.Fill()

		}

		if stroke != "" {

			o.dc.SetLineWidth(lw)

			o.dc.SetHexColor(stroke)

			o.dc.DrawCircle(x, y, r)

			o.dc.Stroke()

		}

		if fill != "" && stroke != "" {

			o.log = append(o.log, fmt.Sprintf("<circle cx=\"%g\" cy=\"%g\" r=\"%g\" fill=%q stroke=%q stroke-width=\"%g\"/>", x, y, r, fill, stroke, lw))

		} else if fill != "" {

			o.log = append(o.log, fmt.Sprintf("<circle cx=\"%g\" cy=\"%g\" r=\"%g\" fill=%q/>", x, y, r, fill))

		} else if stroke != "" {

			o.log = append(o.log, fmt.Sprintf("<circle cx=\"%g\" cy=\"%g\" r=\"%g\" fill=\"none\" stroke=%q stroke-width=\"%g\"/>", x, y, r, stroke, lw))

		}

		return o

	case "rect":

		x := toFloat(cDrawKw(pos, named, 0, "x"))

		y := toFloat(cDrawKw(pos, named, 1, "y"))

		rw := toFloat(cDrawKw(pos, named, 2, "w"))

		rh := toFloat(cDrawKw(pos, named, 3, "h"))

		fill := cDrawStr(cDrawKw(pos, named, 4, "fill"), "")

		stroke := cDrawStr(cDrawKw(pos, named, 5, "stroke"), "")

		lw := toFloat(cDrawKw(pos, named, 6, "line_width"))

		if lw <= 0 { lw = 1 }

		if fill != "" {

			o.dc.SetHexColor(fill)

			o.dc.DrawRectangle(x, y, rw, rh)

			o.dc.Fill()

		}

		if stroke != "" {

			o.dc.SetLineWidth(lw)

			o.dc.SetHexColor(stroke)

			o.dc.DrawRectangle(x, y, rw, rh)

			o.dc.Stroke()

		}

		if fill != "" && stroke != "" {

			o.log = append(o.log, fmt.Sprintf("<rect x=\"%g\" y=\"%g\" width=\"%g\" height=\"%g\" fill=%q stroke=%q stroke-width=\"%g\"/>", x, y, rw, rh, fill, stroke, lw))

		} else if fill != "" {

			o.log = append(o.log, fmt.Sprintf("<rect x=\"%g\" y=\"%g\" width=\"%g\" height=\"%g\" fill=%q/>", x, y, rw, rh, fill))

		} else if stroke != "" {

			o.log = append(o.log, fmt.Sprintf("<rect x=\"%g\" y=\"%g\" width=\"%g\" height=\"%g\" fill=\"none\" stroke=%q stroke-width=\"%g\"/>", x, y, rw, rh, stroke, lw))

		}

		return o

	case "line":

		x1 := toFloat(cDrawKw(pos, named, 0, "x1"))

		y1 := toFloat(cDrawKw(pos, named, 1, "y1"))

		x2 := toFloat(cDrawKw(pos, named, 2, "x2"))

		y2 := toFloat(cDrawKw(pos, named, 3, "y2"))

		color := cDrawStr(cDrawKw(pos, named, 4, "color"), "#000000")

		lw := toFloat(cDrawKw(pos, named, 5, "width"))

		if lw <= 0 { lw = 1 }

		o.dc.SetHexColor(color)

		o.dc.SetLineWidth(lw)

		o.dc.MoveTo(x1, y1)

		o.dc.LineTo(x2, y2)

		o.dc.Stroke()

		o.log = append(o.log, fmt.Sprintf("<line x1=\"%g\" y1=\"%g\" x2=\"%g\" y2=\"%g\" stroke=%q stroke-width=\"%g\"/>", x1, y1, x2, y2, color, lw))

		return o

	case "text":

		s := cDrawStr(cDrawKw(pos, named, 0, "text"), "")

		x := toFloat(cDrawKw(pos, named, 1, "x"))

		y := toFloat(cDrawKw(pos, named, 2, "y"))

		color := cDrawStr(cDrawKw(pos, named, 3, "color"), "#000000")

		o.dc.SetHexColor(color)

		o.dc.DrawStringAnchored(s, x, y, 0.5, 0.5)

		o.log = append(o.log, fmt.Sprintf("<text x=\"%g\" y=\"%g\" fill=%q text-anchor=\"middle\">%s</text>", x, y, color, html.EscapeString(s)))

		return o

	case "font":

		path := cDrawStr(cDrawKw(pos, named, 0, "path"), "")

		size := toFloat(cDrawKw(pos, named, 1, "size"))

		if size <= 0 { size = 14 }

		if path != "" {

			if err := o.dc.LoadFontFace(path, size); err == nil { o.fontSet = true }

		}

		return o

	case "save":

		path := cDrawStr(cDrawKw(pos, named, 0, "path"), "")

		if path == "" { return cError("E13001", "draw.save: path is required") }

		abs := path

		if !filepath.IsAbs(path) { abs = filepath.Join(cFsWorkDir, path) }

		if err := o.dc.SavePNG(abs); err != nil {

			return cError("E13002", "draw.save failed: "+err.Error())

		}

		return true

	case "png":

		var buf bytes.Buffer

		if err := png.Encode(&buf, o.dc.Image()); err != nil {

			return cError("E13003", "draw.png failed: "+err.Error())

		}

		return buf.Bytes()

	case "svg":

		var sb strings.Builder

		sb.WriteString(fmt.Sprintf("<svg xmlns=\"http://www.w3.org/2000/svg\" width=\"%d\" height=\"%d\">", o.w, o.h))

		for _, e := range o.log {

			sb.WriteString(e)

		}

		sb.WriteString("</svg>")

		return sb.String()

	case "width":

		return float64(o.w)

	case "height":

		return float64(o.h)

	}

	return nil

}
`

// DrawImports lists the Go imports required by RuntimeDraw.
// Each line is a raw Go import clause, embedded in the import ( ... ) block.
const DrawImports = `
	"bytes"
	"fmt"
	"html"
	"image/png"
	"strings"
	"github.com/fogleman/gg"
`

// DrawKeepAlive lists the Go source lines that must be emitted
// as `var _ = <symbol>` to prevent the linker from dropping their packages.
const DrawKeepAlive = `
	gg.NewContext
	png.Encode
`
