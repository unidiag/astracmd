package dashboard

import (
	"context"
	"fmt"
	"main/internal/astra"
	"strings"
)

func dashboardRestartAstra(
	ctx context.Context,
	client *astra.Client,
) error {
	result := client.Restart(ctx)
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
	client *astra.Client,
	stream astra.Stream,
) error {
	streamID := strings.TrimSpace(stream.ID)
	if streamID == "" {
		return fmt.Errorf("stream id is empty")
	}

	result := client.RestartStream(ctx, streamID)
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
	client *astra.Client,
	stream astra.Stream,
) error {
	streamID := strings.TrimSpace(stream.ID)
	if streamID == "" {
		return fmt.Errorf("stream id is empty")
	}

	result := client.DeleteStream(ctx, streamID)
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
	client *astra.Client,
) (astra.Config, error) {
	result := client.Load(ctx)
	if result.Online {
		return result.Config, nil
	}

	if result.Err != nil {
		return astra.Config{}, result.Err
	}

	return astra.Config{}, fmt.Errorf("astra config load failed")
}

func dashboardLoadAstraStatus(
	ctx context.Context,
	client *astra.Client,
) (astra.Status, error) {
	result := client.GetStatus(ctx)
	if result.OK {
		return result.Status, nil
	}

	if result.Err != nil {
		return astra.Status{}, result.Err
	}

	return astra.Status{}, fmt.Errorf("astra status load failed")
}

func dashboardRestartAdapter(
	ctx context.Context,
	client *astra.Client,
	adapter astra.Adapter,
) error {
	adapterID := strings.TrimSpace(adapter.ID)
	if adapterID == "" {
		return fmt.Errorf("adapter id is empty")
	}

	result := client.RestartAdapter(ctx, adapterID)
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
	client *astra.Client,
	adapter astra.Adapter,
) error {
	adapterID := strings.TrimSpace(adapter.ID)
	if adapterID == "" {
		return fmt.Errorf("adapter id is empty")
	}

	result := client.DeleteAdapter(ctx, adapterID)
	if result.OK {
		return nil
	}

	if result.Err != nil {
		return result.Err
	}

	return fmt.Errorf("adapter delete failed: %s", adapter.DisplayName())
}
