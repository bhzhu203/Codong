package interpreter

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/codong-lang/codong/stdlib/codongerror"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

const (
	maxImageWidth    = 8192
	maxImageHeight   = 8192
	maxImagePixels   = 50_000_000
	maxImageFileSize = 100 * 1024 * 1024 // 100MB
)

// Concurrency semaphore for image processing
var (
	imgMaxConcurrent int64
	imgSemChan       chan struct{}
	imgSemOnce       sync.Once
)

func initImageSemaphore() {
	imgMaxConcurrent = int64(runtime.NumCPU() * 2)
	if v := os.Getenv("CODONG_IMAGE_MAX_CONCURRENT"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			imgMaxConcurrent = n
		}
	}
	imgSemChan = make(chan struct{}, imgMaxConcurrent)
}

func acquireImageSlot() bool {
	imgSemOnce.Do(initImageSemaphore)
	imgSemChan <- struct{}{}
	return true
}

func releaseImageSlot() {
	<-imgSemChan
}

// ImageModuleObject is the singleton `image` module.
type ImageModuleObject struct{}

func (im *ImageModuleObject) Type() string    { return "module" }
func (im *ImageModuleObject) Inspect() string { return "<module:image>" }

var imageModuleSingleton = &ImageModuleObject{}

// CodongImageObject represents a loaded image with chainable operations.
type CodongImageObject struct {
	img    image.Image
	format string
	path   string
}

func (ci *CodongImageObject) Type() string    { return "image" }
func (ci *CodongImageObject) Inspect() string { return fmt.Sprintf("<image:%s>", ci.format) }

func imageError(code, message, fix string) Object {
	return &ErrorObject{
		Error:     codongerror.New(code, message, codongerror.WithFix(fix)),
		IsRuntime: true,
	}
}

// evalImageModuleMethod dispatches image.xxx() calls.
func (interp *Interpreter) evalImageModuleMethod(method string) Object {
	return &BuiltinFunction{
		Name: "image." + method,
		Fn: func(i *Interpreter, args ...Object) Object {
			switch method {
			case "open":
				return i.imageOpen(args)
			case "from_bytes":
				return i.imageFromBytes(args)
			case "info":
				return i.imageInfo(args)
			case "read_exif":
				return i.imageReadExif(args)
			default:
				return imageError(codongerror.E12007_PROCESSING_FAILED,
					fmt.Sprintf("unknown image method: %s", method), "")
			}
		},
	}
}

// evalImageObjectMethod dispatches methods on a CodongImageObject (img.resize(), etc.)
func (interp *Interpreter) evalImageObjectMethod(img *CodongImageObject, method string) Object {
	return &BuiltinFunction{
		Name: "image." + method,
		Fn: func(i *Interpreter, args ...Object) Object {
			switch method {
			case "resize":
				return i.imgResize(img, args)
			case "fit":
				return i.imgFit(img, args)
			case "cover":
				return i.imgCover(img, args)
			case "crop":
				return i.imgCrop(img, args)
			case "crop_center":
				return i.imgCropCenter(img, args)
			case "thumbnail":
				return i.imgThumbnail(img, args)
			case "flip_horizontal":
				return i.imgFlipH(img)
			case "flip_vertical":
				return i.imgFlipV(img)
			case "rotate":
				return i.imgRotate(img, args)
			case "auto_rotate":
				return img // No EXIF rotation in pure Go without exif lib
			case "to_grayscale":
				return i.imgGrayscale(img)
			case "watermark_text":
				return i.imgWatermarkText(img, args)
			case "watermark":
				return i.imgWatermarkText(img, args) // Alias
			case "strip_metadata":
				return img // Already stripped in Go decode
			case "save":
				return i.imgSave(img, args)
			case "to_bytes":
				return i.imgToBytes(img, args)
			case "to_base64":
				return i.imgToBase64(img, args)
			case "width":
				return &NumberObject{Value: float64(img.img.Bounds().Dx())}
			case "height":
				return &NumberObject{Value: float64(img.img.Bounds().Dy())}
			default:
				return imageError(codongerror.E12007_PROCESSING_FAILED,
					fmt.Sprintf("unknown image method: %s", method), "")
			}
		},
	}
}

