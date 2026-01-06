package tui

import (
	"fmt"
	"netdis/internal/storage"
	"netdis/internal/tui/styles"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type StatsModel struct {
	store *storage.Storage
	stats map[string]interface{}
	err   error
}

func NewStatsModel(store *storage.Storage) *StatsModel {
	return &StatsModel{
		store: store,
	}
}

func (m *StatsModel) Init() tea.Cmd {
	return m.loadStats()
}

func (m *StatsModel) loadStats() tea.Cmd {
	return func() tea.Msg {
		stats, err := m.store.GetStats()
		if err != nil {
			m.err = err
			return nil
		}
		m.stats = stats
		return nil
	}
}

func (m *StatsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			return m, func() tea.Msg {
				return NavigateMsg{View: ViewMenu}
			}
		case "r":
			return m, m.loadStats()
		}
	}
	return m, nil
}

func (m *StatsModel) View() string {
	var s strings.Builder

	s.WriteString(styles.TitleStyle.Render("📊 Database Statistics"))
	s.WriteString("\n\n")

	if m.err != nil {
		s.WriteString(styles.ErrorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		s.WriteString("\n\n")
		s.WriteString(styles.RenderHelp("Press Esc to return to menu"))
		return s.String()
	}

	if len(m.stats) == 0 {
		s.WriteString(styles.WarningStyle.Render("No statistics available"))
		s.WriteString("\n\n")
		s.WriteString(styles.RenderHelp("Press Esc to return to menu"))
		return s.String()
	}

	s.WriteString(m.renderStatsGrid())
	s.WriteString("\n\n")

	s.WriteString(styles.RenderHelp("r: Refresh • Esc: Back to Menu"))

	return s.String()
}

func (m *StatsModel) renderStatsGrid() string {
	var items []string

	statOrder := []struct {
		key   string
		label string
		icon  string
	}{
		{"total-scans", "Total Scans", "📊"},
		{"total-devices", "Total Devices", "📱"},
		{"unique-devices", "Unique Devices", "🔢"},
		{"total-networks", "Networks Scanned", "🌐"},
		{"oldest-scan", "Oldest Scan", "📅"},
		{"newest-scan", "Newest Scan", "🆕"},
	}

	for _, stat := range statOrder {
		if val, ok := m.stats[stat.key]; ok {
			item := m.renderStatItem(stat.icon, stat.label, fmt.Sprintf("%v", val))
			items = append(items, item)
		}
	}

	var rows []string
	for i := 0; i < len(items); i += 2 {
		if i+1 < len(items) {
			rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, items[i], "  ", items[i+1]))
		} else {
			rows = append(rows, items[i])
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func (m *StatsModel) renderStatItem(icon, label, value string) string {
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Border).
		Padding(1, 2).
		Width(35)

	iconStyle := lipgloss.NewStyle().
		Foreground(styles.Primary).
		Bold(true)

	labelStyle := lipgloss.NewStyle().
		Foreground(styles.Muted)

	valueStyle := lipgloss.NewStyle().
		Foreground(styles.Success).
		Bold(true)

	content := fmt.Sprintf("%s %s\n%s",
		iconStyle.Render(icon),
		labelStyle.Render(label),
		valueStyle.Render(value))

	return boxStyle.Render(content)
}
