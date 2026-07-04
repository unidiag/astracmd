package main

import (
	"fmt"
	"main/internal/astra"
	"sort"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func FillLogTable(table *tview.Table, items []astra.AstraLogItem, dimmed bool, maxRows int) {
	table.Clear()

	if maxRows <= 0 {
		return
	}

	color := tcell.ColorWhite
	if dimmed {
		color = tcell.ColorDarkGray
	}

	start := 0
	if len(items) > maxRows {
		start = len(items) - maxRows
	}

	row := 0

	for _, item := range items[start:] {
		itemColor := color

		switch item.Type {
		case 1:
			itemColor = tcell.ColorYellow
		case 2:
			itemColor = tcell.ColorRed
		case 3:
			itemColor = dashboardDisabledColor
		}

		if dimmed {
			itemColor = tcell.ColorDarkGray
		}

		text := fmt.Sprintf(
			"%s %s",
			time.Unix(item.Time, 0).Format("15:04:05"),
			item.Text,
		)

		table.SetCell(row, 0,
			tview.NewTableCell(text).
				SetTextColor(itemColor).
				SetExpansion(1),
		)

		row++
	}
}

func FillStreamsTable(
	table *tview.Table,
	streams []astra.Stream,
	states map[string]astra.StreamState,
	dimmed bool,
) {
	table.Clear()

	sort.SliceStable(streams, func(i, j int) bool {
		return strings.ToLower(streams[i].DisplayName()) < strings.ToLower(streams[j].DisplayName())
	})

	for row, stream := range streams {
		color := tcell.ColorWhite

		if !stream.Enable {
			color = dashboardDisabledColor
		}

		if dimmed {
			color = dashboardDisabledColor
		}

		nameText := fmt.Sprintf("%d. %s", row+1, stream.DisplayName())

		table.SetCell(row, 0,
			tview.NewTableCell(nameText).
				SetTextColor(color).
				SetExpansion(1),
		)

		state, ok := states[stream.ID]

		errorText := ""
		errorColor := tcell.ColorYellow

		bitrateText := "-"
		bitrateColor := dashboardDisabledColor

		if ok {
			if state.CCError > 0 && state.PESError > 0 {
				errorText = fmt.Sprintf("CC:%d PES:%d", state.CCError, state.PESError)
			} else if state.CCError > 0 {
				errorText = fmt.Sprintf("CC:%d", state.CCError)
			} else if state.PESError > 0 {
				errorText = fmt.Sprintf("PES:%d", state.PESError)
			}

			bitrateText = fmt.Sprintf("%d kbps", state.Bitrate)

			if state.Onair && state.Bitrate > 0 {
				bitrateColor = tcell.ColorGreen
			} else {
				bitrateColor = tcell.ColorRed
			}

			if state.Scrambled {
				bitrateColor = tcell.ColorOrange
			}
		}

		if dimmed {
			errorColor = tcell.ColorDarkGray
			bitrateColor = tcell.ColorDarkGray
		}

		table.SetCell(row, 1,
			tview.NewTableCell(errorText).
				SetTextColor(errorColor).
				SetAlign(tview.AlignCenter).
				SetExpansion(0),
		)

		table.SetCell(row, 2,
			tview.NewTableCell(bitrateText).
				SetTextColor(bitrateColor).
				SetAlign(tview.AlignRight).
				SetExpansion(0),
		)
	}
}

func FilterLogItemsByAdapter(
	items []astra.AstraLogItem,
	streams []astra.Stream,
	adapterNumber int,
) []astra.AstraLogItem {
	result := make([]astra.AstraLogItem, 0)

	for _, item := range items {
		if isAdapterLogItem(item, adapterNumber) {
			result = append(result, item)
			continue
		}

		for _, stream := range streams {
			if logItemMatchesStream(item.Text, stream) {
				result = append(result, item)
				break
			}
		}
	}

	return result
}

func NewDashboardTable(title string) *tview.Table {
	table := tview.NewTable()
	table.SetBorder(true)
	table.SetTitle(" " + title + " ")
	table.SetTitleAlign(tview.AlignCenter)
	table.SetSelectable(true, false)
	table.SetEvaluateAllRows(true)

	return table
}

func FillAdaptersTable(
	table *tview.Table,
	adapters []astra.Adapter,
	states map[string]astra.AdapterState,
	dimmed bool,
) {
	table.Clear()

	specialColor := tcell.ColorWhite
	if dimmed {
		specialColor = tcell.ColorDarkGray
	}

	table.SetCell(0, 0,
		tview.NewTableCell("ALL").
			SetTextColor(specialColor).
			SetAlign(tview.AlignCenter).
			SetExpansion(1),
	)

	table.SetCell(1, 0,
		tview.NewTableCell("OUTSIDE").
			SetTextColor(specialColor).
			SetAlign(tview.AlignCenter).
			SetExpansion(1),
	)

	sort.SliceStable(adapters, func(i, j int) bool {
		if adapters[i].Adapter == adapters[j].Adapter {
			return adapters[i].Name < adapters[j].Name
		}

		return adapters[i].Adapter < adapters[j].Adapter
	})

	for i, adapter := range adapters {
		row := 2 + i*2

		color := tcell.ColorWhite
		infoColor := dashboardDisabledColor

		if !adapter.Enable {
			color = dashboardDisabledColor
			infoColor = tcell.ColorDarkGray
		}

		if dimmed {
			color = tcell.ColorDarkGray
			infoColor = tcell.ColorDarkGray
		}

		text := fmt.Sprintf("%d. %s", i+1, adapter.DisplayName())

		table.SetCell(row, 0,
			tview.NewTableCell(text).
				SetTextColor(color).
				SetExpansion(1),
		)

		state, ok := states[adapter.ID]

		infoText := "   BER:- UNC:- S:- Q:- - Mbps"
		if ok {
			infoText = fmt.Sprintf(
				"   BER:%s UNC:%s S:%d%% Q:%d%% %.1f Mbps",
				astra.FormatAdapterCounter(state.BER),
				astra.FormatAdapterCounter(state.UNC),
				astra.NormalizeAdapterSignal(state.Signal),
				astra.NormalizeAdapterSignal(state.SNR),
				float64(state.Bitrate)/1000.0,
			)

			if state.BER > 0 || state.UNC > 0 {
				infoColor = tcell.ColorYellow
			}

			if state.BER > 100 || state.UNC > 100 {
				infoColor = tcell.ColorRed
			}

			if state.Bitrate <= 0 {
				infoColor = dashboardDisabledColor
			}
		}

		if dimmed {
			infoColor = dashboardDisabledColor
		}

		table.SetCell(row+1, 0,
			tview.NewTableCell(infoText).
				SetTextColor(infoColor).
				SetSelectable(false).
				SetExpansion(1),
		)
	}
}
