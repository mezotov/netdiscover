package tui

import (
	"netdis/internal/storage"

	tea "github.com/charmbracelet/bubbletea"
)

type ViewState int

const (
	ViewMenu ViewState = iota
	ViewScan
	ViewSearch
	ViewHistory
	ViewStats
	ViewSettings
)

type App struct {
	store       *storage.Storage
	currentView ViewState
	menu        *MenuModel
	scan        *ScanModel
	search      *SearchModel
	history     *HistoryModel
	stats       *StatsModel
	width       int
	height      int
	err         error
}

func NewApp(store *storage.Storage) *App {
	return &App{
		store:       store,
		currentView: ViewMenu,
		menu:        NewMenuModel(),
	}
}

func (a *App) Init() tea.Cmd {
	return nil
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		return a, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if a.currentView == ViewMenu {
				return a, tea.Quit
			}
			a.currentView = ViewMenu
			return a, nil
		}

	case NavigateMsg:
		a.currentView = msg.View
		switch msg.View {
		case ViewScan:
			if a.scan == nil {
				a.scan = NewScanModel(a.store)
			}
			return a, a.scan.Init()
		case ViewSearch:
			if a.search == nil {
				a.search = NewSearchModel(a.store)
			}
			return a, a.search.Init()
		case ViewHistory:
			if a.history == nil {
				a.history = NewHistoryModel(a.store)
			}
			return a, a.history.Init()
		case ViewStats:
			if a.stats == nil {
				a.stats = NewStatsModel(a.store)
			}
			return a, a.stats.Init()
		}
		return a, nil
	}

	var cmd tea.Cmd
	switch a.currentView {
	case ViewMenu:
		var m tea.Model
		m, cmd = a.menu.Update(msg)
		a.menu = m.(*MenuModel)
	case ViewScan:
		if a.scan != nil {
			var m tea.Model
			m, cmd = a.scan.Update(msg)
			a.scan = m.(*ScanModel)
		}
	case ViewSearch:
		if a.search != nil {
			var m tea.Model
			m, cmd = a.search.Update(msg)
			a.search = m.(*SearchModel)
		}
	case ViewHistory:
		if a.history != nil {
			var m tea.Model
			m, cmd = a.history.Update(msg)
			a.history = m.(*HistoryModel)
		}
	case ViewStats:
		if a.stats != nil {
			var m tea.Model
			m, cmd = a.stats.Update(msg)
			a.stats = m.(*StatsModel)
		}
	}

	return a, cmd
}

func (a *App) View() string {
	if a.err != nil {
		return "Error: " + a.err.Error() + "\n\nPress q to quit."
	}

	switch a.currentView {
	case ViewMenu:
		return a.menu.View()
	case ViewScan:
		if a.scan != nil {
			return a.scan.View()
		}
	case ViewSearch:
		if a.search != nil {
			return a.search.View()
		}
	case ViewHistory:
		if a.history != nil {
			return a.history.View()
		}
	case ViewStats:
		if a.stats != nil {
			return a.stats.View()
		}
	}

	return "Loading..."
}

type NavigateMsg struct {
	View ViewState
}
