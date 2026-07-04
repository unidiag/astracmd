package astra

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var dvbInputRe = regexp.MustCompile(`^dvb://([^#?]+)`)

type StreamState struct {
	Bitrate   int
	Onair     bool
	CCError   int
	PESError  int
	Scrambled bool
	InputID   int
}

type AdapterState struct {
	Signal   int
	SignalDB int
	Bitrate  int
	UNC      int
	SNRDB    int
	SNR      int
	BER      int
	Status   int
}

type Config struct {
	GID      int64     `json:"gid"`
	Streams  []Stream  `json:"make_stream"`
	Adapters []Adapter `json:"dvb_tune"`
	Softcams []Softcam `json:"softcam"`
}

type Stream struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Type   string   `json:"type"`
	Enable bool     `json:"enable"`
	Input  []string `json:"input"`
	Output []string `json:"output"`

	HbbtvURL string `json:"hbbtv_url"`

	SetPNR  string `json:"set_pnr"`
	SetTSID string `json:"set_tsid"`

	Map       string `json:"map"`
	Filter    string `json:"filter"`
	FilterNot string `json:"filter~"`

	ServiceName     string `json:"service_name"`
	ServiceProvider string `json:"service_provider"`
}

type Adapter struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Type         string `json:"type"`
	Adapter      int    `json:"adapter"`
	Device       int    `json:"device"`
	Frequency    string `json:"frequency"`
	Polarization string `json:"polarization"`
	Symbolrate   string `json:"symbolrate"`
	Bandwidth    string `json:"bandwidth"`
	Hierarchy    string `json:"hierarchy"`
	Modulation   string `json:"modulation"`
	DiseqcMode   string `json:"diseqc_mode,omitempty"` // "cmd"
	Diseqc       string `json:"diseqc,omitempty"`      // "t W50 [E0 31 6E E3 13] W30 T"  75 deg
	Lof1         string `json:"lof1"`
	Lof2         string `json:"lof2"`
	Slof         string `json:"slof"`
	Enable       bool   `json:"enable"`
}

type Softcam struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Host       string `json:"host"`
	Port       string `json:"port"`
	User       string `json:"user"`
	Pass       string `json:"pass"`
	Key        string `json:"key,omitempty"`
	DisableEMM bool   `json:"disable_emm,omitempty"`
}

func (c Softcam) DisplayName() string {
	name := strings.TrimSpace(c.Name)
	if name != "" {
		return name
	}

	id := strings.TrimSpace(c.ID)
	if id != "" {
		return id
	}

	return "New SoftCAM"
}

func BuildAdapterStreamMap(cfg Config) map[string][]Stream {
	result := make(map[string][]Stream)

	for _, stream := range cfg.Streams {
		adapterID := stream.FirstAdapterID()
		if adapterID == "" {
			result[""] = append(result[""], stream)
			continue
		}

		result[adapterID] = append(result[adapterID], stream)
	}

	for key := range result {
		sort.SliceStable(result[key], func(i, j int) bool {
			return strings.ToLower(result[key][i].Name) < strings.ToLower(result[key][j].Name)
		})
	}

	return result
}

func (s Stream) FirstAdapterID() string {
	for _, input := range s.Input {
		m := dvbInputRe.FindStringSubmatch(input)
		if len(m) == 2 {
			return strings.TrimSpace(m[1])
		}
	}

	return ""
}

func (a Adapter) DisplayName() string {
	name := strings.TrimSpace(a.Name)
	if name == "" {
		name = strings.TrimSpace(a.ID)
	}

	tp := a.DisplayTransponder()
	if tp == "" {
		return fmt.Sprintf("%s #%d", name, a.Adapter)
	}

	return fmt.Sprintf("%s #%d %s", name, a.Adapter, tp)
}

func (a Adapter) DisplayTransponder() string {
	tpType := strings.ToUpper(strings.TrimSpace(a.Type))
	frequency := strings.TrimSpace(a.Frequency)
	polarization := strings.ToUpper(strings.TrimSpace(a.Polarization))
	symbolrate := strings.TrimSpace(a.Symbolrate)

	if frequency == "" {
		return ""
	}

	switch tpType {
	case "S", "S2":
		if polarization == "" && symbolrate == "" {
			return frequency
		}

		return fmt.Sprintf("%s:%s:%s:%s", tpType, frequency, polarization, symbolrate)

	case "T", "T2":
		return fmt.Sprintf("%s:%s", tpType, frequency)

	case "C":
		if symbolrate == "" {
			return fmt.Sprintf("%s:%s", tpType, frequency)
		}

		return fmt.Sprintf("%s:%s:%s", tpType, frequency, symbolrate)

	default:
		return frequency
	}
}

func (s Stream) DisplayName() string {
	name := strings.TrimSpace(s.Name)
	if name == "" {
		name = s.ID
	}

	return name
}
