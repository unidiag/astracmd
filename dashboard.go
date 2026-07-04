package main

import (
	"context"
	"fmt"

	"main/internal/astra"
	"main/internal/dashboard"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type dashboardConfigStore struct {
	cfg *Config
}

type FunctionKeyAction struct {
	Label  string
	Handle func()
}

func (s dashboardConfigStore) Save() error {
	if s.cfg == nil {
		return nil
	}

	return s.cfg.Save()
}

func (s dashboardConfigStore) UpdateConnectionDebug(id string, enabled bool) {
	if s.cfg == nil {
		return
	}

	for i := range s.cfg.Connections {
		if s.cfg.Connections[i].ID == id {
			s.cfg.Connections[i].Debug = enabled
			return
		}
	}
}

func (ui *UI) SetDashboardCancel(cancel context.CancelFunc) {
	ui.dashboardCancel = cancel
}

func (ui *UI) ShowDashboard(conn astra.Connection) {
	dashboard.Show(ui.dashboardOptions(conn))
}

func (ui *UI) NewFunctionKeyBar(actions map[int]FunctionKeyAction) *tview.Table {
	table := tview.NewTable()
	table.SetSelectable(true, false)
	table.SetEvaluateAllRows(true)

	for i := 1; i <= 10; i++ {
		action, active := actions[i]

		label := ""
		if active {
			label = action.Label
		}

		buttonCol := (i - 1) * 2
		spaceCol := buttonCol + 1

		text := fmt.Sprintf(" F%d %-7s", i, label)

		cell := tview.NewTableCell(text).
			SetAlign(tview.AlignCenter).
			SetExpansion(1).
			SetSelectable(active)

		if active {
			if i == 2 {
				cell.SetTextColor(tcell.ColorWhite)
				cell.SetBackgroundColor(tcell.ColorRed)
				cell.SetSelectedStyle(
					tcell.StyleDefault.
						Foreground(tcell.ColorWhite).
						Background(tcell.ColorRed),
				)
			} else {
				cell.SetTextColor(tcell.ColorWhite)
				cell.SetBackgroundColor(tcell.ColorBlack)
			}
		} else {
			cell.SetTextColor(tcell.NewRGBColor(95, 95, 95)) // gray color
			cell.SetBackgroundColor(tcell.ColorBlack)
		}

		table.SetCell(0, buttonCol, cell)

		if i < 10 {
			spaceCell := tview.NewTableCell(" ").
				SetTextColor(tcell.ColorBlack).
				SetBackgroundColor(tcell.ColorBlack).
				SetExpansion(0).
				SetSelectable(false)

			table.SetCell(0, spaceCol, spaceCell)
		}
	}

	table.SetSelectedFunc(func(_ int, col int) {
		if col%2 != 0 {
			return
		}

		key := col/2 + 1

		action, ok := actions[key]
		if !ok || action.Handle == nil {
			return
		}

		action.Handle()
	})

	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if ui.HandleGlobalKeys(event) {
			return nil
		}

		if event.Key() == tcell.KeyF1 {
			if action, ok := actions[1]; ok && action.Handle != nil {
				action.Handle()
				return nil
			}
		}

		return event
	})

	return table
}

func (ui *UI) dashboardOptions(conn astra.Connection) dashboard.Options {
	return dashboard.Options{
		App:        ui.app,
		Pages:      ui.pages,
		Connection: conn,
		Config: dashboardConfigStore{
			cfg: ui.cfg,
		},

		Debug: debug,

		SetMain: ui.setMain,

		ShowError:       ui.ShowError,
		ShowConnections: ui.ShowConnections,
		Quit:            ui.Quit,

		StopDashboardTimer: ui.StopDashboardTimer,
		SetDashboardCancel: ui.SetDashboardCancel,

		HandleGlobalKeys: ui.HandleGlobalKeys,

		NewFunctionKeyBar: func(actions map[int]dashboard.FunctionKeyAction) tview.Primitive {
			converted := make(map[int]FunctionKeyAction, len(actions))

			for key, action := range actions {
				converted[key] = FunctionKeyAction{
					Label:  action.Label,
					Handle: action.Handle,
				}
			}

			return ui.NewFunctionKeyBar(converted)
		},
	}
}

func (s dashboardConfigStore) ServiceProvider() string {
	if s.cfg == nil {
		return ""
	}

	return s.cfg.ServiceProvider()
}
