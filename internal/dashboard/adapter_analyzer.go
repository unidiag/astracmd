package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"main/internal/astra"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func ShowAdapterAnalyzerDialog(
	opt Options,
	conn astra.Connection,
	adapter astra.Adapter,
	existingStreams []astra.Stream,
	onScanOK func(astra.Adapter, int),
	onError func(error),
) {
	adapterID := strings.TrimSpace(adapter.ID)
	if adapterID == "" {
		opt.ShowError("Adapter ID is empty", nil)
		return
	}

	knownStreams := append([]astra.Stream(nil), existingStreams...)
	scanInProgress := false

	ctx, cancel := context.WithCancel(context.Background())

	status := tview.NewTextView()
	status.SetDynamicColors(true)
	status.SetTextAlign(tview.AlignCenter)
	status.SetText("[yellow]Starting adapter analyzer...[-]")

	table := tview.NewTable()
	table.SetBorder(false)
	table.SetSelectable(false, false)
	table.SetEvaluateAllRows(true)

	statusFlags := tview.NewTextView()
	statusFlags.SetDynamicColors(true)
	statusFlags.SetTextAlign(tview.AlignCenter)
	statusFlags.SetText("[gray]SIGNAL CARRIER FEC SYNC LOCK[-]")

	setAdapterAnalyzerLoading(table)

	bars := newAdapterAnalyzerBarsView()

	diseqcInput := tview.NewInputField()
	diseqcInput.SetLabel("DiSEqC command ")
	diseqcInput.SetLabelColor(tcell.ColorGray)
	//diseqcInput.SetTextColor(tcell.ColorWhite)
	diseqcInput.SetFieldTextColor(tcell.ColorBlack)
	diseqcInput.SetFieldBackgroundColor(tcell.ColorGreen)
	diseqcInput.SetFieldWidth(40)
	diseqcInput.SetPlaceholder(defaultDiseqcBytes)
	diseqcInput.SetText(diseqcInputTextFromConfig(adapter.Diseqc))

	sendDiseqcButton := tview.NewButton("SEND")

	diseqcRow := tview.NewFlex()
	diseqcRow.SetDirection(tview.FlexColumn)
	diseqcRow.AddItem(tview.NewBox(), 2, 0, false)
	diseqcRow.AddItem(diseqcInput, 0, 1, true)
	diseqcRow.AddItem(tview.NewBox(), 1, 0, false)
	diseqcRow.AddItem(sendDiseqcButton, 8, 0, true)
	diseqcRow.AddItem(tview.NewBox(), 2, 0, false)

	footer := tview.NewTextView()
	footer.SetDynamicColors(true)
	footer.SetTextAlign(tview.AlignCenter)
	footer.SetText("[gray]Space — scan & add new    Tab — DiSEqC    Esc — close[-]")

	body := tview.NewFlex()
	body.SetDirection(tview.FlexRow)
	body.SetBorder(true)
	body.SetTitle(fmt.Sprintf(" Adapter analyzer: %s ", tview.Escape(adapter.DisplayName())))
	body.SetTitleAlign(tview.AlignCenter)

	statusFlagsSpacer := tview.NewBox()

	body.AddItem(status, 1, 0, false)
	body.AddItem(table, 3, 0, false)
	body.AddItem(statusFlags, 1, 0, false)
	body.AddItem(statusFlagsSpacer, 1, 0, false)
	body.AddItem(bars, 8, 0, false)

	body.AddItem(tview.NewBox(), 1, 0, false)
	body.AddItem(diseqcRow, 1, 0, false)
	body.AddItem(tview.NewBox(), 1, 0, false)

	body.AddItem(footer, 1, 0, false)

	closeDialog := func() {
		cancel()
		opt.Pages.RemovePage(PageDialog)
	}

	runScan := func() {

		if denyRestrictedStatus(status) {
			return
		}

		if scanInProgress {
			return
		}

		scanInProgress = true

		status.SetText("[yellow]Scanning adapter and adding new streams...[-]")
		footer.SetText("[yellow]Scanning... please wait    Esc — close[-]")

		parsed := adapter
		scanDelay := 3 * time.Second

		go func() {
			result := astra.AstraScanAddStreams(
				context.Background(),
				conn,
				parsed,
				knownStreams,
				scanDelay,
				opt.ServiceProvider(),
			)

			opt.App.QueueUpdateDraw(func() {
				scanInProgress = false
				footer.SetText("[gray]Space — scan & add new    Esc — close[-]")

				if !result.OK {
					if result.Err != nil {
						status.SetText("[red]Scan failed[-]")

						if onError != nil {
							onError(result.Err)
						}

						return
					}

					status.SetText("[red]Scan failed[-]")

					if onError != nil {
						onError(fmt.Errorf("scan failed"))
					}

					return
				}

				if len(result.Streams) > 0 {
					knownStreams = append(knownStreams, result.Streams...)
				}

				if result.Count == 0 {
					status.SetText("[green]Scan complete[-] [gray]no new streams[-]")
				} else {
					status.SetText(fmt.Sprintf(
						"[green]Scan complete[-] [gray]added:%d[-]",
						result.Count,
					))
				}

				if onScanOK != nil {
					onScanOK(parsed, result.Count)
				}
			})
		}()
	}

	// send diseqc cmd
	sendDiseqcCommand := func() {

		if denyRestrictedStatus(status) {
			return
		}

		rawCommand := strings.TrimSpace(diseqcInput.GetText())

		cmdAdapter := adapter
		cmdAdapter.Enable = true

		if rawCommand == "" {
			cmdAdapter.Diseqc = ""
			cmdAdapter.DiseqcMode = ""

			status.SetText("[yellow]Saving adapter without DiSEqC command...[-]")
		} else {
			cmd, err := buildDiseqcCommand(rawCommand, adapter.Polarization)
			if err != nil {
				status.SetText("[red]" + err.Error() + "[-]")
				return
			}

			cmdAdapter.DiseqcMode = "cmd"
			cmdAdapter.Diseqc = cmd

			status.SetText("[yellow]Sending DiSEqC command...[-]")
		}

		go func() {
			result := astra.AstraSaveAdapter(context.Background(), conn, cmdAdapter)

			opt.App.QueueUpdateDraw(func() {
				if !result.OK {
					status.SetText("[red]DiSEqC command failed[-]")

					if onError != nil {
						onError(result.Err)
					}

					return
				}

				if rawCommand == "" {
					status.SetText("[green]Adapter saved without DiSEqC command[-]")
				} else {
					status.SetText("[green]DiSEqC command sent[-]")
				}

				opt.App.SetFocus(body)
			})
		}()
	}

	sendDiseqcButton.SetSelectedFunc(sendDiseqcCommand)

	diseqcInput.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			sendDiseqcCommand()

		case tcell.KeyEsc:
			closeDialog()
		}
	})

	body.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if opt.HandleGlobalKeys(event) {
			return nil
		}

		switch event.Key() {
		case tcell.KeyEsc:
			closeDialog()
			return nil

		case tcell.KeyTab:
			switch opt.App.GetFocus() {
			case diseqcInput:
				opt.App.SetFocus(sendDiseqcButton)
			case sendDiseqcButton:
				opt.App.SetFocus(body)
			default:
				opt.App.SetFocus(diseqcInput)
			}

			return nil

		case tcell.KeyRune:
			if event.Rune() == ' ' {
				if opt.App.GetFocus() == diseqcInput {
					return event
				}

				runScan()
				return nil
			}
		}

		return event
	})

	opt.Pages.AddPage(PageDialog, centerPrimitive(body, 76, 20), true, true)
	opt.App.SetFocus(body)

	go runAdapterAnalyzer(opt, ctx, conn, adapter, status, table, statusFlags, bars)
}

