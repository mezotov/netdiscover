package tui

import (
	"context"
	"fmt"
	"net"
	"netdis/internal/model"
	"netdis/internal/network"
	"netdis/internal/scanner"
	"netdis/internal/storage"
	"netdis/internal/tui/styles"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ScanModel struct {
	store              *storage.Storage
	scanning           bool
	devices            []model.Device
	spinner            spinner.Model
	table              table.Model
	status             string
	err                error
	iface              *net.Interface
	network            *net.IPNet
	detectServices     bool
	watchMode          bool
	scanCount          int
	startTime          time.Time
	ctx                context.Context
	cancel             context.CancelFunc
	selectedOption     int
	showingOptions     bool
	selectingInterface bool
	availableIfaces    []network.InterfaceInfo
	selectedIface      int
}

func NewScanModel(store *storage.Storage) *ScanModel {
	s := spinner.New()
	s.Spinner = spinner.Globe
	s.Style = lipgloss.NewStyle().Foreground(styles.Primary)

	columns := []table.Column{
		{Title: "IP Address", Width: 15},
		{Title: "MAC Address", Width: 17},
		{Title: "Hostname", Width: 25},
		{Title: "Manufacturer", Width: 20},
		{Title: "Status", Width: 10},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(20),
	)

	sStyle := table.DefaultStyles()
	sStyle.Header = sStyle.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true)
	sStyle.Selected = sStyle.Selected.
		Foreground(styles.Primary).
		Bold(true)
	t.SetStyles(sStyle)

	return &ScanModel{
		store:          store,
		spinner:        s,
		table:          t,
		status:         "Ready to scan",
		detectServices: false,
		watchMode:      false,
		showingOptions: true,
	}
}

func (m *ScanModel) Init() tea.Cmd {
	ifaces, err := network.GetAllInterfaces()
	if err != nil {
		m.err = err
		return nil
	}
	m.availableIfaces = ifaces
	m.selectingInterface = true
	m.status = "Select a network interface to scan"

	return m.spinner.Tick
}

type scanCompleteMsg struct {
	devices []model.Device
	err     error
}

type deviceFoundMsg struct {
	device model.Device
}

func (m *ScanModel) startScan() tea.Cmd {
	return func() tea.Msg {
		m.ctx, m.cancel = context.WithCancel(context.Background())

		s := scanner.New(m.network, m.detectServices)
		devices := s.Scan()

		return scanCompleteMsg{
			devices: devices,
			err:     nil,
		}
	}
}

