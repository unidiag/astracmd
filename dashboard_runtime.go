package main

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/rivo/tview"
)

type DashboardRuntime struct {
	ui     *UI
	conn   AstraConnection
	ctx    context.Context
	cancel context.CancelFunc

	forceOfflineUntil atomic.Int64
	loadDone          atomic.Bool

	activePane  int
	dimmed      bool
	isRendering bool

	debugLogEnabled bool

	currentConfig    AstraConfig
	currentStreamMap map[string][]AstraStream

	currentLogItems []AstraLogItem
	lastLogID       int64
	visibleStreams  []AstraStream

	streamStates  map[string]AstraStreamState
	adapterStates map[string]AstraAdapterState

	selectedStreamIDs map[string]bool

	nameView    *tview.TextView
	statusView  *tview.TextView
	versionView *tview.TextView

	header      *tview.Flex
	mainColumns *tview.Flex
	root        *tview.Flex

	adaptersTable *tview.Table
	streamsTable  *tview.Table
	logTable      *tview.Table
}

func NewDashboardRuntime(
	ui *UI,
	conn AstraConnection,
	ctx context.Context,
	cancel context.CancelFunc,
) *DashboardRuntime {
	rt := &DashboardRuntime{
		ui:     ui,
		conn:   conn,
		ctx:    ctx,
		cancel: cancel,

		activePane:      dashboardPaneAdapters,
		debugLogEnabled: conn.Debug,

		currentStreamMap: make(map[string][]AstraStream),
		streamStates:     make(map[string]AstraStreamState),
		adapterStates:    make(map[string]AstraAdapterState),

		selectedStreamIDs: make(map[string]bool),
	}

	rt.nameView = tview.NewTextView()
	rt.nameView.SetDynamicColors(true)
	rt.nameView.SetTextAlign(tview.AlignLeft)
	rt.nameView.SetText(fmt.Sprintf("[white]%s[-]", tview.Escape(conn.Name)))

	rt.statusView = tview.NewTextView()
	rt.statusView.SetDynamicColors(true)
	rt.statusView.SetTextAlign(tview.AlignCenter)
	rt.statusView.SetText("[yellow]CHECKING[-]")

	rt.versionView = tview.NewTextView()
	rt.versionView.SetDynamicColors(true)
	rt.versionView.SetTextAlign(tview.AlignRight)
	rt.versionView.SetText("[gray]Astra: checking...[-]")

	rt.header = tview.NewFlex()
	rt.header.SetDirection(tview.FlexColumn)
	rt.header.AddItem(rt.nameView, 0, 1, false)
	rt.header.AddItem(rt.statusView, 0, 1, false)
	rt.header.AddItem(rt.versionView, 0, 1, false)

	rt.adaptersTable = NewDashboardTable("Adapters")
	rt.streamsTable = NewDashboardTable("Streams")
	rt.logTable = NewDashboardTable("Log")
	rt.logTable.SetSelectable(false, false)

	rt.mainColumns = tview.NewFlex()
	rt.mainColumns.SetDirection(tview.FlexColumn)
	rt.mainColumns.AddItem(rt.adaptersTable, 0, 1, true)
	rt.mainColumns.AddItem(rt.streamsTable, 0, 2, false)
	rt.mainColumns.AddItem(rt.logTable, 0, 2, false)

	return rt
}

func (rt *DashboardRuntime) QueueUpdateDraw(fn func()) {
	if fn == nil {
		return
	}

	rt.ui.app.QueueUpdateDraw(fn)
}

func (rt *DashboardRuntime) BuildRoot(functionKeys tview.Primitive) *tview.Flex {
	rt.root = tview.NewFlex()
	rt.root.SetDirection(tview.FlexRow)
	rt.root.AddItem(rt.header, 1, 0, false)
	rt.root.AddItem(rt.mainColumns, 0, 1, true)
	rt.root.AddItem(functionKeys, 1, 0, false)

	return rt.root
}

func (rt *DashboardRuntime) SetOffline(message string) {
	dashboardSetOffline(rt.statusView, rt.versionView, message)
}

