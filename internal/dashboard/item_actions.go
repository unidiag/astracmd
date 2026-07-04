package dashboard

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
	opt Options,
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

		opt.App.QueueUpdateDraw(func() {
			if err != nil {
				opt.ShowError(err.Error(), nil)
				return
			}

			dashboardSetActionMessage(versionView, "green", donePrefix, itemName)

			if onDone != nil {
				onDone()
			}
		})
	}()
}
