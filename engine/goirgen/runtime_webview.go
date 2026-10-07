package goirgen

// RuntimeWebview contains the Go source for the Webview module fragment
// (webview_go + WebKitGTK). Concatenated by the generator into the final
// main.go. Requires CGO. Runs the GTK event loop on the main goroutine,
// so calls block until the window is closed.
const RuntimeWebview = `
// --- Webview Module (webview_go) ---

func cWebviewOpen(url Value, title Value, w Value, h Value) Value {

	wv := webview.New(false)

	defer wv.Destroy()

	wv.SetTitle(toString(title))

	wv.SetSize(int(toFloat(w)), int(toFloat(h)), webview.HintNone)

	wv.Navigate(toString(url))

	wv.Run()

	return nil

}

func cWebviewHtml(html Value, title Value, w Value, h Value) Value {

	wv := webview.New(false)

	defer wv.Destroy()

	wv.SetTitle(toString(title))

	wv.SetSize(int(toFloat(w)), int(toFloat(h)), webview.HintNone)

	// Exposed to page scripts as __cResize(width, height): lets the page
	// grow/shrink the native window (e.g. expand a collapsed history area).
	// The binding callback runs on the UI thread, so SetSize is safe here.

	wv.Bind("__cResize", func(w, h float64) {

		wv.SetSize(int(w), int(h), webview.HintNone)

	})

	// Page → Go bridge for native file dialogs and lifecycle. Bound
	// functions are exposed to the page as window.__cXxx(...) and return a
	// Promise that resolves with the JSON-encoded result (see webview_go).
	// Dialog implementations live in RuntimeWebviewDialogsLinux (cgo/GTK)
	// or RuntimeWebviewDialogsFallback, selected by GOOS at generation time.

	wv.Bind("__cLog", func(msg string) {

		fmt.Println("[webview] " + msg)

	})

	wv.Bind("__cOpenFile", func(title string) string {

		return cWvOpenFile(title)

	})

	wv.Bind("__cSaveFile", func(path, content, suggested string) string {

		return cWvSaveFile(path, content, suggested)

	})

	wv.Bind("__cReadFile", func(path string) string {

		b, err := os.ReadFile(path)

		if err != nil {

			return ""

		}

		return string(b)

	})

	wv.Bind("__cExit", func() {

		cWvExit()

	})

	// System clipboard bridge. From the page: window.__cClipboardRead()
	// resolves (Promise) with the current text or "" when unavailable;
	// window.__cClipboardWrite(text) replaces it. Linux uses GTK CLIPBOARD
	// selection (cgo); other platforms no-op so pages can fall back to
	// navigator.clipboard / document.execCommand('copy').

	wv.Bind("__cClipboardRead", func() string {

		return cWvClipboardRead()

	})

	wv.Bind("__cClipboardWrite", func(text string) {

		cWvClipboardWrite(text)

	})

	wv.SetHtml(toString(html))

	wv.Run()

	return nil

}
`

// WebviewCgoPreamble is injected verbatim into the import block of the
// generated main.go (Linux only) when the webview module is used. It
// declares the C helpers for native file dialogs; gtk+-3.0 is already a
// build dependency of webkit2gtk-4.0, which webview_go links against.
// The preamble comment must stay directly above the "C" import line.
const WebviewCgoPreamble = `	/*
#cgo pkg-config: gtk+-3.0
#include <gtk/gtk.h>

// Native open/save file dialog for webview pages. Returns a newly
// allocated path (free with wv_free) or NULL if the user cancelled.
static char *wv_file_dialog(const char *title, int is_save, const char *initial) {
    GtkFileChooserAction action = is_save ? GTK_FILE_CHOOSER_ACTION_SAVE
                                          : GTK_FILE_CHOOSER_ACTION_OPEN;
    const char *accept_label = is_save ? "保存" : "打开";
    GtkWidget *dialog = gtk_file_chooser_dialog_new(
        title, NULL, action,
        "取消", GTK_RESPONSE_CANCEL,
        accept_label, GTK_RESPONSE_ACCEPT,
        (char *)NULL);
    if (is_save && initial != NULL && initial[0] != '\0') {
        gtk_file_chooser_set_current_name(GTK_FILE_CHOOSER(dialog), initial);
    }
    int resp = gtk_dialog_run(GTK_DIALOG(dialog));
    char *path = NULL;
    if (resp == GTK_RESPONSE_ACCEPT) {
        path = gtk_file_chooser_get_filename(GTK_FILE_CHOOSER(dialog));
    }
    gtk_widget_destroy(dialog);
    return path;
}

static void wv_free(char *p) {
    if (p != NULL) g_free(p);
}

// Synchronous system-clipboard access for webview pages.
// wv_clipboard_read returns a newly allocated string (free with wv_free)
// or NULL when the clipboard holds no text. Both calls briefly spin a
// nested GTK main loop until the clipboard owner responds, so they must
// only be invoked from the UI thread (webview binding callbacks are).
static char *wv_clipboard_read(void) {
    GtkClipboard *clip = gtk_clipboard_get(GDK_SELECTION_CLIPBOARD);
    if (!gtk_clipboard_wait_is_text_available(clip)) return NULL;
    return gtk_clipboard_wait_for_text(clip);
}

static void wv_clipboard_write(const char *text) {
    if (text == NULL || text[0] == '\0') return;
    GtkClipboard *clip = gtk_clipboard_get(GDK_SELECTION_CLIPBOARD);
    gtk_clipboard_set_text(clip, text, -1);
}
*/
	"C"
`