func (m *ScanModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.selectingInterface {
			switch msg.String() {
			case "up", "k":
				if m.selectedIface > 0 {
					m.selectedIface--
				}
			case "down", "j":
				if m.selectedIface < len(m.availableIfaces)-1 {
					m.selectedIface++
				}
			case "enter", " ":
				selected := m.availableIfaces[m.selectedIface]
				m.iface = selected.Interface
				m.network = selected.IPNet
				m.selectingInterface = false
				m.showingOptions = true
				m.status = fmt.Sprintf("Ready to scan %s on %s", selected.Network, selected.Interface.Name)
			case "esc", "q":
				return m, func() tea.Msg {
					return NavigateMsg{View: ViewMenu}
				}
			}
		} else if m.showingOptions {
			switch msg.String() {
			case "up", "k":
				if m.selectedOption > 0 {
					m.selectedOption--
				}
			case "down", "j":
				if m.selectedOption < 2 {
					m.selectedOption++
				}
			case "enter", " ":
				switch m.selectedOption {
				case 0:
					m.showingOptions = false
					m.scanning = true
					m.startTime = time.Now()
					m.scanCount++
					m.status = "Scanning network..."
					return m, tea.Batch(m.spinner.Tick, m.startScan())
				case 1:
					m.detectServices = !m.detectServices
				case 2:
					m.watchMode = !m.watchMode
				}
			case "esc", "q":
				if m.scanning && m.cancel != nil {
					m.cancel()
					m.scanning = false
					m.status = "Scan cancelled"
				}
				return m, func() tea.Msg {
					return NavigateMsg{View: ViewMenu}
				}
			}
		} else {
			switch msg.String() {
			case "esc", "q":
				if m.scanning && m.cancel != nil {
					m.cancel()
					m.scanning = false
					m.status = "Scan cancelled"
				} else {
					return m, func() tea.Msg {
						return NavigateMsg{View: ViewMenu}
					}
				}
			case "r":
				if !m.scanning {
					m.showingOptions = true
					m.devices = nil
					m.table.SetRows([]table.Row{})
				}
			}
			m.table, cmd = m.table.Update(msg)
			return m, cmd
		}

	case scanCompleteMsg:
		m.scanning = false
		if msg.err != nil {
			m.err = msg.err
			m.status = "Scan failed"
		} else {
			m.devices = msg.devices
			duration := time.Since(m.startTime)
			m.status = fmt.Sprintf("Scan completed in %v - Found %d devices", duration, len(msg.devices))

			rows := make([]table.Row, len(msg.devices))
			for i, device := range msg.devices {
				hostname := device.Hostname
				if hostname == "" {
					hostname = "-"
				}
				mac := device.MAC
				if mac == "" {
					mac = "-"
				}
				rows[i] = table.Row{
					device.IP,
					mac,
					hostname,
					device.Manufacturer,
					device.Status,
				}
			}
			m.table.SetRows(rows)

			result := model.ScanResult{
				TimeStamp: time.Now(),
				Network:   m.network.String(),
				Interface: m.iface.Name,
				Duration:  duration.String(),
				Total:     len(msg.devices),
				Devices:   msg.devices,
			}
			err := m.store.SaveScanResult(result)
			if err != nil {
				m.err = err
				m.status += " (Failed to save scan result)"
			}

			if m.watchMode {
				time.Sleep(30 * time.Second)
				return m, m.startScan()
			}
		}
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *ScanModel) View() string {
	var s strings.Builder

	s.WriteString(styles.TitleStyle.Render("🔍 Network Scanner"))
	s.WriteString("\n\n")

	if m.err != nil {
		s.WriteString(styles.ErrorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		s.WriteString("\n\n")
		s.WriteString(styles.RenderHelp("Press Esc to return to menu"))
		return s.String()
	}

	if m.selectingInterface {
		s.WriteString(styles.SubtitleStyle.Render("Select Network Interface:"))
		s.WriteString("\n\n")

		if len(m.availableIfaces) == 0 {
			s.WriteString(styles.WarningStyle.Render("No network interfaces found"))
			s.WriteString("\n\n")
			s.WriteString(styles.RenderHelp("Press Esc to return to menu"))
			return s.String()
		}

		for i, iface := range m.availableIfaces {
			ifaceInfo := fmt.Sprintf("%s - %s (%s)",
				iface.Interface.Name,
				iface.IP,
				iface.Network)

			if i == m.selectedIface {
				s.WriteString(styles.SelectedMenuItemStyle.Render("▶ " + ifaceInfo))
			} else {
				s.WriteString(styles.MenuItemStyle.Render("  " + ifaceInfo))
			}
			s.WriteString("\n")
		}

		s.WriteString("\n")
		s.WriteString(styles.RenderHelp("↑/↓: Navigate • Enter: Select • Esc: Back"))
		return s.String()
	}

	if m.network != nil {
		s.WriteString(styles.InfoStyle.Render(fmt.Sprintf("Network: %s", m.network.String())))
		s.WriteString("\n")
		s.WriteString(styles.InfoStyle.Render(fmt.Sprintf("Interface: %s", m.iface.Name)))
		s.WriteString("\n\n")
	}

	if m.showingOptions {
		s.WriteString(styles.SubtitleStyle.Render("Scan Options:"))
		s.WriteString("\n\n")

		options := []string{
			"▶ Start Scan",
			fmt.Sprintf("Service Detection: %s", boolToStatus(m.detectServices)),
			fmt.Sprintf("Watch Mode: %s", boolToStatus(m.watchMode)),
		}

		for i, opt := range options {
			if i == m.selectedOption {
				s.WriteString(styles.SelectedMenuItemStyle.Render("▶ " + opt))
			} else {
				s.WriteString(styles.MenuItemStyle.Render("  " + opt))
			}
			s.WriteString("\n")
		}

		s.WriteString("\n")
		s.WriteString(styles.RenderHelp("↑/↓: Navigate • Enter: Select/Toggle • Esc: Back"))
	} else {
		if m.scanning {
			s.WriteString(m.spinner.View() + " " + m.status)
			s.WriteString("\n\n")
		} else {
			s.WriteString(styles.SuccessStyle.Render(m.status))
			s.WriteString("\n\n")
		}

		if len(m.devices) > 0 {
			s.WriteString(styles.BaseStyle.Render(m.table.View()))
			s.WriteString("\n\n")
		} else if !m.scanning {
			s.WriteString(styles.WarningStyle.Render("No devices found"))
			s.WriteString("\n\n")
		}

		if !m.scanning {
			s.WriteString(styles.RenderHelp("r: New Scan • Esc: Back to Menu"))
		} else {
			s.WriteString(styles.RenderHelp("Esc: Cancel Scan"))
		}
	}

	return s.String()
}

func boolToStatus(b bool) string {
	if b {
		return styles.SuccessStyle.Render("ON")
	}
	mutedStyle := lipgloss.NewStyle().Foreground(styles.Muted)
	return mutedStyle.Render("OFF")
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
