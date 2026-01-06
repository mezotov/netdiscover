package tui

import (
	"netdis/internal/storage"
	"netdis/internal/tui/styles"

	tea "github.com/charmbracelet/bubbletea"
)

type SettingsModel struct {
	store *storage.Storage
}

func NewSettingsModel(store *storage.Storage) *SettingsModel {
	return &SettingsModel{
		store: store,
	}
}

func (m *SettingsModel) Init() tea.Cmd {
	return nil
}

func (m *SettingsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			return m, func() tea.Msg {
				return NavigateMsg{View: ViewMenu}
			}
		}
	}
	return m, nil
}

func (m *SettingsModel) View() string {
	s := styles.TitleStyle.Render("⚙️  Settings")
	s += "\n\n"
	s += styles.InfoStyle.Render("Settings view coming soon...")
	s += "\n\n"
	s += styles.RenderHelp("Press Esc to return to menu")
	return s
}
