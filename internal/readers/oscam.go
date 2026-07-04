package readers

import (
	"os"
	"path/filepath"
	"strings"
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

func readTailLines(path string, maxBytes int64, maxLines int) (string, error) {
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

	if len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}

	return strings.Join(lines, "\n"), nil
}
