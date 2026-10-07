package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	buttonStyle = lipgloss.NewStyle().Padding(1, 2).Background(lipgloss.Color("63")).Foreground(lipgloss.Color("15"))
)

type model struct {
	textInput textinput.Model
	clicked   bool
	quitting  bool
}

func initialModel() model {
	ti := textinput.New()
	ti.Placeholder = "输入点什么..."
	ti.Focus()
	ti.CharLimit = 50
	ti.Width = 30

	return model{
		textInput: ti,
		clicked:   false,
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.quitting = true
			return m, tea.Quit
		case tea.KeyEnter:
			m.clicked = true
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.quitting {
		return "再见！👋\n"
	}

	s := titleStyle.Render("你好 Codong!") + "\n\n"
	s += "欢迎使用终端图形界面\n\n"
	
	if m.clicked {
		s += buttonStyle.Render("按钮被点击了！🎉") + "\n\n"
	} else {
		s += buttonStyle.Render("按 Enter 点击按钮") + "\n\n"
	}
	
	s += m.textInput.View() + "\n\n"
	s += "按 Ctrl+C 退出"
	return s
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("错误：%v\n", err)
		os.Exit(1)
	}
}
