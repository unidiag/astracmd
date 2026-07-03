package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func dashboardGenerateSoftcamID(existing []AstraSoftcam) string {
	used := make(map[string]bool)

	for _, cam := range existing {
		id := strings.TrimSpace(cam.ID)
		if id != "" {
			used[id] = true
		}
	}

	for i := 1; i < 10000; i++ {
		id := fmt.Sprintf("cam%d", i)
		if !used[id] {
			return id
		}
	}

	return "cam"
}

func dashboardIsSoftcamKeyInput(value string) bool {
	if len(value) > 28 {
		return false
	}

	for _, r := range value {
		if r >= '0' && r <= '9' {
			continue
		}

		if r >= 'a' && r <= 'f' {
			continue
		}

		if r >= 'A' && r <= 'F' {
			continue
		}

		return false
	}

	return true
}

func dashboardIsSoftcamKeyValid(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return true
	}

	return len(value) == 28 && dashboardIsSoftcamKeyInput(value)
}

func (ui *UI) ShowSoftCAMDialog(
	conn AstraConnection,
	config AstraConfig,
	onSaved func(),
	onClose func(),
) {
	ui.pages.RemovePage(pageDialog)

	isUpdating := false
	isNewCam := true
	softcams := config.Softcams

	cam := AstraSoftcam{
		ID:   dashboardGenerateSoftcamID(softcams),
		Name: "",
		Type: "newcamd",
	}

	name := strings.TrimSpace(cam.Name)
	address := strings.TrimSpace(cam.Host)
	port := strings.TrimSpace(cam.Port)
	login := strings.TrimSpace(cam.User)
	password := strings.TrimSpace(cam.Pass)
	key := strings.TrimSpace(cam.Key)
	disableEMM := cam.DisableEMM
	remove := false

	form := tview.NewForm()
	form.SetBorder(true)
	form.SetTitle(" SoftCAM ")
	form.SetTitleAlign(tview.AlignCenter)
	form.SetButtonsAlign(tview.AlignCenter)

	updateTitle := func() {
		if isNewCam {
			form.SetTitle(" SoftCAM ")
			return
		}

		id := strings.TrimSpace(cam.ID)
		if id != "" {
			form.SetTitle(" SoftCAM #" + id + " ")
			return
		}

		form.SetTitle(" SoftCAM ")
	}

	updateTitle()

	options := make([]string, 0, len(softcams)+1)
	optionIndexes := make([]int, 0, len(softcams)+1)

	options = append(options, "##### NEW #####")
	optionIndexes = append(optionIndexes, -1)

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
		item := softcams[sourceIndex]

		options = append(options, tview.Escape(item.DisplayName()))
		optionIndexes = append(optionIndexes, sourceIndex)
	}

	selectedOption := 0

	nameField := tview.NewInputField().
		SetText(name).
		SetFieldWidth(64).
		SetChangedFunc(func(value string) {
			if isUpdating {
				return
			}
			name = value
		})

	addressField := tview.NewInputField().
		SetText(address).
		SetFieldWidth(64).
		SetChangedFunc(func(value string) {
			if isUpdating {
				return
			}
			address = value
		})

	portField := tview.NewInputField().
		SetText(port).
		SetFieldWidth(64).
		SetAcceptanceFunc(tview.InputFieldInteger).
		SetChangedFunc(func(value string) {
			if isUpdating {
				return
			}
			port = value
		})

	loginField := tview.NewInputField().
		SetText(login).
		SetFieldWidth(64).
		SetChangedFunc(func(value string) {
			if isUpdating {
				return
			}
			login = value
		})

	passwordField := tview.NewInputField().
		SetText(password).
		SetFieldWidth(64).
		SetChangedFunc(func(value string) {
			if isUpdating {
				return
			}
			password = value
		})

	keyField := tview.NewInputField().
		SetText(key).
		SetFieldWidth(64).
		SetAcceptanceFunc(func(value string, lastChar rune) bool {
			return dashboardIsSoftcamKeyInput(value)
		}).
		SetChangedFunc(func(value string) {
			if isUpdating {
				return
			}
			key = value
		})

	disableEMMField := tview.NewCheckbox().
		SetChecked(disableEMM).
		SetChangedFunc(func(value bool) {
			disableEMM = value
		})

	removeField := tview.NewCheckbox().
		SetChecked(remove).
		SetChangedFunc(func(value bool) {
			remove = value
		})

	loadCamToFields := func(item AstraSoftcam, newCam bool) {
		isUpdating = true
		defer func() {
			isUpdating = false
		}()

		isNewCam = newCam
		cam = item

		name = strings.TrimSpace(item.Name)
		address = strings.TrimSpace(item.Host)
		port = strings.TrimSpace(item.Port)
		login = strings.TrimSpace(item.User)
		password = strings.TrimSpace(item.Pass)
		key = strings.TrimSpace(item.Key)
		disableEMM = item.DisableEMM
		remove = false

		nameField.SetText(name)
		addressField.SetText(address)
		portField.SetText(port)
		loginField.SetText(login)
		passwordField.SetText(password)
		keyField.SetText(key)
		disableEMMField.SetChecked(disableEMM)
		removeField.SetChecked(remove)

		updateTitle()
	}

	form.AddDropDown("SOFTCAM", options, selectedOption, func(_ string, index int) {
		if index < 0 || index >= len(optionIndexes) {
			return
		}

		sourceIndex := optionIndexes[index]
		if sourceIndex < 0 {
			loadCamToFields(AstraSoftcam{
				ID:   dashboardGenerateSoftcamID(softcams),
				Name: "",
				Type: "newcamd",
			}, true)
			return
		}

		loadCamToFields(softcams[sourceIndex], false)
	})

	form.AddFormItem(nameField.SetLabel("NAME"))
	form.AddFormItem(addressField.SetLabel("ADDRESS"))
	form.AddFormItem(portField.SetLabel("PORT"))
	form.AddFormItem(loginField.SetLabel("LOGIN"))
	form.AddFormItem(passwordField.SetLabel("PASSWORD"))
	form.AddFormItem(keyField.SetLabel("KEY"))
	form.AddFormItem(disableEMMField.SetLabel("Disable EMM"))
	form.AddFormItem(removeField.SetLabel("[red]Remove[-]"))

	closeDialog := func() {
		ui.pages.RemovePage(pageDialog)
		if onClose != nil {
			onClose()
		}
	}

	showSoftCAMTestModal := func(title string, message string) {
		modal := tview.NewModal().
			SetText(message).
			AddButtons([]string{"OK"}).
			SetDoneFunc(func(buttonIndex int, buttonLabel string) {
				ui.pages.RemovePage(pageDialog + "_test")
				ui.app.SetFocus(form)
			})

		modal.SetTitle(" " + title + " ")
		modal.SetTitleAlign(tview.AlignCenter)

		ui.pages.AddPage(pageDialog+"_test", modal, true, true)
		ui.app.SetFocus(modal)
	}

	formatSoftCAMTestSuccess := func(result AstraTestSoftcamResult) string {
		var b strings.Builder

		b.WriteString("SoftCAM test successful\n\n")
		b.WriteString(fmt.Sprintf("Status:   %d\n", result.Status))
		b.WriteString(fmt.Sprintf("CAID:     %d\n", result.CAID))
		b.WriteString(fmt.Sprintf("AU:       %v\n", result.AU))
		b.WriteString(fmt.Sprintf("UA:       %s\n", result.UA))
		b.WriteString(fmt.Sprintf("ECM rate: %d\n", result.ECMRate))
		b.WriteString(fmt.Sprintf("EMM rate: %d\n", result.EMMRate))

		if len(result.Idents) > 0 {
			b.WriteString("\nIdents:\n")
			for _, ident := range result.Idents {
				b.WriteString(fmt.Sprintf("  ID: %s  SA: %s\n", ident.ID, ident.SA))
			}
		}

		return b.String()
	}

	//  █████╗ ██████╗ ██████╗ ██╗  ██╗   ██╗
	// ██╔══██╗██╔══██╗██╔══██╗██║  ╚██╗ ██╔╝
	// ███████║██████╔╝██████╔╝██║   ╚████╔╝
	// ██╔══██║██╔═══╝ ██╔═══╝ ██║    ╚██╔╝
	// ██║  ██║██║     ██║     ███████╗██║
	// ╚═╝  ╚═╝╚═╝     ╚═╝     ╚══════╝╚═╝

	applySoftCAM := func() {
		currentID := strings.TrimSpace(cam.ID)
		if currentID == "" || isNewCam {
			currentID = dashboardGenerateSoftcamID(softcams)
		}

		if remove && !isNewCam {
			result := AstraRemoveSoftcam(context.Background(), conn, config.GID, currentID)
			if !result.OK {
				if result.Err != nil {
					ui.ShowError(result.Err.Error(), form)
				} else {
					ui.ShowError("SoftCAM remove failed", form)
				}
				return
			}

			ui.pages.RemovePage(pageDialog)

			if onSaved != nil {
				onSaved()
			}

			return
		}

		currentType := strings.TrimSpace(cam.Type)
		if currentType == "" {
			currentType = "newcamd"
		}

		key = strings.TrimSpace(key)

		if !dashboardIsSoftcamKeyValid(key) {
			ui.ShowError("SoftCAM key must be empty or exactly 28 HEX characters", form)
			return
		}

		savedCam := AstraSoftcam{
			ID:         currentID,
			Name:       strings.TrimSpace(name),
			Type:       currentType,
			Host:       strings.TrimSpace(address),
			Port:       strings.TrimSpace(port),
			User:       strings.TrimSpace(login),
			Pass:       strings.TrimSpace(password),
			Key:        strings.TrimSpace(key),
			DisableEMM: disableEMM,
		}

		if savedCam.Name == "" {
			ui.ShowError("SoftCAM name is required", form)
			return
		}

		if savedCam.Host == "" {
			ui.ShowError("SoftCAM address is required", form)
			return
		}

		if savedCam.Port == "" {
			ui.ShowError("SoftCAM port is required", form)
			return
		}

		if savedCam.User == "" {
			ui.ShowError("SoftCAM login is required", form)
			return
		}

		if savedCam.Pass == "" {
			ui.ShowError("SoftCAM password is required", form)
			return
		}

		result := AstraSaveSoftcam(context.Background(), conn, config.GID, savedCam)
		if !result.OK {
			if result.Err != nil {
				ui.ShowError(result.Err.Error(), form)
			} else {
				ui.ShowError("SoftCAM save failed", form)
			}
			return
		}

		ui.pages.RemovePage(pageDialog)

		if onSaved != nil {
			onSaved()
		}
	}

	// ████████╗███████╗███████╗████████╗
	// ╚══██╔══╝██╔════╝██╔════╝╚══██╔══╝
	//    ██║   █████╗  ███████╗   ██║
	//    ██║   ██╔══╝  ╚════██║   ██║
	//    ██║   ███████╗███████║   ██║
	//    ╚═╝   ╚══════╝╚══════╝   ╚═╝

	testInProgress := false
	var testButton *tview.Button
	var testSpinnerCancel context.CancelFunc

	setTestButtonIdleStyle := func() {
		if testButton == nil {
			return
		}

		testButton.SetLabel("TEST")
		testButton.SetLabelColor(tcell.ColorWhite)
		testButton.SetBackgroundColor(tcell.ColorDarkCyan)
		testButton.SetLabelColorActivated(tcell.ColorBlack)
		testButton.SetBackgroundColorActivated(tcell.ColorWhite)
	}

	setTestButtonTestingStyle := func() {
		if testButton == nil {
			return
		}

		testButton.SetLabelColor(tcell.ColorBlack)
		testButton.SetBackgroundColor(tcell.ColorGreen)
		testButton.SetLabelColorActivated(tcell.ColorBlack)
		testButton.SetBackgroundColorActivated(tcell.ColorGreen)
	}

	startTestSpinner := func() {
		if testSpinnerCancel != nil {
			testSpinnerCancel()
		}

		spinnerCtx, cancel := context.WithCancel(context.Background())
		testSpinnerCancel = cancel

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
						if testButton != nil {
							testButton.SetLabel("TEST " + frame)
						}
					})
				}
			}
		}()
	}

	stopTestSpinner := func() {
		if testSpinnerCancel != nil {
			testSpinnerCancel()
			testSpinnerCancel = nil
		}

		setTestButtonIdleStyle()
	}

	testSoftCAM := func() {
		if testInProgress {
			return
		}

		currentID := strings.TrimSpace(cam.ID)
		if currentID == "" || isNewCam {
			currentID = dashboardGenerateSoftcamID(softcams)
		}

		currentType := strings.TrimSpace(cam.Type)
		if currentType == "" {
			currentType = "newcamd"
		}

		key = strings.TrimSpace(key)

		if !dashboardIsSoftcamKeyValid(key) {
			showSoftCAMTestModal("SoftCAM test failed", "SoftCAM key must be empty or exactly 28 HEX characters")
			return
		}

		testCam := AstraSoftcam{
			ID:         currentID,
			Name:       strings.TrimSpace(name),
			Type:       currentType,
			Host:       strings.TrimSpace(address),
			Port:       strings.TrimSpace(port),
			User:       strings.TrimSpace(login),
			Pass:       strings.TrimSpace(password),
			Key:        strings.TrimSpace(key),
			DisableEMM: disableEMM,
		}

		if testCam.Name == "" {
			showSoftCAMTestModal("SoftCAM test failed", "SoftCAM name is required")
			return
		}

		if testCam.Host == "" {
			showSoftCAMTestModal("SoftCAM test failed", "SoftCAM address is required")
			return
		}

		if testCam.Port == "" {
			showSoftCAMTestModal("SoftCAM test failed", "SoftCAM port is required")
			return
		}

		if testCam.User == "" {
			showSoftCAMTestModal("SoftCAM test failed", "SoftCAM login is required")
			return
		}

		if testCam.Pass == "" {
			showSoftCAMTestModal("SoftCAM test failed", "SoftCAM password is required")
			return
		}

		testInProgress = true

		if testButton != nil {
			testButton.SetLabel("TEST |")
			setTestButtonTestingStyle()
		}

		startTestSpinner()

		go func() {
			result := AstraTestSoftcam(context.Background(), conn, testCam)

			ui.app.QueueUpdateDraw(func() {
				testInProgress = false
				stopTestSpinner()

				if !result.OK {
					if result.Err != nil {
						showSoftCAMTestModal("SoftCAM test failed", result.Err.Error())
						return
					}

					showSoftCAMTestModal("SoftCAM test failed", result.ErrorMsg)
					return
				}

				showSoftCAMTestModal("SoftCAM test", formatSoftCAMTestSuccess(result))
			})
		}()
	}

	//  ██████╗██╗      ██████╗ ███╗   ██╗███████╗
	// ██╔════╝██║     ██╔═══██╗████╗  ██║██╔════╝
	// ██║     ██║     ██║   ██║██╔██╗ ██║█████╗
	// ██║     ██║     ██║   ██║██║╚██╗██║██╔══╝
	// ╚██████╗███████╗╚██████╔╝██║ ╚████║███████╗
	//  ╚═════╝╚══════╝ ╚═════╝ ╚═╝  ╚═══╝╚══════╝

	cloneSoftCAM := func() {
		sourceID := strings.TrimSpace(cam.ID)
		if sourceID == "" || isNewCam {
			ui.ShowError("Select existing SoftCAM to clone", form)
			return
		}

		newID := dashboardGenerateSoftcamID(softcams)

		newName := strings.TrimSpace(name)
		if newName == "" {
			newName = strings.TrimSpace(cam.Name)
		}
		if newName == "" {
			newName = "SoftCAM"
		}
		newName += " [CLONE]"

		clonedCam := AstraSoftcam{
			ID:         newID,
			Name:       newName,
			Type:       strings.TrimSpace(cam.Type),
			Host:       strings.TrimSpace(address),
			Port:       strings.TrimSpace(port),
			User:       strings.TrimSpace(login),
			Pass:       strings.TrimSpace(password),
			Key:        strings.TrimSpace(key),
			DisableEMM: disableEMM,
		}

		if clonedCam.Type == "" {
			clonedCam.Type = "newcamd"
		}

		selectedOption = 0
		loadCamToFields(clonedCam, true)
	}

	// ███████╗████████╗██████╗ ███████╗ █████╗ ███╗   ███╗███████╗
	// ██╔════╝╚══██╔══╝██╔══██╗██╔════╝██╔══██╗████╗ ████║██╔════╝
	// ███████╗   ██║   ██████╔╝█████╗  ███████║██╔████╔██║███████╗
	// ╚════██║   ██║   ██╔══██╗██╔══╝  ██╔══██║██║╚██╔╝██║╚════██║
	// ███████║   ██║   ██║  ██║███████╗██║  ██║██║ ╚═╝ ██║███████║
	// ╚══════╝   ╚═╝   ╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝╚═╝     ╚═╝╚══════╝

	showSoftCAMStreamsModal := func(camID string) {
		camID = strings.TrimSpace(camID)
		if camID == "" {
			ui.ShowError("SoftCAM id is empty", form)
			return
		}

		type streamUsage struct {
			Name string
			ID   string
		}

		usedBy := make([]streamUsage, 0)

		for _, stream := range config.Streams {
			streamUsesCAM := false

			for _, input := range stream.Input {
				inputCAMID := dashboardInputCAMID(input)
				if inputCAMID == camID {
					streamUsesCAM = true
					break
				}
			}

			if !streamUsesCAM {
				continue
			}

			name := strings.TrimSpace(stream.Name)
			if name == "" {
				name = "Unnamed stream"
			}

			usedBy = append(usedBy, streamUsage{
				Name: name,
				ID:   strings.TrimSpace(stream.ID),
			})
		}

		var b strings.Builder

		lenUsedBy := len(usedBy)

		if lenUsedBy > 0 {
			b.WriteString("\n")

			for i, item := range usedBy {
				if item.ID != "" {
					b.WriteString(fmt.Sprintf("%d. %s  #%s\n", i+1, item.Name, item.ID))
				} else {
					b.WriteString(fmt.Sprintf("%d. %s\n", i+1, item.Name))
				}
			}
		} else {
			b.WriteString("Not found streams!")
		}

		modal := tview.NewModal().
			SetText(b.String()).
			AddButtons([]string{"OK"}).
			SetDoneFunc(func(buttonIndex int, buttonLabel string) {
				ui.pages.RemovePage(pageDialog + "_streams")
				ui.app.SetFocus(form)
			})

		modal.SetTitle(fmt.Sprintf("SoftCAM streams #%s\n\n", camID))
		modal.SetTitleAlign(tview.AlignCenter)

		ui.pages.AddPage(pageDialog+"_streams", modal, true, true)
		ui.app.SetFocus(modal)
	}

	showStreams := func() {
		if isNewCam {
			ui.ShowError("Select existing SoftCAM to view streams", form)
			return
		}

		camID := strings.TrimSpace(cam.ID)
		if camID == "" {
			ui.ShowError("SoftCAM id is empty", form)
			return
		}

		showSoftCAMStreamsModal(camID)
	}

	// ██████╗ ██╗   ██╗████████╗████████╗ ██████╗ ███╗   ██╗███████╗
	// ██╔══██╗██║   ██║╚══██╔══╝╚══██╔══╝██╔═══██╗████╗  ██║██╔════╝
	// ██████╔╝██║   ██║   ██║      ██║   ██║   ██║██╔██╗ ██║███████╗
	// ██╔══██╗██║   ██║   ██║      ██║   ██║   ██║██║╚██╗██║╚════██║
	// ██████╔╝╚██████╔╝   ██║      ██║   ╚██████╔╝██║ ╚████║███████║
	// ╚═════╝  ╚═════╝    ╚═╝      ╚═╝    ╚═════╝ ╚═╝  ╚═══╝╚══════╝

	form.AddButton("APPLY", applySoftCAM)
	form.AddButton("TEST", testSoftCAM)
	form.AddButton("CLONE", cloneSoftCAM)
	form.AddButton("STREAMS", showStreams)
	form.AddButton("CANCEL", closeDialog)

	testButton = form.GetButton(1)
	setTestButtonIdleStyle()

	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if ui.HandleGlobalKeys(event) {
			return nil
		}

		switch event.Key() {
		case tcell.KeyEsc:
			closeDialog()
			return nil
		}

		return event
	})

	ui.pages.AddPage(pageDialog, centerPrimitive(form, 88, 24), true, true)
	ui.app.SetFocus(form)
}
