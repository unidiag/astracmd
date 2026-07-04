package main

import (
	"main/internal/astra"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func dashboardGetSelectedAdapterID(
	adaptersTable *tview.Table,
	adapters []astra.AstraAdapter,
) string {
	row, _ := adaptersTable.GetSelection()

	switch row {
	case 0:
		return dashboardAdapterAll

	case 1:
		return dashboardAdapterOutside
	}

	adapterIndex := (row - 2) / 2
	if adapterIndex < 0 || adapterIndex >= len(adapters) {
		return dashboardAdapterAll
	}

	return adapters[adapterIndex].ID
}

func dashboardGetSelectedAdapter(
	adaptersTable *tview.Table,
	adapters []astra.AstraAdapter,
) (astra.AstraAdapter, bool) {
	row, _ := adaptersTable.GetSelection()
	if row < 2 {
		return astra.AstraAdapter{}, false
	}

	adapterIndex := (row - 2) / 2
	if adapterIndex < 0 || adapterIndex >= len(adapters) {
		return astra.AstraAdapter{}, false
	}

	return adapters[adapterIndex], true
}

func dashboardRenderAdapters(
	adaptersTable *tview.Table,
	adapters []astra.AstraAdapter,
	adapterStates map[string]astra.AstraAdapterState,
	dimmed bool,
) {
	FillAdaptersTable(adaptersTable, adapters, adapterStates, dimmed)

	row, _ := adaptersTable.GetSelection()
	maxRow := len(adapters)*2 + 1

	if row < 0 || row > maxRow {
		adaptersTable.Select(0, 0)
	}
}

func dashboardBuildVisibleStreams(
	adapterID string,
	config astra.AstraConfig,
	streamMap map[string][]astra.AstraStream,
) []astra.AstraStream {
	switch adapterID {
	case dashboardAdapterAll:
		return append([]astra.AstraStream(nil), config.Streams...)

	case dashboardAdapterOutside:
		return append([]astra.AstraStream(nil), streamMap[""]...)

	default:
		return append([]astra.AstraStream(nil), streamMap[adapterID]...)
	}
}

func dashboardRenderStreams(
	table *tview.Table,
	streams []astra.AstraStream,
	states map[string]astra.AstraStreamState,
	selectedStreamIDs map[string]bool,
	dimmed bool,
) {
	FillStreamsTable(table, streams, states, dimmed)

	for row, stream := range streams {
		streamID := strings.TrimSpace(stream.ID)
		if streamID == "" {
			continue
		}

		if !selectedStreamIDs[streamID] {
			continue
		}

		color := tcell.ColorGreen
		if dimmed {
			color = dashboardDisabledColor
		}

		nameCell := table.GetCell(row, 0)
		if nameCell != nil {
			nameCell.SetTextColor(color)
		}

		for col := 1; col < 10; col++ {
			cell := table.GetCell(row, col)
			if cell == nil {
				continue
			}

			cell.SetTextColor(color)
		}
	}

	row, _ := table.GetSelection()
	if row < 0 || row >= len(streams) {
		if len(streams) > 0 {
			table.Select(0, 0)
		}
	}
}

func dashboardGetSelectedStream(
	streamsTable *tview.Table,
	visibleStreams []astra.AstraStream,
) (astra.AstraStream, bool) {
	if len(visibleStreams) == 0 {
		return astra.AstraStream{}, false
	}

	row, _ := streamsTable.GetSelection()
	if row < 0 || row >= len(visibleStreams) {
		return astra.AstraStream{}, false
	}

	return visibleStreams[row], true
}

func dashboardGetFilteredLogItems(
	activePane int,
	currentLogItems []astra.AstraLogItem,
	selectedAdapterID string,
	selectedAdapter astra.AstraAdapter,
	hasSelectedAdapter bool,
	selectedStream astra.AstraStream,
	hasSelectedStream bool,
	streamMap map[string][]astra.AstraStream,
) []astra.AstraLogItem {
	if activePane == dashboardPaneStreams {
		if hasSelectedStream {
			return FilterLogItemsByStream(currentLogItems, selectedStream)
		}

		return currentLogItems
	}

	switch selectedAdapterID {
	case dashboardAdapterAll:
		return currentLogItems

	case dashboardAdapterOutside:
		return FilterLogItemsByStreams(currentLogItems, streamMap[""])

	default:
		if hasSelectedAdapter {
			return FilterLogItemsByAdapter(
				currentLogItems,
				streamMap[selectedAdapterID],
				selectedAdapter.Adapter,
			)
		}

		return FilterLogItemsByStreams(currentLogItems, streamMap[selectedAdapterID])
	}
}

func dashboardRenderLog(
	logTable *tview.Table,
	items []astra.AstraLogItem,
	dimmed bool,
) {
	FillLogTable(logTable, items, dimmed, dashboardGetLogMaxRows(logTable))
}