func runAdapterAnalyzer(
	opt Options,
	ctx context.Context,
	conn astra.Connection,
	adapter astra.Adapter,
	status *tview.TextView,
	table *tview.Table,
	statusFlags *tview.TextView,
	bars *adapterAnalyzerBarsView,
) {
	ws, err := astra.AstraConnectWebSocket(ctx, conn)
	if err != nil {
		opt.App.QueueUpdateDraw(func() {
			status.SetText("[red]WebSocket error[-]")
			setAdapterAnalyzerError(table, err)
		})
		return
	}

	defer ws.Close()

	opt.App.QueueUpdateDraw(func() {
		status.SetText("[green]WebSocket connected[-]")
	})

	messages := make(chan astra.AstraWSMessage, 32)
	go ws.ReadLoop(ctx, messages)

	adapterID := strings.TrimSpace(adapter.ID)

	for msg := range messages {
		if msg.Err != nil {
			opt.App.QueueUpdateDraw(func() {
				status.SetText("[red]WebSocket stopped[-]")
				setAdapterAnalyzerError(table, msg.Err)
			})
			return
		}

		var envelope astra.AstraWSEnvelope
		if err := json.Unmarshal(msg.Raw, &envelope); err != nil {
			continue
		}

		if envelope.Scope != "adapter_event" {
			continue
		}

		var event astra.AstraWSAdapterEvent
		if err := json.Unmarshal(msg.Raw, &event); err != nil {
			continue
		}

		if strings.TrimSpace(event.DVBID) != adapterID {
			continue
		}

		state := astra.AdapterState{
			Signal:   event.Signal,
			SignalDB: event.SignalDB,
			Bitrate:  event.Bitrate,
			UNC:      event.UNC,
			SNRDB:    event.SNRDB,
			SNR:      event.SNR,
			BER:      event.BER,
			Status:   event.Status,
		}

		opt.App.QueueUpdateDraw(func() {
			status.SetText(formatAdapterAnalyzerStatus(adapterID, state))
			statusFlags.SetText(formatAdapterStatusFlagsColored(state.Status))
			setAdapterAnalyzerTable(table, state)
			bars.SetState(state)
		})
	}
}

