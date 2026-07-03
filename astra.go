package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const astraRequestTimeout = 10 * time.Second

type AstraVersionResult struct {
	Version string
	Commit  string
	Online  bool
	Err     error
}

func (r AstraVersionResult) DisplayVersion() string {
	version := strings.TrimSpace(r.Version)
	commit := strings.TrimSpace(r.Commit)

	switch {
	case version != "" && commit != "":
		return version + " / " + commit
	case version != "":
		return version
	case commit != "":
		return commit
	default:
		return "unknown"
	}
}

type AstraLoadResult struct {
	Config AstraConfig
	Online bool
	Err    error
}

type AstraRestartResult struct {
	OK  bool
	Err error
}

type AstraSetLicenseResult struct {
	OK  bool
	Err error
}

type AstraStatusResult struct {
	Status AstraStatus
	OK     bool
	Err    error
}

type AstraStatus struct {
	License AstraLicense `json:"license"`
}

type AstraLicense struct {
	Type    int    `json:"type"`
	ID      string `json:"id"`
	Email   string `json:"email"`
	U       string `json:"u"`
	Message string `json:"message"`
	Expire  int64  `json:"expire"`
}

type AstraRestartItemResult struct {
	OK  bool
	Err error
}

type AstraDeleteItemResult struct {
	OK  bool
	Err error
}

func AstraVersion(ctx context.Context, conn AstraConnection) AstraVersionResult {
	raw, err := astraControlRequest(ctx, conn, []byte(`{"cmd":"version"}`))
	if err != nil {
		return AstraVersionResult{Online: false, Err: err}
	}

	version, commit := parseAstraVersion(raw)
	if version == "" && commit == "" {
		return AstraVersionResult{
			Online: false,
			Err:    fmt.Errorf("invalid version response: %s", string(raw)),
		}
	}

	return AstraVersionResult{
		Version: version,
		Commit:  commit,
		Online:  true,
	}
}

func parseAstraVersion(raw []byte) (string, string) {
	var resp struct {
		Version string `json:"version"`
		Commit  string `json:"commit"`
	}

	if err := json.Unmarshal(raw, &resp); err != nil {
		return "", ""
	}

	return strings.TrimSpace(resp.Version), strings.TrimSpace(resp.Commit)
}

func AstraLoad(ctx context.Context, conn AstraConnection) AstraLoadResult {
	raw, err := astraControlRequest(ctx, conn, []byte(`{"cmd":"load"}`))
	if err != nil {
		return AstraLoadResult{Online: false, Err: err}
	}

	var cfg AstraConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return AstraLoadResult{Online: false, Err: err}
	}

	return AstraLoadResult{
		Config: cfg,
		Online: true,
	}
}

func AstraRestart(ctx context.Context, conn AstraConnection) AstraRestartResult {
	raw, err := astraControlRequest(ctx, conn, []byte(`{"cmd":"restart"}`))
	if err != nil {
		return AstraRestartResult{OK: false, Err: err}
	}

	var resp struct {
		Restart string `json:"restart"`
	}

	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &resp); err != nil {
			return AstraRestartResult{OK: false, Err: err}
		}

		if strings.TrimSpace(resp.Restart) != "ok" {
			return AstraRestartResult{
				OK:  false,
				Err: fmt.Errorf("unexpected restart response: %s", string(raw)),
			}
		}
	}

	return AstraRestartResult{OK: true}
}

func AstraSetLicense(ctx context.Context, conn AstraConnection, license string) AstraSetLicenseResult {
	payload := struct {
		Cmd     string `json:"cmd"`
		License string `json:"license"`
	}{
		Cmd:     "set-license",
		License: license,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return AstraSetLicenseResult{OK: false, Err: err}
	}

	_, err = astraControlRequest(ctx, conn, body)
	if err != nil {
		return AstraSetLicenseResult{OK: false, Err: err}
	}

	return AstraSetLicenseResult{OK: true}
}

func AstraGetStatus(ctx context.Context, conn AstraConnection) AstraStatusResult {
	raw, err := astraControlRequest(ctx, conn, []byte(`{"cmd":"status"}`))
	if err != nil {
		return AstraStatusResult{OK: false, Err: err}
	}

	var status AstraStatus
	if err := json.Unmarshal(raw, &status); err != nil {
		return AstraStatusResult{OK: false, Err: err}
	}

	return AstraStatusResult{
		Status: status,
		OK:     true,
	}
}

