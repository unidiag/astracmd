package readers

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/rivo/tview"
)

var (
	oscamLogDatePrefixRe = regexp.MustCompile(`^\d{4}[/-]\d{2}[/-]\d{2}\s+`)
	oscamLogThreadIDRe   = regexp.MustCompile(`^(\d{2}:\d{2}:\d{2})\s+[0-9A-Fa-f]{8}\s+`)
	oscamLogClientFlagRe = regexp.MustCompile(`^(\d{2}:\d{2}:\d{2})\s+c\s+`)
	oscamLogECMTimeRe    = regexp.MustCompile(`\((\d+)\s*ms\)`)
	oscamLogCacheRe      = regexp.MustCompile(`\scache\d+\s`)
)

func oscamConfigPathFromCommand(command string) string {
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return ""
	}

	for i := 0; i < len(fields); i++ {
		field := strings.TrimSpace(fields[i])

		switch field {
		case "-c", "--config-dir":
			if i+1 < len(fields) {
				return filepath.Join(fields[i+1], "oscam.conf")
			}
		}

		if strings.HasPrefix(field, "-c") && len(field) > 2 {
			return filepath.Join(strings.TrimPrefix(field, "-c"), "oscam.conf")
		}

		if strings.HasPrefix(field, "--config-dir=") {
			return filepath.Join(strings.TrimPrefix(field, "--config-dir="), "oscam.conf")
		}
	}

	return ""
}

func oscamLogPathFromConfig(configText string, configDir string) string {
	lines := strings.Split(configText, "\n")
	inGlobal := false

	for _, line := range lines {
		clean := strings.TrimSpace(line)
		if clean == "" || strings.HasPrefix(clean, "#") || strings.HasPrefix(clean, ";") {
			continue
		}

		if strings.HasPrefix(clean, "[") && strings.HasSuffix(clean, "]") {
			section := strings.ToLower(strings.Trim(clean, "[] "))
			inGlobal = section == "global"
			continue
		}

		if !inGlobal {
			continue
		}

		key, value, ok := strings.Cut(clean, "=")
		if !ok {
			continue
		}

		if strings.EqualFold(strings.TrimSpace(key), "logfile") {
			logPath := strings.TrimSpace(value)
			if logPath == "" || strings.EqualFold(logPath, "stdout") {
				return ""
			}

			if filepath.IsAbs(logPath) {
				return logPath
			}

			return filepath.Join(configDir, logPath)
		}
	}

	return ""
}

func readTailLines(path string, maxBytes int64, maxLines int, filterText string) (string, error) {
	if maxLines <= 0 {
		maxLines = 1
	}

	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return "", err
	}

	size := info.Size()
	offset := int64(0)

	if size > maxBytes {
		offset = size - maxBytes
	}

	if _, err := file.Seek(offset, 0); err != nil {
		return "", err
	}

	data := make([]byte, size-offset)

	n, err := file.Read(data)
	if err != nil {
		return "", err
	}

	text := string(data[:n])
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	text = strings.TrimRight(text, "\n")

	if text == "" {
		return "", nil
	}

	lines := strings.Split(text, "\n")

	if offset > 0 && len(lines) > 0 {
		lines = lines[1:]
	}

	filterText = strings.TrimSpace(filterText)
	filterLower := strings.ToLower(filterText)

	prepared := make([]string, 0, len(lines))

	for _, line := range lines {
		line = cleanLogLine(line)

		if line == "" {
			continue
		}

		if filterLower != "" && !strings.Contains(strings.ToLower(line), filterLower) {
			continue
		}

		prepared = append(prepared, colorOscamLogLine(line, filterText))
	}

	if len(prepared) > maxLines {
		prepared = prepared[len(prepared)-maxLines:]
	}

	return strings.Join(prepared, "\n"), nil
}

func cleanLogLine(line string) string {
	line = strings.ReplaceAll(line, "\t", " ")
	line = strings.Join(strings.Fields(line), " ")

	line = oscamLogDatePrefixRe.ReplaceAllString(line, "")
	line = oscamLogThreadIDRe.ReplaceAllString(line, "$1 ")
	line = oscamLogClientFlagRe.ReplaceAllString(line, "$1 ")

	return line
}

func colorOscamLogLine(line string, filterText string) string {
	color := oscamLogLineColor(line)
	filterText = strings.TrimSpace(filterText)

	if filterText == "" {
		line = tview.Escape(line)

		if color == "" {
			return line
		}

		return fmt.Sprintf("[%s]%s[-:-:-]", color, line)
	}

	return highlightLogSubstring(line, filterText, color)
}

func oscamLogLineColor(line string) string {
	switch {
	case strings.Contains(line, " not found "):
		return "red"

	case oscamLogCacheRe.MatchString(line):
		return "gray"

	case strings.Contains(line, " found "):
		matches := oscamLogECMTimeRe.FindStringSubmatch(line)

		if len(matches) != 2 {
			return ""
		}

		ms, err := strconv.Atoi(matches[1])
		if err != nil {
			return ""
		}

		switch {
		case ms > 2500:
			return "red"
		case ms > 1000:
			return "yellow"
		default:
			return "green"
		}
	}

	return ""
}

func highlightLogSubstring(line string, filterText string, baseColor string) string {
	filterText = strings.TrimSpace(filterText)

	if filterText == "" {
		return tview.Escape(line)
	}

	lineLower := strings.ToLower(line)
	filterLower := strings.ToLower(filterText)

	var b strings.Builder

	pos := 0
	filterLen := len(filterText)

	writeNormal := func(text string) {
		if text == "" {
			return
		}

		text = tview.Escape(text)

		if baseColor == "" {
			b.WriteString(text)
			return
		}

		b.WriteString("[")
		b.WriteString(baseColor)
		b.WriteString("]")
		b.WriteString(text)
		b.WriteString("[-:-:-]")
	}

	writeMatch := func(text string) {
		if text == "" {
			return
		}

		b.WriteString("[black:white:b]")
		b.WriteString(tview.Escape(text))
		b.WriteString("[-:-:-]")
	}

	for {
		index := strings.Index(lineLower[pos:], filterLower)
		if index < 0 {
			break
		}

		start := pos + index
		end := start + filterLen

		writeNormal(line[pos:start])
		writeMatch(line[start:end])

		pos = end
	}

	writeNormal(line[pos:])

	return b.String()
}
