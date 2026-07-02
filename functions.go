package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type FunctionKeyAction struct {
	Label  string
	Handle func()
}

func (ui *UI) NewFunctionKeyBar(actions map[int]FunctionKeyAction) *tview.Table {
	table := tview.NewTable()
	table.SetSelectable(true, false)
	table.SetEvaluateAllRows(true)

	for i := 1; i <= 10; i++ {
		action, active := actions[i]

		label := ""
		if active {
			label = action.Label
		}

		buttonCol := (i - 1) * 2
		spaceCol := buttonCol + 1

		text := fmt.Sprintf(" F%d %-7s", i, label)

		cell := tview.NewTableCell(text).
			SetAlign(tview.AlignCenter).
			SetExpansion(1).
			SetSelectable(active)

		if active {
			if i == 2 {
				cell.SetTextColor(tcell.ColorWhite)
				cell.SetBackgroundColor(tcell.ColorRed)
				cell.SetSelectedStyle(
					tcell.StyleDefault.
						Foreground(tcell.ColorWhite).
						Background(tcell.ColorRed),
				)
			} else {
				cell.SetTextColor(tcell.ColorWhite)
				cell.SetBackgroundColor(tcell.ColorBlack)
			}
		} else {
			cell.SetTextColor(dashboardDisabledColor)
			cell.SetBackgroundColor(tcell.ColorBlack)
		}

		table.SetCell(0, buttonCol, cell)

		if i < 10 {
			spaceCell := tview.NewTableCell(" ").
				SetTextColor(tcell.ColorBlack).
				SetBackgroundColor(tcell.ColorBlack).
				SetExpansion(0).
				SetSelectable(false)

			table.SetCell(0, spaceCol, spaceCell)
		}
	}

	table.SetSelectedFunc(func(_ int, col int) {
		if col%2 != 0 {
			return
		}

		key := col/2 + 1

		action, ok := actions[key]
		if !ok || action.Handle == nil {
			return
		}

		action.Handle()
	})

	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if ui.HandleGlobalKeys(event) {
			return nil
		}

		if event.Key() == tcell.KeyF1 {
			if action, ok := actions[1]; ok && action.Handle != nil {
				action.Handle()
				return nil
			}
		}

		return event
	})

	return table
}

func formatUnixTime(ts int64) string {
	if ts <= 0 {
		return "unknown"
	}

	return time.Unix(ts, 0).Format("02.01.2006 15:04:05")
}

func isValidLicense(value string) bool {
	if len([]rune(value)) != 32 {
		return false
	}

	for _, r := range value {
		if r >= '0' && r <= '9' {
			continue
		}

		if r >= 'a' && r <= 'f' {
			continue
		}

		if r >= 'A' && r <= 'F' {
			continue
		}

		return false
	}

	return true
}

func formatAdapterCounter(value int) string {
	if value > 100 {
		return "99+"
	}

	return fmt.Sprintf("%d", value)
}

func normalizeAdapterSignal(value int) int {
	if value <= 100 {
		return value
	}

	if value <= 0 {
		return 0
	}

	return 65535 / value
}

var logChannelNameRe = regexp.MustCompile(`\[([^\]]+)\]`)
var logInputSuffixRe = regexp.MustCompile(`\s+[io]/\d+$`)

func normalizeLogChannelName(value string) string {
	return strings.TrimSpace(value)
}

func logItemMatchesStreamName(text string, streamName string) bool {
	streamName = normalizeLogChannelName(streamName)
	if streamName == "" {
		return false
	}

	if strings.Contains(text, "["+streamName+"]") {
		return true
	}

	if strings.Contains(text, "["+streamName+" i/") {
		return true
	}

	if strings.Contains(text, "["+streamName+" o/") {
		return true
	}

	return false
}

func logItemMatchesStream(text string, stream AstraStream) bool {
	return logItemMatchesStreamName(text, stream.DisplayName())
}

func FilterLogItemsByStreams(items []AstraLogItem, streams []AstraStream) []AstraLogItem {
	if len(streams) == 0 {
		return nil
	}

	result := make([]AstraLogItem, 0)

	for _, item := range items {
		for _, stream := range streams {
			if logItemMatchesStream(item.Text, stream) {
				result = append(result, item)
				break
			}
		}
	}

	return result
}

func FilterLogItemsByStream(items []AstraLogItem, stream AstraStream) []AstraLogItem {
	result := make([]AstraLogItem, 0)

	for _, item := range items {
		if logItemMatchesStream(item.Text, stream) {
			result = append(result, item)
		}
	}

	return result
}

var logDVBInputRe = regexp.MustCompile(`^dvb_input\s+(\d+):\d+$`)

func extractLogBracketValue(text string) string {
	match := logChannelNameRe.FindStringSubmatch(text)
	if len(match) < 2 {
		return ""
	}

	return strings.TrimSpace(match[1])
}

func isAdapterLogItem(item AstraLogItem, adapterNumber int) bool {
	match := logDVBInputRe.FindStringSubmatch(item.Text)
	if len(match) < 2 {
		return false
	}

	n, err := strconv.Atoi(match[1])
	if err != nil {
		return false
	}

	return n == adapterNumber
}

func dashboardStreamServiceName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}

	if dashboardContainsCyrillic(name) {
		return dashboardTranslitCyrillic(name)
	}

	return name
}

func dashboardContainsCyrillic(value string) bool {
	for _, r := range value {
		if unicode.In(r, unicode.Cyrillic) {
			return true
		}
	}

	return false
}

func dashboardTranslitCyrillic(value string) string {
	replacer := strings.NewReplacer(
		"А", "A",
		"Б", "B",
		"В", "V",
		"Г", "G",
		"Д", "D",
		"Е", "E",
		"Ё", "Yo",
		"Ж", "Zh",
		"З", "Z",
		"И", "I",
		"Й", "Y",
		"К", "K",
		"Л", "L",
		"М", "M",
		"Н", "N",
		"О", "O",
		"П", "P",
		"Р", "R",
		"С", "S",
		"Т", "T",
		"У", "U",
		"Ф", "F",
		"Х", "Kh",
		"Ц", "Ts",
		"Ч", "Ch",
		"Ш", "Sh",
		"Щ", "Sch",
		"Ъ", "",
		"Ы", "Y",
		"Ь", "",
		"Э", "E",
		"Ю", "Yu",
		"Я", "Ya",

		"а", "a",
		"б", "b",
		"в", "v",
		"г", "g",
		"д", "d",
		"е", "e",
		"ё", "yo",
		"ж", "zh",
		"з", "z",
		"и", "i",
		"й", "y",
		"к", "k",
		"л", "l",
		"м", "m",
		"н", "n",
		"о", "o",
		"п", "p",
		"р", "r",
		"с", "s",
		"т", "t",
		"у", "u",
		"ф", "f",
		"х", "kh",
		"ц", "ts",
		"ч", "ch",
		"ш", "sh",
		"щ", "sch",
		"ъ", "",
		"ы", "y",
		"ь", "",
		"э", "e",
		"ю", "yu",
		"я", "ya",
	)

	return replacer.Replace(value)
}
