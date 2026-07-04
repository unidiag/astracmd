package main

import (
	"context"
	"fmt"
	"main/internal/astra"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (ui *UI) ConfirmRestartAstra(conn astra.AstraConnection, onOK func(), onError func(error)) {
	modal := tview.NewModal()
	modal.SetText("Are you sure restart Astra?")
	modal.AddButtons([]string{"Restart", "Cancel"})

	modal.SetDoneFunc(func(_ int, label string) {
		if label != "Restart" {
			ui.pages.RemovePage(pageDialog)
			return
		}

		ui.pages.RemovePage(pageDialog)

		go func() {
			client := astra.NewAstraClient(conn)
			err := dashboardRestartAstra(context.Background(), client)

			ui.app.QueueUpdateDraw(func() {
				if err == nil {
					if onOK != nil {
						onOK()
					}
					return
				}

				if onError != nil {
					onError(err)
				}
			})
		}()
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

	ui.pages.AddPage(pageDialog, centerPrimitive(modal, 50, 10), true, true)
	ui.app.SetFocus(modal)
}

func (ui *UI) ShowLicenseDialog(conn astra.AstraConnection, onOK func(), onError func(error)) {
	license := ""

	emailView := tview.NewTextView()
	emailView.SetDynamicColors(true)
	emailView.SetTextAlign(tview.AlignLeft)
	emailView.SetWrap(false)
	emailView.SetText(" [gray]Email: loading...[-]")

	expireView := tview.NewTextView()
	expireView.SetDynamicColors(true)
	expireView.SetTextAlign(tview.AlignLeft)
	expireView.SetWrap(false)
	expireView.SetText(" [gray]Expire: loading...[-]")

	form := tview.NewForm()
	form.SetButtonsAlign(tview.AlignCenter)

	licenseField := tview.NewInputField()
	licenseField.SetLabel("License")
	licenseField.SetFieldWidth(42)
	licenseField.SetChangedFunc(func(value string) {
		license = value
	})

	form.AddFormItem(licenseField)

	form.AddButton("Apply", func() {
		license = strings.TrimSpace(license)

		if license == "" {
			ui.ShowError("license is required", form)
			return
		}

		if !astra.IsValidLicense(license) {
			ui.ShowError("license must be exactly 32 hex characters", form)
			return
		}

		ui.pages.RemovePage(pageDialog)

		go func() {
			client := astra.NewAstraClient(conn)
			result := client.SetLicense(context.Background(), license)

			ui.app.QueueUpdateDraw(func() {
				if result.OK {
					if onOK != nil {
						onOK()
					}
					return
				}

				if result.Err != nil && onError != nil {
					onError(result.Err)
				}
			})
		}()
	})

	form.AddButton("Cancel", func() {
		ui.pages.RemovePage(pageDialog)
	})

	body := tview.NewFlex()
	body.SetDirection(tview.FlexRow)
	body.SetBorder(true)
	body.SetTitle(" Astra license ")
	body.SetTitleAlign(tview.AlignCenter)

	spacer := tview.NewBox()

	body.AddItem(spacer, 1, 0, false)
	body.AddItem(emailView, 1, 0, false)
	body.AddItem(expireView, 1, 0, false)
	body.AddItem(form, 6, 0, true)

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

	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
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

	ui.pages.AddPage(pageDialog, centerPrimitive(body, 76, 11), true, true)
	ui.app.SetFocus(form)

	go func() {
		client := astra.NewAstraClient(conn)
		status, err := dashboardLoadAstraStatus(context.Background(), client)

		ui.app.QueueUpdateDraw(func() {
			if err != nil {
				emailView.SetText(fmt.Sprintf(
					"[red]Email: %s[-]",
					tview.Escape(err.Error()),
				))
				expireView.SetText(" [red]Expire: unknown[-]")
				return
			}

			lic := status.License

			license = strings.TrimSpace(lic.ID)
			licenseField.SetText(license)

			email := strings.TrimSpace(lic.Email)
			if email == "" {
				email = "unknown"
			}

			emailView.SetText(fmt.Sprintf(
				" [white]Email: [-][green]%s[-]",
				tview.Escape(email),
			))

			expireView.SetText(fmt.Sprintf(
				" [white]Expire: [-][green]%s[-]",
				tview.Escape(formatUnixTime(lic.Expire)),
			))
		})
	}()
}
