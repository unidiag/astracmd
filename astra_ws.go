package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

type AstraWSMessage struct {
	Raw []byte
	Err error
}

type AstraWSClient struct {
	conn *websocket.Conn
}

type AstraWSEnvelope struct {
	Scope string `json:"scope"`
}

type AstraWSLogEvent struct {
	Scope string         `json:"scope"`
	Log   []AstraLogItem `json:"log"`
}

type AstraWSStreamEvent struct {
	Scope     string `json:"scope"`
	ChannelID string `json:"channel_id"`
	PESError  int    `json:"pes_error"`
	CCError   int    `json:"cc_error"`
	Scrambled bool   `json:"scrambled"`
	Bitrate   int    `json:"bitrate"`
	Onair     bool   `json:"onair"`
	InputID   int    `json:"input_id"`
}

type AstraWSAdapterEvent struct {
	Scope    string `json:"scope"`
	DVBID    string `json:"dvb_id"`
	Signal   int    `json:"signal"`
	SignalDB int    `json:"signal_db"`
	Bitrate  int    `json:"bitrate"`
	UNC      int    `json:"unc"`
	SNRDB    int    `json:"snr_db"`
	SNR      int    `json:"snr"`
	BER      int    `json:"ber"`
	Status   int    `json:"status"`
}

type AstraWSSysInfoEvent struct {
	Scope         string `json:"scope"`
	Uptime        int64  `json:"uptime"`
	CPUTotalUsage int    `json:"cpu_total_usage"`
}

func AstraConnectWebSocket(ctx context.Context, conn AstraConnection) (*AstraWSClient, error) {
	wsURL := url.URL{
		Scheme: "ws",
		Host:   conn.Addr(),
		Path:   "/control/event/",
	}

	headers := http.Header{}
	headers.Set("Cookie", fmt.Sprintf(
		"auth=%s",
		url.QueryEscape(conn.Login+":"+conn.Password),
	))

	dialer := websocket.Dialer{
		HandshakeTimeout: 5 * time.Second,
	}

	ws, _, err := dialer.DialContext(ctx, wsURL.String(), headers)
	if err != nil {
		return nil, err
	}

	authMessage := struct {
		Scope string `json:"scope"`
		Auth  string `json:"auth"`
	}{
		Scope: "auth",
		Auth:  conn.Login + ":" + conn.Password,
	}

	if err := ws.WriteJSON(authMessage); err != nil {
		_ = ws.Close()
		return nil, err
	}

	return &AstraWSClient{
		conn: ws,
	}, nil
}

func (c *AstraWSClient) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}

	return c.conn.Close()
}

func (c *AstraWSClient) ReadLoop(ctx context.Context, out chan<- AstraWSMessage) {
	defer close(out)

	if c == nil || c.conn == nil {
		out <- AstraWSMessage{Err: fmt.Errorf("websocket client is nil")}
		return
	}

	done := make(chan struct{})

	go func() {
		select {
		case <-ctx.Done():
			_ = c.conn.Close()
		case <-done:
		}
	}()

	defer close(done)

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				out <- AstraWSMessage{Err: err}
				return
			}
		}

		select {
		case <-ctx.Done():
			return

		case out <- AstraWSMessage{Raw: data}:
		}
	}
}
