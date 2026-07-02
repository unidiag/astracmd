package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const (
	pageMain   = "main"
	pageDialog = "dialog"
	pageError  = "error"
)

type UI struct {
	app   *tview.Application
	cfg   *Config
	pages *tview.Pages

	dashboardCancel context.CancelFunc
}

func (ui *UI) StopDashboardTimer() {
	if ui.dashboardCancel != nil {
		ui.dashboardCancel()
		ui.dashboardCancel = nil
	}
}

func NewUI(app *tview.Application, cfg *Config) *UI {
	return &UI{
		app:   app,
		cfg:   cfg,
		pages: tview.NewPages(),
	}
}

func (ui *UI) Quit() {
	ui.StopDashboardTimer()
	ui.app.Stop()
}

func (ui *UI) HandleGlobalKeys(event *tcell.EventKey) bool {
	switch event.Key() {
	case tcell.KeyF10, tcell.KeyCtrlC:
		ui.Quit()
		return true
	}

	return false
}

func (ui *UI) ShowConnections() {
	ui.StopDashboardTimer()

	ui.pages.RemovePage(pageDialog)

	title := tview.NewTextView()
	title.SetDynamicColors(true)
	title.SetTextAlign(tview.AlignCenter)
	title.SetText(fmt.Sprintf(
		"[::b]%s[::-]  v%s\n[gray]BUILD: %s[-]\n[gray]CONFIG: %s[-]",
		APPNAME,
		VERSION,
		BUILD,
		ui.cfg.Path,
	))

	table := tview.NewTable()
	table.SetBorder(true)
	table.SetTitle(" Astra connections ")
	table.SetTitleAlign(tview.AlignCenter)
	table.SetSelectable(true, false)
	table.SetEvaluateAllRows(true)

	for i := range ui.cfg.Connections {
		conn := ui.cfg.Connections[i]

		nameCell := tview.NewTableCell(conn.Name).
			SetTextColor(tcell.ColorWhite).
			SetExpansion(1)

		dsnCell := tview.NewTableCell(conn.DisplayMaskedDSN()).
			SetTextColor(tcell.ColorGreen).
			SetAlign(tview.AlignRight).
			SetExpansion(1)

		table.SetCell(i, 0, nameCell)
		table.SetCell(i, 1, dsnCell)
	}

	newRow := len(ui.cfg.Connections)

	newCell := tview.NewTableCell("(F7) + New connection").
		SetTextColor(tcell.ColorYellow).
		SetExpansion(1)

	newDescCell := tview.NewTableCell("create a new Cesbo Astra connection").
		SetTextColor(tcell.ColorGreen).
		SetAlign(tview.AlignRight).
		SetExpansion(1)

	table.SetCell(newRow, 0, newCell)
	table.SetCell(newRow, 1, newDescCell)

	table.SetSelectedFunc(func(row int, _ int) {
		if row >= 0 && row < len(ui.cfg.Connections) {
			ui.ShowDashboard(ui.cfg.Connections[row])
			return
		}

		if row == len(ui.cfg.Connections) {
			ui.ShowConnectionForm(nil)
			return
		}
	})

	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if ui.HandleGlobalKeys(event) {
			return nil
		}

		switch event.Key() {
		case tcell.KeyEsc:
			ui.Quit()
			return nil

		case tcell.KeyF7:
			ui.ShowConnectionForm(nil)
			return nil

		case tcell.KeyF4:
			row, _ := table.GetSelection()
			if row >= 0 && row < len(ui.cfg.Connections) {
				conn := ui.cfg.Connections[row]
				ui.ShowConnectionForm(&conn)
			}
			return nil

		case tcell.KeyF8, tcell.KeyDelete:
			row, _ := table.GetSelection()
			if row >= 0 && row < len(ui.cfg.Connections) {
				ui.ConfirmDelete(ui.cfg.Connections[row])
			}
			return nil
		}

		return event
	})

	help := tview.NewTextView()
	help.SetDynamicColors(true)
	help.SetTextAlign(tview.AlignCenter)
	help.SetText("[gray]Enter / Left click — open · F4 — edit · F7 — new · F8 — delete · ESC — quit[-]")

	body := tview.NewFlex()
	body.SetDirection(tview.FlexRow)
	body.AddItem(title, 4, 0, false)
	body.AddItem(table, 0, 1, true)
	body.AddItem(help, 2, 0, false)

	ui.setMain(centerPrimitive(body, 96, 24))
	ui.app.SetFocus(table)
}

