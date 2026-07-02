package main

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (ui *UI) ShowDashboardConfirm(
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
		ui.pages.RemovePage(pageDialog)

		if label != confirmLabel {
			return
		}

		if onConfirm != nil {
			onConfirm()
		}
	})

	modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if ui.HandleGlobalKeys(event) {
			return nil
		}

		switch event.Key() {
		case tcell.KeyEsc:
			ui.pages.RemovePage(pageDialog)
			return nil
		}

		return event
	})

	ui.pages.AddPage(pageDialog, centerPrimitive(modal, width, height), true, true)
	ui.app.SetFocus(modal)
}
