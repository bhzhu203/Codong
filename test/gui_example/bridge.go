package bridge

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"sync"
)

var (
	guiApp     fyne.App
	mainWindow fyne.Window
	mu         sync.Mutex
)

// GUIInit 初始化 GUI 应用
// 在 codong.toml 中注册为：gui.init = { fn = "bridge.GUIInit", ... }
// Codong 调用：result = gui_init("标题", 400, 300)
func GUIInit(title string, width, height int) map[string]interface{} {
	mu.Lock()
	defer mu.Unlock()

	if guiApp == nil {
		guiApp = app.New()
	}

	mainWindow = guiApp.NewWindow(title)
	mainWindow.Resize(fyne.NewSize(float32(width), float32(height)))

	return map[string]interface{}{
		"error": nil,
	}
}

// GUIShowLabel 显示标签
// Codong 调用：result = gui_show_label("欢迎文字")
func GUIShowLabel(text string) map[string]interface{} {
	mu.Lock()
	defer mu.Unlock()

	if mainWindow == nil {
		return map[string]interface{}{
			"error": "GUI not initialized",
		}
	}

	label := widget.NewLabel(text)
	mainWindow.SetContent(label)
	mainWindow.Show()

	return map[string]interface{}{
		"error": nil,
	}
}

// GUIShowButton 显示带按钮的窗口
// Codong 调用：result = gui_show_button("标签文字", "按钮文字")
func GUIShowButton(labelText, buttonText string) map[string]interface{} {
	mu.Lock()
	defer mu.Unlock()

	if mainWindow == nil {
		return map[string]interface{}{
			"error": "GUI not initialized",
		}
	}

	label := widget.NewLabel(labelText)
	button := widget.NewButton(buttonText, func() {
		label.SetText("按钮被点击了！🎉")
	})

	content := container.NewVBox(label, button)
	mainWindow.SetContent(content)
	mainWindow.Show()

	return map[string]interface{}{
		"error": nil,
	}
}

// GUIRun 运行 GUI 主循环
// Codong 调用：result = gui_run()
func GUIRun() map[string]interface{} {
	mu.Lock()
	defer mu.Unlock()

	if guiApp == nil {
		return map[string]interface{}{
			"error": "GUI not initialized",
		}
	}

	guiApp.Run()

	return map[string]interface{}{
		"error": nil,
	}
}

// GUIClose 关闭 GUI
// Codong 调用：result = gui_close()
func GUIClose() map[string]interface{} {
	mu.Lock()
	defer mu.Unlock()

	if mainWindow != nil {
		mainWindow.Close()
	}

	return map[string]interface{}{
		"error": nil,
	}
}
