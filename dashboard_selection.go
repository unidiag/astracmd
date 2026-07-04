package main

import (
	"main/internal/astra"

	"github.com/rivo/tview"
)

func dashboardAdapterLogicalIndex(row int) int {
	switch {
	case row <= 0:
		return 0

	case row == 1:
		return 1

	default:
		return 2 + ((row - 2) / 2)
	}
}

func dashboardAdapterRowFromLogicalIndex(logicalIndex int) int {
	switch {
	case logicalIndex <= 0:
		return 0

	case logicalIndex == 1:
		return 1

	default:
		adapterIndex := logicalIndex - 2
		return 2 + adapterIndex*2
	}
}

func dashboardMoveAdapterSelection(
	adaptersTable *tview.Table,
	adapters []astra.Adapter,
	delta int,
) {
	row, _ := adaptersTable.GetSelection()

	logicalIndex := dashboardAdapterLogicalIndex(row)
	maxLogicalIndex := len(adapters) + 1

	logicalIndex += delta

	if logicalIndex < 0 {
		logicalIndex = 0
	}

	if logicalIndex > maxLogicalIndex {
		logicalIndex = maxLogicalIndex
	}

	nextRow := dashboardAdapterRowFromLogicalIndex(logicalIndex)
	adaptersTable.Select(nextRow, 0)
}

func dashboardNormalizeAdapterSelectionRow(
	adaptersTable *tview.Table,
	adapters []astra.Adapter,
	row int,
) bool {
	maxRow := len(adapters)*2 + 1
	if row < 0 || row > maxRow {
		return false
	}

	if row >= 2 && (row-2)%2 == 1 {
		adaptersTable.Select(row-1, 0)
		return false
	}

	return true
}
