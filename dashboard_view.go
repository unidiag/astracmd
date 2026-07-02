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
	if logTable == nil {
		return
	}

	title := " LOG "

	if debugLogEnabled {
		title = " DEBUG "
	}

	logTable.SetTitle(title)

	if dimmed {
		logTable.SetTitleColor(tcell.ColorDarkGray)
		return
	}

	logTable.SetTitleColor(tcell.ColorWhite)
}

func dashboardUpdateBorders(
	adaptersTable *tview.Table,
	streamsTable *tview.Table,
	logTable *tview.Table,
	activePane int,
	dimmed bool,
) {
	borderColor := tcell.ColorDarkCyan
	titleColor := tcell.ColorWhite

	if dimmed {
		borderColor = tcell.ColorDarkGray
		titleColor = tcell.ColorDarkGray
	}

	adaptersTable.SetBorderColor(borderColor)
	streamsTable.SetBorderColor(borderColor)
	logTable.SetBorderColor(borderColor)

	adaptersTable.SetTitleColor(titleColor)
	streamsTable.SetTitleColor(titleColor)
	logTable.SetTitleColor(titleColor)
}

func dashboardGetLogMaxRows(logTable *tview.Table) int {
	if logTable == nil {
		return 1
	}

	_, _, _, height := logTable.GetInnerRect()
	if height <= 0 {
		return 1
	}

	return height
}
