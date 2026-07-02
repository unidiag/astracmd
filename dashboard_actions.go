package main

import (
	"context"
	"fmt"
	"strings"
)

func dashboardRestartAstra(
	ctx context.Context,
	conn AstraConnection,
) error {
	result := AstraRestart(ctx, conn)
	if result.OK {
		return nil
	}

	if result.Err != nil {
		return result.Err
	}

	return fmt.Errorf("astra restart failed")
}

func dashboardRestartStream(
	ctx context.Context,
	conn AstraConnection,
	stream AstraStream,
) error {
	streamID := strings.TrimSpace(stream.ID)
	if streamID == "" {
		return fmt.Errorf("stream id is empty")
	}

	result := AstraRestartStream(ctx, conn, streamID)
	if result.OK {
		return nil
	}

	if result.Err != nil {
		return result.Err
	}

	return fmt.Errorf("stream restart failed: %s", stream.DisplayName())
}

func dashboardDeleteStream(
	ctx context.Context,
	conn AstraConnection,
	stream AstraStream,
) error {
	streamID := strings.TrimSpace(stream.ID)
	if streamID == "" {
		return fmt.Errorf("stream id is empty")
	}

	result := AstraDeleteStream(ctx, conn, streamID)
	if result.OK {
		return nil
	}

	if result.Err != nil {
		return result.Err
	}

	return fmt.Errorf("stream delete failed: %s", stream.DisplayName())
}

func dashboardReloadConfig(
	ctx context.Context,
	conn AstraConnection,
) (AstraConfig, error) {
	result := AstraLoad(ctx, conn)
	if result.Online {
		return result.Config, nil
	}

	if result.Err != nil {
		return AstraConfig{}, result.Err
	}

	return AstraConfig{}, fmt.Errorf("astra config load failed")
}

func dashboardLoadAstraStatus(
	ctx context.Context,
	conn AstraConnection,
) (AstraStatus, error) {
	result := AstraGetStatus(ctx, conn)
	if result.OK {
		return result.Status, nil
	}

	if result.Err != nil {
		return AstraStatus{}, result.Err
	}

	return AstraStatus{}, fmt.Errorf("astra status load failed")
}

func dashboardRestartAdapter(
	ctx context.Context,
	conn AstraConnection,
	adapter AstraAdapter,
) error {
	adapterID := strings.TrimSpace(adapter.ID)
	if adapterID == "" {
		return fmt.Errorf("adapter id is empty")
	}

	result := AstraRestartAdapter(ctx, conn, adapterID)
	if result.OK {
		return nil
	}

	if result.Err != nil {
		return result.Err
	}

	return fmt.Errorf("adapter restart failed: %s", adapter.DisplayName())
}

func dashboardDeleteAdapter(
	ctx context.Context,
	conn AstraConnection,
	adapter AstraAdapter,
) error {
	adapterID := strings.TrimSpace(adapter.ID)
	if adapterID == "" {
		return fmt.Errorf("adapter id is empty")
	}

	result := AstraDeleteAdapter(ctx, conn, adapterID)
	if result.OK {
		return nil
	}

	if result.Err != nil {
		return result.Err
	}

	return fmt.Errorf("adapter delete failed: %s", adapter.DisplayName())
}
