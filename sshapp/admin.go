package sshapp

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

type adminModel struct {
	term       string
	profile    string
	width      uint
	height     uint
	isDarkMode bool
}

func (m adminModel) Init() tea.Cmd {
	return nil
}

func (m adminModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m adminModel) View() string {
	colorScheme := "Light mode"
	if m.isDarkMode {
		colorScheme = "Dark mode"
	}
	return fmt.Sprintf("%s %s %s [w,h][%d,%d] Admin", m.term, m.profile, colorScheme, m.width, m.height)
}