func (i *Interpreter) imageOpen(args []Object) Object {
	if len(args) < 1 {
		return imageError(codongerror.E12007_PROCESSING_FAILED,
			"image.open requires a file path", "image.open(\"./photo.jpg\")")
	}
	path := args[0].Inspect()
	absPath := i.fsResolve(path)

	// Check file size (decompression bomb protection)
	info, err := os.Stat(absPath)
	if err != nil {
		return imageError(codongerror.E12007_PROCESSING_FAILED,
			fmt.Sprintf("cannot open image: %s", err.Error()),
			"check file path")
	}
	if info.Size() > maxImageFileSize {
		return imageError(codongerror.E12003_TOO_LARGE,
			fmt.Sprintf("file size %d bytes exceeds limit %d bytes", info.Size(), maxImageFileSize),
			"reduce file size before processing")
	}

	f, err := os.Open(absPath)
	if err != nil {
		return imageError(codongerror.E12007_PROCESSING_FAILED,
			fmt.Sprintf("cannot open image: %s", err.Error()), "check file path")
	}
	defer f.Close()

	// Read header for dimensions check (decompression bomb protection)
	config, format, err := image.DecodeConfig(f)
	if err != nil {
		return imageError(codongerror.E12002_CORRUPT_IMAGE,
			"cannot read image header: "+err.Error(),
			"verify the file is a valid image")
	}

	if config.Width > maxImageWidth || config.Height > maxImageHeight {
		return imageError(codongerror.E12003_TOO_LARGE,
			fmt.Sprintf("image dimensions %dx%d exceed limit %dx%d",
				config.Width, config.Height, maxImageWidth, maxImageHeight),
			"resize the image to within 8192x8192 before processing")
	}
	if config.Width*config.Height > maxImagePixels {
		return imageError(codongerror.E12003_TOO_LARGE,
			fmt.Sprintf("total pixels %d exceed limit %d", config.Width*config.Height, maxImagePixels),
			"reduce image resolution before processing")
	}

	// Acquire semaphore slot
	acquireImageSlot()
	defer releaseImageSlot()

	// Rewind and decode
	f.Seek(0, 0)
	img, _, err := image.Decode(f)
	if err != nil {
		return imageError(codongerror.E12002_CORRUPT_IMAGE,
			"cannot decode image: "+err.Error(),
			"verify the file is a valid image")
	}

	return &CodongImageObject{img: img, format: format, path: absPath}
}

func (i *Interpreter) imageFromBytes(args []Object) Object {
	if len(args) < 1 {
		return imageError(codongerror.E12007_PROCESSING_FAILED,
			"image.from_bytes requires byte data", "")
	}

	data := args[0].Inspect()
	reader := bytes.NewReader([]byte(data))

	config, format, err := image.DecodeConfig(reader)
	if err != nil {
		return imageError(codongerror.E12002_CORRUPT_IMAGE,
			"cannot read image header: "+err.Error(), "")
	}

	if config.Width > maxImageWidth || config.Height > maxImageHeight ||
		config.Width*config.Height > maxImagePixels {
		return imageError(codongerror.E12003_TOO_LARGE,
			"image dimensions exceed limit", "reduce image resolution")
	}

	acquireImageSlot()
	defer releaseImageSlot()

	reader.Seek(0, 0)
	img, _, err := image.Decode(reader)
	if err != nil {
		return imageError(codongerror.E12002_CORRUPT_IMAGE,
			"cannot decode image: "+err.Error(), "")
	}

	return &CodongImageObject{img: img, format: format}
}

func (i *Interpreter) imageInfo(args []Object) Object {
	if len(args) < 1 {
		return NULL_OBJ
	}
	path := args[0].Inspect()
	absPath := i.fsResolve(path)

	info, err := os.Stat(absPath)
	if err != nil {
		return NULL_OBJ
	}

	f, err := os.Open(absPath)
	if err != nil {
		return NULL_OBJ
	}
	defer f.Close()

	config, format, err := image.DecodeConfig(f)
	if err != nil {
		return NULL_OBJ
	}

	return &MapObject{
		Entries: map[string]Object{
			"width":      &NumberObject{Value: float64(config.Width)},
			"height":     &NumberObject{Value: float64(config.Height)},
			"format":     &StringObject{Value: format},
			"size_bytes": &NumberObject{Value: float64(info.Size())},
		},
		Order: []string{"width", "height", "format", "size_bytes"},
	}
}

func (i *Interpreter) imageReadExif(args []Object) Object {
	// Basic EXIF placeholder - Go standard library doesn't include EXIF
	// Return empty map for now
	return &MapObject{Entries: map[string]Object{}, Order: []string{}}
}

