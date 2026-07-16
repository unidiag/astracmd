package astra

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type AstraSetLogResult struct {
	OK  bool
	Err error
}

type AstraLogResult struct {
	Items []AstraLogItem
	Debug bool
	OK    bool
	Err   error
}

type AstraLogResponse struct {
	Info  any            `json:"info"`
	Log   []AstraLogItem `json:"log"`
	Debug bool           `json:"debug"`
}

type AstraLogItem struct {
	ID   int64  `json:"id"`
	Time int64  `json:"time"`
	Type int    `json:"type"`
	Text string `json:"text"`

	Level   string `json:"level,omitempty"`
	Message string `json:"message,omitempty"`
}

func AstraSetLog(ctx context.Context, conn Connection, debug bool) AstraSetLogResult {
	payload := struct {
		Cmd string `json:"cmd"`
		Set struct {
			Debug bool `json:"debug"`
		} `json:"set"`
	}{
		Cmd: "set-log",
	}

	payload.Set.Debug = debug

	body, err := json.Marshal(payload)
	if err != nil {
		return AstraSetLogResult{OK: false, Err: err}
	}

	raw, err := controlRequest(ctx, conn, body)
	if err != nil {
		return AstraSetLogResult{OK: false, Err: err}
	}

	var data struct {
		SetLog string `json:"set-log"`
	}

	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &data); err != nil {
			return AstraSetLogResult{OK: false, Err: err}
		}

		if strings.TrimSpace(data.SetLog) != "ok" {
			return AstraSetLogResult{
				OK:  false,
				Err: fmt.Errorf("unexpected set-log response: %s", string(raw)),
			}
		}
	}

	return AstraSetLogResult{OK: true}
}

func AstraLog(ctx context.Context, conn Connection) AstraLogResult {
	raw, err := controlRequest(ctx, conn, []byte(`{"cmd":"log"}`))
	if err != nil {
		return AstraLogResult{
			OK:  false,
			Err: err,
		}
	}

	var data AstraLogResponse
	if err := json.Unmarshal(raw, &data); err != nil {
		return AstraLogResult{
			OK: false,
			Err: fmt.Errorf(
				"parse astra log response: %w; raw: %s",
				err,
				string(raw),
			),
		}
	}

	for i := range data.Log {
		item := &data.Log[i]

		if item.Text == "" {
			item.Text = item.Message
		}

		if item.Type == 0 {
			item.Type = astraLogTypeFromLevel(item.Level)
		}

		if item.ID == 0 {
			item.ID = item.Time*1_000_000 + int64(i)
		}
	}

	return AstraLogResult{
		Items: data.Log,
		Debug: data.Debug,
		OK:    true,
	}
}

func astraLogTypeFromLevel(level string) int {
	switch strings.ToUpper(strings.TrimSpace(level)) {
	case "ERROR":
		return 3

	case "WARN", "WARNING":
		return 2

	case "DEBUG":
		return 0

	default:
		return 1
	}
}
