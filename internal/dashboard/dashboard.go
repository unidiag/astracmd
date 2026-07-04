package dashboard

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"main/internal/astra"

	"github.com/davecgh/go-spew/spew"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func Show(opt Options) {
	opt.StopDashboardTimer()
	opt.RemoveDialog()

	conn := opt.Connection

	ctx, cancel := context.WithCancel(context.Background())
	opt.SetDashboardCancel(cancel)

	rt := NewDashboardRuntime(opt, conn, ctx, cancel)

	currentFocus := func() tview.Primitive {
		if opt.App == nil {
			return nil
		}

		return opt.App.GetFocus()
	}

	restrictedDenied := func() {
		opt.ShowError(
			"Restricted mode: run astracmd as root to change Astra config",
			currentFocus(),
		)
	}

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
						opt.ShowError(result.Err.Error(), nil)
					} else {
						opt.ShowError("Astra: offline", nil)
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
				opt.ShowError(err.Error(), nil)
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

				if opt.Config != nil {
					opt.Config.UpdateConnectionDebug(conn.ID, enabled)

					if err := opt.Config.Save(); err != nil {
						opt.ShowError(err.Error(), nil)
						return
					}
				}

				rt.UpdateLogTitle()
			},
			func(err error) {
				opt.ShowError(err.Error(), nil)
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

				OnAdapterState: func(adapterID string, state astra.AdapterState) {
					rt.adapterStates[adapterID] = state
					rt.RenderAdapters()
				},

				OnStreamState: func(streamID string, state astra.StreamState) {
					rt.streamStates[streamID] = state
					rt.RenderStreams()
				},
			},
		)
	}

	confirmRestart := func() {
		ConfirmRestartAstra(
			opt,
			conn,
			func() {
				rt.forceOfflineUntil.Store(time.Now().Add(10 * time.Second).UnixNano())
				rt.SetRestarting()
			},
			func(err error) {
				opt.ShowError(err.Error(), nil)
			},
		)
	}

	showLicenseDialog := func() {
		ShowLicenseDialog(
			opt,
			conn,
			func() {
				rt.versionView.SetText("[green]License applied[-]")
			},
			func(err error) {
				opt.ShowError(err.Error(), nil)
			},
		)
	}

	showSoftCAMDialog := func() {
		if !CanChangeAstraConfig() {
			restrictedDenied()
			return
		}

		ShowSoftCAMDialog(
			opt,
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

	showAdapterDialog := func(editAdapter *astra.Adapter) {
		ShowAdapterDialog(
			opt,
			conn,
			editAdapter,
			rt.currentConfig.Adapters,
			rt.currentConfig.Streams,
			func(saved astra.Adapter) {
				rt.versionView.SetText(fmt.Sprintf(
					"[green]Adapter saved: %s[-]",
					tview.Escape(saved.DisplayName()),
				))

				loadAstraConfig()
			},
			func(adapter astra.Adapter, count int) {
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
				opt.ShowError(err.Error(), currentFocus())
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

		ShowAdapterAnalyzerDialog(
			opt,
			conn,
			adapter,
			rt.currentConfig.Streams,
			func(adapter astra.Adapter, count int) {
				loadAstraConfig()
			},
			func(err error) {
				opt.ShowError(err.Error(), currentFocus())
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

	showStreamDialog := func(editStream *astra.Stream) {
		ShowStreamDialog(
			opt,
			conn,
			editStream,
			rt.currentConfig.Streams,
			rt.currentConfig.Softcams,
			func(saved astra.Stream) {
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
				opt.ShowError(err.Error(), nil)
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
			opt.ShowError("MPTS streams are not supported in this version of astracmd", nil)
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

		ShowStreamAnalyzerDialog(opt, conn, stream)
	}

	openSelectedDashboardItem := func() {
		switch rt.activePane {
		case dashboardPaneAdapters:
			openSelectedAdapterAnalyzer()

		case dashboardPaneStreams:
			openSelectedStreamAnalyzer()
		}
	}

	newDashboardItem := func() {
		if !CanChangeAstraConfig() {
			restrictedDenied()
			return
		}

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
			opt,
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
			opt,
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

		ShowDashboardConfirm(
			opt,
			fmt.Sprintf(
				"Delete adapter?\n\n%s\n\nDanger: this will delete the adapter and all related streams.",
				adapterName,
			),
			"Delete",
			70,
			12,
			func() {
				dashboardRunAsyncItemAction(
					opt,
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

			streams = []astra.Stream{stream}
		}

		if len(streams) == 1 {
			streamName := streams[0].DisplayName()

			ShowDashboardConfirm(
				opt,
				fmt.Sprintf(
					"Delete stream?\n\n%s",
					streamName,
				),
				"Delete",
				60,
				10,
				func() {
					dashboardRunAsyncItemAction(
						opt,
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

		ShowDashboardConfirm(
			opt,
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
					opt,
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
		if !CanChangeAstraConfig() {
			restrictedDenied()
			return
		}

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

	functionKeys := opt.NewFunctionKeyBar(map[int]FunctionKeyAction{
		1: {
			Label: "Help",
			Handle: func() {
				ShowHelp(opt)
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
				opt.Quit()
			},
		},
	})

	root := rt.BuildRoot(functionKeys)

	handleDashboardKeys := func(event *tcell.EventKey) *tcell.EventKey {
		handled := dashboardHandleKeys(
			event,
			DashboardKeyActions{
				ShowHelp: func() {
					ShowHelp(opt)
				},

				Restart: func() {
					confirmRestart()
				},

				OpenItem: func() {
					openSelectedDashboardItem()
				},

				RestrictedDenied: func() {
					restrictedDenied()
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
					opt.StopDashboardTimer()
					opt.ShowConnections()
				},

				Quit: func() {
					opt.Quit()
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
			opt.HandleGlobalKeys,
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

	opt.SetMain(root)

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

func debugSave(opt Options, some any) {
	if !opt.Debug {
		return
	}

	text := "DEBUG " + time.Now().Format("2006-01-02 15:04:05") + "\n\n"
	text += spew.Sdump(some)

	_ = os.WriteFile("debug.txt", []byte(text), 0644)
}