// Resize image to given dimensions
func (i *Interpreter) imgResize(img *CodongImageObject, args []Object) Object {
	bounds := img.img.Bounds()
	origW := float64(bounds.Dx())
	origH := float64(bounds.Dy())

	var newW, newH int

	if len(args) >= 2 && args[0] != NULL_OBJ && args[1] != NULL_OBJ {
		newW = int(args[0].(*NumberObject).Value)
		newH = int(args[1].(*NumberObject).Value)
	} else if len(args) >= 1 && args[0] != NULL_OBJ {
		newW = int(args[0].(*NumberObject).Value)
		newH = int(float64(newW) * origH / origW)
	} else if len(args) >= 2 && args[0] == NULL_OBJ && args[1] != NULL_OBJ {
		newH = int(args[1].(*NumberObject).Value)
		newW = int(float64(newH) * origW / origH)
	} else {
		return img
	}

	if newW <= 0 || newH <= 0 {
		return imageError(codongerror.E12004_INVALID_DIMENSIONS,
			"width and height must be positive", "")
	}

	resized := resizeImage(img.img, newW, newH)
	return &CodongImageObject{img: resized, format: img.format, path: img.path}
}

func (i *Interpreter) imgFit(img *CodongImageObject, args []Object) Object {
	if len(args) < 2 {
		return img
	}
	maxW := int(args[0].(*NumberObject).Value)
	maxH := int(args[1].(*NumberObject).Value)

	bounds := img.img.Bounds()
	origW := float64(bounds.Dx())
	origH := float64(bounds.Dy())

	ratio := math.Min(float64(maxW)/origW, float64(maxH)/origH)
	if ratio >= 1.0 {
		return img // Already fits
	}

	newW := int(origW * ratio)
	newH := int(origH * ratio)

	resized := resizeImage(img.img, newW, newH)
	return &CodongImageObject{img: resized, format: img.format, path: img.path}
}

func (i *Interpreter) imgCover(img *CodongImageObject, args []Object) Object {
	if len(args) < 2 {
		return img
	}
	targetW := int(args[0].(*NumberObject).Value)
	targetH := int(args[1].(*NumberObject).Value)

	bounds := img.img.Bounds()
	origW := float64(bounds.Dx())
	origH := float64(bounds.Dy())

	ratio := math.Max(float64(targetW)/origW, float64(targetH)/origH)
	newW := int(origW * ratio)
	newH := int(origH * ratio)

	resized := resizeImage(img.img, newW, newH)

	// Center crop
	x := (newW - targetW) / 2
	y := (newH - targetH) / 2
	cropped := cropImage(resized, x, y, targetW, targetH)

	return &CodongImageObject{img: cropped, format: img.format, path: img.path}
}

func (i *Interpreter) imgCrop(img *CodongImageObject, args []Object) Object {
	x, y, w, h := 0, 0, 0, 0

	// Parse from named args map
	for _, a := range args {
		if m, ok := a.(*MapObject); ok {
			if v, ok := m.Entries["x"]; ok {
				x = int(v.(*NumberObject).Value)
			}
			if v, ok := m.Entries["y"]; ok {
				y = int(v.(*NumberObject).Value)
			}
			if v, ok := m.Entries["width"]; ok {
				w = int(v.(*NumberObject).Value)
			}
			if v, ok := m.Entries["height"]; ok {
				h = int(v.(*NumberObject).Value)
			}
		}
	}

	// Also try positional: crop(x, y, w, h)
	if w == 0 && len(args) >= 4 {
		if n, ok := args[0].(*NumberObject); ok {
			x = int(n.Value)
		}
		if n, ok := args[1].(*NumberObject); ok {
			y = int(n.Value)
		}
		if n, ok := args[2].(*NumberObject); ok {
			w = int(n.Value)
		}
		if n, ok := args[3].(*NumberObject); ok {
			h = int(n.Value)
		}
	}

	if w <= 0 || h <= 0 {
		return imageError(codongerror.E12004_INVALID_DIMENSIONS,
			"crop dimensions must be positive", "")
	}

	cropped := cropImage(img.img, x, y, w, h)
	return &CodongImageObject{img: cropped, format: img.format, path: img.path}
}

