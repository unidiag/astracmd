package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type AstraStreamScanInitResult struct {
	OK     bool
	ScanID string
	Err    error
}

type AstraStreamScanTotal struct {
	BitrateLimit int  `json:"bitrate_limit"`
	CCErrors     int  `json:"cc_errors"`
	PESErrors    int  `json:"pes_errors"`
	Packets      int  `json:"packets"`
	Scrambled    bool `json:"scrambled"`
	SCErrors     int  `json:"sc_errors"`
	Bitrate      int  `json:"bitrate"`
	PCRErrors    int  `json:"pcr_errors"`
}

type AstraStreamScanEvent struct {
	Scope string `json:"scope"`
	Data  struct {
		OnAir bool                 `json:"on_air"`
		Total AstraStreamScanTotal `json:"total"`
	} `json:"data"`
}

type AstraStreamScanProgram struct {
	PNR int `json:"pnr"`
	PID int `json:"pid"`
}

type AstraStreamScanDescriptor struct {
	TypeID          int    `json:"type_id"`
	TypeName        string `json:"type_name"`
	Lang            string `json:"lang"`
	Language        string `json:"language"`
	ServiceName     string `json:"service_name"`
	ServiceProvider string `json:"service_provider"`
	ServiceTypeID   int    `json:"service_type_id"`
	EventName       string `json:"event_name"`
	TextChar        string `json:"text_char"`
	Text            string `json:"text"`
}

type AstraStreamScanPMTItem struct {
	PID         int                         `json:"pid"`
	TypeID      int                         `json:"type_id"`
	TypeName    string                      `json:"type_name"`
	Descriptors []AstraStreamScanDescriptor `json:"descriptors"`
}

type AstraStreamScanService struct {
	SID         int                         `json:"sid"`
	Descriptors []AstraStreamScanDescriptor `json:"descriptors"`
}

type AstraStreamScanEITEvent struct {
	EventID       int                         `json:"event_id"`
	StartUT       int64                       `json:"start_ut"`
	StopUT        int64                       `json:"stop_ut"`
	CAMode        int                         `json:"ca_mode"`
	RunningStatus int                         `json:"running_status"`
	Descriptors   []AstraStreamScanDescriptor `json:"descriptors"`
}

type AstraStreamScanPSIData struct {
	PSI string `json:"psi"`

	PID     int `json:"pid"`
	TSID    int `json:"tsid"`
	ONID    int `json:"onid"`
	SID     int `json:"sid"`
	TableID int `json:"table_id"`
	Version int `json:"version"`

	// PAT
	Programs []AstraStreamScanProgram `json:"programs"`

	// PMT
	PNR    int `json:"pnr"`
	PCRPID int `json:"pcr_pid"`

	PMT     []AstraStreamScanPMTItem `json:"pmt"`
	Streams []AstraStreamScanPMTItem `json:"streams"`
	Items   []AstraStreamScanPMTItem `json:"items"`

	// SDT
	Services []AstraStreamScanService `json:"services"`

	// EIT
	Events []AstraStreamScanEITEvent `json:"events"`
}

type AstraStreamScanPSIEvent struct {
	Scope string                 `json:"scope"`
	Data  AstraStreamScanPSIData `json:"data"`
}

func AstraStreamScanInit(ctx context.Context, conn AstraConnection, streamID string, wsID int64) AstraStreamScanInitResult {
	streamID = strings.TrimSpace(streamID)
	if streamID == "" {
		return AstraStreamScanInitResult{
			OK:  false,
			Err: fmt.Errorf("stream id is empty"),
		}
	}

	if wsID <= 0 {
		return AstraStreamScanInitResult{
			OK:  false,
			Err: fmt.Errorf("websocket id is empty"),
		}
	}

	payload := struct {
		Cmd  string `json:"cmd"`
		Scan string `json:"scan"`
		WS   int64  `json:"ws"`
	}{
		Cmd:  "scan-init",
		Scan: "stream://" + streamID,
		WS:   wsID,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return AstraStreamScanInitResult{
			OK:  false,
			Err: err,
		}
	}

	raw, err := astraControlRequest(ctx, conn, body)
	if err != nil {
		return AstraStreamScanInitResult{
			OK:  false,
			Err: err,
		}
	}

	var resp struct {
		ScanInit string `json:"scan-init"`
		ID       string `json:"id"`
	}

	if err := json.Unmarshal(raw, &resp); err != nil {
		return AstraStreamScanInitResult{
			OK:  false,
			Err: fmt.Errorf("parse stream scan-init response: %w; raw: %s", err, string(raw)),
		}
	}

	if strings.TrimSpace(resp.ScanInit) != "ok" {
		return AstraStreamScanInitResult{
			OK:  false,
			Err: fmt.Errorf("unexpected stream scan-init response: %s", string(raw)),
		}
	}

	if strings.TrimSpace(resp.ID) == "" {
		return AstraStreamScanInitResult{
			OK:  false,
			Err: fmt.Errorf("empty stream scan id in scan-init response"),
		}
	}

	return AstraStreamScanInitResult{
		OK:     true,
		ScanID: resp.ID,
	}
}
