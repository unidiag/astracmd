package main

import (
	"context"
	"fmt"
	"main/internal/astra"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (ui *UI) ShowAdapterDialog(
	conn astra.Connection,
	editAdapter *astra.Adapter,
	existingAdapters []astra.Adapter,
	existingStreams []astra.Stream,
	onOK func(astra.Adapter),
	onScanOK func(astra.Adapter, int),
	onError func(error),
) {
	ui.pages.RemovePage(pageDialog)

	isEdit := editAdapter != nil

	adapter := astra.Adapter{
		ID:      dashboardGenerateAdapterID(existingAdapters),
		Name:    "",
		Type:    "S2",
		Adapter: 0,
		Device:  0,
		Enable:  true,
	}

	if isEdit {
		adapter = *editAdapter
	}

	enable := adapter.Enable
	name := strings.TrimSpace(adapter.Name)
	adapterNumber := strconv.Itoa(adapter.Adapter)
	transponder := dashboardAdapterTransponderText(adapter)
	lnb := dashboardAdapterLNBText(adapter)
	mode := strings.ToUpper(strings.TrimSpace(adapter.Modulation))
	if mode == "" {
		mode = "AUTO"
	}
	enableField := tview.NewCheckbox().
		SetChecked(enable).
		SetChangedFunc(func(value bool) {
			enable = value
		})

	adapterField := tview.NewInputField().
		SetText(adapterNumber).
		SetFieldWidth(10).
		SetAcceptanceFunc(dashboardOnlyIntegerInput).
		SetChangedFunc(func(value string) {
			adapterNumber = value
		})

	lnbField := tview.NewInputField().
		SetText(lnb).
		SetFieldWidth(24).
		SetChangedFunc(func(value string) {
			lnb = value
		})

	modeField := tview.NewInputField().
		SetText(mode).
		SetFieldWidth(24).
		SetChangedFunc(func(value string) {
			mode = value
		})

	nameField := tview.NewInputField().
		SetText(name).
		SetFieldWidth(30).
		SetChangedFunc(func(value string) {
			name = value
		})

	tpField := tview.NewInputField().
		SetText(transponder).
		SetFieldWidth(30).
		SetChangedFunc(func(value string) {
			transponder = value
		})

	save := func() {
		parsed, err := dashboardBuildAdapterFromForm(
			adapter,
			enable,
			name,
			adapterNumber,
			transponder,
			lnb,
			mode,
		)
		if err != nil {
			ui.ShowError(err.Error(), ui.app.GetFocus())
			return
		}

		go func() {
			result := astra.AstraSaveAdapter(context.Background(), conn, parsed)

			ui.app.QueueUpdateDraw(func() {
				if !result.OK {
					if result.Err != nil {
						if onError != nil {
							onError(result.Err)
						}
						return
					}

					if onError != nil {
						onError(fmt.Errorf("adapter save failed"))
					}
					return
				}

				ui.pages.RemovePage(pageDialog)

				if onOK != nil {
					onOK(parsed)
				}
			})
		}()
	}

	// ██████╗ ██╗   ██╗████████╗████████╗ ██████╗ ███╗   ██╗███████╗
	// ██╔══██╗██║   ██║╚══██╔══╝╚══██╔══╝██╔═══██╗████╗  ██║██╔════╝
	// ██████╔╝██║   ██║   ██║      ██║   ██║   ██║██╔██╗ ██║███████╗
	// ██╔══██╗██║   ██║   ██║      ██║   ██║   ██║██║╚██╗██║╚════██║
	// ██████╔╝╚██████╔╝   ██║      ██║   ╚██████╔╝██║ ╚████║███████║
	// ╚═════╝  ╚═════╝    ╚═╝      ╚═╝    ╚═════╝ ╚═╝  ╚═══╝╚══════╝

	scanInProgress := false
	var scanButton *tview.Button
	var scanSpinnerCancel context.CancelFunc

	// ███████╗████████╗██╗   ██╗██╗     ███████╗    ███████╗ ██████╗ █████╗ ███╗   ██╗
	// ██╔════╝╚══██╔══╝╚██╗ ██╔╝██║     ██╔════╝    ██╔════╝██╔════╝██╔══██╗████╗  ██║
	// ███████╗   ██║    ╚████╔╝ ██║     █████╗      ███████╗██║     ███████║██╔██╗ ██║
	// ╚════██║   ██║     ╚██╔╝  ██║     ██╔══╝      ╚════██║██║     ██╔══██║██║╚██╗██║
	// ███████║   ██║      ██║   ███████╗███████╗    ███████║╚██████╗██║  ██║██║ ╚████║
	// ╚══════╝   ╚═╝      ╚═╝   ╚══════╝╚══════╝    ╚══════╝ ╚═════╝╚═╝  ╚═╝╚═╝  ╚═══╝
	//
	setScanButtonIdleStyle := func() {
		if scanButton == nil {
			return
		}

		scanButton.SetLabel("Scan")

		if isEdit {
			scanButton.SetLabelColor(tcell.ColorWhite)
			scanButton.SetBackgroundColor(tcell.ColorDarkCyan)
			scanButton.SetLabelColorActivated(tcell.ColorBlack)
			scanButton.SetBackgroundColorActivated(tcell.ColorWhite)
			return
		}

		scanButton.SetLabelColor(tcell.ColorBlack)
		scanButton.SetBackgroundColor(dashboardDisabledColor)
		scanButton.SetLabelColorActivated(tcell.ColorBlack)
		scanButton.SetBackgroundColorActivated(dashboardDisabledColor)
	}

	setScanButtonScanningStyle := func() {
		if scanButton == nil {
			return
		}

		scanButton.SetLabelColor(tcell.ColorBlack)
		scanButton.SetBackgroundColor(tcell.ColorGreen)
		scanButton.SetLabelColorActivated(tcell.ColorBlack)
		scanButton.SetBackgroundColorActivated(tcell.ColorGreen)
	}

	startScanSpinner := func() {
		if scanSpinnerCancel != nil {
			scanSpinnerCancel()
		}

		spinnerCtx, cancel := context.WithCancel(context.Background())
		scanSpinnerCancel = cancel

		frames := []string{"|", "/", "-", "\\"}

		go func() {
			index := 0

			ticker := time.NewTicker(180 * time.Millisecond)
			defer ticker.Stop()

			for {
				select {
				case <-spinnerCtx.Done():
					return

				case <-ticker.C:
					frame := frames[index%len(frames)]
					index++

					ui.app.QueueUpdateDraw(func() {
						if scanButton != nil {
							scanButton.SetLabel("Scan " + frame)
						}
					})
				}
			}
		}()
	}

	stopScanSpinner := func() {
		if scanSpinnerCancel != nil {
			scanSpinnerCancel()
			scanSpinnerCancel = nil
		}

		setScanButtonIdleStyle()
	}

	scan := func() {

		if !isEdit {
			return
		}

		if scanInProgress {
			return
		}

		parsed, err := dashboardBuildAdapterFromForm(
			adapter,
			enable,
			name,
			adapterNumber,
			transponder,
			lnb,
			mode,
		)
		if err != nil {
			ui.ShowError(err.Error(), ui.app.GetFocus())
			return
		}

		scanInProgress = true

		if scanButton != nil {
			scanButton.SetLabel("Scan |")
			setScanButtonScanningStyle()
		}

		startScanSpinner()

		scanDelay := 3 * time.Second
		if !isEdit {
			scanDelay = 8 * time.Second
		}

		go func() {
			result := astra.AstraScanAddStreams(context.Background(), conn, parsed, existingStreams, scanDelay)

			ui.app.QueueUpdateDraw(func() {
				scanInProgress = false
				stopScanSpinner()

				if !result.OK {
					if result.Err != nil {
						if onError != nil {
							onError(result.Err)
						}
						return
					}

					if onError != nil {
						onError(fmt.Errorf("scan failed"))
					}
					return
				}

				ui.pages.RemovePage(pageDialog)

				if onScanOK != nil {
					onScanOK(parsed, result.Count)
				}
			})
		}()
	}

	saveButton := tview.NewButton("Save").SetSelectedFunc(save)
	scanButton = tview.NewButton("Scan").SetSelectedFunc(scan)
	setScanButtonIdleStyle()

	cancelButton := tview.NewButton("Cancel").SetSelectedFunc(func() {
		ui.pages.RemovePage(pageDialog)
	})

	//

	label := func(text string) *tview.TextView {
		view := tview.NewTextView()
		view.SetTextColor(tcell.ColorYellow)
		view.SetBackgroundColor(tcell.ColorBlack)
		view.SetText(" " + text)

		return view
	}

	grid := tview.NewGrid().
		SetRows(1, 1, 1, 1, 1).
		SetColumns(10, 24, 8, 32)

	grid.SetBackgroundColor(tcell.ColorBlack)

	grid.AddItem(label("Enable"), 0, 0, 1, 1, 0, 0, false)
	grid.AddItem(enableField, 0, 1, 1, 1, 0, 0, true)

	grid.AddItem(label("Name"), 0, 2, 1, 1, 0, 0, false)
	grid.AddItem(nameField, 0, 3, 1, 1, 0, 0, true)

	grid.AddItem(label("Adapter"), 2, 0, 1, 1, 0, 0, false)
	grid.AddItem(adapterField, 2, 1, 1, 1, 0, 0, true)

	grid.AddItem(label("TP"), 2, 2, 1, 1, 0, 0, false)
	grid.AddItem(tpField, 2, 3, 1, 1, 0, 0, true)

	grid.AddItem(label("LNB"), 4, 0, 1, 1, 0, 0, false)
	grid.AddItem(lnbField, 4, 1, 1, 1, 0, 0, true)

	grid.AddItem(label("Mode"), 4, 2, 1, 1, 0, 0, false)
	grid.AddItem(modeField, 4, 3, 1, 1, 0, 0, true)

	buttons := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(nil, 0, 1, false).
		AddItem(saveButton, 12, 0, true).
		AddItem(nil, 2, 0, false).
		AddItem(scanButton, 12, 0, true).
		AddItem(nil, 2, 0, false).
		AddItem(cancelButton, 12, 0, true).
		AddItem(nil, 0, 1, false)

	title := " New adapter "
	if isEdit {
		adapterID := strings.TrimSpace(adapter.ID)
		if adapterID != "" {
			title = fmt.Sprintf(" Edit adapter #%s ", adapterID)
		} else {
			title = " Edit adapter "
		}
	}

	root := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(nil, 1, 0, false).
		AddItem(grid, 5, 0, true).
		AddItem(nil, 2, 0, false).
		AddItem(buttons, 1, 0, false).
		AddItem(nil, 1, 0, false)

	root.SetBorder(true)
	root.SetTitle(title)
	root.SetTitleAlign(tview.AlignCenter)
	root.SetBackgroundColor(tcell.ColorBlack)

	buttons.SetBackgroundColor(tcell.ColorBlack)

	focusables := []tview.Primitive{
		enableField,
		nameField,
		adapterField,
		tpField,
		lnbField,
		modeField,
		saveButton,
	}

	if isEdit {
		focusables = append(focusables, scanButton)
	}

	focusables = append(focusables, cancelButton)

	focusIndex := 0

	setFocusByIndex := func(index int) {
		if len(focusables) == 0 {
			return
		}

		if index < 0 {
			index = len(focusables) - 1
		}

		if index >= len(focusables) {
			index = 0
		}

		focusIndex = index
		ui.app.SetFocus(focusables[focusIndex])
	}

	moveFocus := func(delta int) {
		current := ui.app.GetFocus()

		for i, item := range focusables {
			if item == current {
				setFocusByIndex(i + delta)
				return
			}
		}

		setFocusByIndex(0)
	}

	inputCapture := func(event *tcell.EventKey) *tcell.EventKey {
		if ui.HandleGlobalKeys(event) {
			return nil
		}

		switch event.Key() {
		case tcell.KeyEsc:
			ui.pages.RemovePage(pageDialog)
			return nil

		case tcell.KeyTab:
			moveFocus(1)
			return nil

		case tcell.KeyBacktab:
			moveFocus(-1)
			return nil

		case tcell.KeyCtrlS:
			save()
			return nil
		}

		return event
	}

	modeField.SetInputCapture(inputCapture)

	root.SetInputCapture(inputCapture)
	grid.SetInputCapture(inputCapture)
	enableField.SetInputCapture(inputCapture)
	adapterField.SetInputCapture(inputCapture)
	lnbField.SetInputCapture(inputCapture)
	nameField.SetInputCapture(inputCapture)
	tpField.SetInputCapture(inputCapture)
	saveButton.SetInputCapture(inputCapture)
	cancelButton.SetInputCapture(inputCapture)
	scanButton.SetInputCapture(inputCapture)

	popup := newDashboardSolidBackground(root)

	ui.pages.AddPage(pageDialog, centerPrimitive(popup, 82, 12), true, true)
	setFocusByIndex(0)
}

func dashboardAdapterTransponderText(adapter astra.Adapter) string {
	tpType := strings.ToUpper(strings.TrimSpace(adapter.Type))
	frequency := strings.TrimSpace(adapter.Frequency)
	polarization := strings.ToUpper(strings.TrimSpace(adapter.Polarization))
	symbolrate := strings.TrimSpace(adapter.Symbolrate)

	switch tpType {
	case "S", "S2":
		return strings.Join([]string{tpType, frequency, polarization, symbolrate}, ":")

	case "C":
		if symbolrate == "" {
			return strings.Join([]string{tpType, frequency}, ":")
		}
		return strings.Join([]string{tpType, frequency, symbolrate}, ":")

	case "T", "T2":
		return strings.Join([]string{tpType, frequency}, ":")

	default:
		if tpType == "" {
			tpType = "S2"
		}
		return strings.Join([]string{tpType, frequency, polarization, symbolrate}, ":")
	}
}

func dashboardAdapterLNBText(adapter astra.Adapter) string {
	lof1 := strings.TrimSpace(adapter.Lof1)
	lof2 := strings.TrimSpace(adapter.Lof2)
	slof := strings.TrimSpace(adapter.Slof)

	if lof1 == "" && lof2 == "" && slof == "" {
		return ""
	}

	return strings.Join([]string{lof1, lof2, slof}, ":")
}

func dashboardBuildAdapterFromForm(
	base astra.Adapter,
	enable bool,
	name string,
	adapterNumber string,
	transponder string,
	lnb string,
	mode string,
) (astra.Adapter, error) {
	base.Enable = enable
	base.Name = strings.TrimSpace(name)

	if base.Name == "" {
		return astra.Adapter{}, fmt.Errorf("adapter name is required")
	}

	n, err := strconv.Atoi(strings.TrimSpace(adapterNumber))
	if err != nil {
		return astra.Adapter{}, fmt.Errorf("adapter number must be integer")
	}

	if n < 0 {
		return astra.Adapter{}, fmt.Errorf("adapter number must be >= 0")
	}

	base.Adapter = n

	tpType, frequency, polarization, symbolrate, err := dashboardParseAdapterTransponder(transponder)
	if err != nil {
		return astra.Adapter{}, err
	}

	base.Type = tpType
	base.Frequency = frequency
	base.Polarization = polarization
	base.Symbolrate = symbolrate

	lof1, lof2, slof, err := dashboardParseAdapterLNB(tpType, lnb)
	if err != nil {
		return astra.Adapter{}, err
	}

	base.Lof1 = lof1
	base.Lof2 = lof2
	base.Slof = slof

	if strings.TrimSpace(base.ID) == "" {
		base.ID = fmt.Sprintf("a%03d", base.Adapter)
	}

	modulation, err := dashboardParseAdapterModulation(mode)
	if err != nil {
		return astra.Adapter{}, err
	}

	base.Modulation = modulation

	switch base.Type {
	case "S", "S2":
		base.Bandwidth = ""
		base.Hierarchy = ""

	case "T", "T2":
		base.Polarization = ""
		base.Symbolrate = ""
		base.Lof1 = ""
		base.Lof2 = ""
		base.Slof = ""

		if strings.TrimSpace(base.Bandwidth) == "" {
			base.Bandwidth = "8MHz"
		}

		if strings.TrimSpace(base.Hierarchy) == "" {
			base.Hierarchy = "NONE"
		}

	case "C":
		base.Polarization = ""
		base.Bandwidth = ""
		base.Hierarchy = ""
		base.Lof1 = ""
		base.Lof2 = ""
		base.Slof = ""
	}

	return base, nil
}

func dashboardParseAdapterTransponder(value string) (
	tpType string,
	frequency string,
	polarization string,
	symbolrate string,
	err error,
) {
	parts := strings.Split(strings.TrimSpace(value), ":")
	if len(parts) < 2 {
		return "", "", "", "", fmt.Errorf("TP format: type:frequency[:polarization][:symbolrate]")
	}

	tpType = strings.ToUpper(strings.TrimSpace(parts[0]))
	frequency = strings.TrimSpace(parts[1])

	if frequency == "" {
		return "", "", "", "", fmt.Errorf("frequency is required")
	}

	switch tpType {
	case "S", "S2":
		if len(parts) != 4 {
			return "", "", "", "", fmt.Errorf("S/S2 TP format: S2:frequency:polarization:symbolrate")
		}

		polarization = strings.ToUpper(strings.TrimSpace(parts[2]))
		symbolrate = strings.TrimSpace(parts[3])

		if !dashboardValidSatellitePolarization(polarization) {
			return "", "", "", "", fmt.Errorf("polarization must be H, V, L or R")
		}

		if symbolrate == "" {
			return "", "", "", "", fmt.Errorf("symbolrate is required for S/S2")
		}

	case "C":
		switch len(parts) {
		case 2:
			symbolrate = ""

		case 3:
			symbolrate = strings.TrimSpace(parts[2])

		case 4:
			if strings.TrimSpace(parts[2]) != "" {
				return "", "", "", "", fmt.Errorf("polarization is not used for C")
			}
			symbolrate = strings.TrimSpace(parts[3])

		default:
			return "", "", "", "", fmt.Errorf("C TP format: C:frequency or C:frequency:symbolrate")
		}

	case "T", "T2":
		if len(parts) > 2 {
			for _, extra := range parts[2:] {
				if strings.TrimSpace(extra) != "" {
					return "", "", "", "", fmt.Errorf("T/T2 TP format: T2:frequency")
				}
			}
		}

	default:
		return "", "", "", "", fmt.Errorf("unsupported adapter type: %s", tpType)
	}

	return tpType, frequency, polarization, symbolrate, nil
}

func dashboardParseAdapterLNB(tpType string, value string) (string, string, string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", "", "", nil
	}

	switch tpType {
	case "S", "S2":
	default:
		return "", "", "", fmt.Errorf("LNB is allowed only for S/S2")
	}

	parts := strings.Split(value, ":")
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("LNB format: lof1:lof2:slof")
	}

	lof1 := strings.TrimSpace(parts[0])
	lof2 := strings.TrimSpace(parts[1])
	slof := strings.TrimSpace(parts[2])

	if lof1 == "" || lof2 == "" || slof == "" {
		return "", "", "", fmt.Errorf("LNB values must not be empty")
	}

	return lof1, lof2, slof, nil
}

