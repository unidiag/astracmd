package astra

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type AstraSaveSoftcamResult struct {
	OK  bool
	Err error
}

type astraSetSoftcamPayload struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Host       string `json:"host"`
	Port       string `json:"port"`
	User       string `json:"user"`
	Pass       string `json:"pass"`
	Key        string `json:"key,omitempty"`
	DisableEMM bool   `json:"disable_emm"`
}

type astraRemoveSoftcamPayload struct {
	Remove bool `json:"remove"`
}

type AstraTestSoftcamResult struct {
	OK       bool
	Status   int
	CAID     int
	ECMRate  int
	EMMRate  int
	AU       bool
	UA       string
	Idents   []AstraTestSoftcamIdent
	ErrorMsg string
	Err      error
}

type AstraTestSoftcamIdent struct {
	SA string `json:"sa"`
	ID string `json:"id"`
}

type astraTestSoftcamResponse struct {
	Status  int    `json:"status"`
	Error   string `json:"error"`
	CAID    int    `json:"caid"`
	ECMRate int    `json:"ecm_rate"`
	EMMRate int    `json:"emm_rate"`
	Info    struct {
		Idents []AstraTestSoftcamIdent `json:"idents"`
		CAID   int                     `json:"caid"`
		AU     bool                    `json:"au"`
		UA     string                  `json:"ua"`
	} `json:"info"`
}

func AstraTestSoftcam(
	ctx context.Context,
	conn Connection,
	softcam Softcam,
) AstraTestSoftcamResult {
	softcamID := strings.TrimSpace(softcam.ID)
	if softcamID == "" {
		return AstraTestSoftcamResult{
			OK:       false,
			ErrorMsg: "softcam id is empty",
		}
	}

	payload := struct {
		Cmd    string  `json:"cmd"`
		Config Softcam `json:"config"`
	}{
		Cmd:    "test-softcam",
		Config: softcam,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return AstraTestSoftcamResult{OK: false, Err: err}
	}

	responseBody, err := astraControlRequest(ctx, conn, body)
	if err != nil {
		return AstraTestSoftcamResult{OK: false, Err: err}
	}

	var response astraTestSoftcamResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return AstraTestSoftcamResult{OK: false, Err: err}
	}

	if response.Status == 0 {
		errorMsg := strings.TrimSpace(response.Error)
		if errorMsg == "" {
			errorMsg = "SoftCAM test failed"
		}

		return AstraTestSoftcamResult{
			OK:       false,
			Status:   response.Status,
			ErrorMsg: errorMsg,
		}
	}

	caid := response.CAID
	if caid == 0 {
		caid = response.Info.CAID
	}

	return AstraTestSoftcamResult{
		OK:      true,
		Status:  response.Status,
		CAID:    caid,
		ECMRate: response.ECMRate,
		EMMRate: response.EMMRate,
		AU:      response.Info.AU,
		UA:      response.Info.UA,
		Idents:  response.Info.Idents,
	}
}

func AstraSaveSoftcam(
	ctx context.Context,
	conn Connection,
	gid int64,
	softcam Softcam,
) AstraSaveSoftcamResult {
	if gid == 0 {
		return AstraSaveSoftcamResult{
			OK:  false,
			Err: fmt.Errorf("config gid is empty"),
		}
	}

	softcamID := strings.TrimSpace(softcam.ID)
	if softcamID == "" {
		return AstraSaveSoftcamResult{
			OK:  false,
			Err: fmt.Errorf("softcam id is empty"),
		}
	}

	payload := struct {
		Cmd     string                 `json:"cmd"`
		GID     int64                  `json:"gid"`
		ID      string                 `json:"id"`
		Softcam astraSetSoftcamPayload `json:"softcam"`
	}{
		Cmd: "set-softcam",
		GID: gid,
		ID:  softcamID,
		Softcam: astraSetSoftcamPayload{
			ID:         softcamID,
			Name:       strings.TrimSpace(softcam.Name),
			Type:       strings.TrimSpace(softcam.Type),
			Host:       strings.TrimSpace(softcam.Host),
			Port:       strings.TrimSpace(softcam.Port),
			User:       strings.TrimSpace(softcam.User),
			Pass:       strings.TrimSpace(softcam.Pass),
			Key:        strings.TrimSpace(softcam.Key),
			DisableEMM: softcam.DisableEMM,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return AstraSaveSoftcamResult{OK: false, Err: err}
	}

	_, err = astraControlRequest(ctx, conn, body)
	if err != nil {
		return AstraSaveSoftcamResult{OK: false, Err: err}
	}

	return AstraSaveSoftcamResult{OK: true}
}

func AstraRemoveSoftcam(
	ctx context.Context,
	conn Connection,
	gid int64,
	softcamID string,
) AstraSaveSoftcamResult {
	if gid == 0 {
		return AstraSaveSoftcamResult{
			OK:  false,
			Err: fmt.Errorf("config gid is empty"),
		}
	}

	softcamID = strings.TrimSpace(softcamID)
	if softcamID == "" {
		return AstraSaveSoftcamResult{
			OK:  false,
			Err: fmt.Errorf("softcam id is empty"),
		}
	}

	payload := struct {
		Cmd     string                    `json:"cmd"`
		GID     int64                     `json:"gid"`
		ID      string                    `json:"id"`
		Softcam astraRemoveSoftcamPayload `json:"softcam"`
	}{
		Cmd: "set-softcam",
		GID: gid,
		ID:  softcamID,
		Softcam: astraRemoveSoftcamPayload{
			Remove: true,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return AstraSaveSoftcamResult{OK: false, Err: err}
	}

	_, err = astraControlRequest(ctx, conn, body)
	if err != nil {
		return AstraSaveSoftcamResult{OK: false, Err: err}
	}

	return AstraSaveSoftcamResult{OK: true}
}