func (ui *UI) ShowConnectionForm(editConn *AstraConnection) {
	ui.pages.RemovePage(pageDialog)

	isEdit := editConn != nil

	conn := AstraConnection{
		Login:     "admin",
		Interface: "127.0.0.1",
		Port:      8000,
	}

	if isEdit {
		conn = *editConn
	}

	form := tview.NewForm()
	form.SetBorder(true)

	if isEdit {
		form.SetTitle(" Edit connection ")
	} else {
		form.SetTitle(" New connection ")
	}

	form.SetTitleAlign(tview.AlignCenter)

	form.AddInputField("Name", conn.Name, 42, nil, func(value string) {
		conn.Name = value
	})

	form.AddInputField("Login", conn.Login, 42, nil, func(value string) {
		conn.Login = value
	})

	form.AddPasswordField("Password", conn.Password, 42, '*', func(value string) {
		conn.Password = value
	})

	form.AddInputField("Address", conn.Interface, 42, nil, func(value string) {
		conn.Interface = value
	})

	form.AddInputField("Port", strconv.Itoa(conn.Port), 10, onlyPortInput, func(value string) {
		port, _ := strconv.Atoi(strings.TrimSpace(value))
		conn.Port = port
	})

	form.AddButton("Save", func() {
		if err := validateConnection(conn); err != nil {
			ui.ShowError(err.Error(), form)
			return
		}

		if isEdit {
			conn.ID = editConn.ID
		} else {
			conn.ID = ui.cfg.NextID()
		}

		ui.cfg.UpsertConnection(conn)

		if err := ui.cfg.Save(); err != nil {
			ui.ShowError(err.Error(), form)
			return
		}

		ui.ShowConnections()
	})

	form.AddButton("Cancel", func() {
		ui.ShowConnections()
	})

	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if ui.HandleGlobalKeys(event) {
			return nil
		}

		if event.Key() == tcell.KeyEsc {
			ui.ShowConnections()
			return nil
		}

		return event
	})

	ui.setMain(centerPrimitive(form, 76, 18))
	ui.app.SetFocus(form)
}

func onlyPortInput(text string, _ rune) bool {
	text = strings.TrimSpace(text)

	if text == "" {
		return true
	}

	port, err := strconv.Atoi(text)
	if err != nil {
		return false
	}

	return port >= 0 && port <= 65535
}

func validateConnection(conn AstraConnection) error {
	if strings.TrimSpace(conn.Name) == "" {
		return fmt.Errorf("connection name is required")
	}

	if strings.TrimSpace(conn.Login) == "" {
		return fmt.Errorf("login is required")
	}

	if strings.TrimSpace(conn.Interface) == "" {
		return fmt.Errorf("address is required")
	}

	if conn.Port <= 0 || conn.Port > 65535 {
		return fmt.Errorf("invalid port")
	}

	return nil
}

func (ui *UI) ConfirmDelete(conn AstraConnection) {
	modal := tview.NewModal()
	modal.SetText(fmt.Sprintf("Delete connection?\n\n%s\n%s", conn.Name, conn.DisplayMaskedDSN()))
	modal.AddButtons([]string{"Delete", "Cancel"})

	modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if ui.HandleGlobalKeys(event) {
			return nil
		}

		return event
	})

	modal.SetDoneFunc(func(_ int, buttonLabel string) {
		ui.pages.RemovePage(pageDialog)

		if buttonLabel != "Delete" {
			ui.ShowConnections()
			return
		}

		ui.cfg.DeleteConnection(conn.ID)

		if err := ui.cfg.Save(); err != nil {
			ui.ShowError(err.Error(), nil)
			return
		}

		ui.ShowConnections()
	})

	ui.pages.AddPage(pageDialog, modal, false, true)
	ui.app.SetFocus(modal)
}

func (ui *UI) ShowError(text string, returnFocus tview.Primitive) {
	modal := tview.NewModal()
	modal.SetText(text)
	modal.AddButtons([]string{"OK"})

	modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if ui.HandleGlobalKeys(event) {
			return nil
		}

		return event
	})

	modal.SetDoneFunc(func(_ int, _ string) {
		ui.pages.RemovePage(pageError)

		if returnFocus != nil {
			ui.app.SetFocus(returnFocus)
		}
	})

	ui.pages.RemovePage(pageError)
	ui.pages.AddPage(pageError, modal, false, true)
	ui.app.SetFocus(modal)
}

func (ui *UI) setMain(p tview.Primitive) {
	ui.pages.RemovePage(pageMain)
	ui.pages.AddPage(pageMain, p, true, true)
	ui.pages.SwitchToPage(pageMain)
}

func centerPrimitive(p tview.Primitive, width int, height int) tview.Primitive {
	row := tview.NewFlex()
	row.SetDirection(tview.FlexRow)
	row.AddItem(nil, 0, 1, false)
	row.AddItem(p, height, 1, true)
	row.AddItem(nil, 0, 1, false)

	root := tview.NewFlex()
	root.SetDirection(tview.FlexColumn)
	root.AddItem(nil, 0, 1, false)
	root.AddItem(row, width, 1, true)
	root.AddItem(nil, 0, 1, false)

	return root
}
