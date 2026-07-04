package dashboard

import (
	"context"
	"fmt"
	"main/internal/astra"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func ConfirmRestartAstra(opt Options, conn astra.Connection, onOK func(), onError func(error)) {
	modal := tview.NewModal()
	modal.SetText("Are you sure restart Astra?")
	modal.AddButtons([]string{"Restart", "Cancel"})

	modal.SetDoneFunc(func(_ int, label string) {
		if label != "Restart" {
			opt.Pages.RemovePage(PageDialog)
			return
		}

		opt.Pages.RemovePage(PageDialog)

		go func() {
			client := astra.NewClient(conn)
			err := dashboardRestartAstra(context.Background(), client)

			opt.App.QueueUpdateDraw(func() {
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

	opt.Pages.AddPage(PageDialog, centerPrimitive(modal, 50, 10), true, true)
	opt.App.SetFocus(modal)
}

func ShowLicenseDialog(opt Options, conn astra.Connection, onOK func(), onError func(error)) {
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
		if !CanChangeAstraConfig() {
			opt.ShowError(
				"Restricted mode: run astracmd as root to change Astra license",
				form,
			)
			return
		}

		license = strings.TrimSpace(license)

		if license == "" {
			opt.ShowError("license is required", form)
			return
		}

		if !astra.IsValidLicense(license) {
			opt.ShowError("license must be exactly 32 hex characters", form)
			return
		}

		opt.Pages.RemovePage(PageDialog)

		go func() {
			client := astra.NewClient(conn)
			result := client.SetLicense(context.Background(), license)

			opt.App.QueueUpdateDraw(func() {
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
		opt.Pages.RemovePage(PageDialog)
	})

	if !CanChangeAstraConfig() {
		applyButton := form.GetButton(0)
		if applyButton != nil {
			applyButton.SetLabelColor(tcell.ColorBlack)
			applyButton.SetBackgroundColor(tcell.ColorGray)
			applyButton.SetLabelColorActivated(tcell.ColorBlack)
			applyButton.SetBackgroundColorActivated(tcell.ColorGray)
		}
	}

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

	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
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

	opt.Pages.AddPage(PageDialog, centerPrimitive(body, 76, 11), true, true)
	opt.App.SetFocus(form)

	go func() {
		client := astra.NewClient(conn)
		status, err := dashboardLoadAstraStatus(context.Background(), client)

		opt.App.QueueUpdateDraw(func() {
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

func formatUnixTime(ts int64) string {
	if ts <= 0 {
		return "unknown"
	}

	return time.Unix(ts, 0).Format("02.01.2006 15:04:05")
}