func formatAdapterAnalyzerStatus(adapterID string, state astra.AdapterState) string {
	statusColor := "green"
	if state.Bitrate <= 0 || state.Signal <= 0 || state.SNR <= 0 {
		statusColor = "red"
	}

	if state.BER > 0 || state.UNC > 0 {
		statusColor = "yellow"
	}

	if state.BER > 100 || state.UNC > 100 {
		statusColor = "red"
	}

	return fmt.Sprintf(
		"[%s]%s[-] [gray]status:%d[-]",
		statusColor,
		tview.Escape(adapterID),
		state.Status,
	)
}

func setAdapterAnalyzerLoading(table *tview.Table) {
	table.Clear()

	table.SetCell(0, 0, tview.NewTableCell(" Waiting for adapter data...").
		SetTextColor(tcell.ColorYellow).
		SetExpansion(1))
}

func setAdapterAnalyzerError(table *tview.Table, err error) {
	table.Clear()

	text := "unknown error"
	if err != nil {
		text = err.Error()
	}

	table.SetCell(0, 0, tview.NewTableCell(" "+text).
		SetTextColor(tcell.ColorRed).
		SetExpansion(1))
}

func setAdapterAnalyzerTable(table *tview.Table, state astra.AdapterState) {
	table.Clear()

	rows := []struct {
		Name  string
		Value string
		Color tcell.Color
	}{
		{
			Name:  "Bitrate",
			Value: fmt.Sprintf("%d kbps", state.Bitrate),
			Color: adapterAnalyzerBitrateColor(state.Bitrate),
		},
		{
			Name:  "BER / UNC",
			Value: fmt.Sprintf("%s / %s", astra.FormatAdapterCounter(state.BER), astra.FormatAdapterCounter(state.UNC)),
			Color: adapterAnalyzerErrorColor(state.BER, state.UNC),
		},
	}

	for row, item := range rows {
		table.SetCell(row, 0, tview.NewTableCell(" "+item.Name).
			SetTextColor(tcell.ColorGray).
			SetExpansion(1))

		table.SetCell(row, 1, tview.NewTableCell(item.Value).
			SetTextColor(item.Color).
			SetAlign(tview.AlignRight).
			SetExpansion(1).
			SetExpansion(1))
	}
}

func adapterAnalyzerBitrateColor(bitrate int) tcell.Color {
	if bitrate > 0 {
		return tcell.ColorGreen
	}

	return tcell.ColorRed
}

func adapterAnalyzerSignalColor(signal int, snr int) tcell.Color {
	signal = astra.NormalizeAdapterSignal(signal)
	snr = astra.NormalizeAdapterSignal(snr)

	if signal <= 0 || snr <= 0 {
		return tcell.ColorRed
	}

	if signal < 40 || snr < 40 {
		return tcell.ColorYellow
	}

	return tcell.ColorGreen
}

func adapterAnalyzerErrorColor(ber int, unc int) tcell.Color {
	if ber > 100 || unc > 100 {
		return tcell.ColorRed
	}

	if ber > 0 || unc > 0 {
		return tcell.ColorYellow
	}

	return tcell.ColorWhite
}

func formatAdapterAnalyzerDB(value int) string {
	if value == 0 {
		return "-"
	}

	return fmt.Sprintf("%.1f dB", float64(value)/100.0)
}

type adapterAnalyzerBarsView struct {
	*tview.Box
	state astra.AdapterState
}

func newAdapterAnalyzerBarsView() *adapterAnalyzerBarsView {
	return &adapterAnalyzerBarsView{
		Box: tview.NewBox(),
	}
}

func (v *adapterAnalyzerBarsView) SetState(state astra.AdapterState) {
	v.state = state
}

