package sshadmin

import (
	tea "github.com/charmbracelet/bubbletea"
)

type AdminModel struct {
	Profile ProfileInfo
}

func (m AdminModel) Init() tea.Cmd {
	return nil
}

func (m AdminModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m AdminModel) View() string {
	return m.Profile.String()
}
