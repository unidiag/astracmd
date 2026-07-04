package main

import (
	"context"
	"fmt"
	"main/internal/astra"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (ui *UI) ShowDashboard(conn astra.AstraConnection) {
	ui.StopDashboardTimer()
	ui.pages.RemovePage(pageDialog)

	ctx, cancel := context.WithCancel(context.Background())
	ui.dashboardCancel = cancel

	rt := NewDashboardRuntime(ui, conn, ctx, cancel)

	var loadAstraConfig func()
	var loadAstraConfigAfter func(afterLoad func())

	loadAstraConfigAfter = func(afterLoad func()) {
		activePane := rt.activePane

		rt.SetDimmed(true)

		go func() {
			time.Sleep(180 * time.Millisecond)

			result := rt.client.Load(context.Background())

			rt.QueueUpdateDraw(func() {
				rt.SetDimmed(false)

				if !result.Online {
					if result.Err != nil {
						ui.ShowError(result.Err.Error(), nil)
					} else {
						ui.ShowError("Astra: offline", nil)
					}
					return
				}

				rt.currentConfig = result.Config
				rt.currentStreamMap = astra.BuildAdapterStreamMap(rt.currentConfig)
				rt.CleanupMarkedStreams()

				rt.RenderTables()

				if afterLoad != nil {
					afterLoad()
					return
				}

				rt.SetActivePane(activePane)
			})
		}()
	}

	loadAstraConfig = func() {
		loadAstraConfigAfter(nil)
	}

	loadAstraLog := func() {
		dashboardLoadLogAsync(
			context.Background(),
			conn,
			rt.QueueUpdateDraw,
			func(items []astra.AstraLogItem) {
				rt.UpdateLogTitle()
				rt.AppendLogItems(items)
			},
			func(err error) {
				ui.ShowError(err.Error(), nil)
			},
		)
	}

	setAstraDebugLog := func(enabled bool) {
		dashboardSetDebugLogAsync(
			context.Background(),
			conn,
			enabled,
			rt.QueueUpdateDraw,
			func(enabled bool) {
				rt.debugLogEnabled = enabled
				conn.Debug = enabled

				for i := range ui.cfg.Connections {
					if ui.cfg.Connections[i].ID == conn.ID {
						ui.cfg.Connections[i].Debug = enabled
						break
					}
				}

				if err := ui.cfg.Save(); err != nil {
					ui.ShowError(err.Error(), nil)
					return
				}

				rt.UpdateLogTitle()
			},
			func(err error) {
				ui.ShowError(err.Error(), nil)
			},
		)
	}

	toggleAstraDebugLog := func() {
		setAstraDebugLog(!rt.debugLogEnabled)
	}

	startAstraWebSocket := func() {
		dashboardStartAstraWebSocket(
			ctx,
			conn,
			rt.QueueUpdateDraw,
			DashboardWebSocketHandlers{
				OnLogItems: func(items []astra.AstraLogItem) {
					rt.AppendLogItems(items)
				},

				OnAdapterState: func(adapterID string, state astra.AstraAdapterState) {
					rt.adapterStates[adapterID] = state
					rt.RenderAdapters()
				},

				OnStreamState: func(streamID string, state astra.AstraStreamState) {
					rt.streamStates[streamID] = state
					rt.RenderStreams()
				},
			},
		)
	}

	confirmRestart := func() {
		ui.ConfirmRestartAstra(
			conn,
			func() {
				rt.forceOfflineUntil.Store(time.Now().Add(10 * time.Second).UnixNano())
				rt.SetRestarting()
			},
			func(err error) {
				ui.ShowError(err.Error(), nil)
			},
		)
	}

	showLicenseDialog := func() {
		ui.ShowLicenseDialog(
			conn,
			func() {
				rt.versionView.SetText("[green]License applied[-]")
			},
			func(err error) {
				ui.ShowError(err.Error(), nil)
			},
		)
	}

	showSoftCAMDialog := func() {
		ui.ShowSoftCAMDialog(
			conn,
			rt.currentConfig,
			func() {
				loadAstraConfig()
			},
			func() {
				rt.RenderTables()
				rt.UpdateBorders()
			},
		)
	}

	//  █████╗ ██████╗  █████╗ ██████╗ ████████╗███████╗██████╗     ██████╗ ██╗ █████╗ ██╗      ██████╗  ██████╗
	// ██╔══██╗██╔══██╗██╔══██╗██╔══██╗╚══██╔══╝██╔════╝██╔══██╗    ██╔══██╗██║██╔══██╗██║     ██╔═══██╗██╔════╝
	// ███████║██║  ██║███████║██████╔╝   ██║   █████╗  ██████╔╝    ██║  ██║██║███████║██║     ██║   ██║██║  ███╗
	// ██╔══██║██║  ██║██╔══██║██╔═══╝    ██║   ██╔══╝  ██╔══██╗    ██║  ██║██║██╔══██║██║     ██║   ██║██║   ██║
	// ██║  ██║██████╔╝██║  ██║██║        ██║   ███████╗██║  ██║    ██████╔╝██║██║  ██║███████╗╚██████╔╝╚██████╔╝
	// ╚═╝  ╚═╝╚═════╝ ╚═╝  ╚═╝╚═╝        ╚═╝   ╚══════╝╚═╝  ╚═╝    ╚═════╝ ╚═╝╚═╝  ╚═╝╚══════╝ ╚═════╝  ╚═════╝

	showAdapterDialog := func(editAdapter *astra.AstraAdapter) {
		ui.ShowAdapterDialog(
			conn,
			editAdapter,
			rt.currentConfig.Adapters,
			rt.currentConfig.Streams,
			func(saved astra.AstraAdapter) {
				rt.versionView.SetText(fmt.Sprintf(
					"[green]Adapter saved: %s[-]",
					tview.Escape(saved.DisplayName()),
				))
				loadAstraConfig()
			},
			func(adapter astra.AstraAdapter, count int) {
				rt.versionView.SetText(fmt.Sprintf(
					"[green]Scan completed: %d stream(s) added for %s[-]",
					count,
					tview.Escape(adapter.DisplayName()),
				))

				loadAstraConfigAfter(func() {
					rt.SetActivePane(dashboardPaneStreams)
				})
			},
			func(err error) {
				ui.ShowError(err.Error(), ui.app.GetFocus())
			},
		)
	}

	newAdapter := func() {
		showAdapterDialog(nil)
	}

	openSelectedAdapterAnalyzer := func() {
		if rt.activePane != dashboardPaneAdapters {
			return
		}

		adapter, ok := rt.SelectedAdapter()
		if !ok {
			return
		}

		ui.ShowAdapterAnalyzerDialog(
			conn,
			adapter,
			rt.currentConfig.Streams,
			func(adapter astra.AstraAdapter, count int) {
				loadAstraConfig()
			},
			func(err error) {
				ui.ShowError(err.Error(), ui.app.GetFocus())
			},
		)
	}

	editSelectedAdapter := func() {
		if rt.activePane != dashboardPaneAdapters {
			return
		}

		adapter, ok := rt.SelectedAdapter()
		if !ok {
			return
		}

		showAdapterDialog(&adapter)
	}

	// ███████╗████████╗██████╗ ███████╗ █████╗ ███╗   ███╗    ██████╗ ██╗ █████╗ ██╗      ██████╗  ██████╗
	// ██╔════╝╚══██╔══╝██╔══██╗██╔════╝██╔══██╗████╗ ████║    ██╔══██╗██║██╔══██╗██║     ██╔═══██╗██╔════╝
	// ███████╗   ██║   ██████╔╝█████╗  ███████║██╔████╔██║    ██║  ██║██║███████║██║     ██║   ██║██║  ███╗
	// ╚════██║   ██║   ██╔══██╗██╔══╝  ██╔══██║██║╚██╔╝██║    ██║  ██║██║██╔══██║██║     ██║   ██║██║   ██║
	// ███████║   ██║   ██║  ██║███████╗██║  ██║██║ ╚═╝ ██║    ██████╔╝██║██║  ██║███████╗╚██████╔╝╚██████╔╝
	// ╚══════╝   ╚═╝   ╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝╚═╝     ╚═╝    ╚═════╝ ╚═╝╚═╝  ╚═╝╚══════╝ ╚═════╝  ╚═════╝

	showStreamDialog := func(editStream *astra.AstraStream) {
		ui.ShowStreamDialog(
			conn,
			editStream,
			rt.currentConfig.Streams,
			rt.currentConfig.Softcams,
			func(saved astra.AstraStream) {
				activePane := rt.activePane
				rt.versionView.SetText(fmt.Sprintf(
					"[green]Stream saved: %s[-]",
					tview.Escape(saved.DisplayName()),
				))
				loadAstraConfigAfter(func() {
					rt.SetActivePane(activePane)
				})
			},
			func() {
				rt.RenderTables()
			},
			func(err error) {
				ui.ShowError(err.Error(), nil)
			},
		)
	}

	newStream := func() {
		showStreamDialog(nil)
	}

	editSelectedStream := func() {
		if rt.activePane != dashboardPaneStreams {
			return
		}

		stream, ok := rt.SelectedStream()
		if !ok {
			return
		}

		if strings.EqualFold(strings.TrimSpace(stream.Type), "mpts") {
			ui.ShowError("MPTS streams are not supported in this version of astracmd", nil)
			return
		}

		showStreamDialog(&stream)
	}

	openSelectedStreamAnalyzer := func() {
		if rt.activePane != dashboardPaneStreams {
			return
		}

		stream, ok := rt.SelectedStream()
		if !ok {
			return
		}

		ui.ShowStreamAnalyzerDialog(conn, stream)
	}

	openSelectedDashboardItem := func() {
		switch rt.activePane {
		case dashboardPaneAdapters:
			openSelectedAdapterAnalyzer()

		case dashboardPaneStreams:
			openSelectedStreamAnalyzer()
		}
	}

	// ███╗   ██╗███████╗██╗    ██╗        ██╗    ███████╗██████╗ ██╗████████╗
	// ████╗  ██║██╔════╝██║    ██║       ██╔╝    ██╔════╝██╔══██╗██║╚══██╔══╝
	// ██╔██╗ ██║█████╗  ██║ █╗ ██║      ██╔╝     █████╗  ██║  ██║██║   ██║
	// ██║╚██╗██║██╔══╝  ██║███╗██║     ██╔╝      ██╔══╝  ██║  ██║██║   ██║
	// ██║ ╚████║███████╗╚███╔███╔╝    ██╔╝       ███████╗██████╔╝██║   ██║
	// ╚═╝  ╚═══╝╚══════╝ ╚══╝╚══╝     ╚═╝        ╚══════╝╚═════╝ ╚═╝   ╚═╝

	newDashboardItem := func() {
		switch rt.activePane {
		case dashboardPaneAdapters:
			newAdapter()

		case dashboardPaneStreams:
			newStream()
		}
	}

	editSelectedDashboardItem := func() {
		switch rt.activePane {
		case dashboardPaneAdapters:
			editSelectedAdapter()

		case dashboardPaneStreams:
			editSelectedStream()
		}
	}

	restartSelectedAdapter := func() {
		adapter, ok := rt.SelectedAdapter()
		if !ok {
			return
		}

		adapterName := adapter.DisplayName()

		dashboardRunAsyncItemAction(
			ui,
			rt.versionView,
			"Restart adapter",
			"Adapter restarted",
			adapterName,
			func(ctx context.Context) error {
				return dashboardRestartAdapter(ctx, rt.client, adapter)
			},
			func() {
				loadAstraConfig()
			},
		)
	}

	restartSelectedStream := func() {
		stream, ok := rt.SelectedStream()
		if !ok {
			return
		}

		streamName := stream.DisplayName()

		dashboardRunAsyncItemAction(
			ui,
			rt.versionView,
			"Restart stream",
			"Stream restarted",
			streamName,
			func(ctx context.Context) error {
				return dashboardRestartStream(ctx, rt.client, stream)
			},
			func() {
				loadAstraConfig()
			},
		)
	}

	restartSelectedDashboardItem := func() {
		switch rt.activePane {
		case dashboardPaneAdapters:
			restartSelectedAdapter()

		case dashboardPaneStreams:
			restartSelectedStream()
		}
	}

	deleteSelectedAdapter := func() {
		adapter, ok := rt.SelectedAdapter()
		if !ok {
			return
		}

		adapterName := adapter.DisplayName()

		ui.ShowDashboardConfirm(
			fmt.Sprintf(
				"Delete adapter?\n\n%s\n\nDanger: this will delete the adapter and all related streams.",
				adapterName,
			),
			"Delete",
			70,
			12,
			func() {
				dashboardRunAsyncItemAction(
					ui,
					rt.versionView,
					"Delete adapter",
					"Adapter deleted",
					adapterName,
					func(ctx context.Context) error {
						return dashboardDeleteAdapter(ctx, rt.client, adapter)
					},
					func() {
						loadAstraConfig()
					},
				)
			},
		)
	}

	deleteSelectedStream := func() {
		streams := rt.MarkedStreams()

		if len(streams) == 0 {
			stream, ok := rt.SelectedStream()
			if !ok {
				return
			}

			streams = []astra.AstraStream{stream}
		}

		if len(streams) == 1 {
			streamName := streams[0].DisplayName()

			ui.ShowDashboardConfirm(
				fmt.Sprintf(
					"Delete stream?\n\n%s",
					streamName,
				),
				"Delete",
				60,
				10,
				func() {
					dashboardRunAsyncItemAction(
						ui,
						rt.versionView,
						"Delete stream",
						"Stream deleted",
						streamName,
						func(ctx context.Context) error {
							return dashboardDeleteStream(ctx, rt.client, streams[0])
						},
						func() {
							rt.ClearMarkedStreams()
							loadAstraConfig()
						},
					)
				},
			)

			return
		}

		ui.ShowDashboardConfirm(
			fmt.Sprintf(
				"Delete selected streams?\n\n%d streams will be deleted.",
				len(streams),
			),
			"Delete",
			60,
			10,
			func() {
				title := fmt.Sprintf("%d streams", len(streams))

				dashboardRunAsyncItemAction(
					ui,
					rt.versionView,
					"Delete streams",
					"Streams deleted",
					title,
					func(ctx context.Context) error {
						for _, stream := range streams {
							if err := dashboardDeleteStream(ctx, rt.client, stream); err != nil {
								return fmt.Errorf("%s: %w", stream.DisplayName(), err)
							}
						}

						return nil
					},
					func() {
						rt.ClearMarkedStreams()
						loadAstraConfig()
					},
				)
			},
		)
	}

	deleteSelectedDashboardItem := func() {
		switch rt.activePane {
		case dashboardPaneAdapters:
			deleteSelectedAdapter()

		case dashboardPaneStreams:
			deleteSelectedStream()
		}
	}

	rt.streamsTable.SetSelectionChangedFunc(func(row int, _ int) {
		if rt.isRendering {
			return
		}

		if row < 0 || row >= len(rt.visibleStreams) {
			return
		}

		rt.RenderLog()
		rt.UpdateBorders()
	})

	rt.adaptersTable.SetSelectionChangedFunc(func(row int, _ int) {
		if rt.isRendering {
			return
		}

		if !dashboardNormalizeAdapterSelectionRow(
			rt.adaptersTable,
			rt.currentConfig.Adapters,
			row,
		) {
			return
		}

		rt.RenderStreams()
		rt.RenderLog()
		rt.UpdateBorders()
	})

	functionKeys := ui.NewFunctionKeyBar(map[int]FunctionKeyAction{
		1: {
			Label: "Help",
			Handle: func() {
				ui.ShowHelp()
			},
		},
		2: {
			Label: "Restart",
			Handle: func() {
				confirmRestart()
			},
		},
		3: {
			Label: "SoftCAM",
			Handle: func() {
				showSoftCAMDialog()
			},
		},
		4: {
			Label: "Edit",
			Handle: func() {
				editSelectedDashboardItem()
			},
		},
		5: {
			Label: "Reload",
			Handle: func() {
				loadAstraConfig()
			},
		},
		6: {
			Label: "Debug",
			Handle: func() {
				toggleAstraDebugLog()
			},
		},
		7: {
			Label: "New",
			Handle: func() {
				newDashboardItem()
			},
		},
		8: {
			Label: "Delete",
			Handle: func() {
				deleteSelectedDashboardItem()
			},
		},
		9: {
			Label: "License",
			Handle: func() {
				showLicenseDialog()
			},
		},
		10: {
			Label: "Quit",
			Handle: func() {
				ui.Quit()
			},
		},
	})

	root := rt.BuildRoot(functionKeys)

	handleDashboardKeys := func(event *tcell.EventKey) *tcell.EventKey {
		handled := dashboardHandleKeys(
			event,
			DashboardKeyActions{
				ShowHelp: func() {
					ui.ShowHelp()
				},

				Restart: func() {
					confirmRestart()
				},

				OpenItem: func() {
					openSelectedDashboardItem()
				},

				ToggleStreamMark: func() {
					rt.ToggleSelectedStreamMark()
				},

				SoftCAM: func() {
					showSoftCAMDialog()
				},

				Reload: func() {
					loadAstraConfig()
				},

				Debug: func() {
					toggleAstraDebugLog()
				},

				Delete: func() {
					deleteSelectedDashboardItem()
				},

				License: func() {
					showLicenseDialog()
				},

				Back: func() {
					ui.StopDashboardTimer()
					ui.ShowConnections()
				},

				Quit: func() {
					ui.Quit()
				},

				RestartItem: func() {
					restartSelectedDashboardItem()
				},

				MoveAdapterUp: func() {
					rt.MoveAdapterSelection(-1)
				},

				MoveAdapterDown: func() {
					rt.MoveAdapterSelection(1)
				},

				SetAdaptersPane: func() {
					rt.SetActivePane(dashboardPaneAdapters)
				},

				SetStreamsPane: func() {
					rt.SetActivePane(dashboardPaneStreams)
				},

				GetActivePane: func() int {
					return rt.activePane
				},

				NewItem: func() {
					newDashboardItem()
				},

				EditItem: func() {
					editSelectedDashboardItem()
				},

				MarkAllStreams: func() {
					rt.MarkAllVisibleStreams()
				},
			},
			ui.HandleGlobalKeys,
		)

		if handled {
			return nil
		}

		return event
	}

	root.SetInputCapture(handleDashboardKeys)
	rt.adaptersTable.SetInputCapture(handleDashboardKeys)
	rt.streamsTable.SetInputCapture(handleDashboardKeys)
	rt.logTable.SetInputCapture(handleDashboardKeys)

	ui.setMain(root)

	rt.RenderTables()
	rt.SetActivePane(dashboardPaneAdapters)

	dashboardStartVersionWatcher(
		ctx,
		conn,
		&rt.forceOfflineUntil,
		rt.QueueUpdateDraw,
		5*time.Second,
		DashboardVersionCallbacks{
			SetRestarting: func() {
				rt.SetRestarting()
			},

			SetOnline: func(version string) {
				rt.SetOnline(version)
			},

			SetOffline: func(message string) {
				rt.SetOffline(message)
			},

			OnFirstOnline: func() {
				if rt.loadDone.CompareAndSwap(false, true) {
					setAstraDebugLog(rt.debugLogEnabled)
					loadAstraConfig()
					loadAstraLog()
					startAstraWebSocket()
				}
			},
		},
	)
}
