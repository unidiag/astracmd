package main

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func dashboardSetOffline(statusView, versionView *tview.TextView, message string) {
	statusView.SetText("[red]OFFLINE[-]")
	versionView.SetText(fmt.Sprintf("[red]%s[-]", tview.Escape(message)))
}

func dashboardSetOnline(statusView, versionView *tview.TextView, version string) {
	statusView.SetText("[green]ONLINE[-]")
	versionView.SetText(fmt.Sprintf("[green]Astra: %s[-]", tview.Escape(version)))
}

func dashboardSetRestarting(statusView, versionView *tview.TextView) {
	statusView.SetText("[yellow]RESTARTING[-]")
	versionView.SetText("[yellow]Astra: restarting...[-]")
}

func dashboardUpdateLogTitle(
	logTable *tview.Table,
	dimmed bool,
	debugLogEnabled bool,
) {
	if dimmed {
		logTable.SetTitle(" Log: Reloading... ")
		return
	}

	if debugLogEnabled {
		logTable.SetTitle(" Log: Debug ON ")
		return
	}

	logTable.SetTitle(" Log: Debug OFF ")
}

func dashboardUpdateBorders(
	adaptersTable *tview.Table,
	streamsTable *tview.Table,
	logTable *tview.Table,
	activePane int,
	dimmed bool,
) {
	activeColor := tcell.ColorYellow
	inactiveColor := tcell.ColorDarkCyan

	if dimmed {
		activeColor = tcell.ColorDarkGray
		inactiveColor = tcell.ColorDarkGray
	}

	adaptersTable.SetBorderColor(inactiveColor)
	streamsTable.SetBorderColor(inactiveColor)
	logTable.SetBorderColor(inactiveColor)

	adaptersTable.SetTitleColor(tcell.ColorWhite)
	streamsTable.SetTitleColor(tcell.ColorWhite)
	logTable.SetTitleColor(tcell.ColorWhite)

	if dimmed {
		adaptersTable.SetTitleColor(tcell.ColorDarkGray)
		streamsTable.SetTitleColor(tcell.ColorDarkGray)
		logTable.SetTitleColor(tcell.ColorDarkGray)
		return
	}

	switch activePane {
	case dashboardPaneAdapters:
		adaptersTable.SetBorderColor(activeColor)

	case dashboardPaneStreams:
		streamsTable.SetBorderColor(activeColor)
	}
}

func dashboardGetLogMaxRows(logTable *tview.Table) int {
	_, _, _, height := logTable.GetInnerRect()
	if height <= 0 {
		return 1
	}

	return height
}