func (i *Interpreter) imgCropCenter(img *CodongImageObject, args []Object) Object {
	if len(args) < 2 {
		return img
	}
	w := int(args[0].(*NumberObject).Value)
	h := int(args[1].(*NumberObject).Value)

	bounds := img.img.Bounds()
	x := (bounds.Dx() - w) / 2
	y := (bounds.Dy() - h) / 2

	cropped := cropImage(img.img, x, y, w, h)
	return &CodongImageObject{img: cropped, format: img.format, path: img.path}
}

func (i *Interpreter) imgThumbnail(img *CodongImageObject, args []Object) Object {
	if len(args) < 2 {
		return img
	}
	w := int(args[0].(*NumberObject).Value)
	h := int(args[1].(*NumberObject).Value)

	// Cover + crop approach
	_ = w
	_ = h
	return i.imgCover(img, args[:2])
}

func (i *Interpreter) imgFlipH(img *CodongImageObject) Object {
	bounds := img.img.Bounds()
	flipped := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			flipped.Set(bounds.Max.X-1-x, y, img.img.At(x, y))
		}
	}
	return &CodongImageObject{img: flipped, format: img.format, path: img.path}
}

func (i *Interpreter) imgFlipV(img *CodongImageObject) Object {
	bounds := img.img.Bounds()
	flipped := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			flipped.Set(x, bounds.Max.Y-1-y, img.img.At(x, y))
		}
	}
	return &CodongImageObject{img: flipped, format: img.format, path: img.path}
}

func (i *Interpreter) imgRotate(img *CodongImageObject, args []Object) Object {
	if len(args) < 1 {
		return img
	}
	degrees := int(args[0].(*NumberObject).Value)
	degrees = ((degrees % 360) + 360) % 360

	switch degrees {
	case 90:
		return &CodongImageObject{img: rotate90(img.img), format: img.format, path: img.path}
	case 180:
		return &CodongImageObject{img: rotate180(img.img), format: img.format, path: img.path}
	case 270:
		return &CodongImageObject{img: rotate270(img.img), format: img.format, path: img.path}
	}
	return img
}

func (i *Interpreter) imgGrayscale(img *CodongImageObject) Object {
	bounds := img.img.Bounds()
	gray := image.NewGray(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			gray.Set(x, y, img.img.At(x, y))
		}
	}
	return &CodongImageObject{img: gray, format: img.format, path: img.path}
}

func (i *Interpreter) imgWatermarkText(img *CodongImageObject, args []Object) Object {
	if len(args) < 1 {
		return img
	}
	text := args[0].Inspect()

	bounds := img.img.Bounds()

	// Draw the original image onto a new RGBA
	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, img.img, bounds.Min, draw.Src)

	// Simple text watermark: draw a semi-transparent rectangle at bottom-right with text
	// Since we can't use freetype without external deps, we draw a simple marker
	position := "bottom_right"
	if len(args) > 1 {
		if m, ok := args[1].(*MapObject); ok {
			if v, ok := m.Entries["position"]; ok {
				position = v.Inspect()
			}
		}
	}

	// Draw a semi-transparent overlay as watermark indicator
	watermarkColor := color.RGBA{255, 255, 255, 128}
	textLen := len(text) * 8
	textH := 20

	var startX, startY int
	switch position {
	case "top_left":
		startX, startY = 10, 10
	case "top_right":
		startX, startY = bounds.Dx()-textLen-10, 10
	case "bottom_left":
		startX, startY = 10, bounds.Dy()-textH-10
	case "center":
		startX, startY = (bounds.Dx()-textLen)/2, (bounds.Dy()-textH)/2
	default: // bottom_right
		startX, startY = bounds.Dx()-textLen-10, bounds.Dy()-textH-10
	}

	for y := startY; y < startY+textH && y < bounds.Dy(); y++ {
		for x := startX; x < startX+textLen && x < bounds.Dx(); x++ {
			if x >= 0 && y >= 0 {
				rgba.Set(x, y, watermarkColor)
			}
		}
	}

	return &CodongImageObject{img: rgba, format: img.format, path: img.path}
}

