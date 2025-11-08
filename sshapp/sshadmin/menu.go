package sshadmin

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type menuItem struct {
	title  string
	action func()
}

func (i menuItem) Title() string {
	return i.title
}
func (i menuItem) Description() string { return "" }
func (i menuItem) FilterValue() string {
	return i.title
}

type menu struct {
	list list.Model
}

func newMenu(items []menuItem) menu {
	listItems := make([]list.Item, len(items))
	for i := range len(items) {
		listItems[i] = items[i]
	}

	list := list.New(listItems, newMenuItemDelegate(), 0, 0)
	list.SetShowStatusBar(false)
	list.SetFilteringEnabled(false)
	list.SetShowHelp(false)
	list.SetShowTitle(false)
	list.Styles.Title = titleStyle

	return menu{
		list: list,
	}
}

func (m menu) Init() tea.Cmd {
	return nil
}

func (m menu) Update(msg tea.Msg) (menu, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		w, h := appStyle.GetFrameSize()
		m.list.SetSize(msg.Width-w, msg.Height-h)
	case tea.KeyMsg:
		switch keys := msg.String(); keys {
		case "enter":
			if item, ok := m.list.SelectedItem().(menuItem); ok && item.action != nil {
				item.action()
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)

	return m, tea.Batch(cmd)
}

func (m menu) View() string {
	return m.list.View()
}

func newMenuItemDelegate() list.DefaultDelegate {
	delegate := list.NewDefaultDelegate()
	delegate.UpdateFunc = func(m1 tea.Msg, m2 *list.Model) tea.Cmd {
		return nil
	}
	delegate.ShowDescription = false
	return delegate
}