func AstraRestartStream(ctx context.Context, conn AstraConnection, id string) AstraRestartItemResult {
	return AstraRestartItem(ctx, conn, "restart-stream", id)
}

func AstraRestartAdapter(ctx context.Context, conn AstraConnection, id string) AstraRestartItemResult {
	return AstraRestartItem(ctx, conn, "restart-adapter", id)
}

func AstraRestartItem(ctx context.Context, conn AstraConnection, cmd string, id string) AstraRestartItemResult {
	payload := struct {
		Cmd string `json:"cmd"`
		ID  string `json:"id"`
	}{
		Cmd: cmd,
		ID:  id,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return AstraRestartItemResult{OK: false, Err: err}
	}

	_, err = astraControlRequest(ctx, conn, body)
	if err != nil {
		return AstraRestartItemResult{OK: false, Err: err}
	}

	return AstraRestartItemResult{OK: true}
}

type AstraSaveAdapterResult struct {
	OK  bool
	Err error
}

type astraSetAdapterPayload struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Type         string `json:"type"`
	Adapter      int    `json:"adapter"`
	Device       int    `json:"device"`
	Frequency    string `json:"frequency"`
	Polarization string `json:"polarization,omitempty"`
	Symbolrate   string `json:"symbolrate,omitempty"`
	Bandwidth    string `json:"bandwidth,omitempty"`
	Hierarchy    string `json:"hierarchy,omitempty"`
	Modulation   string `json:"modulation,omitempty"`
	Lof1         string `json:"lof1,omitempty"`
	Lof2         string `json:"lof2,omitempty"`
	Slof         string `json:"slof,omitempty"`
	Enable       bool   `json:"enable"`
}

//  █████╗ ██████╗  █████╗ ██████╗ ████████╗███████╗██████╗
// ██╔══██╗██╔══██╗██╔══██╗██╔══██╗╚══██╔══╝██╔════╝██╔══██╗
// ███████║██║  ██║███████║██████╔╝   ██║   █████╗  ██████╔╝
// ██╔══██║██║  ██║██╔══██║██╔═══╝    ██║   ██╔══╝  ██╔══██╗
// ██║  ██║██████╔╝██║  ██║██║        ██║   ███████╗██║  ██║
// ╚═╝  ╚═╝╚═════╝ ╚═╝  ╚═╝╚═╝        ╚═╝   ╚══════╝╚═╝  ╚═╝

