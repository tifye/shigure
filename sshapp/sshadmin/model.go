package sshadmin

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tifye/shigure/api"
)

var (
	appStyle   = lipgloss.NewStyle().Padding(1, 2)
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(lipgloss.Color("#25A065")).
			Padding(0, 1)
)

type AdminModel struct {
	profile ProfileInfo
	menu    menu
	result  *string
}

func NewAdminModel(profile ProfileInfo) AdminModel {
	result := ""

	items := []menuItem{
		{
			title: "Generate token",
			action: func() {
				token, _ := api.GenerateToken(os.Getenv("JWT_SIGNING_KEY"))
				*&result = token
			},
		},
		{
			title: "Clean youtube activity",
		},
		{
			title: "Mark repository redacted",
		},
	}

	return AdminModel{
		profile: profile,
		menu:    newMenu(items),
		result:  &result,
	}
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

	var cmd tea.Cmd
	m.menu, cmd = m.menu.Update(msg)

	return m, tea.Batch(cmd)
}

func (m AdminModel) View() string {
	return lipgloss.JoinVertical(
		lipgloss.Center,
		appStyle.Render(m.menu.View()),
		lipgloss.NewStyle().Render(*m.result),
	)
}
