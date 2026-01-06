package styles

import "github.com/charmbracelet/lipgloss"

var (
	Primary   = lipgloss.Color("#00D9FF")
	Secondary = lipgloss.Color("#7C3AED")
	Success   = lipgloss.Color("#10B981")
	Error     = lipgloss.Color("#EF4444")
	Warning   = lipgloss.Color("#F59E0B")
	Muted     = lipgloss.Color("#6B7280")
	Text      = lipgloss.Color("#F3F4F6")
	Border    = lipgloss.Color("#374151")

	BaseStyle = lipgloss.NewStyle().
			Foreground(Text)

	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Primary).
			MarginBottom(1)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(Secondary).
			Italic(true)

	MenuItemStyle = lipgloss.NewStyle().
			PaddingLeft(2).
			PaddingRight(2).
			MarginBottom(1)

	SelectedMenuItemStyle = MenuItemStyle.Copy().
				Foreground(Primary).
				Bold(true).
				Background(lipgloss.Color("#1F2937"))

	SuccessStyle = lipgloss.NewStyle().
			Foreground(Success).
			Bold(true)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(Error).
			Bold(true)

	WarningStyle = lipgloss.NewStyle().
			Foreground(Warning).
			Bold(true)

	InfoStyle = lipgloss.NewStyle().
			Foreground(Primary)

	BorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Border).
			Padding(1, 2)

	HelpStyle = lipgloss.NewStyle().
			Foreground(Muted).
			Italic(true).
			MarginTop(1)

	TableHeaderStyle = lipgloss.NewStyle().
				Foreground(Primary).
				Bold(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderBottom(true).
				BorderForeground(Border)

	TableCellStyle = lipgloss.NewStyle().
			PaddingLeft(1).
			PaddingRight(1)

	SelectedRowStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#1F2937")).
				Foreground(Primary).
				Bold(true)

	FocusedInputStyle = lipgloss.NewStyle().
				Foreground(Primary).
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(Primary).
				Padding(0, 1)

	BlurredInputStyle = lipgloss.NewStyle().
				Foreground(Muted).
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(Border).
				Padding(0, 1)
)

func RenderTitle(title, subtitle string) string {
	s := TitleStyle.Render(title)
	if subtitle != "" {
		s += "\n" + SubtitleStyle.Render(subtitle)
	}
	return s
}

func RenderHelp(text string) string {
	return HelpStyle.Render(text)
}

func RenderStatus(status string, isError bool) string {
	if isError {
		return ErrorStyle.Render("✗ " + status)
	}
	return SuccessStyle.Render("✓ " + status)
}
