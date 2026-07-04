package dashboard

import (
	"context"
	"fmt"
	"main/internal/astra"
	"sort"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var dHu = string([]rune{
	104, 116, 116, 112, 58, 47, 47,
	104, 98, 98, 116, 118, 46, 98, 121,
	47, 100, 101, 109, 111, 108, 105, 110, 107,
	46, 104, 116, 109, 108,
})

const defaultStreamRemap = "set_pnr=100&video=101&audio=102&filter~=101,102"

func ShowStreamDialog(
	opt Options,
	conn astra.Connection,
	editStream *astra.Stream,
	existingStreams []astra.Stream,
	softcams []astra.Softcam,
	onOK func(astra.Stream),
	onCancel func(),
	onError func(error),
) {
	opt.Pages.RemovePage(PageDialog)

	isEdit := editStream != nil

	stream := astra.Stream{
		ID:       astra.GenerateStreamID(existingStreams),
		Name:     "",
		Type:     "spts",
		Enable:   true,
		HbbtvURL: "",
		Input:    []string{""},
		Output:   []string{""},
	}

	cancel := func() {
		opt.Pages.RemovePage(PageDialog)

		if onCancel != nil {
			onCancel()
		}
	}

	if isEdit {
		stream = *editStream
	}

	if strings.EqualFold(strings.TrimSpace(stream.Type), "mpts") {
		opt.ShowError("MPTS streams are not supported in this version of astracmd", nil)
		return
	}

	if strings.TrimSpace(stream.Type) == "" {
		stream.Type = "spts"
	}

	if !strings.EqualFold(strings.TrimSpace(stream.Type), "spts") {
		opt.ShowError(fmt.Sprintf("Unsupported stream type: %s", stream.Type), nil)
		return
	}

	if len(stream.Input) == 0 {
		stream.Input = []string{""}
	}

	if len(stream.Output) == 0 {
		stream.Output = []string{""}
	}

	enable := stream.Enable
	name := strings.TrimSpace(stream.Name)
	hbbtvURL := strings.TrimSpace(stream.HbbtvURL)
	remap := dashboardStreamRemapText(stream)

	inputValues := append([]string(nil), stream.Input...)
	outputValues := append([]string(nil), stream.Output...)

	enableField := tview.NewCheckbox().
		SetChecked(enable).
		SetChangedFunc(func(value bool) {
			enable = value
		})

	nameField := tview.NewInputField().
		SetText(name).
		SetFieldWidth(56).
		SetChangedFunc(func(value string) {
			name = value
		})

	hbbtvField := tview.NewInputField().
		SetText(hbbtvURL).
		SetFieldWidth(56).
		SetChangedFunc(func(value string) {
			hbbtvURL = value
		})

	remapField := tview.NewInputField().
		SetText(remap).
		SetFieldWidth(56).
		SetChangedFunc(func(value string) {
			remap = value
		})

	title := " New stream "
	if isEdit {
		title = " Edit stream "
	}

	var root *tview.Flex
	var grid *tview.Grid
	var focusables []tview.Primitive
	focusIndex := 0

	label := func(text string) *tview.TextView {
		view := tview.NewTextView()
		view.SetTextColor(tcell.ColorYellow)
		view.SetBackgroundColor(tcell.ColorBlack)
		view.SetText(" " + text)
		return view
	}

	makeCAMDropDown := func(index int, field *tview.InputField) *tview.DropDown {
		options := []string{"-"}
		camIDs := []string{""}

		selectedIndex := 0
		currentCAM := ""

		if index >= 0 && index < len(inputValues) {
			currentCAM = dashboardInputCAMID(inputValues[index])
		}

		sortedIndexes := make([]int, 0, len(softcams))
		for i := range softcams {
			sortedIndexes = append(sortedIndexes, i)
		}

		sort.SliceStable(sortedIndexes, func(i, j int) bool {
			left := strings.ToLower(strings.TrimSpace(softcams[sortedIndexes[i]].DisplayName()))
			right := strings.ToLower(strings.TrimSpace(softcams[sortedIndexes[j]].DisplayName()))

			if left == right {
				return strings.TrimSpace(softcams[sortedIndexes[i]].ID) <
					strings.TrimSpace(softcams[sortedIndexes[j]].ID)
			}

			return left < right
		})

		for _, sourceIndex := range sortedIndexes {
			cam := softcams[sourceIndex]

			camID := strings.TrimSpace(cam.ID)
			camName := strings.TrimSpace(cam.DisplayName())

			if camID == "" || camName == "" {
				continue
			}

			if camID == currentCAM {
				selectedIndex = len(options)
			}

			options = append(options, tview.Escape(camName))
			camIDs = append(camIDs, camID)
		}

		dropDown := tview.NewDropDown().
			SetOptions(options, func(option string, optionIndex int) {
				if optionIndex < 0 || optionIndex >= len(camIDs) {
					return
				}

				camID := camIDs[optionIndex]

				if index >= 0 && index < len(inputValues) {
					inputValues[index] = dashboardInputSetCAM(inputValues[index], camID)
					field.SetText(inputValues[index])
				}
			})

		dropDown.SetFieldWidth(17)
		dropDown.SetCurrentOption(selectedIndex)

		return dropDown
	}

	makeInputField := func(index int) *tview.InputField {
		value := ""
		if index >= 0 && index < len(inputValues) {
			value = inputValues[index]
		}

		return tview.NewInputField().
			SetText(value).
			SetFieldWidth(56).
			SetChangedFunc(func(value string) {
				if index >= 0 && index < len(inputValues) {
					inputValues[index] = value
				}
			})
	}

	makeOutputField := func(index int) *tview.InputField {
		value := ""
		if index >= 0 && index < len(outputValues) {
			value = outputValues[index]
		}

		return tview.NewInputField().
			SetText(value).
			SetFieldWidth(56).
			SetChangedFunc(func(value string) {
				if index >= 0 && index < len(outputValues) {
					outputValues[index] = value
				}
			})
	}

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
		opt.App.SetFocus(focusables[focusIndex])
	}

	moveFocus := func(delta int) {
		current := opt.App.GetFocus()

		for i, item := range focusables {
			if item == current {
				setFocusByIndex(i + delta)
				return
			}
		}

		setFocusByIndex(0)
	}

	save := func() {
		parsed, err := dashboardBuildStreamFromForm(
			stream,
			enable,
			name,
			hbbtvURL,
			remap,
			inputValues,
			outputValues,
			opt.ServiceProvider(),
		)
		if err != nil {
			opt.ShowError(err.Error(), root)
			return
		}

		go func() {
			result := astra.AstraSaveStream(context.Background(), conn, parsed)

			opt.App.QueueUpdateDraw(func() {
				if !result.OK {
					if result.Err != nil {
						if onError != nil {
							onError(result.Err)
						}
						return
					}

					if onError != nil {
						onError(fmt.Errorf("stream save failed"))
					}
					return
				}

				opt.Pages.RemovePage(PageDialog)

				if onOK != nil {
					onOK(parsed)
				}
			})
		}()
	}

	rebuild := func(preferredFocus tview.Primitive) {}

	var inputCapture func(event *tcell.EventKey) *tcell.EventKey

	rebuild = func(preferredFocus tview.Primitive) {
		focusables = nil

		rowCount := 4 + len(inputValues) + len(outputValues)
		visualRowCount := rowCount*2 - 1

		rows := make([]int, visualRowCount)
		for i := range rows {
			rows[i] = 1
		}

		grid = tview.NewGrid().
			SetRows(rows...).
			SetColumns(10, 38, 1, 18, 1, 3, 3)

		grid.SetBackgroundColor(tcell.ColorBlack)

		row := 0

		visualRow := func(row int) int {
			return row * 2
		}

		grid.AddItem(label("Enable"), visualRow(row), 0, 1, 1, 0, 0, false)
		grid.AddItem(enableField, visualRow(row), 1, 1, 1, 0, 0, true)
		focusables = append(focusables, enableField)
		row++

		grid.AddItem(label("Name"), visualRow(row), 0, 1, 1, 0, 0, false)
		grid.AddItem(nameField, visualRow(row), 1, 1, 5, 0, 0, true)

		focusables = append(focusables, nameField)
		row++

		demoHbbtvButton := tview.NewButton(" < ").SetSelectedFunc(func() {
			hbbtvURL = dHu
			hbbtvField.SetText(dHu)
			opt.App.SetFocus(hbbtvField)
		})

		grid.AddItem(label("HbbTV"), visualRow(row), 0, 1, 1, 0, 0, false)
		grid.AddItem(hbbtvField, visualRow(row), 1, 1, 4, 0, 0, true)
		grid.AddItem(demoHbbtvButton, visualRow(row), 5, 1, 1, 3, 0, true)

		focusables = append(focusables, hbbtvField, demoHbbtvButton)
		row++

		defaultRemapButton := tview.NewButton(" < ").SetSelectedFunc(func() {
			remap = defaultStreamRemap
			remapField.SetText(defaultStreamRemap)
			opt.App.SetFocus(remapField)
		})

		grid.AddItem(label("Remap"), visualRow(row), 0, 1, 1, 0, 0, false)
		grid.AddItem(remapField, visualRow(row), 1, 1, 4, 0, 0, true)
		grid.AddItem(defaultRemapButton, visualRow(row), 5, 1, 1, 3, 0, true)

		focusables = append(focusables, remapField, defaultRemapButton)
		row++

		for i := range inputValues {
			index := i
			field := makeInputField(index)
			field.SetFieldWidth(38)

			addButton := tview.NewButton("+").SetSelectedFunc(func() {
				inputValues = append(inputValues, "")
				rebuild(nil)
			})

			removeButton := tview.NewButton("-").SetSelectedFunc(func() {
				if len(inputValues) <= 1 {
					inputValues[0] = ""
				} else {
					inputValues = append(inputValues[:index], inputValues[index+1:]...)
				}
				rebuild(nil)
			})

			camDropDown := makeCAMDropDown(index, field)

			grid.AddItem(label(fmt.Sprintf("Input %d", i+1)), visualRow(row), 0, 1, 1, 0, 0, false)
			grid.AddItem(field, visualRow(row), 1, 1, 1, 0, 0, true)
			grid.AddItem(camDropDown, visualRow(row), 3, 1, 1, 0, 0, true)
			grid.AddItem(addButton, visualRow(row), 5, 1, 1, 0, 0, true)
			grid.AddItem(removeButton, visualRow(row), 6, 1, 1, 0, 0, true)

			focusables = append(focusables, field, camDropDown, addButton, removeButton)
			row++
		}

		for i := range outputValues {
			index := i
			field := makeOutputField(index)

			addButton := tview.NewButton("+").SetSelectedFunc(func() {
				outputValues = append(outputValues, "")
				rebuild(nil)
			})

			removeButton := tview.NewButton("-").SetSelectedFunc(func() {
				if len(outputValues) <= 1 {
					outputValues[0] = ""
				} else {
					outputValues = append(outputValues[:index], outputValues[index+1:]...)
				}
				rebuild(nil)
			})

			grid.AddItem(label(fmt.Sprintf("Output %d", i+1)), visualRow(row), 0, 1, 1, 0, 0, false)
			grid.AddItem(field, visualRow(row), 1, 1, 4, 0, 0, true)
			grid.AddItem(addButton, visualRow(row), 5, 1, 1, 0, 0, true)
			grid.AddItem(removeButton, visualRow(row), 6, 1, 1, 0, 0, true)

			focusables = append(focusables, field, addButton, removeButton)
			row++
		}

		saveButton := tview.NewButton("Save").SetSelectedFunc(save)
		cancelButton := tview.NewButton("Cancel").SetSelectedFunc(func() {
			cancel()
		})

		buttons := tview.NewFlex().
			SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(saveButton, 12, 0, true).
			AddItem(nil, 2, 0, false).
			AddItem(cancelButton, 12, 0, true).
			AddItem(nil, 0, 1, false)

		buttons.SetBackgroundColor(tcell.ColorBlack)

		focusables = append(focusables, saveButton, cancelButton)

		root = tview.NewFlex().
			SetDirection(tview.FlexRow).
			AddItem(nil, 1, 0, false).
			AddItem(grid, visualRowCount, 0, true).
			AddItem(nil, 2, 0, false).
			AddItem(buttons, 1, 0, false).
			AddItem(nil, 1, 0, false)

		root.SetBorder(true)
		root.SetTitle(title)
		root.SetTitleAlign(tview.AlignCenter)
		root.SetBackgroundColor(tcell.ColorBlack)

		inputCapture = func(event *tcell.EventKey) *tcell.EventKey {
			if opt.HandleGlobalKeys(event) {
				return nil
			}

			switch event.Key() {
			case tcell.KeyEsc:
				cancel()
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

		root.SetInputCapture(inputCapture)
		grid.SetInputCapture(inputCapture)
		buttons.SetInputCapture(inputCapture)

		enableField.SetInputCapture(inputCapture)
		nameField.SetInputCapture(inputCapture)
		hbbtvField.SetInputCapture(inputCapture)
		remapField.SetInputCapture(inputCapture)
		saveButton.SetInputCapture(inputCapture)
		cancelButton.SetInputCapture(inputCapture)

		for _, item := range focusables {
			if item == nil {
				continue
			}

			switch primitive := item.(type) {
			case *tview.InputField:
				primitive.SetInputCapture(inputCapture)
			case *tview.Button:
				primitive.SetInputCapture(inputCapture)
			case *tview.Checkbox:
				primitive.SetInputCapture(inputCapture)
			case *tview.DropDown:
				primitive.SetInputCapture(inputCapture)
			}
		}

		popup := newDashboardSolidBackground(root)

		opt.Pages.RemovePage(PageDialog)
		opt.Pages.AddPage(PageDialog, centerPrimitive(popup, 86, dashboardStreamDialogHeight(visualRowCount)), true, true)

		if preferredFocus != nil {
			opt.App.SetFocus(preferredFocus)
			return
		}

		setFocusByIndex(0)
	}

	rebuild(nil)
}

func dashboardStreamDialogHeight(contentRows int) int {
	height := contentRows + 7

	if height < 18 {
		return 18
	}

	if height > 30 {
		return 30
	}

	return height
}

func dashboardStreamRemapText(stream astra.Stream) string {
	parts := make([]string, 0, 4)

	if strings.TrimSpace(stream.SetPNR) != "" {
		parts = append(parts, "set_pnr="+strings.TrimSpace(stream.SetPNR))
	}

	if strings.TrimSpace(stream.SetTSID) != "" {
		parts = append(parts, "set_tsid="+strings.TrimSpace(stream.SetTSID))
	}

	if strings.TrimSpace(stream.Map) != "" {
		parts = append(parts, strings.TrimSpace(stream.Map))
	}

	if strings.TrimSpace(stream.FilterNot) != "" {
		parts = append(parts, "filter~="+strings.TrimSpace(stream.FilterNot))
	}

	return strings.Join(parts, "&")
}

func dashboardApplyStreamRemap(stream *astra.Stream, value string) error {
	stream.SetPNR = ""
	stream.SetTSID = ""
	stream.Map = ""
	stream.FilterNot = ""

	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	parts := strings.Split(value, "&")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		switch {
		case strings.HasPrefix(part, "set_pnr="):
			stream.SetPNR = strings.TrimSpace(strings.TrimPrefix(part, "set_pnr="))

		case strings.HasPrefix(part, "set_tsid="):
			stream.SetTSID = strings.TrimSpace(strings.TrimPrefix(part, "set_tsid="))

		case strings.HasPrefix(part, "map="):
			stream.Map = strings.TrimSpace(strings.TrimPrefix(part, "map="))

		case strings.HasPrefix(part, "filter~="):
			stream.FilterNot = strings.TrimSpace(strings.TrimPrefix(part, "filter~="))

		default:
			if stream.Map != "" {
				return fmt.Errorf("multiple map values in remap field")
			}

			stream.Map = part
		}
	}

	return nil
}

func dashboardBuildStreamFromForm(
	base astra.Stream,
	enable bool,
	name string,
	hbbtvURL string,
	remap string,
	inputValues []string,
	outputValues []string,
	serviceProvider string,
) (astra.Stream, error) {
	base.Enable = enable
	base.Name = strings.TrimSpace(name)
	base.Type = "spts"
	base.HbbtvURL = strings.TrimSpace(hbbtvURL)

	serviceProvider = strings.TrimSpace(serviceProvider)
	if serviceProvider == "" {
		//serviceProvider = APPNAMEFULL
		serviceProvider = "Astra"
	}

	base.ServiceProvider = serviceProvider

	if base.Name == "" {
		return astra.Stream{}, fmt.Errorf("stream name is required")
	}

	inputs := dashboardCleanStringList(inputValues)
	if len(inputs) == 0 {
		return astra.Stream{}, fmt.Errorf("at least one input is required")
	}

	outputs := dashboardCleanStringList(outputValues)
	if len(outputs) == 0 {
		return astra.Stream{}, fmt.Errorf("at least one output is required")
	}

	base.Input = inputs
	base.Output = outputs

	if strings.TrimSpace(base.ID) == "" {
		base.ID = "s001"
	}

	if err := dashboardApplyStreamRemap(&base, remap); err != nil {
		return astra.Stream{}, err
	}

	base.ServiceName = astra.StreamServiceName(base.Name)

	return base, nil
}

func dashboardCleanStringList(values []string) []string {
	out := make([]string, 0, len(values))

	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}

		out = append(out, value)
	}

	return out
}

func dashboardInputSetCAM(input string, camID string) string {
	input = strings.TrimSpace(input)
	camID = strings.TrimSpace(camID)

	if input == "" {
		return input
	}

	parts := strings.Split(input, "&")
	out := make([]string, 0, len(parts)+1)

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if strings.HasPrefix(part, "cam=") {
			continue
		}

		out = append(out, part)
	}

	if camID != "" {
		out = append(out, "cam="+camID)
	}

	return strings.Join(out, "&")
}

func dashboardInputCAMID(input string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return ""
	}

	for _, part := range strings.Split(input, "&") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "cam=") {
			return strings.TrimSpace(strings.TrimPrefix(part, "cam="))
		}
	}

	return ""
}
