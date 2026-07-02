package main

import (
	"context"
	"fmt"

	"github.com/rivo/tview"
)

func dashboardSetActionMessage(
	versionView *tview.TextView,
	color string,
	prefix string,
	name string,
) {
	if versionView == nil {
		return
	}

	versionView.SetText(fmt.Sprintf(
		"[%s]%s: %s[-]",
		color,
		tview.Escape(prefix),
		tview.Escape(name),
	))
}

func dashboardRunAsyncItemAction(
	ui *UI,
	versionView *tview.TextView,
	startPrefix string,
	donePrefix string,
	itemName string,
	action func(context.Context) error,
	onDone func(),
) {
	dashboardSetActionMessage(versionView, "yellow", startPrefix, itemName)

	go func() {
		err := action(context.Background())

		ui.app.QueueUpdateDraw(func() {
			if err != nil {
				ui.ShowError(err.Error(), nil)
				return
			}

			dashboardSetActionMessage(versionView, "green", donePrefix, itemName)

			if onDone != nil {
				onDone()
			}
		})
	}()
}
