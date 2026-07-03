package main

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var dvbInputRe = regexp.MustCompile(`^dvb://([^#?]+)`)

type AstraStreamState struct {
	Bitrate   int
	Onair     bool
	CCError   int
	PESError  int
	Scrambled bool
	InputID   int
}

type AstraAdapterState struct {
	Signal   int
	SignalDB int
	Bitrate  int
	UNC      int
	SNRDB    int
	SNR      int
	BER      int
	Status   int
}

type AstraConfig struct {
	GID      int64          `json:"gid"`
	Streams  []AstraStream  `json:"make_stream"`
	Adapters []AstraAdapter `json:"dvb_tune"`
	Softcams []AstraSoftcam `json:"softcam"`
}

type AstraStream struct {
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

type AstraAdapter struct {
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
	Lof1         string `json:"lof1"`
	Lof2         string `json:"lof2"`
	Slof         string `json:"slof"`
	Enable       bool   `json:"enable"`
}

type AstraSoftcam struct {
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

func (c AstraSoftcam) DisplayName() string {
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

func BuildAdapterStreamMap(cfg AstraConfig) map[string][]AstraStream {
	result := make(map[string][]AstraStream)

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

func (s AstraStream) FirstAdapterID() string {
	for _, input := range s.Input {
		m := dvbInputRe.FindStringSubmatch(input)
		if len(m) == 2 {
			return strings.TrimSpace(m[1])
		}
	}

	return ""
}

func (a AstraAdapter) DisplayName() string {
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

func (a AstraAdapter) DisplayTransponder() string {
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

func (s AstraStream) DisplayName() string {
	name := strings.TrimSpace(s.Name)
	if name == "" {
		name = s.ID
	}

	return name
}

func NewDashboardTable(title string) *tview.Table {
	table := tview.NewTable()
	table.SetBorder(true)
	table.SetTitle(" " + title + " ")
	table.SetTitleAlign(tview.AlignCenter)
	table.SetSelectable(true, false)
	table.SetEvaluateAllRows(true)

	return table
}

func FillAdaptersTable(
	table *tview.Table,
	adapters []AstraAdapter,
	states map[string]AstraAdapterState,
	dimmed bool,
) {
	table.Clear()

	specialColor := tcell.ColorWhite
	if dimmed {
		specialColor = tcell.ColorDarkGray
	}

	table.SetCell(0, 0,
		tview.NewTableCell("ALL").
			SetTextColor(specialColor).
			SetAlign(tview.AlignCenter).
			SetExpansion(1),
	)

	table.SetCell(1, 0,
		tview.NewTableCell("OUTSIDE").
			SetTextColor(specialColor).
			SetAlign(tview.AlignCenter).
			SetExpansion(1),
	)

	sort.SliceStable(adapters, func(i, j int) bool {
		if adapters[i].Adapter == adapters[j].Adapter {
			return adapters[i].Name < adapters[j].Name
		}

		return adapters[i].Adapter < adapters[j].Adapter
	})

	for i, adapter := range adapters {
		row := 2 + i*2

		color := tcell.ColorWhite
		infoColor := dashboardDisabledColor

		if !adapter.Enable {
			color = dashboardDisabledColor
			infoColor = tcell.ColorDarkGray
		}

		if dimmed {
			color = tcell.ColorDarkGray
			infoColor = tcell.ColorDarkGray
		}

		text := fmt.Sprintf("%d. %s", i+1, adapter.DisplayName())

		table.SetCell(row, 0,
			tview.NewTableCell(text).
				SetTextColor(color).
				SetExpansion(1),
		)

		state, ok := states[adapter.ID]

		infoText := "   BER:- UNC:- S:- Q:- - Mbps"
		if ok {
			infoText = fmt.Sprintf(
				"   BER:%s UNC:%s S:%d%% Q:%d%% %.1f Mbps",
				formatAdapterCounter(state.BER),
				formatAdapterCounter(state.UNC),
				normalizeAdapterSignal(state.Signal),
				normalizeAdapterSignal(state.SNR),
				float64(state.Bitrate)/1000.0,
			)

			if state.BER > 0 || state.UNC > 0 {
				infoColor = tcell.ColorYellow
			}

			if state.BER > 100 || state.UNC > 100 {
				infoColor = tcell.ColorRed
			}

			if state.Bitrate <= 0 {
				infoColor = dashboardDisabledColor
			}
		}

		if dimmed {
			infoColor = dashboardDisabledColor
		}

		table.SetCell(row+1, 0,
			tview.NewTableCell(infoText).
				SetTextColor(infoColor).
				SetSelectable(false).
				SetExpansion(1),
		)
	}
}

func FilterLogItemsByAdapter(
	items []AstraLogItem,
	streams []AstraStream,
	adapterNumber int,
) []AstraLogItem {
	result := make([]AstraLogItem, 0)

	for _, item := range items {
		if isAdapterLogItem(item, adapterNumber) {
			result = append(result, item)
			continue
		}

		for _, stream := range streams {
			if logItemMatchesStream(item.Text, stream) {
				result = append(result, item)
				break
			}
		}
	}

	return result
}

func FillStreamsTable(
	table *tview.Table,
	streams []AstraStream,
	states map[string]AstraStreamState,
	dimmed bool,
) {
	table.Clear()

	sort.SliceStable(streams, func(i, j int) bool {
		return strings.ToLower(streams[i].DisplayName()) < strings.ToLower(streams[j].DisplayName())
	})

	for row, stream := range streams {
		color := tcell.ColorWhite

		if !stream.Enable {
			color = dashboardDisabledColor
		}

		if dimmed {
			color = dashboardDisabledColor
		}

		nameText := fmt.Sprintf("%d. %s", row+1, stream.DisplayName())

		table.SetCell(row, 0,
			tview.NewTableCell(nameText).
				SetTextColor(color).
				SetExpansion(1),
		)

		state, ok := states[stream.ID]

		errorText := ""
		errorColor := tcell.ColorYellow

		bitrateText := "-"
		bitrateColor := dashboardDisabledColor

		if ok {
			if state.CCError > 0 && state.PESError > 0 {
				errorText = fmt.Sprintf("CC:%d PES:%d", state.CCError, state.PESError)
			} else if state.CCError > 0 {
				errorText = fmt.Sprintf("CC:%d", state.CCError)
			} else if state.PESError > 0 {
				errorText = fmt.Sprintf("PES:%d", state.PESError)
			}

			bitrateText = fmt.Sprintf("%d kbps", state.Bitrate)

			if state.Onair && state.Bitrate > 0 {
				bitrateColor = tcell.ColorGreen
			} else {
				bitrateColor = tcell.ColorRed
			}

			if state.Scrambled {
				bitrateColor = tcell.ColorOrange
			}
		}

		if dimmed {
			errorColor = tcell.ColorDarkGray
			bitrateColor = tcell.ColorDarkGray
		}

		table.SetCell(row, 1,
			tview.NewTableCell(errorText).
				SetTextColor(errorColor).
				SetAlign(tview.AlignCenter).
				SetExpansion(0),
		)

		table.SetCell(row, 2,
			tview.NewTableCell(bitrateText).
				SetTextColor(bitrateColor).
				SetAlign(tview.AlignRight).
				SetExpansion(0),
		)
	}
}

func FillLogTable(table *tview.Table, items []AstraLogItem, dimmed bool, maxRows int) {
	table.Clear()

	if maxRows <= 0 {
		return
	}

	color := tcell.ColorWhite
	if dimmed {
		color = tcell.ColorDarkGray
	}

	start := 0
	if len(items) > maxRows {
		start = len(items) - maxRows
	}

	row := 0

	for _, item := range items[start:] {
		itemColor := color

		switch item.Type {
		case 1:
			itemColor = tcell.ColorYellow
		case 2:
			itemColor = tcell.ColorRed
		case 3:
			itemColor = dashboardDisabledColor
		}

		if dimmed {
			itemColor = tcell.ColorDarkGray
		}

		text := fmt.Sprintf(
			"%s %s",
			time.Unix(item.Time, 0).Format("15:04:05"),
			item.Text,
		)

		table.SetCell(row, 0,
			tview.NewTableCell(text).
				SetTextColor(itemColor).
				SetExpansion(1),
		)

		row++
	}
}
