package main

import "github.com/gdamore/tcell/v2"

type DashboardKeyActions struct {
	ShowHelp func()
	Restart  func()
	Reload   func()
	Debug    func()
	Delete   func()
	License  func()
	Back     func()
	Quit     func()

	NewItem  func()
	EditItem func()
	OpenItem func()

	RestartItem func()
	SoftCAM     func()

	ToggleStreamMark func()
	MarkAllStreams   func()

	MoveAdapterUp   func()
	MoveAdapterDown func()

	SetAdaptersPane func()
	SetStreamsPane  func()

	GetActivePane func() int
}

func dashboardRunAction(fn func()) {
	if fn != nil {
		fn()
	}
}

func dashboardTogglePane(actions DashboardKeyActions) {
	if actions.GetActivePane == nil {
		return
	}

	switch actions.GetActivePane() {
	case dashboardPaneAdapters:
		dashboardRunAction(actions.SetStreamsPane)

	default:
		dashboardRunAction(actions.SetAdaptersPane)
	}
}

func dashboardTogglePaneBack(actions DashboardKeyActions) {
	if actions.GetActivePane == nil {
		return
	}

	switch actions.GetActivePane() {
	case dashboardPaneStreams:
		dashboardRunAction(actions.SetAdaptersPane)

	default:
		dashboardRunAction(actions.SetStreamsPane)
	}
}

func dashboardHandleKeys(
	event *tcell.EventKey,
	actions DashboardKeyActions,
	handleGlobalKeys func(*tcell.EventKey) bool,
) bool {
	if event == nil {
		return false
	}

	if actions.GetActivePane != nil && actions.GetActivePane() == dashboardPaneAdapters {
		switch event.Key() {
		case tcell.KeyDown:
			dashboardRunAction(actions.MoveAdapterDown)
			return true

		case tcell.KeyUp:
			dashboardRunAction(actions.MoveAdapterUp)
			return true
		}
	}

	switch event.Key() {
	case tcell.KeyCtrlA:
		dashboardRunAction(actions.MarkAllStreams)
		return true

	case tcell.KeyEnter:
		dashboardRunAction(actions.OpenItem)
		return true

	case tcell.KeyF1:
		dashboardRunAction(actions.ShowHelp)
		return true

	case tcell.KeyF2:
		dashboardRunAction(actions.Restart)
		return true

	case tcell.KeyF3:
		dashboardRunAction(actions.SoftCAM)
		return true

	case tcell.KeyF4:
		dashboardRunAction(actions.EditItem)
		return true

	case tcell.KeyF5:
		dashboardRunAction(actions.Reload)
		return true

	case tcell.KeyF6:
		dashboardRunAction(actions.Debug)
		return true

	case tcell.KeyF7:
		dashboardRunAction(actions.NewItem)
		return true

	case tcell.KeyF8, tcell.KeyDelete:
		dashboardRunAction(actions.Delete)
		return true

	case tcell.KeyInsert:
		dashboardRunAction(actions.ToggleStreamMark)
		return true

	case tcell.KeyF9:
		dashboardRunAction(actions.License)
		return true

	case tcell.KeyF10:
		dashboardRunAction(actions.Quit)
		return true

	case tcell.KeyTab:
		dashboardTogglePane(actions)
		return true

	case tcell.KeyBacktab:
		dashboardTogglePaneBack(actions)
		return true
	}

	switch event.Key() {
	case tcell.KeyRune:
		if event.Rune() == ' ' {
			dashboardRunAction(actions.RestartItem)
			return true
		}

	case tcell.KeyEsc:
		dashboardRunAction(actions.Back)
		return true
	}

	if handleGlobalKeys != nil && handleGlobalKeys(event) {
		return true
	}

	return false
}
