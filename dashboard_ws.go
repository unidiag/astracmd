package main

import (
	"context"
	"encoding/json"
	"strings"
	"time"
)

type DashboardWebSocketHandlers struct {
	OnLogItems     func(items []AstraLogItem)
	OnAdapterState func(adapterID string, state AstraAdapterState)
	OnStreamState  func(streamID string, state AstraStreamState)
}

func dashboardStartAstraWebSocket(
	ctx context.Context,
	conn AstraConnection,
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

			client, err := AstraConnectWebSocket(ctx, conn)
			if err != nil {
				dashboardQueueWebSocketError(queueUpdate, handlers, err)

				select {
				case <-ctx.Done():
					return
				case <-time.After(reconnectDelay):
					continue
				}
			}

			messages := make(chan AstraWSMessage)
			go client.ReadLoop(ctx, messages)

			for msg := range messages {
				if msg.Err != nil {
					dashboardQueueWebSocketError(queueUpdate, handlers, msg.Err)
					break
				}

				var envelope AstraWSEnvelope
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

	item := AstraLogItem{
		ID:   time.Now().UnixNano(),
		Time: time.Now().Unix(),
		Type: 2,
		Text: "[WS] " + err.Error(),
	}

	queueUpdate(func() {
		handlers.OnLogItems([]AstraLogItem{item})
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

	var event AstraWSLogEvent
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

	var event AstraWSAdapterEvent
	if err := json.Unmarshal(raw, &event); err != nil {
		return
	}

	adapterID := strings.TrimSpace(event.DVBID)
	if adapterID == "" {
		return
	}

	state := AstraAdapterState{
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

	var event AstraWSStreamEvent
	if err := json.Unmarshal(raw, &event); err != nil {
		return
	}

	streamID := strings.TrimSpace(event.ChannelID)
	if streamID == "" {
		return
	}

	state := AstraStreamState{
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