// WebviewImports lists the Go imports required by RuntimeWebview.
const WebviewImports = `
	webview "github.com/webview/webview_go"
`

// WebviewKeepAlive lists the Go symbols that must survive linking.
const WebviewKeepAlive = `
	webview.New
`

// RuntimeWebviewDialogsLinux provides native GTK file dialogs for webview
// pages (cgo). Emitted only when GOOS == linux; requires the cgo preamble
// injected by emitRuntime (gtk+-3.0 is already a build dependency of
// webkit2gtk-4.0, which webview_go links against). The C wrapper does all
// GTK type casting so the Go side never touches cgo struct-pointer casts.
const RuntimeWebviewDialogsLinux = `

// --- Webview Dialogs (Linux/GTK via cgo) ---

func cWvOpenFile(title string) string {

	return cWvFileDialog(title, "open", "")

}

func cWvSaveFile(path, content, suggested string) string {

	if path != "" {

		if err := os.WriteFile(path, []byte(content), 0644); err != nil {

			return "error: " + err.Error()

		}

		return "ok"

	}

	p := cWvFileDialog("保存文件", "save", suggested)

	if p == "" {

		return "" // user cancelled

	}

	if err := os.WriteFile(p, []byte(content), 0644); err != nil {

		return "error: " + err.Error()

	}

	return p

}

func cWvExit() {

	os.Exit(0)

}

func cWvClipboardRead() string {

	p := C.wv_clipboard_read()

	if p == nil {

		return ""

	}

	s := C.GoString(p)

	C.wv_free(p)

	return s

}

func cWvClipboardWrite(text string) {

	if text == "" {

		return

	}

	p := C.CString(text)

	defer C.free(unsafe.Pointer(p))

	C.wv_clipboard_write(p)

}

func cWvFileDialog(title, mode, initialName string) string {

	isSave := 0

	if mode == "save" {

		isSave = 1

	}

	titlePtr := C.CString(title)

	defer C.free(unsafe.Pointer(titlePtr))

	var initPtr *C.char

	if initialName != "" {

		initPtr = C.CString(initialName)

		defer C.free(unsafe.Pointer(initPtr))

	}

	cpath := C.wv_file_dialog(titlePtr, C.int(isSave), initPtr)

	if cpath == nil {

		return ""

	}

	p := C.GoString(cpath)

	C.wv_free(cpath)

	return p

}
`

// RuntimeWebviewDialogsFallback is used on non-Linux platforms where the
// cgo GTK dialogs are unavailable. Open returns "" so pages can fall back
// to <input type="file">; Save only works when the page already knows the
// target path (no native save dialog).
const RuntimeWebviewDialogsFallback = `

// --- Webview Dialogs (fallback: no native file dialogs) ---

func cWvOpenFile(title string) string {

	return "" // page falls back to <input type="file">

}

func cWvSaveFile(path, content, suggested string) string {

	if path == "" {

		return "" // no save dialog on this platform; page falls back to download

	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {

		return "error: " + err.Error()

	}

	return "ok"

}

func cWvExit() {

	os.Exit(0)

}

func cWvClipboardRead() string {

	return "" // no native clipboard here; page falls back to navigator.clipboard

}

func cWvClipboardWrite(text string) {

	// no-op; page falls back to document.execCommand('copy')

}
`
