package main

import (
	"context"
	"fmt"
	"main/internal/astra"
)

func dashboardLoadLogAsync(
	ctx context.Context,
	conn astra.Connection,
	queueUpdate func(func()),
	onLoaded func([]astra.AstraLogItem),
	onError func(error),
) {
	go func() {
		result := astra.AstraLog(ctx, conn)

		queueUpdate(func() {
			if !result.OK {
				if onError != nil {
					if result.Err != nil {
						onError(result.Err)
					} else {
						onError(fmt.Errorf("astra log load failed"))
					}
				}
				return
			}

			if onLoaded != nil {
				onLoaded(result.Items)
			}
		})
	}()
}

func dashboardSetDebugLogAsync(
	ctx context.Context,
	conn astra.Connection,
	enabled bool,
	queueUpdate func(func()),
	onSaved func(bool),
	onError func(error),
) {
	go func() {
		result := astra.AstraSetLog(ctx, conn, enabled)

		queueUpdate(func() {
			if !result.OK {
				if result.Err != nil && onError != nil {
					onError(result.Err)
					return
				}

				if onError != nil {
					onError(fmt.Errorf("astra debug log update failed"))
				}
				return
			}

			if onSaved != nil {
				onSaved(enabled)
			}
		})
	}()
}
