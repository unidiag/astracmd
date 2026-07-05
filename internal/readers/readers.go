package readers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func Show(opt Options) {
	table := tview.NewTable()
	table.SetBorder(true)
	table.SetTitle(" Readers ")
	table.SetTitleAlign(tview.AlignCenter)
	table.SetSelectable(true, false)
	table.SetEvaluateAllRows(true)

	logView := tview.NewTextView()
	logView.SetBorder(true)
	logView.SetTitle(" Log ")
	logView.SetTitleAlign(tview.AlignCenter)
	logView.SetDynamicColors(true)
	logView.SetWrap(false)
	logView.SetScrollable(false)

	logFilter := tview.NewInputField()
	logFilter.SetLabel(" Regex: ")
	logFilter.SetFieldWidth(0)
	logFilter.SetBorder(false)
	logFilter.SetFieldBackgroundColor(tcell.ColorBlack)
	logFilter.SetFieldTextColor(tcell.ColorWhite)
	logFilter.SetLabelColor(tcell.ColorYellow)

	status := tview.NewTextView()
	status.SetDynamicColors(true)
	status.SetTextAlign(tview.AlignCenter)
	status.SetText("[gray]Tab — switch Readers/Regex    Enter — oscam.conf    Space — kill process    Esc — close[-]")

	content := tview.NewFlex()
	content.SetDirection(tview.FlexColumn)
	content.AddItem(table, 0, 1, true)

	logPane := tview.NewFlex()
	logPane.SetDirection(tview.FlexRow)
	logPane.SetBorder(true)
	logPane.SetTitle(" Log ")
	logPane.SetTitleAlign(tview.AlignCenter)

	logView.SetBorder(false)

	logPane.AddItem(logFilter, 1, 0, false)
	logPane.AddItem(logView, 0, 1, false)

	content.AddItem(logPane, 0, 2, false)

	body := tview.NewFlex()
	body.SetDirection(tview.FlexRow)
	body.AddItem(content, 0, 1, true)
	body.AddItem(status, 1, 0, false)

	var (
		currentDevices []Device
		currentState   selectedReaderState
		activePane     = readersPaneList
	)

	ctx, cancel := context.WithCancel(context.Background())

	closeDialog := func() {
		cancel()
		opt.Pages.RemovePage(opt.PageName)
	}

	setStatus := func(text string) {
		status.SetText(text)
	}

	updateBorders := func() {
		table.SetBorderColor(tcell.ColorWhite)
		logPane.SetBorderColor(tcell.ColorWhite)
	}

	focusPane := func(pane int) {
		activePane = pane
		updateBorders()

		switch pane {
		case readersPaneLogFilter:
			opt.App.SetFocus(logFilter)

		default:
			opt.App.SetFocus(table)
		}
	}

	loadLog := func() {
		if strings.TrimSpace(currentState.LogPath) == "" {
			logView.SetText("[gray]logfile is not configured[-]")
			return
		}

		_, _, _, height := logView.GetInnerRect()
		if height <= 0 {
			height = 50
		}

		filterText := strings.TrimSpace(logFilter.GetText())

		text, err := readTailLines(currentState.LogPath, readersLogTailBytes, height, filterText)
		if err != nil {
			logView.SetText(fmt.Sprintf("[red]%s[-]", tview.Escape(err.Error())))
			return
		}

		if strings.TrimSpace(text) == "" {
			logView.SetText("[gray]log is empty[-]")
			return
		}

		logView.SetText(text)
	}

	loadSelectedConfig := func(force bool) {
		row, _ := table.GetSelection()
		deviceIndex := row - 1

		if deviceIndex < 0 || deviceIndex >= len(currentDevices) {
			currentState = selectedReaderState{}
			logView.SetText("")
			return
		}

		device := currentDevices[deviceIndex]

		configPath := oscamConfigPathFromCommand(device.ProcessCmd)
		if strings.TrimSpace(configPath) == "" {
			currentState = selectedReaderState{Device: device}
			logView.SetText("[gray]oscam config path not found in process command[-]")
			return
		}

		if currentState.ConfigDirty && !force && currentState.ConfigPath == configPath {
			return
		}

		configText, err := os.ReadFile(configPath)
		if err != nil {
			currentState = selectedReaderState{
				Device:     device,
				ConfigPath: configPath,
			}
			logView.SetText(fmt.Sprintf("[red]%s[-]", tview.Escape(err.Error())))
			return
		}

		logPath := oscamLogPathFromConfig(string(configText), filepath.Dir(configPath))

		currentState = selectedReaderState{
			Device:     device,
			ConfigPath: configPath,
			ConfigText: string(configText),
			LogPath:    logPath,
		}

		loadLog()
	}

	saveConfigText := func(text string) error {
		if strings.TrimSpace(currentState.ConfigPath) == "" {
			return fmt.Errorf("no oscam.conf selected")
		}

		if err := os.WriteFile(currentState.ConfigPath, []byte(text), 0644); err != nil {
			return err
		}

		currentState.ConfigText = text
		currentState.ConfigDirty = false
		currentState.LogPath = oscamLogPathFromConfig(text, filepath.Dir(currentState.ConfigPath))

		loadLog()

		return nil
	}

	loadDevices := func(keepSelection bool) {
		selectedName := ""

		if keepSelection {
			row, _ := table.GetSelection()
			deviceIndex := row - 1
			if deviceIndex >= 0 && deviceIndex < len(currentDevices) {
				selectedName = currentDevices[deviceIndex].Name
			}
		}

		devices, err := ListDevices()
		currentDevices = devices
		renderDevices(table, devices, err)

		if selectedName != "" {
			for i, device := range devices {
				if device.Name == selectedName {
					table.Select(i+1, 0)
					break
				}
			}
		}

		if !currentState.ConfigDirty {
			loadSelectedConfig(false)
		}
	}

	killSelected := func() {
		row, _ := table.GetSelection()
		deviceIndex := row - 1

		if deviceIndex < 0 || deviceIndex >= len(currentDevices) {
			setStatus("[yellow]Select busy COM port first[-]")
			return
		}

		device := currentDevices[deviceIndex]
		if !device.Busy || device.ProcessPID <= 0 {
			setStatus("[yellow]Selected COM port is free[-]")
			return
		}

		processTitle := fmt.Sprintf("%s (%d)", device.ProcessName, device.ProcessPID)

		modal := tview.NewModal()
		modal.SetText(fmt.Sprintf(
			"Kill process?\n\nDevice: %s\nTarget: %s\nProcess: %s\n\nThis is equivalent to kill -9 %d.",
			displayDeviceName(device.Name),
			device.Target,
			processTitle,
			device.ProcessPID,
		))
		modal.AddButtons([]string{"Kill", "Cancel"})

		modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			if opt.HandleGlobalKeys != nil && opt.HandleGlobalKeys(event) {
				return nil
			}

			if event.Key() == tcell.KeyEsc {
				opt.Pages.RemovePage(opt.PageName + "-confirm")
				opt.App.SetFocus(table)
				return nil
			}

			return event
		})

		modal.SetDoneFunc(func(_ int, label string) {
			opt.Pages.RemovePage(opt.PageName + "-confirm")
			opt.App.SetFocus(table)

			if label != "Kill" {
				return
			}

			if err := killProcess(device.ProcessPID); err != nil {
				setStatus(fmt.Sprintf("[red]Kill failed: %s[-]", tview.Escape(err.Error())))
				return
			}

			setStatus(fmt.Sprintf(
				"[green]Killed %s (%d)[-]",
				tview.Escape(device.ProcessName),
				device.ProcessPID,
			))

			loadDevices(true)
		})

		opt.Pages.AddPage(opt.PageName+"-confirm", centerPrimitive(modal, 72, 13), true, true)
		opt.App.SetFocus(modal)
	}

	showConfigModal := func() {
		if strings.TrimSpace(currentState.ConfigPath) == "" {
			setStatus("[yellow]No oscam.conf selected[-]")
			return
		}

		configEditor := tview.NewTextArea()
		configEditor.SetBorder(true)
		configEditor.SetTitle(" " + currentState.ConfigPath + " ")
		configEditor.SetTitleAlign(tview.AlignCenter)
		configEditor.SetText(currentState.ConfigText, false)

		footer := tview.NewTextView()
		footer.SetDynamicColors(true)
		footer.SetTextAlign(tview.AlignCenter)
		footer.SetText("[gray]Ctrl+S — save    Esc — close[-]")

		modalBody := tview.NewFlex()
		modalBody.SetDirection(tview.FlexRow)
		modalBody.AddItem(configEditor, 0, 1, true)
		modalBody.AddItem(footer, 1, 0, false)

		closeConfigModal := func() {
			opt.Pages.RemovePage(opt.PageName + "-config")
			focusPane(activePane)
		}

		configEditor.SetChangedFunc(func() {
			currentState.ConfigDirty = configEditor.GetText() != currentState.ConfigText
		})

		configEditor.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			if opt.HandleGlobalKeys != nil && opt.HandleGlobalKeys(event) {
				return nil
			}

			switch event.Key() {
			case tcell.KeyEsc:
				closeConfigModal()
				return nil

			case tcell.KeyCtrlS:
				text := configEditor.GetText()

				if err := saveConfigText(text); err != nil {
					footer.SetText(fmt.Sprintf("[red]Save failed: %s[-]", tview.Escape(err.Error())))
					return nil
				}

				footer.SetText(fmt.Sprintf("[green]Saved %s[-]", tview.Escape(currentState.ConfigPath)))
				return nil
			}

			return event
		})

		opt.Pages.AddPage(opt.PageName+"-config", centerPrimitive(modalBody, 110, 32), true, true)
		opt.App.SetFocus(configEditor)
	}

	togglePane := func() {
		switch activePane {
		case readersPaneList:
			focusPane(readersPaneLogFilter)

		default:
			focusPane(readersPaneList)
		}
	}

	inputCapture := func(event *tcell.EventKey) *tcell.EventKey {
		if opt.HandleGlobalKeys != nil && opt.HandleGlobalKeys(event) {
			return nil
		}

		switch event.Key() {
		case tcell.KeyEsc:
			closeDialog()
			return nil

		case tcell.KeyTab, tcell.KeyBacktab:
			togglePane()
			return nil

		case tcell.KeyEnter:
			if activePane == readersPaneList {
				showConfigModal()
				return nil
			}

		case tcell.KeyRune:
			if event.Rune() == ' ' && activePane == readersPaneList {
				killSelected()
				return nil
			}
		}

		return event
	}

	table.SetSelectedFunc(func(row int, _ int) {
		if row <= 0 {
			return
		}

		loadSelectedConfig(true)
		showConfigModal()
	})

	table.SetSelectionChangedFunc(func(row int, _ int) {
		if row <= 0 {
			return
		}

		if currentState.ConfigDirty {
			return
		}

		loadSelectedConfig(false)
	})

	logFilter.SetChangedFunc(func(text string) {
		loadLog()
	})

	table.SetInputCapture(inputCapture)

	logFilter.SetInputCapture(inputCapture)
	logPane.SetInputCapture(inputCapture)
	body.SetInputCapture(inputCapture)

	refreshDevicesAndLog := func(keepSelection bool) {
		devices, err := ListDevices()

		opt.App.QueueUpdateDraw(func() {
			select {
			case <-ctx.Done():
				return
			default:
			}

			selectedName := ""

			if keepSelection {
				row, _ := table.GetSelection()
				deviceIndex := row - 1
				if deviceIndex >= 0 && deviceIndex < len(currentDevices) {
					selectedName = currentDevices[deviceIndex].Name
				}
			}

			currentDevices = devices
			renderDevices(table, devices, err)

			if selectedName != "" {
				for i, device := range devices {
					if device.Name == selectedName {
						table.Select(i+1, 0)
						break
					}
				}
			}

			if selectedName == "" && len(devices) > 0 {
				table.Select(1, 0)
			}

			if !currentState.ConfigDirty {
				loadSelectedConfig(false)
			}

			loadLog()
		})
	}

	opt.Pages.AddPage(opt.PageName, body, true, true)
	focusPane(readersPaneList)
	updateBorders()

	go func() {
		refreshDevicesAndLog(false)

		ticker := time.NewTicker(readersRefreshPeriod)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return

			case <-ticker.C:
				refreshDevicesAndLog(true)
			}
		}
	}()
}