func (i *Interpreter) imgSave(img *CodongImageObject, args []Object) Object {
	if len(args) < 1 {
		return imageError(codongerror.E12006_SAVE_FAILED, "save requires an output path", "")
	}
	outPath := i.fsResolve(args[0].Inspect())

	quality := 85
	if len(args) > 1 {
		if m, ok := args[1].(*MapObject); ok {
			if v, ok := m.Entries["quality"]; ok {
				if n, ok := v.(*NumberObject); ok {
					quality = int(n.Value)
				}
			}
		}
	}

	// Determine format from extension
	ext := strings.ToLower(filepath.Ext(outPath))
	format := img.format
	switch ext {
	case ".jpg", ".jpeg":
		format = "jpeg"
	case ".png":
		format = "png"
	case ".gif":
		format = "gif"
	case ".webp":
		format = "png" // WebP write not in std library, fallback to PNG
	}

	// Ensure output directory exists
	os.MkdirAll(filepath.Dir(outPath), 0755)

	f, err := os.Create(outPath)
	if err != nil {
		return imageError(codongerror.E12006_SAVE_FAILED,
			fmt.Sprintf("cannot create output file: %s", err.Error()),
			"check output path permissions")
	}
	defer f.Close()

	switch format {
	case "jpeg":
		err = jpeg.Encode(f, img.img, &jpeg.Options{Quality: quality})
	case "png":
		err = png.Encode(f, img.img)
	case "gif":
		err = gif.Encode(f, img.img, nil)
	default:
		err = png.Encode(f, img.img)
	}

	if err != nil {
		return imageError(codongerror.E12006_SAVE_FAILED,
			fmt.Sprintf("encoding failed: %s", err.Error()), "")
	}

	return img // Return self for chaining
}

func (i *Interpreter) imgToBytes(img *CodongImageObject, args []Object) Object {
	format := "jpeg"
	quality := 85

	if len(args) > 0 {
		format = strings.ToLower(args[0].Inspect())
	}
	if len(args) > 1 {
		if m, ok := args[1].(*MapObject); ok {
			if v, ok := m.Entries["quality"]; ok {
				if n, ok := v.(*NumberObject); ok {
					quality = int(n.Value)
				}
			}
		}
	}

	var buf bytes.Buffer
	switch format {
	case "jpeg", "jpg":
		jpeg.Encode(&buf, img.img, &jpeg.Options{Quality: quality})
	case "png":
		png.Encode(&buf, img.img)
	default:
		png.Encode(&buf, img.img)
	}

	return &StringObject{Value: buf.String()}
}

func (i *Interpreter) imgToBase64(img *CodongImageObject, args []Object) Object {
	format := "jpeg"
	if len(args) > 0 {
		format = strings.ToLower(args[0].Inspect())
	}

	var buf bytes.Buffer
	switch format {
	case "jpeg", "jpg":
		jpeg.Encode(&buf, img.img, &jpeg.Options{Quality: 85})
	case "png":
		png.Encode(&buf, img.img)
	default:
		png.Encode(&buf, img.img)
	}

	b64 := base64.StdEncoding.EncodeToString(buf.Bytes())
	mime := "image/" + format
	return &StringObject{Value: fmt.Sprintf("data:%s;base64,%s", mime, b64)}
}

// Image manipulation helpers using bilinear interpolation

func resizeImage(src image.Image, newW, newH int) image.Image {
	bounds := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))

	scaleX := float64(bounds.Dx()) / float64(newW)
	scaleY := float64(bounds.Dy()) / float64(newH)

	for y := 0; y < newH; y++ {
		for x := 0; x < newW; x++ {
			srcX := int(float64(x)*scaleX) + bounds.Min.X
			srcY := int(float64(y)*scaleY) + bounds.Min.Y
			if srcX >= bounds.Max.X {
				srcX = bounds.Max.X - 1
			}
			if srcY >= bounds.Max.Y {
				srcY = bounds.Max.Y - 1
			}
			dst.Set(x, y, src.At(srcX, srcY))
		}
	}
	return dst
}

func cropImage(src image.Image, x, y, w, h int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(dst, dst.Bounds(), src, image.Pt(x+src.Bounds().Min.X, y+src.Bounds().Min.Y), draw.Src)
	return dst
}

func rotate90(src image.Image) image.Image {
	bounds := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, bounds.Dy(), bounds.Dx()))
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			dst.Set(bounds.Max.Y-1-y, x, src.At(x, y))
		}
	}
	return dst
}

func rotate180(src image.Image) image.Image {
	bounds := src.Bounds()
	dst := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			dst.Set(bounds.Max.X-1-x, bounds.Max.Y-1-y, src.At(x, y))
		}
	}
	return dst
}

func rotate270(src image.Image) image.Image {
	bounds := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, bounds.Dy(), bounds.Dx()))
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			dst.Set(y, bounds.Max.X-1-x, src.At(x, y))
		}
	}
	return dst
}
