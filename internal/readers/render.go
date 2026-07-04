package readers

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func renderDevices(table *tview.Table, devices []Device, err error) {
	table.Clear()

	table.SetCell(0, 0, tview.NewTableCell(" Device").
		SetTextColor(tcell.ColorYellow).
		SetSelectable(false).
		SetExpansion(1))

	table.SetCell(0, 1, tview.NewTableCell(" Target").
		SetTextColor(tcell.ColorYellow).
		SetSelectable(false).
		SetExpansion(1))

	table.SetCell(0, 2, tview.NewTableCell(" Process").
		SetTextColor(tcell.ColorYellow).
		SetSelectable(false).
		SetExpansion(1))

	if err != nil {
		table.SetCell(1, 0, tview.NewTableCell(" Error").
			SetTextColor(tcell.ColorRed).
			SetExpansion(1))

		table.SetCell(1, 1, tview.NewTableCell(fmt.Sprintf(" %s", err.Error())).
			SetTextColor(tcell.ColorRed).
			SetExpansion(1))

		table.SetCell(1, 2, tview.NewTableCell("").
			SetTextColor(tcell.ColorRed).
			SetExpansion(1))

		return
	}

	if len(devices) == 0 {
		table.SetCell(1, 0, tview.NewTableCell(" No devices found").
			SetTextColor(tcell.ColorGray).
			SetExpansion(1))

		table.SetCell(1, 1, tview.NewTableCell(" "+serialByIDPath).
			SetTextColor(tcell.ColorGray).
			SetExpansion(1))

		table.SetCell(1, 2, tview.NewTableCell("").
			SetTextColor(tcell.ColorGray).
			SetExpansion(1))

		return
	}

	for i, device := range devices {
		row := i + 1

		targetText := device.Target
		targetColor := tcell.ColorGreen
		processText := "-"

		if device.Busy {
			targetColor = tcell.ColorYellow
			processText = fmt.Sprintf("%s (%d)", device.ProcessName, device.ProcessPID)
		}

		table.SetCell(row, 0, tview.NewTableCell(" "+displayDeviceName(device.Name)).
			SetTextColor(tcell.ColorWhite).
			SetExpansion(1))

		table.SetCell(row, 1, tview.NewTableCell(" "+targetText).
			SetTextColor(targetColor).
			SetExpansion(1))

		table.SetCell(row, 2, tview.NewTableCell(" "+processText).
			SetTextColor(targetColor).
			SetExpansion(1))
	}
}
