package readers

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const serialByIDPath = "/dev/serial/by-id"

type Options struct {
	App              *tview.Application
	Pages            *tview.Pages
	PageName         string
	HandleGlobalKeys func(*tcell.EventKey) bool
}

type Device struct {
	Name   string
	Path   string
	Target string
}

func Show(opt Options) {
	table := tview.NewTable()
	table.SetBorder(true)
	table.SetTitle(" Readers ")
	table.SetTitleAlign(tview.AlignCenter)
	table.SetSelectable(true, false)
	table.SetEvaluateAllRows(true)

	status := tview.NewTextView()
	status.SetDynamicColors(true)
	status.SetTextAlign(tview.AlignCenter)
	status.SetText("[gray]F5 — reload    Esc — close[-]")

	body := tview.NewFlex()
	body.SetDirection(tview.FlexRow)
	body.AddItem(table, 0, 1, true)
	body.AddItem(status, 1, 0, false)

	closeDialog := func() {
		opt.Pages.RemovePage(opt.PageName)
	}

	load := func() {
		devices, err := ListDevices()
		renderDevices(table, devices, err)
	}

	body.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if opt.HandleGlobalKeys != nil && opt.HandleGlobalKeys(event) {
			return nil
		}

		switch event.Key() {
		case tcell.KeyEsc:
			closeDialog()
			return nil

		case tcell.KeyF5:
			load()
			return nil
		}

		return event
	})

	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if opt.HandleGlobalKeys != nil && opt.HandleGlobalKeys(event) {
			return nil
		}

		switch event.Key() {
		case tcell.KeyEsc:
			closeDialog()
			return nil

		case tcell.KeyF5:
			load()
			return nil
		}

		return event
	})

	load()

	opt.Pages.AddPage(opt.PageName, centerPrimitive(body, 100, 24), true, true)
	opt.App.SetFocus(table)
}

func ListDevices() ([]Device, error) {
	entries, err := os.ReadDir(serialByIDPath)
	if err != nil {
		return nil, err
	}

	devices := make([]Device, 0, len(entries))

	for _, entry := range entries {
		name := entry.Name()
		path := filepath.Join(serialByIDPath, name)

		target := ""

		linkTarget, err := os.Readlink(path)
		if err == nil {
			target = filepath.Clean(filepath.Join(serialByIDPath, linkTarget))
		} else {
			target = err.Error()
		}

		devices = append(devices, Device{
			Name:   name,
			Path:   path,
			Target: target,
		})
	}

	sort.Slice(devices, func(i, j int) bool {
		return devices[i].Name < devices[j].Name
	})

	return devices, nil
}

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

	if err != nil {
		table.SetCell(1, 0, tview.NewTableCell(" Error").
			SetTextColor(tcell.ColorRed).
			SetExpansion(1))

		table.SetCell(1, 1, tview.NewTableCell(fmt.Sprintf(" %s", err.Error())).
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

		return
	}

	for i, device := range devices {
		row := i + 1

		table.SetCell(row, 0, tview.NewTableCell(" "+device.Name).
			SetTextColor(tcell.ColorWhite).
			SetExpansion(1))

		table.SetCell(row, 1, tview.NewTableCell(" "+device.Target).
			SetTextColor(tcell.ColorGreen).
			SetExpansion(1))
	}
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
