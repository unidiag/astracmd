package dashboard

import (
	"context"
	"encoding/json"
	"main/internal/astra"
	"strings"
	"time"
)

type DashboardWebSocketHandlers struct {
	OnLogItems     func(items []astra.AstraLogItem)
	OnAdapterState func(adapterID string, state astra.AdapterState)
	OnStreamState  func(streamID string, state astra.StreamState)
}

func dashboardStartAstraWebSocket(
	ctx context.Context,
	conn astra.Connection,
	queueUpdate func(func()),
	handlers DashboardWebSocketHandlers,
) {
	if queueUpdate == nil {
		queueUpdate = func(fn func()) {
			fn()
		}
	}

	go func() {
		reconnectDelay := 2 * time.Second

		for {
			select {
			case <-ctx.Done():
				return

			default:
			}

			client, err := astra.AstraConnectWebSocket(ctx, conn)
			if err != nil {
				dashboardQueueWebSocketError(queueUpdate, handlers, err)

				select {
				case <-ctx.Done():
					return
				case <-time.After(reconnectDelay):
					continue
				}
			}

			messages := make(chan astra.AstraWSMessage)
			go client.ReadLoop(ctx, messages)

			for msg := range messages {
				if msg.Err != nil {
					dashboardQueueWebSocketError(queueUpdate, handlers, msg.Err)
					break
				}

				var envelope astra.AstraWSEnvelope
				if err := json.Unmarshal(msg.Raw, &envelope); err != nil {
					continue
				}

				switch envelope.Scope {
				case "log_event":
					dashboardHandleWebSocketLogEvent(queueUpdate, handlers, msg.Raw)

				case "adapter_event":
					dashboardHandleWebSocketAdapterEvent(queueUpdate, handlers, msg.Raw)

				case "stream_event":
					dashboardHandleWebSocketStreamEvent(queueUpdate, handlers, msg.Raw)

				case "sysinfo":
					// Not used by dashboard yet.

				case "auth":
					// Not used by dashboard yet.
				}
			}

			_ = client.Close()

			select {
			case <-ctx.Done():
				return

			case <-time.After(reconnectDelay):
			}
		}
	}()
}

func dashboardQueueWebSocketError(
	queueUpdate func(func()),
	handlers DashboardWebSocketHandlers,
	err error,
) {
	if err == nil || handlers.OnLogItems == nil {
		return
	}

	item := astra.AstraLogItem{
		ID:   time.Now().UnixNano(),
		Time: time.Now().Unix(),
		Type: 2,
		Text: "[WS] " + err.Error(),
	}

	queueUpdate(func() {
		handlers.OnLogItems([]astra.AstraLogItem{item})
	})
}

func dashboardHandleWebSocketLogEvent(
	queueUpdate func(func()),
	handlers DashboardWebSocketHandlers,
	raw []byte,
) {
	if handlers.OnLogItems == nil {
		return
	}

	var event astra.AstraWSLogEvent
	if err := json.Unmarshal(raw, &event); err != nil {
		return
	}

	queueUpdate(func() {
		handlers.OnLogItems(event.Log)
	})
}

func dashboardHandleWebSocketAdapterEvent(
	queueUpdate func(func()),
	handlers DashboardWebSocketHandlers,
	raw []byte,
) {
	if handlers.OnAdapterState == nil {
		return
	}

	var event astra.AstraWSAdapterEvent
	if err := json.Unmarshal(raw, &event); err != nil {
		return
	}

	adapterID := strings.TrimSpace(event.DVBID)
	if adapterID == "" {
		return
	}

	state := astra.AdapterState{
		Signal:   event.Signal,
		SignalDB: event.SignalDB,
		Bitrate:  event.Bitrate,
		UNC:      event.UNC,
		SNRDB:    event.SNRDB,
		SNR:      event.SNR,
		BER:      event.BER,
		Status:   event.Status,
	}

	queueUpdate(func() {
		handlers.OnAdapterState(adapterID, state)
	})
}

func dashboardHandleWebSocketStreamEvent(
	queueUpdate func(func()),
	handlers DashboardWebSocketHandlers,
	raw []byte,
) {
	if handlers.OnStreamState == nil {
		return
	}

	var event astra.AstraWSStreamEvent
	if err := json.Unmarshal(raw, &event); err != nil {
		return
	}

	streamID := strings.TrimSpace(event.ChannelID)
	if streamID == "" {
		return
	}

	state := astra.StreamState{
		Bitrate:   event.Bitrate,
		Onair:     event.Onair,
		CCError:   event.CCError,
		PESError:  event.PESError,
		Scrambled: event.Scrambled,
		InputID:   event.InputID,
	}

	queueUpdate(func() {
		handlers.OnStreamState(streamID, state)
	})
}