func (rt *DashboardRuntime) SetOnline(version string) {
	dashboardSetOnline(rt.statusView, rt.versionView, version)
}

func (rt *DashboardRuntime) SetRestarting() {
	dashboardSetRestarting(rt.statusView, rt.versionView)
}

func (rt *DashboardRuntime) UpdateLogTitle() {
	dashboardUpdateLogTitle(
		rt.logTable,
		rt.dimmed,
		rt.debugLogEnabled,
	)
}

func (rt *DashboardRuntime) UpdateBorders() {
	dashboardUpdateBorders(
		rt.adaptersTable,
		rt.streamsTable,
		rt.logTable,
		rt.activePane,
		rt.dimmed,
	)
}

func (rt *DashboardRuntime) SelectedAdapterID() string {
	return dashboardGetSelectedAdapterID(
		rt.adaptersTable,
		rt.currentConfig.Adapters,
	)
}

func (rt *DashboardRuntime) SelectedAdapter() (AstraAdapter, bool) {
	return dashboardGetSelectedAdapter(
		rt.adaptersTable,
		rt.currentConfig.Adapters,
	)
}

func (rt *DashboardRuntime) SelectedStream() (AstraStream, bool) {
	return dashboardGetSelectedStream(
		rt.streamsTable,
		rt.visibleStreams,
	)
}

func (rt *DashboardRuntime) RenderAdapters() {
	dashboardRenderAdapters(
		rt.adaptersTable,
		rt.currentConfig.Adapters,
		rt.adapterStates,
		rt.dimmed,
	)
}

func (rt *DashboardRuntime) RenderStreams() {
	adapterID := rt.SelectedAdapterID()

	rt.visibleStreams = dashboardBuildVisibleStreams(
		adapterID,
		rt.currentConfig,
		rt.currentStreamMap,
	)

	dashboardRenderStreams(
		rt.streamsTable,
		rt.visibleStreams,
		rt.streamStates,
		rt.selectedStreamIDs,
		rt.dimmed,
	)
}

func (rt *DashboardRuntime) RenderLog() {
	selectedAdapterID := rt.SelectedAdapterID()
	selectedAdapter, hasSelectedAdapter := rt.SelectedAdapter()
	selectedStream, hasSelectedStream := rt.SelectedStream()

	items := dashboardGetFilteredLogItems(
		rt.activePane,
		rt.currentLogItems,
		selectedAdapterID,
		selectedAdapter,
		hasSelectedAdapter,
		selectedStream,
		hasSelectedStream,
		rt.currentStreamMap,
	)

	dashboardRenderLog(
		rt.logTable,
		items,
		rt.dimmed,
	)
}

func (rt *DashboardRuntime) RenderTables() {
	if rt.isRendering {
		return
	}

	rt.isRendering = true
	defer func() {
		rt.isRendering = false
	}()

	rt.RenderAdapters()
	rt.RenderStreams()
	rt.RenderLog()
	rt.UpdateLogTitle()
	rt.UpdateBorders()
}

func (rt *DashboardRuntime) SetDimmed(dimmed bool) {
	rt.dimmed = dimmed

	if dimmed {
		rt.adaptersTable.SetTitle(" Adapters: Reloading... ")
		rt.streamsTable.SetTitle(" Streams: Reloading... ")
		rt.logTable.SetTitle(" Log: Reloading... ")
	} else {
		rt.adaptersTable.SetTitle(" Adapters ")
		rt.streamsTable.SetTitle(" Streams ")
		rt.UpdateLogTitle()
	}

	rt.RenderTables()
}

func (rt *DashboardRuntime) SetActivePane(pane int) {
	rt.activePane = pane

	switch rt.activePane {
	case dashboardPaneAdapters:
		rt.ui.app.SetFocus(rt.adaptersTable)

	case dashboardPaneStreams:
		rt.ui.app.SetFocus(rt.streamsTable)
	}

	rt.UpdateBorders()
	rt.RenderLog()
}

func (rt *DashboardRuntime) AppendLogItems(items []AstraLogItem) {
	if len(items) == 0 {
		return
	}

	for _, item := range items {
		if item.ID > rt.lastLogID {
			rt.currentLogItems = append(rt.currentLogItems, item)
			rt.lastLogID = item.ID
		}
	}

	rt.RenderLog()
}