func (v *adapterAnalyzerBarsView) Draw(screen tcell.Screen) {
	v.Box.DrawForSubclass(screen, v)

	x, y, width, _ := v.GetInnerRect()
	if width <= 0 {
		return
	}

	state := v.state

	rows := []struct {
		Label   string
		Value   string
		Percent int
	}{
		{
			Label:   "Signal",
			Value:   fmt.Sprintf("%d%%", astra.NormalizeAdapterSignal(state.Signal)),
			Percent: astra.NormalizeAdapterSignal(state.Signal),
		},
		{
			Label:   "SNR",
			Value:   fmt.Sprintf("%d%%", astra.NormalizeAdapterSignal(state.SNR)),
			Percent: astra.NormalizeAdapterSignal(state.SNR),
		},
		{
			Label:   "Signal dB",
			Value:   formatAdapterAnalyzerDB(state.SignalDB),
			Percent: adapterSignalDBPercent(state.SignalDB),
		},
		{
			Label:   "SNR dB",
			Value:   formatAdapterAnalyzerDB(state.SNRDB),
			Percent: adapterSNRDBPercent(state.SNRDB),
		},
	}

	for i, row := range rows {
		drawAdapterAnalyzerBar(
			screen,
			x+1,
			y+i*2,
			width-2,
			row.Label,
			row.Value,
			row.Percent,
		)
	}
}

func drawAdapterAnalyzerBar(
	screen tcell.Screen,
	x int,
	y int,
	width int,
	label string,
	value string,
	percent int,
) {
	if width <= 0 {
		return
	}

	percent = clampAdapterPercent(percent)

	labelWidth := 10
	valueWidth := 10

	barWidth := width - labelWidth - valueWidth - 4
	if barWidth < 4 {
		barWidth = 4
	}

	labelStyle := tcell.StyleDefault.Foreground(tcell.ColorGray)
	valueStyle := adapterBarStyle(percent)
	emptyStyle := tcell.StyleDefault.Foreground(tcell.ColorDarkSlateGray)

	drawText(screen, x, y, label, labelStyle)

	barX := x + labelWidth + 1

	screen.SetContent(barX, y, '[', nil, tcell.StyleDefault.Foreground(tcell.ColorGray))
	barX++

	filled := barWidth * percent / 100

	for i := 0; i < barWidth; i++ {
		style := emptyStyle
		ch := '░'

		if i < filled {
			style = valueStyle
			ch = '█'
		}

		screen.SetContent(barX+i, y, ch, nil, style)
	}

	screen.SetContent(barX+barWidth, y, ']', nil, tcell.StyleDefault.Foreground(tcell.ColorGray))

	valueX := barX + barWidth + 2
	drawText(screen, valueX, y, value, valueStyle)
}

func drawText(screen tcell.Screen, x int, y int, text string, style tcell.Style) {
	for i, ch := range []rune(text) {
		screen.SetContent(x+i, y, ch, nil, style)
	}
}

func adapterBarStyle(percent int) tcell.Style {
	switch {
	case percent <= 0:
		return tcell.StyleDefault.Foreground(tcell.ColorRed)

	case percent < 40:
		return tcell.StyleDefault.Foreground(tcell.ColorYellow)

	default:
		return tcell.StyleDefault.Foreground(tcell.ColorGreen)
	}
}

func clampAdapterPercent(value int) int {
	if value < 0 {
		return 0
	}

	if value > 100 {
		return 100
	}

	return value
}

func adapterSignalDBPercent(value int) int {
	if value == 0 {
		return 0
	}

	db := float64(value) / 100.0

	// For signal level we map -100..0 dB to 0..100%.
	percent := int((db + 100.0) * 100.0 / 100.0)

	return clampAdapterPercent(percent)
}

func adapterSNRDBPercent(value int) int {
	if value == 0 {
		return 0
	}

	db := float64(value) / 100.0

	// For SNR we map 0..30 dB to 0..100%.
	percent := int(db * 100.0 / 30.0)

	return clampAdapterPercent(percent)
}

func formatAdapterStatusFlagsColored(status int) string {
	flags := []struct {
		Mask int
		Name string
	}{
		{0x01, "SIGNAL"},
		{0x02, "CARRIER"},
		{0x04, "FEC"},
		{0x08, "SYNC"},
		{0x10, "LOCK"},
	}

	parts := make([]string, 0, len(flags))

	for _, flag := range flags {
		if status&flag.Mask != 0 {
			parts = append(parts, fmt.Sprintf("[black:green:b] %s [-:-:-]", flag.Name))
			continue
		}

		parts = append(parts, fmt.Sprintf("[white:red:b] %s [-:-:-]", flag.Name))
	}

	return strings.Join(parts, " ")
}

