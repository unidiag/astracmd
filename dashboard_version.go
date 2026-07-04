package main

import (
	"context"
	"main/internal/astra"
	"sync/atomic"
	"time"
)

type DashboardVersionCallbacks struct {
	SetRestarting func()
	SetOnline     func(version string)
	SetOffline    func(message string)

	OnFirstOnline func()
}

func dashboardStartVersionWatcher(
	ctx context.Context,
	conn astra.AstraConnection,
	forceOfflineUntil *atomic.Int64,
	queueUpdate func(func()),
	interval time.Duration,
	callbacks DashboardVersionCallbacks,
) {
	if interval <= 0 {
		interval = 5 * time.Second
	}

	update := func() {
		result := astra.AstraVersion(ctx, conn)

		queueUpdate(func() {
			if forceOfflineUntil != nil && time.Now().UnixNano() < forceOfflineUntil.Load() {
				if callbacks.SetRestarting != nil {
					callbacks.SetRestarting()
				}
				return
			}

			if result.Online {
				if callbacks.SetOnline != nil {
					callbacks.SetOnline(result.DisplayVersion())
				}

				if callbacks.OnFirstOnline != nil {
					callbacks.OnFirstOnline()
				}

				return
			}

			if result.Err != nil {
				if callbacks.SetOffline != nil {
					callbacks.SetOffline(result.Err.Error())
				}
				return
			}

			if callbacks.SetOffline != nil {
				callbacks.SetOffline("Astra: offline")
			}
		})
	}

	go func() {
		update()

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return

			case <-ticker.C:
				update()
			}
		}
	}()
}