func AstraSaveAdapter(ctx context.Context, conn AstraConnection, adapter AstraAdapter) AstraSaveAdapterResult {
	payload := struct {
		Cmd     string                 `json:"cmd"`
		ID      string                 `json:"id"`
		Adapter astraSetAdapterPayload `json:"adapter"`
	}{
		Cmd: "set-adapter",
		ID:  adapter.ID,
		Adapter: astraSetAdapterPayload{
			ID:           adapter.ID,
			Name:         adapter.Name,
			Type:         adapter.Type,
			Adapter:      adapter.Adapter,
			Device:       adapter.Device,
			Frequency:    adapter.Frequency,
			Polarization: adapter.Polarization,
			Symbolrate:   adapter.Symbolrate,
			Bandwidth:    adapter.Bandwidth,
			Hierarchy:    adapter.Hierarchy,
			Modulation:   adapter.Modulation,
			Lof1:         adapter.Lof1,
			Lof2:         adapter.Lof2,
			Slof:         adapter.Slof,
			Enable:       adapter.Enable,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return AstraSaveAdapterResult{OK: false, Err: err}
	}

	_, err = astraControlRequest(ctx, conn, body)
	if err != nil {
		return AstraSaveAdapterResult{OK: false, Err: err}
	}

	return AstraSaveAdapterResult{OK: true}
}

func AstraDeleteAdapter(ctx context.Context, conn AstraConnection, id string) AstraDeleteItemResult {
	payload := struct {
		Cmd     string `json:"cmd"`
		ID      string `json:"id"`
		Adapter struct {
			Remove bool `json:"remove"`
		} `json:"adapter"`
	}{
		Cmd: "set-adapter",
		ID:  id,
	}

	payload.Adapter.Remove = true

	body, err := json.Marshal(payload)
	if err != nil {
		return AstraDeleteItemResult{OK: false, Err: err}
	}

	_, err = astraControlRequest(ctx, conn, body)
	if err != nil {
		return AstraDeleteItemResult{OK: false, Err: err}
	}

	return AstraDeleteItemResult{OK: true}
}

// ███████╗████████╗██████╗ ███████╗ █████╗ ███╗   ███╗
// ██╔════╝╚══██╔══╝██╔══██╗██╔════╝██╔══██╗████╗ ████║
// ███████╗   ██║   ██████╔╝█████╗  ███████║██╔████╔██║
// ╚════██║   ██║   ██╔══██╗██╔══╝  ██╔══██║██║╚██╔╝██║
// ███████║   ██║   ██║  ██║███████╗██║  ██║██║ ╚═╝ ██║
// ╚══════╝   ╚═╝   ╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝╚═╝     ╚═╝

type AstraSaveStreamResult struct {
	OK  bool
	Err error
}

type astraSetStreamPayload struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Type   string   `json:"type"`
	Enable bool     `json:"enable"`
	Input  []string `json:"input"`
	Output []string `json:"output,omitempty"`

	HbbtvURL string `json:"hbbtv_url"`

	SetPNR  string `json:"set_pnr,omitempty"`
	SetTSID string `json:"set_tsid,omitempty"`

	Map       string `json:"map,omitempty"`
	Filter    string `json:"filter,omitempty"`
	FilterNot string `json:"filter~,omitempty"`

	ServiceName     string `json:"service_name,omitempty"`
	ServiceProvider string `json:"service_provider,omitempty"`
}

func AstraSaveStream(ctx context.Context, conn AstraConnection, stream AstraStream) AstraSaveStreamResult {
	payload := struct {
		Cmd    string                `json:"cmd"`
		ID     string                `json:"id"`
		Stream astraSetStreamPayload `json:"stream"`
	}{
		Cmd: "set-stream",
		ID:  stream.ID,
		Stream: astraSetStreamPayload{
			ID:     stream.ID,
			Name:   stream.Name,
			Type:   stream.Type,
			Enable: stream.Enable,
			Input:  stream.Input,
			Output: stream.Output,

			HbbtvURL: stream.HbbtvURL,

			SetPNR:  stream.SetPNR,
			SetTSID: stream.SetTSID,

			Map:       stream.Map,
			Filter:    stream.Filter,
			FilterNot: stream.FilterNot,

			ServiceName:     stream.ServiceName,
			ServiceProvider: stream.ServiceProvider,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return AstraSaveStreamResult{OK: false, Err: err}
	}

	_, err = astraControlRequest(ctx, conn, body)
	if err != nil {
		return AstraSaveStreamResult{OK: false, Err: err}
	}

	return AstraSaveStreamResult{OK: true}
}

func AstraDeleteStream(ctx context.Context, conn AstraConnection, id string) AstraDeleteItemResult {
	payload := struct {
		Cmd    string `json:"cmd"`
		ID     string `json:"id"`
		Stream struct {
			Remove bool `json:"remove"`
		} `json:"stream"`
	}{
		Cmd: "set-stream",
		ID:  id,
	}

	payload.Stream.Remove = true

	body, err := json.Marshal(payload)
	if err != nil {
		return AstraDeleteItemResult{OK: false, Err: err}
	}

	_, err = astraControlRequest(ctx, conn, body)
	if err != nil {
		return AstraDeleteItemResult{OK: false, Err: err}
	}

	return AstraDeleteItemResult{OK: true}
}

func astraControlRequest(ctx context.Context, conn AstraConnection, body []byte) ([]byte, error) {
	url := fmt.Sprintf("http://%s:%d/control/", conn.Interface, conn.Port)

	reqCtx, cancel := context.WithTimeout(ctx, astraRequestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(conn.Login, conn.Password)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: astraRequestTimeout,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(raw))
	}

	return raw, nil
}
