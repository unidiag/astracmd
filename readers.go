package main

import (
	"main/internal/dashboard"
	"main/internal/readers"
)

func (ui *UI) ShowReaders() {
	if !dashboard.CanChangeConnections() {
		ui.ShowRestrictedConnectionsError(ui.app.GetFocus())
		return
	}

	readers.Show(readers.Options{
		App:              ui.app,
		Pages:            ui.pages,
		PageName:         pageDialog,
		ConfigPath:       ui.cfg.Path,
		HandleGlobalKeys: ui.HandleGlobalKeys,
	})
}
