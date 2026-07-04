package dashboard

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func ShowDashboardConfirm(
	opt Options,
	text string,
	confirmLabel string,
	width int,
	height int,
	onConfirm func(),
) {
	modal := tview.NewModal()
	modal.SetText(text)
	modal.AddButtons([]string{confirmLabel, "Cancel"})

	modal.SetDoneFunc(func(_ int, label string) {
		opt.Pages.RemovePage(PageDialog)

		if label != confirmLabel {
			return
		}

		if onConfirm != nil {
			onConfirm()
		}
	})

	modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if opt.HandleGlobalKeys(event) {
			return nil
		}

		switch event.Key() {
		case tcell.KeyEsc:
			opt.Pages.RemovePage(PageDialog)
			return nil
		}

		return event
	})

	opt.Pages.AddPage(PageDialog, centerPrimitive(modal, width, height), true, true)
	opt.App.SetFocus(modal)
}
