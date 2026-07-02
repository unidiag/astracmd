package main

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (ui *UI) ShowHelp() {
	text := tview.NewTextView()
	text.SetDynamicColors(true)
	text.SetScrollable(true)
	text.SetWrap(true)
	text.SetText(HelpText)
	text.SetBorder(true)
	text.SetTitle(" Help ")
	text.SetTitleAlign(tview.AlignCenter)

	footer := tview.NewTextView()
	footer.SetDynamicColors(true)
	footer.SetTextAlign(tview.AlignCenter)
	footer.SetText("[gray]Esc — close · Up/Down/PgUp/PgDn — scroll · F10 — quit[-]")

	body := tview.NewFlex()
	body.SetDirection(tview.FlexRow)
	body.AddItem(text, 0, 1, true)
	body.AddItem(footer, 1, 0, false)

	body.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
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

	text.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
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

	ui.pages.AddPage(pageDialog, centerPrimitive(body, 90, 28), true, true)
	ui.app.SetFocus(text)
}
