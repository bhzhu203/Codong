package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

var (
	guiApp        fyne.App
	mainWindow    fyne.Window
	timerLabel    *widget.Label
	startBtn      *widget.Button
	resetBtn      *widget.Button
	minuteEntry   *widget.Entry
	secondEntry   *widget.Entry
	remainingSecs int
	isRunning     bool
	mu            sync.Mutex
	animWin       fyne.Window
	updateCh      chan func()
)

func sendNotification(title, message string) {
	switch runtime.GOOS {
	case "linux":
		exec.Command("notify-send", title, message).Run()
	case "darwin":
		script := fmt.Sprintf(`display notification "%s" with title "%s"`, message, title)
		exec.Command("osascript", "-e", script).Run()
	case "windows":
		exec.Command("powershell", "-Command", fmt.Sprintf(`[System.Windows.Forms.MessageBox]::Show("%s", "%s")`, message, title)).Run()
	}
}

func showTimeUpAnimation() {
	updateCh <- func() {
		if animWin != nil {
			animWin.Close()
		}
		animApp := app.New()
		animWin = animApp.NewWindow("Time is up!")
		animWin.Resize(fyne.NewSize(400, 300))
		animLabel := widget.NewLabel("Time is up!")
		animLabel.TextStyle = fyne.TextStyle{Bold: true}
		animLabel.Alignment = fyne.TextAlignCenter
		closeBtn := widget.NewButton("Close", func() {
			if animWin != nil {
				animWin.Close()
				animWin = nil
			}
		})
		content := container.NewVBox(
			widget.NewLabel(""),
			animLabel,
			widget.NewLabel(""),
			closeBtn,
			widget.NewLabel(""),
		)
		animWin.SetContent(content)
		animWin.Show()
	}
}

func formatTime(seconds int) string {
	mins := seconds / 60
	secs := seconds % 60
	return fmt.Sprintf("%02d:%02d", mins, secs)
}

func toggleTimer() {
	mu.Lock()
	if isRunning {
		isRunning = false
		startBtn.SetText("Start")
		mu.Unlock()
		return
	}
	if remainingSecs <= 0 {
		mins := 0
		secs := 0
		fmt.Sscanf(minuteEntry.Text, "%d", &mins)
		fmt.Sscanf(secondEntry.Text, "%d", &secs)
		remainingSecs = mins*60 + secs
		if remainingSecs <= 0 {
			mu.Unlock()
			dialog.ShowInformation("Info", "Please enter valid time", mainWindow)
			return
		}
	}
	isRunning = true
	startBtn.SetText("Pause")
	mu.Unlock()

	go func() {
		for isRunning && remainingSecs > 0 {
			time.Sleep(1 * time.Second)
			mu.Lock()
			secs := remainingSecs
			remainingSecs--
			mu.Unlock()
			secsVal := secs
			updateCh <- func() {
				if secsVal <= 10 {
					timerLabel.SetText("Last 10s: " + formatTime(secsVal))
				} else {
					timerLabel.SetText(formatTime(secsVal))
				}
			}
		}
		updateCh <- func() {
			timerLabel.SetText("00:00")
			isRunning = false
			startBtn.SetText("Start")
			startBtn.Enable()
			go sendNotification("Time is up!", "Your timer has finished")
			go showTimeUpAnimation()
		}
	}()
}

func resetTimer() {
	updateCh <- func() {
		mu.Lock()
		defer mu.Unlock()
		isRunning = false
		remainingSecs = 0
		timerLabel.SetText("00:00")
		startBtn.SetText("Start")
		startBtn.Enable()
	}
}

func main() {
	guiApp = app.New()
	mainWindow = guiApp.NewWindow("Codong Timer")
	mainWindow.Resize(fyne.NewSize(500, 400))

	updateCh = make(chan func{}, 100)

	go func() {
		for fn := range updateCh {
			fn()
		}
	}()

	titleLabel := widget.NewLabel("Timer")
	titleLabel.TextStyle = fyne.TextStyle{Bold: true}
	titleLabel.Alignment = fyne.TextAlignCenter

	timerLabel = widget.NewLabel("00:00")
	timerLabel.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
	timerLabel.Alignment = fyne.TextAlignCenter

	minuteEntry = widget.NewEntry()
	minuteEntry.SetPlaceHolder("Min")
	minuteEntry.Text = "1"

	secondEntry = widget.NewEntry()
	secondEntry.SetPlaceHolder("Sec")
	secondEntry.Text = "0"

	inputForm := container.NewGridWithColumns(2,
		container.NewVBox(widget.NewLabel("Minutes:"), minuteEntry),
		container.NewVBox(widget.NewLabel("Seconds:"), secondEntry),
	)

	startBtn = widget.NewButton("Start", toggleTimer)
	startBtn.Icon = theme.MediaPlayIcon()

	resetBtn = widget.NewButton("Reset", resetTimer)
	resetBtn.Icon = theme.ViewRefreshIcon()

	buttons := container.NewHBox(startBtn, resetBtn)

	quickBtn1 := widget.NewButton("1 min", func() {
		minuteEntry.SetText("1")
		secondEntry.SetText("0")
	})
	quickBtn5 := widget.NewButton("5 min", func() {
		minuteEntry.SetText("5")
		secondEntry.SetText("0")
	})
	quickBtn10 := widget.NewButton("10 min", func() {
		minuteEntry.SetText("10")
		secondEntry.SetText("0")
	})
	quickBtn25 := widget.NewButton("25 min", func() {
		minuteEntry.SetText("25")
		secondEntry.SetText("0")
	})

	quickPresets := container.NewGridWithColumns(4, quickBtn1, quickBtn5, quickBtn10, quickBtn25)

	content := container.NewVBox(
		titleLabel,
		widget.NewSeparator(),
		timerLabel,
		widget.NewSeparator(),
		inputForm,
		buttons,
		widget.NewSeparator(),
		widget.NewLabel("Quick presets:"),
		quickPresets,
	)

	mainWindow.SetContent(content)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		if animWin != nil {
			animWin.Close()
		}
		os.Exit(0)
	}()

	mainWindow.ShowAndRun()
}
