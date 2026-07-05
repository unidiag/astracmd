package readers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const readerSectionPrefix = "reader:"

func loadReaderSections(path string) (map[string]ReaderSection, error) {
	out := make(map[string]ReaderSection)

	if strings.TrimSpace(path) == "" {
		return out, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return out, err
	}

	lines := strings.Split(string(data), "\n")

	var current *ReaderSection

	flush := func() {
		if current == nil {
			return
		}

		current.SerialByID = normalizeSerialByID(current.SerialByID)
		current.Name = strings.TrimSpace(current.Name)
		current.OscamBin = strings.TrimSpace(current.OscamBin)
		current.OscamDir = strings.TrimSpace(current.OscamDir)

		if current.SerialByID != "" {
			if current.OscamBin == "" {
				current.OscamBin = "oscam"
			}

			out[current.SerialByID] = *current
		}
	}

	for _, raw := range lines {
		line := strings.TrimSpace(raw)

		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			flush()

			section := strings.TrimSpace(strings.Trim(line, "[]"))

			if strings.HasPrefix(section, readerSectionPrefix) {
				serial := strings.TrimPrefix(section, readerSectionPrefix)

				current = &ReaderSection{
					Section:    section,
					Enabled:    true,
					SerialByID: normalizeSerialByID(serial),
					OscamBin:   "oscam",
				}
			} else {
				current = nil
			}

			continue
		}

		if current == nil {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		value = cleanINIValue(value)

		switch strings.ToLower(key) {
		case "enabled":
			current.Enabled = readBoolINI(value, true)

		case "name":
			current.Name = value

		case "oscam_bin":
			current.OscamBin = value

		case "oscam_dir":
			current.OscamDir = value
		}
	}

	flush()

	return out, nil
}

func saveReaderSection(path string, section ReaderSection) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("config path is empty")
	}

	section.SerialByID = normalizeSerialByID(section.SerialByID)
	section.Name = strings.TrimSpace(section.Name)
	section.OscamBin = strings.TrimSpace(section.OscamBin)
	section.OscamDir = strings.TrimSpace(section.OscamDir)

	if section.SerialByID == "" {
		return fmt.Errorf("serial_by_id is required")
	}

	if section.OscamBin == "" {
		section.OscamBin = "oscam"
	}

	section.Section = readerSectionPrefix + section.SerialByID

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")

	start := -1
	end := len(lines)

	for i, raw := range lines {
		line := strings.TrimSpace(raw)

		if !strings.HasPrefix(line, "[") || !strings.HasSuffix(line, "]") {
			continue
		}

		name := strings.TrimSpace(strings.Trim(line, "[]"))

		if start >= 0 {
			end = i
			break
		}

		if name == section.Section {
			start = i
			continue
		}

	}

	block := []string{
		"[" + section.Section + "]",
		fmt.Sprintf("enabled=%t", section.Enabled),
		"name=" + section.Name,
		"oscam_bin=" + section.OscamBin,
		"oscam_dir=" + section.OscamDir,
	}

	var updated []string

	if start >= 0 {
		updated = append(updated, lines[:start]...)
		updated = append(updated, block...)
		updated = append(updated, lines[end:]...)
	} else {
		updated = append(updated, lines...)

		if len(updated) > 0 && strings.TrimSpace(updated[len(updated)-1]) != "" {
			updated = append(updated, "")
		}

		updated = append(updated, block...)
	}

	return os.WriteFile(path, []byte(strings.Join(updated, "\n")), 0644)
}

func cleanINIValue(value string) string {
	value = strings.TrimSpace(value)

	if idx := strings.Index(value, "#"); idx >= 0 {
		value = strings.TrimSpace(value[:idx])
	}

	if idx := strings.Index(value, ";"); idx >= 0 {
		value = strings.TrimSpace(value[:idx])
	}

	return strings.Trim(value, `"'`)
}

func normalizeSerialByID(value string) string {
	value = strings.TrimSpace(value)

	if value == "" {
		return ""
	}

	value = strings.TrimPrefix(value, serialByIDPath+"/")

	if strings.Contains(value, "/") {
		value = filepath.Base(value)
	}

	return value
}

func applyReaderSections(devices []Device, sections map[string]ReaderSection) {
	for i := range devices {
		section, ok := sections[devices[i].Name]
		if !ok {
			continue
		}

		if strings.TrimSpace(section.Name) != "" {
			devices[i].DisplayName = section.Name
		}
	}
}

func readBoolINI(value string, defaultValue bool) bool {
	value = strings.ToLower(strings.TrimSpace(value))

	switch value {
	case "1", "true", "yes", "on", "enabled":
		return true

	case "0", "false", "no", "off", "disabled":
		return false
	}

	return defaultValue
}