func (rt *DashboardRuntime) MoveAdapterSelection(delta int) {
	dashboardMoveAdapterSelection(
		rt.adaptersTable,
		rt.currentConfig.Adapters,
		delta,
	)

	rt.RenderStreams()
	rt.RenderLog()
	rt.UpdateBorders()
}

func (rt *DashboardRuntime) ToggleSelectedStreamMark() {
	if rt.activePane != dashboardPaneStreams {
		return
	}

	stream, ok := rt.SelectedStream()
	if !ok {
		return
	}

	streamID := strings.TrimSpace(stream.ID)
	if streamID == "" {
		return
	}

	row, col := rt.streamsTable.GetSelection()

	if rt.selectedStreamIDs[streamID] {
		delete(rt.selectedStreamIDs, streamID)
	} else {
		rt.selectedStreamIDs[streamID] = true
	}

	rt.RenderStreams()

	nextRow := row + 1
	if nextRow >= len(rt.visibleStreams) {
		nextRow = len(rt.visibleStreams) - 1
	}

	if nextRow >= 0 {
		rt.streamsTable.Select(nextRow, col)
	}
}

func (rt *DashboardRuntime) MarkedStreams() []AstraStream {
	if len(rt.selectedStreamIDs) == 0 {
		return nil
	}

	items := make([]AstraStream, 0, len(rt.selectedStreamIDs))

	for _, stream := range rt.visibleStreams {
		streamID := strings.TrimSpace(stream.ID)
		if streamID == "" {
			continue
		}

		if rt.selectedStreamIDs[streamID] {
			items = append(items, stream)
		}
	}

	return items
}

func (rt *DashboardRuntime) ClearMarkedStreams() {
	if len(rt.selectedStreamIDs) == 0 {
		return
	}

	rt.selectedStreamIDs = make(map[string]bool)
	rt.RenderStreams()
}

func (rt *DashboardRuntime) CleanupMarkedStreams() {
	if len(rt.selectedStreamIDs) == 0 {
		return
	}

	exists := make(map[string]bool)

	for _, stream := range rt.currentConfig.Streams {
		streamID := strings.TrimSpace(stream.ID)
		if streamID != "" {
			exists[streamID] = true
		}
	}

	for streamID := range rt.selectedStreamIDs {
		if !exists[streamID] {
			delete(rt.selectedStreamIDs, streamID)
		}
	}
}

func (rt *DashboardRuntime) IsStreamMarked(stream AstraStream) bool {
	streamID := strings.TrimSpace(stream.ID)
	if streamID == "" {
		return false
	}

	return rt.selectedStreamIDs[streamID]
}

func (rt *DashboardRuntime) MarkAllVisibleStreams() {
	if rt.activePane != dashboardPaneStreams {
		return
	}

	if len(rt.visibleStreams) == 0 {
		return
	}

	allMarked := true
	hasStreams := false

	for _, stream := range rt.visibleStreams {
		streamID := strings.TrimSpace(stream.ID)
		if streamID == "" {
			continue
		}

		hasStreams = true

		if !rt.selectedStreamIDs[streamID] {
			allMarked = false
			break
		}
	}

	if !hasStreams {
		return
	}

	if allMarked {
		for _, stream := range rt.visibleStreams {
			streamID := strings.TrimSpace(stream.ID)
			if streamID == "" {
				continue
			}

			delete(rt.selectedStreamIDs, streamID)
		}
	} else {
		for _, stream := range rt.visibleStreams {
			streamID := strings.TrimSpace(stream.ID)
			if streamID == "" {
				continue
			}

			rt.selectedStreamIDs[streamID] = true
		}
	}

	rt.RenderStreams()
}

func (rt *DashboardRuntime) RestoreActivePane(pane int) {
	switch pane {
	case dashboardPaneAdapters, dashboardPaneStreams:
		rt.SetActivePane(pane)

	default:
		rt.SetActivePane(dashboardPaneAdapters)
	}
}