// ██████╗ ██╗███████╗███████╗ ██████╗  ██████╗    ██╗  ██╗███████╗██╗     ██████╗ ███████╗██████╗ ███████╗
// ██╔══██╗██║██╔════╝██╔════╝██╔═══██╗██╔════╝    ██║  ██║██╔════╝██║     ██╔══██╗██╔════╝██╔══██╗██╔════╝
// ██║  ██║██║███████╗█████╗  ██║   ██║██║         ███████║█████╗  ██║     ██████╔╝█████╗  ██████╔╝███████╗
// ██║  ██║██║╚════██║██╔══╝  ██║▄▄ ██║██║         ██╔══██║██╔══╝  ██║     ██╔═══╝ ██╔══╝  ██╔══██╗╚════██║
// ██████╔╝██║███████║███████╗╚██████╔╝╚██████╗    ██║  ██║███████╗███████╗██║     ███████╗██║  ██║███████║
// ╚═════╝ ╚═╝╚══════╝╚══════╝ ╚══▀▀═╝  ╚═════╝    ╚═╝  ╚═╝╚══════╝╚══════╝╚═╝     ╚══════╝╚═╝  ╚═╝╚══════╝

const defaultDiseqcBytes = "E0 31 6E E0 00"

func diseqcInputTextFromConfig(current string) string {
	current = strings.TrimSpace(current)
	if current == "" {
		return ""
	}

	start := strings.Index(current, "[")
	stop := strings.Index(current, "]")

	if start >= 0 && stop > start {
		value := strings.TrimSpace(current[start+1 : stop])
		if value != "" {
			return value
		}
	}

	return current
}

func buildDiseqcCommand(input string, polarization string) (string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", nil
	}

	if looksLikeFullDiseqcCommand(input) {
		return normalizeFullDiseqcCommand(input)
	}

	bytesText, err := normalizeDiseqcBytes(input)
	if err != nil {
		return "", err
	}

	cmd := "t W50 [" + bytesText + "]"

	switch strings.ToUpper(strings.TrimSpace(polarization)) {
	case "V", "R":
		cmd += " W30 T"
	}

	return cmd, nil
}

func looksLikeFullDiseqcCommand(input string) bool {
	input = strings.TrimSpace(input)

	return strings.Contains(input, "[") && strings.Contains(input, "]")
}

func normalizeFullDiseqcCommand(input string) (string, error) {
	input = strings.TrimSpace(input)

	start := strings.Index(input, "[")
	stop := strings.Index(input, "]")

	if start < 0 || stop <= start {
		return "", fmt.Errorf("invalid full DiSEqC command")
	}

	rawBytes := strings.TrimSpace(input[start+1 : stop])

	bytesText, err := normalizeDiseqcBytes(rawBytes)
	if err != nil {
		return "", err
	}

	before := strings.TrimSpace(input[:start])
	after := strings.TrimSpace(input[stop+1:])

	result := before + " [" + bytesText + "]"
	if after != "" {
		result += " " + after
	}

	return result, nil
}

func normalizeDiseqcBytes(input string) (string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		input = defaultDiseqcBytes
	}

	input = strings.Trim(input, "[]")
	input = strings.TrimSpace(input)
	input = strings.ToUpper(input)

	if strings.Contains(input, " ") {
		parts := strings.Fields(input)

		for _, part := range parts {
			if len(part) != 2 || !isHexByte(part) {
				return "", fmt.Errorf("invalid DiSEqC byte: %s", part)
			}
		}

		return strings.Join(parts, " "), nil
	}

	if len(input)%2 != 0 {
		return "", fmt.Errorf("DiSEqC command must contain even number of hex digits")
	}

	if !isHexString(input) {
		return "", fmt.Errorf("DiSEqC command contains non-hex characters")
	}

	parts := make([]string, 0, len(input)/2)
	for i := 0; i < len(input); i += 2 {
		parts = append(parts, input[i:i+2])
	}

	return strings.Join(parts, " "), nil
}

func isHexByte(value string) bool {
	return len(value) == 2 && isHexString(value)
}

func isHexString(value string) bool {
	for _, r := range value {
		switch {
		case r >= '0' && r <= '9':
		case r >= 'A' && r <= 'F':
		case r >= 'a' && r <= 'f':
		default:
			return false
		}
	}

	return true
}
