package tui

import (
	"fmt"
	"netdis/internal/model"
	"netdis/internal/storage"
	"netdis/internal/tui/styles"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type SearchModel struct {
	store         *storage.Storage
	ipInput       textinput.Model
	macInput      textinput.Model
	hostnameInput textinput.Model
	vendorInput   textinput.Model
	focusIndex    int
	results       []model.Device
	searching     bool
	err           error
}

func NewSearchModel(store *storage.Storage) *SearchModel {
	ipInput := textinput.New()
	ipInput.Placeholder = "e.g., 192.168.1"
	ipInput.CharLimit = 15

	macInput := textinput.New()
	macInput.Placeholder = "e.g., aa:bb:cc"
	macInput.CharLimit = 17

	hostnameInput := textinput.New()
	hostnameInput.Placeholder = "e.g., laptop"
	hostnameInput.CharLimit = 50

	vendorInput := textinput.New()
	vendorInput.Placeholder = "e.g., Apple"
	vendorInput.CharLimit = 50

	ipInput.Focus()

	return &SearchModel{
		store:         store,
		ipInput:       ipInput,
		macInput:      macInput,
		hostnameInput: hostnameInput,
		vendorInput:   vendorInput,
		focusIndex:    0,
	}
}

func (m *SearchModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *SearchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			return m, func() tea.Msg {
				return NavigateMsg{View: ViewMenu}
			}
		case "tab", "down":
			m.focusIndex = (m.focusIndex + 1) % 4
			return m, m.updateFocus()
		case "shift+tab", "up":
			m.focusIndex--
			if m.focusIndex < 0 {
				m.focusIndex = 3
			}
			return m, m.updateFocus()
		case "enter":
			return m, m.performSearch()
		case "ctrl+r":
			m.clearInputs()
			m.results = nil
			return m, nil
		}
	}

	var cmd tea.Cmd
	switch m.focusIndex {
	case 0:
		m.ipInput, cmd = m.ipInput.Update(msg)
	case 1:
		m.macInput, cmd = m.macInput.Update(msg)
	case 2:
		m.hostnameInput, cmd = m.hostnameInput.Update(msg)
	case 3:
		m.vendorInput, cmd = m.vendorInput.Update(msg)
	}

	return m, cmd
}

func (m *SearchModel) updateFocus() tea.Cmd {
	cmds := make([]tea.Cmd, 4)

	for i := 0; i < 4; i++ {
		if i == m.focusIndex {
			switch i {
			case 0:
				cmds[i] = m.ipInput.Focus()
			case 1:
				cmds[i] = m.macInput.Focus()
			case 2:
				cmds[i] = m.hostnameInput.Focus()
			case 3:
				cmds[i] = m.vendorInput.Focus()
			}
		} else {
			switch i {
			case 0:
				m.ipInput.Blur()
			case 1:
				m.macInput.Blur()
			case 2:
				m.hostnameInput.Blur()
			case 3:
				m.vendorInput.Blur()
			}
		}
	}

	return tea.Batch(cmds...)
}

func (m *SearchModel) performSearch() tea.Cmd {
	return func() tea.Msg {
		filter := model.SearchFilter{
			IP:           m.ipInput.Value(),
			MAC:          m.macInput.Value(),
			Hostname:     m.hostnameInput.Value(),
			Manufacturer: m.vendorInput.Value(),
			Limit:        100,
		}

		devices, err := m.store.SearchDevices(filter)
		if err != nil {
			m.err = err
			return nil
		}

		m.results = devices
		return nil
	}
}

func (m *SearchModel) clearInputs() {
	m.ipInput.SetValue("")
	m.macInput.SetValue("")
	m.hostnameInput.SetValue("")
	m.vendorInput.SetValue("")
}

func (m *SearchModel) View() string {
	var s strings.Builder

	s.WriteString(styles.TitleStyle.Render("🔎 Search Devices"))
	s.WriteString("\n\n")

	s.WriteString(m.renderInput("IP Address:", m.ipInput, 0))
	s.WriteString(m.renderInput("MAC Address:", m.macInput, 1))
	s.WriteString(m.renderInput("Hostname:", m.hostnameInput, 2))
	s.WriteString(m.renderInput("Vendor:", m.vendorInput, 3))

	s.WriteString("\n")

	if m.err != nil {
		s.WriteString(styles.ErrorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		s.WriteString("\n\n")
	}

	if m.results != nil {
		s.WriteString(styles.SubtitleStyle.Render(fmt.Sprintf("Found %d devices:", len(m.results))))
		s.WriteString("\n\n")

		if len(m.results) > 0 {
			s.WriteString(m.renderResultsTable())
		} else {
			s.WriteString(styles.WarningStyle.Render("No devices found matching your criteria"))
		}
		s.WriteString("\n\n")
	}

	s.WriteString(styles.RenderHelp("Tab: Next Field • Enter: Search • Ctrl+R: Clear • Esc: Back"))

	return s.String()
}

func (m *SearchModel) renderInput(label string, input textinput.Model, index int) string {
	var s strings.Builder

	style := styles.BlurredInputStyle
	if m.focusIndex == index {
		style = styles.FocusedInputStyle
	}

	s.WriteString(styles.InfoStyle.Render(label))
	s.WriteString("\n")
	s.WriteString(style.Render(input.View()))
	s.WriteString("\n\n")

	return s.String()
}

func (m *SearchModel) renderResultsTable() string {
	var s strings.Builder

	header := fmt.Sprintf("%-15s %-17s %-25s %-20s",
		"IP Address", "MAC Address", "Hostname", "Manufacturer")
	s.WriteString(styles.TableHeaderStyle.Render(header))
	s.WriteString("\n")

	limit := len(m.results)
	if limit > 20 {
		limit = 20
	}

	for i := 0; i < limit; i++ {
		device := m.results[i]
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

	if len(m.results) > 20 {
		mutedStyle := lipgloss.NewStyle().Foreground(styles.Muted)
		s.WriteString(mutedStyle.Render(fmt.Sprintf("\n... and %d more", len(m.results)-20)))
	}

	return s.String()
}
