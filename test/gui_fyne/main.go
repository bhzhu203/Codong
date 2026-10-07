package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func main() {
	// 创建应用
	myApp := app.New()

	// 创建窗口
	myWindow := myApp.NewWindow("你好 Codong!")

	// 创建标签
	label := widget.NewLabel("欢迎使用 Codong 图形界面!")

	// 创建按钮
	button := widget.NewButton("点击我", func() {
		label.SetText("按钮被点击了！🎉")
	})

	// 创建输入框
	entry := widget.NewEntry()
	entry.SetPlaceHolder("输入点什么...")

	// 创建垂直布局
	content := container.NewVBox(
		label,
		button,
		entry,
		widget.NewSeparator(),
		widget.NewLabel("按 Ctrl+C 退出"),
	)

	// 设置窗口内容
	myWindow.SetContent(content)
	myWindow.Resize(fyne.NewSize(400, 300))

	// 运行应用
	myApp.Run()
}