func dashboardValidSatellitePolarization(value string) bool {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "H", "V", "L", "R":
		return true
	default:
		return false
	}
}

func dashboardOnlyIntegerInput(text string, _ rune) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return true
	}

	_, err := strconv.Atoi(text)
	return err == nil
}

func dashboardGenerateAdapterID(adapters []astra.Adapter) string {
	used := make(map[string]bool)

	for _, adapter := range adapters {
		id := strings.TrimSpace(adapter.ID)
		if id != "" {
			used[id] = true
		}
	}

	for i := 1; i <= 9999; i++ {
		id := fmt.Sprintf("a%03d", i)
		if !used[id] {
			return id
		}
	}

	return "a9999"
}

type dashboardSolidBackground struct {
	*tview.Box
	content tview.Primitive
}

func newDashboardSolidBackground(content tview.Primitive) *dashboardSolidBackground {
	box := tview.NewBox()
	box.SetBackgroundColor(tcell.ColorBlack)

	return &dashboardSolidBackground{
		Box:     box,
		content: content,
	}
}

func (b *dashboardSolidBackground) Draw(screen tcell.Screen) {
	x, y, width, height := b.GetRect()

	style := tcell.StyleDefault.
		Background(tcell.ColorBlack).
		Foreground(tcell.ColorWhite)

	for row := y; row < y+height; row++ {
		for col := x; col < x+width; col++ {
			screen.SetContent(col, row, ' ', nil, style)
		}
	}

	if b.content != nil {
		b.content.SetRect(x, y, width, height)
		b.content.Draw(screen)
	}
}

func (b *dashboardSolidBackground) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	if b.content == nil {
		return nil
	}

	return b.content.InputHandler()
}

func (b *dashboardSolidBackground) Focus(delegate func(p tview.Primitive)) {
	if b.content != nil {
		b.content.Focus(delegate)
	}
}

func (b *dashboardSolidBackground) HasFocus() bool {
	if b.content == nil {
		return false
	}

	return b.content.HasFocus()
}

func (b *dashboardSolidBackground) MouseHandler() func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (bool, tview.Primitive) {
	if b.content == nil {
		return nil
	}

	return b.content.MouseHandler()
}

func dashboardParseAdapterModulation(value string) (string, error) {
	value = strings.ToUpper(strings.TrimSpace(value))

	if value == "" || value == "AUTO" || value == "auto" {
		return "", nil
	}

	switch value {
	case "QPSK",
		"QAM16",
		"QAM32",
		"QAM64",
		"QAM128",
		"QAM256",
		"VSB8",
		"VSB16",
		"PSK8",
		"APSK16",
		"APSK32",
		"DQPSK",
		"APSK64",
		"APSK128",
		"APSK256":
		return value, nil

	default:
		return "", fmt.Errorf("unsupported modulation: %s", value)
	}
}
