package tui

import (
	"netdis/internal/tui/styles"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type MenuModel struct {
	choices  []string
	cursor   int
	selected int
}

func NewMenuModel() *MenuModel {
	return &MenuModel{
		choices: []string{
			"🔍 Scan Network",
			"🔎 Search Devices",
			"📜 View History",
			"📊 Statistics",
			"⚙️  Settings",
			"❌ Exit",
		},
		cursor:   0,
		selected: -1,
	}
}

func (m *MenuModel) Init() tea.Cmd {
	return nil
}

func (m *MenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}
		case "enter", " ":
			m.selected = m.cursor
			switch m.cursor {
			case 0:
				return m, func() tea.Msg {
					return NavigateMsg{View: ViewScan}
				}
			case 1:
				return m, func() tea.Msg {
					return NavigateMsg{View: ViewSearch}
				}
			case 2:
				return m, func() tea.Msg {
					return NavigateMsg{View: ViewHistory}
				}
			case 3:
				return m, func() tea.Msg {
					return NavigateMsg{View: ViewStats}
				}
			case 4:
				return m, func() tea.Msg {
					return NavigateMsg{View: ViewSettings}
				}
			case 5:
				return m, tea.Quit
			}
		}
	}
	return m, nil
}

func (m *MenuModel) View() string {
	banner := lipgloss.NewStyle().
		Foreground(styles.Primary).
		Bold(true).
		Render(`
 ███╗   ██╗███████╗████████╗██████╗ ██╗███████╗ ██████╗ ██████╗ ██╗   ██╗███████╗██████╗ 
 ████╗  ██║██╔════╝╚══██╔══╝██╔══██╗██║██╔════╝██╔════╝██╔═══██╗██║   ██║██╔════╝██╔══██╗
 ██╔██╗ ██║█████╗     ██║   ██║  ██║██║███████╗██║     ██║   ██║██║   ██║█████╗  ██████╔╝
 ██║╚██╗██║██╔══╝     ██║   ██║  ██║██║╚════██║██║     ██║   ██║╚██╗ ██╔╝██╔══╝  ██╔══██╗
 ██║ ╚████║███████╗   ██║   ██████╔╝██║███████║╚██████╗╚██████╔╝ ╚████╔╝ ███████╗██║  ██║
 ╚═╝  ╚═══╝╚══════╝   ╚═╝   ╚═════╝ ╚═╝╚══════╝ ╚═════╝ ╚═════╝   ╚═══╝  ╚══════╝╚═╝  ╚═╝
`)

	subtitle := styles.SubtitleStyle.Render("Network Device Discovery Tool - Interactive TUI")

	s := banner + "\n" + subtitle + "\n\n"

	for i, choice := range m.choices {
		cursor := "  "
		if m.cursor == i {
			cursorStyle := lipgloss.NewStyle().Foreground(styles.Primary)
			cursor = cursorStyle.Render("▶ ")
			s += styles.SelectedMenuItemStyle.Render(cursor+choice) + "\n"
		} else {
			s += styles.MenuItemStyle.Render(cursor+choice) + "\n"
		}
	}

	s += "\n" + styles.RenderHelp("↑/↓: Navigate • Enter: Select • q: Quit")

	return lipgloss.Place(
		80, 30,
		lipgloss.Center, lipgloss.Center,
		s,
	)
}
