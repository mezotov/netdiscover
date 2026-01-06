package tui

import (
	"fmt"
	"netdis/internal/model"
	"netdis/internal/storage"
	"netdis/internal/tui/styles"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type HistoryModel struct {
	store    *storage.Storage
	history  []model.ScanResult
	cursor   int
	selected int
	err      error
}

func NewHistoryModel(store *storage.Storage) *HistoryModel {
	return &HistoryModel{
		store:    store,
		cursor:   0,
		selected: -1,
	}
}

func (m *HistoryModel) Init() tea.Cmd {
	return m.loadHistory()
}

func (m *HistoryModel) loadHistory() tea.Cmd {
	return func() tea.Msg {
		history, err := m.store.GetScanHistory(50)
		if err != nil {
			m.err = err
			return nil
		}
		m.history = history
		return nil
	}
}

func (m *HistoryModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			if m.selected >= 0 {
				m.selected = -1
				return m, nil
			}
			return m, func() tea.Msg {
				return NavigateMsg{View: ViewMenu}
			}
		case "up", "k":
			if m.selected < 0 && m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.selected < 0 && m.cursor < len(m.history)-1 {
				m.cursor++
			}
		case "enter", " ":
			if m.selected < 0 && len(m.history) > 0 {
				m.selected = m.cursor
			}
		case "r":
			return m, m.loadHistory()
		}
	}
	return m, nil
}

func (m *HistoryModel) View() string {
	var s strings.Builder

	s.WriteString(styles.TitleStyle.Render("📜 Scan History"))
	s.WriteString("\n\n")

	if m.err != nil {
		s.WriteString(styles.ErrorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		s.WriteString("\n\n")
		s.WriteString(styles.RenderHelp("Press Esc to return to menu"))
		return s.String()
	}

	if len(m.history) == 0 {
		s.WriteString(styles.WarningStyle.Render("No scan history found"))
		s.WriteString("\n\n")
		s.WriteString(styles.RenderHelp("Press Esc to return to menu"))
		return s.String()
	}

	if m.selected >= 0 {
		scan := m.history[m.selected]
		s.WriteString(styles.SubtitleStyle.Render(fmt.Sprintf("Scan #%d Details", scan.ID)))
		s.WriteString("\n\n")

		s.WriteString(styles.InfoStyle.Render(fmt.Sprintf("Time: %s", scan.TimeStamp.Format("2006-01-02 15:04:05"))))
		s.WriteString("\n")
		s.WriteString(styles.InfoStyle.Render(fmt.Sprintf("Network: %s", scan.Network)))
		s.WriteString("\n")
		s.WriteString(styles.InfoStyle.Render(fmt.Sprintf("Interface: %s", scan.Interface)))
		s.WriteString("\n")
		s.WriteString(styles.InfoStyle.Render(fmt.Sprintf("Duration: %s", scan.Duration)))
		s.WriteString("\n")
		s.WriteString(styles.InfoStyle.Render(fmt.Sprintf("Devices Found: %d", scan.Total)))
		s.WriteString("\n\n")

		if len(scan.Devices) > 0 {
			s.WriteString(m.renderDeviceTable(scan.Devices))
		}

		s.WriteString("\n\n")
		s.WriteString(styles.RenderHelp("Esc: Back to List"))
	} else {
		s.WriteString(styles.SubtitleStyle.Render(fmt.Sprintf("Last %d scans:", len(m.history))))
		s.WriteString("\n\n")

		header := fmt.Sprintf("%-5s %-20s %-18s %-15s %-10s %-8s",
			"ID", "Timestamp", "Network", "Interface", "Duration", "Devices")
		s.WriteString(styles.TableHeaderStyle.Render(header))
		s.WriteString("\n")

		for i, scan := range m.history {
			row := fmt.Sprintf("%-5d %-20s %-18s %-15s %-10s %-8d",
				scan.ID,
				scan.TimeStamp.Format("2006-01-02 15:04:05"),
				scan.Network,
				truncate(scan.Interface, 15),
				scan.Duration,
				scan.Total)

			if i == m.cursor {
				s.WriteString(styles.SelectedRowStyle.Render(row))
			} else {
				s.WriteString(styles.TableCellStyle.Render(row))
			}
			s.WriteString("\n")
		}

		s.WriteString("\n")
		s.WriteString(styles.RenderHelp("↑/↓: Navigate • Enter: View Details • r: Refresh • Esc: Back"))
	}

	return s.String()
}

func (m *HistoryModel) renderDeviceTable(devices []model.Device) string {
	var s strings.Builder

	header := fmt.Sprintf("%-15s %-17s %-25s %-20s",
		"IP Address", "MAC Address", "Hostname", "Manufacturer")
	s.WriteString(styles.TableHeaderStyle.Render(header))
	s.WriteString("\n")

	limit := len(devices)
	if limit > 15 {
		limit = 15
	}

	for i := 0; i < limit; i++ {
		device := devices[i]
		hostname := device.Hostname
		if hostname == "" {
			hostname = "-"
		}
		mac := device.MAC
		if mac == "" {
			mac = "-"
		}

		row := fmt.Sprintf("%-15s %-17s %-25s %-20s",
			device.IP, mac, truncate(hostname, 25), truncate(device.Manufacturer, 20))
		s.WriteString(styles.TableCellStyle.Render(row))
		s.WriteString("\n")
	}

	if len(devices) > 15 {
		mutedStyle := lipgloss.NewStyle().Foreground(styles.Muted)
		s.WriteString(mutedStyle.Render(fmt.Sprintf("\n... and %d more", len(devices)-15)))
	}

	return s.String()
}
