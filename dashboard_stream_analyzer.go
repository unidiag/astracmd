package main

import (
	"context"
	"encoding/json"
	"fmt"
	"main/internal/astra"
	"sort"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (ui *UI) ShowStreamAnalyzerDialog(conn astra.Connection, stream astra.Stream) {
	streamID := strings.TrimSpace(stream.ID)
	if streamID == "" {
		ui.ShowError("Stream ID is empty", nil)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())

	status := tview.NewTextView()
	status.SetDynamicColors(true)
	status.SetTextAlign(tview.AlignCenter)
	status.SetText("[yellow]Starting stream analyzer...[-]")

	table := tview.NewTable()
	table.SetBorder(false)
	table.SetSelectable(false, false)
	table.SetEvaluateAllRows(true)

	setScanTableLoading(table)

	footer := tview.NewTextView()
	footer.SetDynamicColors(true)
	footer.SetTextAlign(tview.AlignCenter)
	footer.SetText("[gray]Esc — close[-]")

	body := tview.NewFlex()
	body.SetDirection(tview.FlexRow)
	body.SetBorder(true)
	body.SetTitle(fmt.Sprintf(" Stream analyzer: %s ", tview.Escape(stream.DisplayName())))
	body.SetTitleAlign(tview.AlignCenter)
	body.AddItem(status, 1, 0, false)
	body.AddItem(table, 0, 1, false)
	body.AddItem(footer, 1, 0, false)

	closeDialog := func() {
		cancel()
		ui.pages.RemovePage(pageDialog)
	}

	body.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
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

	ui.pages.AddPage(pageDialog, centerPrimitive(body, 76, 26), true, true)
	ui.app.SetFocus(body)

	go ui.runStreamAnalyzer(ctx, conn, stream, status, table)
}

func (ui *UI) runStreamAnalyzer(
	ctx context.Context,
	conn astra.Connection,
	stream astra.Stream,
	status *tview.TextView,
	table *tview.Table,
) {
	ws, wsID, err := astra.AstraConnectWebSocketWithID(ctx, conn)
	if err != nil {
		ui.app.QueueUpdateDraw(func() {
			status.SetText("[red]WebSocket error[-]")
			setScanTableError(table, err)
		})
		return
	}

	defer ws.Close()

	initResult := astra.AstraStreamScanInit(ctx, conn, stream.ID, wsID)
	if !initResult.OK {
		ui.app.QueueUpdateDraw(func() {
			status.SetText("[red]scan-init failed[-]")
			setScanTableError(table, initResult.Err)
		})
		return
	}

	ui.app.QueueUpdateDraw(func() {
		status.SetText(fmt.Sprintf(
			"[green]scan-init OK[-] [gray]%s · ws:%d[-]",
			tview.Escape(initResult.ScanID),
			wsID,
		))
	})

	messages := make(chan astra.AstraWSMessage, 32)
	go ws.ReadLoop(ctx, messages)

	var emptyFrames int

	var state streamAnalyzerState

	for msg := range messages {
		if msg.Err != nil {
			ui.app.QueueUpdateDraw(func() {
				status.SetText("[red]WebSocket stopped[-]")
				setScanTableError(table, msg.Err)
			})
			return
		}

		var envelope astra.AstraWSEnvelope
		if err := json.Unmarshal(msg.Raw, &envelope); err != nil {
			continue
		}

		if envelope.Scope != "scan" {
			continue
		}

		var totalEvent astra.AstraStreamScanEvent
		if err := json.Unmarshal(msg.Raw, &totalEvent); err == nil {
			total := totalEvent.Data.Total

			if !isEmptyStreamScanTotal(total) {
				state.Total = total
				state.OnAir = totalEvent.Data.OnAir
				state.HasTotal = true
				emptyFrames = 0

				ui.app.QueueUpdateDraw(func() {
					status.SetText(formatScanStatus(state.OnAir, state.Total.Scrambled, initResult.ScanID, wsID))
					setScanAnalyzerTable(table, state)
				})

				continue
			}

			if state.HasTotal {
				emptyFrames++
				if emptyFrames < streamScanEmptyFramesBeforeReset {
					continue
				}

				state.Total = total
				state.OnAir = totalEvent.Data.OnAir

				ui.app.QueueUpdateDraw(func() {
					status.SetText(formatScanStatus(state.OnAir, state.Total.Scrambled, initResult.ScanID, wsID))
					setScanAnalyzerTable(table, state)
				})
			}
		}

		var psiEvent astra.AstraStreamScanPSIEvent
		if err := json.Unmarshal(msg.Raw, &psiEvent); err != nil {
			continue
		}

		psi := strings.ToLower(strings.TrimSpace(psiEvent.Data.PSI))
		if psi == "" {
			continue
		}

		state.ApplyPSI(psiEvent.Data)

		ui.app.QueueUpdateDraw(func() {
			status.SetText(formatScanStatus(state.OnAir, state.Total.Scrambled, initResult.ScanID, wsID))
			setScanAnalyzerTable(table, state)
		})
	}
}

const streamScanEmptyFramesBeforeReset = 100

type streamAnalyzerState struct {
	HasTotal bool
	Total    astra.AstraStreamScanTotal
	OnAir    bool

	PAT *astra.AstraStreamScanPSIData
	PMT *astra.AstraStreamScanPSIData
	SDT *astra.AstraStreamScanPSIData

	EPG []streamAnalyzerEPGItem
}

type streamAnalyzerEPGItem struct {
	StartUT int64
	StopUT  int64
	Title   string
	Text    string
	Lang    string
}

func (s *streamAnalyzerState) ApplyPSI(data astra.AstraStreamScanPSIData) {
	switch strings.ToLower(strings.TrimSpace(data.PSI)) {
	case "pat":
		s.PAT = &data

	case "pmt":
		s.PMT = &data

	case "sdt":
		s.SDT = &data

	case "eit":
		s.applyEIT(data)
	}
}

func (s *streamAnalyzerState) applyEIT(data astra.AstraStreamScanPSIData) {
	if len(data.Events) == 0 {
		return
	}

	existing := make(map[int64]streamAnalyzerEPGItem, len(s.EPG)+len(data.Events))
	for _, item := range s.EPG {
		key := item.StartUT
		if key > 0 {
			existing[key] = item
		}
	}

	for _, event := range data.Events {
		item := streamAnalyzerEPGItem{
			StartUT: event.StartUT,
			StopUT:  event.StopUT,
		}

		for _, desc := range event.Descriptors {
			switch desc.TypeName {
			case "short_event_descriptor":
				item.Title = strings.TrimSpace(desc.EventName)
				item.Text = strings.TrimSpace(desc.TextChar)
				item.Lang = strings.TrimSpace(desc.Lang)

			case "extended_event_descriptor":
				if item.Text == "" {
					item.Text = strings.TrimSpace(desc.Text)
				}
				if item.Lang == "" {
					item.Lang = strings.TrimSpace(desc.Lang)
				}
			}
		}

		if item.Title == "" {
			continue
		}

		existing[item.StartUT] = item
	}

	s.EPG = s.EPG[:0]
	for _, item := range existing {
		s.EPG = append(s.EPG, item)
	}

	sort.Slice(s.EPG, func(i, j int) bool {
		return s.EPG[i].StartUT < s.EPG[j].StartUT
	})

	if len(s.EPG) > 8 {
		s.EPG = s.EPG[:8]
	}
}

func isEmptyStreamScanTotal(total astra.AstraStreamScanTotal) bool {
	return total.BitrateLimit == 0 &&
		total.CCErrors == 0 &&
		total.PESErrors == 0 &&
		total.Packets == 0 &&
		!total.Scrambled &&
		total.SCErrors == 0 &&
		total.Bitrate == 0 &&
		total.PCRErrors == 0
}

func formatScanStatus(onAir bool, scrambled bool, scanID string, wsID int64) string {
	onAirText := "[red]OFF AIR[-]"
	if onAir {
		onAirText = "[green]ON AIR[-]"
	}

	scrambledText := ""
	if scrambled {
		scrambledText = " [red](Scrambled)[-]"
	}

	return onAirText + scrambledText
}

func setScanTableLoading(table *tview.Table) {
	table.Clear()
	table.SetCell(0, 0, tview.NewTableCell("Waiting for scan data...").
		SetTextColor(tcell.ColorYellow).
		SetExpansion(1))
}

func setScanTableError(table *tview.Table, err error) {
	table.Clear()

	text := "unknown error"
	if err != nil {
		text = err.Error()
	}

	table.SetCell(0, 0, tview.NewTableCell(text).
		SetTextColor(tcell.ColorRed).
		SetExpansion(1))
}

func setScanAnalyzerTable(table *tview.Table, state streamAnalyzerState) {
	table.Clear()

	row := 0

	row = addScanTotalRows(table, row, state.Total)
	row = addScanSpacer(table, row)

	if state.PAT != nil {
		row = addPATRows(table, row, *state.PAT)
		row = addScanSpacer(table, row)
	}

	if state.PMT != nil {
		row = addPMTRows(table, row, *state.PMT)
		row = addScanSpacer(table, row)
	}

	if state.SDT != nil {
		row = addSDTRows(table, row, *state.SDT)
		row = addScanSpacer(table, row)
	}

	if len(state.EPG) > 0 {
		addEPGRows(table, row, state.EPG)
	}
}

func addScanTotalRows(table *tview.Table, row int, total astra.AstraStreamScanTotal) int {
	rows := []struct {
		Name  string
		Value string
		Color tcell.Color
	}{
		{
			Name:  "Bitrate / Packets",
			Value: formatBitratePackets(total),
			Color: tcell.ColorWhite,
		},
		{
			Name:  "CC / PES / PCR",
			Value: fmt.Sprintf("%d / %d / %d", total.CCErrors, total.PESErrors, total.PCRErrors),
			Color: scanErrorColor(total.CCErrors + total.PESErrors),
		},
	}

	for _, item := range rows {
		table.SetCell(row, 0, tview.NewTableCell(" "+item.Name).
			SetTextColor(tcell.ColorGray).
			SetExpansion(1))

		table.SetCell(row, 1, tview.NewTableCell(item.Value).
			SetTextColor(item.Color).
			SetAlign(tview.AlignRight).
			SetExpansion(1))

		row++
	}

	return row
}

func addPATRows(table *tview.Table, row int, data astra.AstraStreamScanPSIData) int {
	table.SetCell(row, 0, tview.NewTableCell(fmt.Sprintf(" PAT TSID:%d", data.TSID)).
		SetTextColor(tcell.ColorLightCyan).
		SetAttributes(tcell.AttrBold).
		SetExpansion(1))
	row++

	for _, program := range data.Programs {
		table.SetCell(row, 0, tview.NewTableCell(fmt.Sprintf("   PNR:%d PID:%d", program.PNR, program.PID)).
			SetTextColor(tcell.ColorWhite).
			SetExpansion(1))
		row++
	}

	return row
}

func addPMTRows(table *tview.Table, row int, data astra.AstraStreamScanPSIData) int {
	items := getPMTItems(data)

	title := fmt.Sprintf(" PMT PNR:%d", data.PNR)

	if data.PCRPID > 0 {
		title += fmt.Sprintf("  PCR PID:%d", data.PCRPID)
	}

	table.SetCell(row, 0, tview.NewTableCell(title).
		SetTextColor(tcell.ColorLightCyan).
		SetAttributes(tcell.AttrBold).
		SetExpansion(1))
	row++

	for _, item := range items {
		line := formatPMTItem(item)

		table.SetCell(row, 0, tview.NewTableCell("   "+line).
			SetTextColor(tcell.ColorWhite).
			SetExpansion(1))
		row++
	}

	return row
}

func addSDTRows(table *tview.Table, row int, data astra.AstraStreamScanPSIData) int {
	table.SetCell(row, 0, tview.NewTableCell(fmt.Sprintf(" SDT TSID:%d", data.TSID)).
		SetTextColor(tcell.ColorLightCyan).
		SetAttributes(tcell.AttrBold).
		SetExpansion(1))
	row++

	for _, service := range data.Services {
		for _, desc := range service.Descriptors {
			if desc.TypeName != "service" {
				continue
			}

			if strings.TrimSpace(desc.ServiceProvider) != "" {
				table.SetCell(row, 0, tview.NewTableCell("   Provider: "+desc.ServiceProvider).
					SetTextColor(tcell.ColorWhite).
					SetExpansion(1))
				row++
			}

			if strings.TrimSpace(desc.ServiceName) != "" {
				table.SetCell(row, 0, tview.NewTableCell("   Service: "+desc.ServiceName).
					SetTextColor(tcell.ColorWhite).
					SetExpansion(1))
				row++
			}
		}
	}

	return row
}

func addEPGRows(table *tview.Table, row int, items []streamAnalyzerEPGItem) int {
	table.SetCell(row, 0, tview.NewTableCell(" EPG").
		SetTextColor(tcell.ColorLightCyan).
		SetAttributes(tcell.AttrBold).
		SetExpansion(1))
	row++

	for _, item := range items {
		line := formatEPGItem(item)

		table.SetCell(row, 0, tview.NewTableCell("   "+line).
			SetTextColor(tcell.ColorWhite).
			SetExpansion(1))
		row++
	}

	return row
}

func addScanSpacer(table *tview.Table, row int) int {
	table.SetCell(row, 0, tview.NewTableCell(""))
	return row + 1
}

func scanErrorColor(value int) tcell.Color {
	if value > 0 {
		return tcell.ColorRed
	}

	return tcell.ColorWhite
}

func scanValueColor(ok bool) tcell.Color {
	if ok {
		return tcell.ColorGreen
	}

	return tcell.ColorRed
}

func formatPMTItem(item astra.AstraStreamScanPMTItem) string {
	kind := formatPMTType(item.TypeName, item.TypeID)
	lang := getPMTItemLang(item)

	line := fmt.Sprintf("%s PID:%d", kind, item.PID)

	if lang != "" {
		line += " " + lang
	}

	return line
}

func formatPMTType(typeName string, typeID int) string {
	name := strings.ToLower(typeName)

	switch {
	case strings.Contains(name, "video"):
		if typeID == 0x1b {
			return "VIDEO H.264"
		}
		return "VIDEO"

	case strings.Contains(name, "audio"):
		return "AUDIO"

	case strings.Contains(name, "teletext"):
		return "TTX"

	case strings.Contains(name, "private"):
		return "DATA"

	default:
		if typeName != "" {
			return strings.ToUpper(typeName)
		}
		return fmt.Sprintf("TYPE:0x%02X", typeID)
	}
}

func formatEPGItem(item streamAnalyzerEPGItem) string {
	start := formatEPGTime(item.StartUT)
	stop := formatEPGTime(item.StopUT)

	title := truncateScanText(strings.TrimSpace(item.Title), 30)
	return fmt.Sprintf("%s-%s %s", start, stop, title)
}

func formatEPGTime(ut int64) string {
	if ut <= 0 {
		return "--:--"
	}

	return time.Unix(ut, 0).Local().Format("15:04")
}

func truncateScanText(s string, limit int) string {
	s = strings.TrimSpace(s)
	if limit <= 0 {
		return ""
	}

	runes := []rune(s)
	if len(runes) <= limit {
		return s
	}

	return string(runes[:limit]) + "..."
}

func getPMTItemLang(item astra.AstraStreamScanPMTItem) string {
	for _, desc := range item.Descriptors {
		lang := strings.TrimSpace(desc.Lang)
		if lang != "" {
			return lang
		}

		lang = strings.TrimSpace(desc.Language)
		if lang != "" {
			return lang
		}
	}

	return ""
}

func getPMTItems(data astra.AstraStreamScanPSIData) []astra.AstraStreamScanPMTItem {
	if len(data.PMT) > 0 {
		return data.PMT
	}

	if len(data.Streams) > 0 {
		return data.Streams
	}

	if len(data.Items) > 0 {
		return data.Items
	}

	return nil
}

func formatBitratePackets(total astra.AstraStreamScanTotal) string {
	bitrateColor := "green"
	if total.Bitrate <= 0 {
		bitrateColor = "red"
	}

	return fmt.Sprintf(
		"[%s]%d kbps[-] / [white]%d[-]",
		bitrateColor,
		total.Bitrate,
		total.Packets,
	)
}
