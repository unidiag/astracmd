package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

type AstraScanAddStreamsResult struct {
	OK     bool
	ScanID string
	Count  int
	Err    error
}

type astraScanInitResponse struct {
	ScanInit string `json:"scan-init"`
	ID       string `json:"id"`
}

type astraScanCheckResponse struct {
	ScanCheck string           `json:"scan-check"`
	Scan      []astraScanTable `json:"scan"`
}

type astraScanTable struct {
	PSI      string              `json:"psi"`
	PNR      int                 `json:"pnr"`
	Services []astraScanService  `json:"services"`
	Streams  []astraScanESStream `json:"streams"`
}

type astraScanService struct {
	SID         int                   `json:"sid"`
	Descriptors []astraScanDescriptor `json:"descriptors"`
}

type astraScanDescriptor struct {
	TypeID          int    `json:"type_id"`
	TypeName        string `json:"type_name"`
	ServiceTypeID   int    `json:"service_type_id"`
	ServiceName     string `json:"service_name"`
	ServiceProvider string `json:"service_provider"`
}

type astraScanESStream struct {
	TypeID   int    `json:"type_id"`
	TypeName string `json:"type_name"`
	PID      int    `json:"pid"`
}

type astraScanAdapterPayload struct {
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
	Format       string `json:"format"`
}

func AstraScanAddStreams(
	ctx context.Context,
	conn AstraConnection,
	adapter AstraAdapter,
	existingStreams []AstraStream,
	checkDelay time.Duration,
) AstraScanAddStreamsResult {
	scanID, err := AstraScanInit(ctx, conn, adapter)
	if err != nil {
		return AstraScanAddStreamsResult{OK: false, Err: err}
	}

	if checkDelay <= 0 {
		checkDelay = 5 * time.Second
	}

	time.Sleep(checkDelay)

	scan, err := AstraScanCheck(ctx, conn, scanID)
	if err != nil {
		return AstraScanAddStreamsResult{OK: false, ScanID: scanID, Err: err}
	}

	streams := BuildStreamsFromScan(adapter, existingStreams, scan.Scan)
	if len(streams) == 0 {
		return AstraScanAddStreamsResult{
			OK:     false,
			ScanID: scanID,
			Err:    fmt.Errorf("no TV streams found in scan result"),
		}
	}

	count := 0

	for _, stream := range streams {
		result := AstraSaveStream(ctx, conn, stream)
		if !result.OK {
			if result.Err != nil {
				return AstraScanAddStreamsResult{
					OK:     false,
					ScanID: scanID,
					Count:  count,
					Err:    result.Err,
				}
			}

			return AstraScanAddStreamsResult{
				OK:     false,
				ScanID: scanID,
				Count:  count,
				Err:    fmt.Errorf("stream save failed: %s", stream.DisplayName()),
			}
		}

		count++
	}

	return AstraScanAddStreamsResult{
		OK:     true,
		ScanID: scanID,
		Count:  count,
	}
}

func AstraScanInit(ctx context.Context, conn AstraConnection, adapter AstraAdapter) (string, error) {
	payload := struct {
		Cmd  string                  `json:"cmd"`
		Scan astraScanAdapterPayload `json:"scan"`
	}{
		Cmd: "scan-init",
		Scan: astraScanAdapterPayload{
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
			Format:       "dvb",
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	raw, err := astraControlRequest(ctx, conn, body)
	if err != nil {
		return "", err
	}

	var out astraScanInitResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("parse scan-init response: %w; raw: %s", err, string(raw))
	}

	if strings.TrimSpace(out.ScanInit) != "ok" {
		return "", fmt.Errorf("unexpected scan-init response: %s", string(raw))
	}

	if strings.TrimSpace(out.ID) == "" {
		return "", fmt.Errorf("empty scan id in scan-init response")
	}

	return out.ID, nil
}

func AstraScanCheck(ctx context.Context, conn AstraConnection, scanID string) (astraScanCheckResponse, error) {
	payload := struct {
		Cmd string `json:"cmd"`
		ID  string `json:"id"`
	}{
		Cmd: "scan-check",
		ID:  scanID,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return astraScanCheckResponse{}, err
	}

	raw, err := astraControlRequest(ctx, conn, body)
	if err != nil {
		return astraScanCheckResponse{}, err
	}

	var out astraScanCheckResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return astraScanCheckResponse{}, fmt.Errorf("parse scan-check response: %w; raw: %s", err, string(raw))
	}

	if strings.TrimSpace(out.ScanCheck) != "ok" {
		return astraScanCheckResponse{}, fmt.Errorf("unexpected scan-check response: %s", string(raw))
	}

	if len(out.Scan) == 0 {
		return astraScanCheckResponse{}, fmt.Errorf("empty scan-check result")
	}

	return out, nil
}

func BuildStreamsFromScan(
	adapter AstraAdapter,
	existingStreams []AstraStream,
	tables []astraScanTable,
) []AstraStream {
	serviceNames := make(map[int]string)
	videoPNRs := make(map[int]bool)

	for _, table := range tables {
		switch strings.ToLower(strings.TrimSpace(table.PSI)) {
		case "sdt":
			for _, service := range table.Services {
				name := astraScanServiceName(service)
				if name != "" {
					serviceNames[service.SID] = name
				}
			}

		case "pmt":
			if table.PNR > 0 && astraScanHasVideo(table.Streams) {
				videoPNRs[table.PNR] = true
			}
		}
	}

	pnrs := make([]int, 0, len(videoPNRs))
	for pnr := range videoPNRs {
		pnrs = append(pnrs, pnr)
	}

	sort.Ints(pnrs)

	streams := make([]AstraStream, 0, len(pnrs))
	used := make([]AstraStream, 0, len(existingStreams))
	used = append(used, existingStreams...)

	for _, pnr := range pnrs {
		channelName := strings.TrimSpace(serviceNames[pnr])
		if channelName == "" {
			channelName = fmt.Sprintf("PNR %d", pnr)
		}

		streamID := dashboardGenerateStreamID(used)

		stream := AstraStream{
			ID:     streamID,
			Name:   fmt.Sprintf("[%d] %s", pnr, strings.TrimLeft(channelName, "_")),
			Type:   "spts",
			Enable: false,

			Input: []string{
				fmt.Sprintf("dvb://%s#pnr=%d", adapter.ID, pnr),
			},

			// Output is intentionally omitted. The stream is created disabled
			// and can be configured later by the user.
			Output: nil,

			// Use the same default HbbTV URL as in the new stream dialog.
			HbbtvURL: dHu,

			// Default remap for scanned streams.
			SetTSID:   "1",
			SetPNR:    "100",
			Map:       "video=101,audio=102",
			FilterNot: "101,102",

			ServiceName:     dashboardStreamServiceName(channelName),
			ServiceProvider: fmt.Sprintf("%s v.%s", APPNAME, VERSION),
		}

		streams = append(streams, stream)
		used = append(used, stream)
	}

	return streams
}

func astraScanServiceName(service astraScanService) string {
	for _, descriptor := range service.Descriptors {
		if descriptor.TypeID == 72 || strings.EqualFold(descriptor.TypeName, "service") {
			name := strings.TrimSpace(descriptor.ServiceName)
			if name != "" {
				return name
			}
		}
	}

	return ""
}

func astraScanHasVideo(streams []astraScanESStream) bool {
	for _, stream := range streams {
		typeName := strings.ToUpper(strings.TrimSpace(stream.TypeName))

		if typeName == "VIDEO" {
			return true
		}

		switch stream.TypeID {
		case 1, 2, 16, 27, 36:
			return true
		}
	}

	return false
}
