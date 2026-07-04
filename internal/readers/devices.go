package readers

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func ListDevices() ([]Device, error) {
	entries, err := os.ReadDir(serialByIDPath)
	if err != nil {
		return nil, err
	}

	devices := make([]Device, 0, len(entries))

	for _, entry := range entries {
		name := entry.Name()
		path := filepath.Join(serialByIDPath, name)

		target := ""

		linkTarget, err := os.Readlink(path)
		if err == nil {
			target = filepath.Clean(filepath.Join(serialByIDPath, linkTarget))
		} else {
			target = err.Error()
		}

		processName, processCmd, processPID, busy := serialDeviceOwner(path)

		devices = append(devices, Device{
			Name:        name,
			Path:        path,
			Target:      target,
			Busy:        busy,
			ProcessPID:  processPID,
			ProcessName: processName,
			ProcessCmd:  processCmd,
		})
	}

	sort.Slice(devices, func(i, j int) bool {
		return devices[i].Name < devices[j].Name
	})

	return devices, nil
}

func serialDeviceOwner(devicePath string) (string, string, int, bool) {
	resolvedDevice, err := filepath.EvalSymlinks(devicePath)
	if err != nil {
		return "", "", 0, false
	}

	resolvedDevice = filepath.Clean(resolvedDevice)

	procEntries, err := os.ReadDir("/proc")
	if err != nil {
		return "", "", 0, false
	}

	for _, procEntry := range procEntries {
		if !procEntry.IsDir() {
			continue
		}

		pid, err := strconv.Atoi(procEntry.Name())
		if err != nil {
			continue
		}

		fdDir := filepath.Join("/proc", procEntry.Name(), "fd")

		fdEntries, err := os.ReadDir(fdDir)
		if err != nil {
			continue
		}

		for _, fdEntry := range fdEntries {
			fdPath := filepath.Join(fdDir, fdEntry.Name())

			fdTarget, err := os.Readlink(fdPath)
			if err != nil {
				continue
			}

			resolvedTarget, err := filepath.EvalSymlinks(fdTarget)
			if err != nil {
				continue
			}

			if filepath.Clean(resolvedTarget) == resolvedDevice {
				return processNameByPID(pid), processCommandLineByPID(pid), pid, true
			}
		}
	}

	return "", "", 0, false
}

func processNameByPID(pid int) string {
	commPath := filepath.Join("/proc", strconv.Itoa(pid), "comm")

	data, err := os.ReadFile(commPath)
	if err == nil {
		name := strings.TrimSpace(string(data))
		if name != "" {
			return name
		}
	}

	cmdlinePath := filepath.Join("/proc", strconv.Itoa(pid), "cmdline")

	data, err = os.ReadFile(cmdlinePath)
	if err == nil {
		parts := strings.Split(string(data), "\x00")
		if len(parts) > 0 {
			name := strings.TrimSpace(filepath.Base(parts[0]))
			if name != "" {
				return name
			}
		}
	}

	return "unknown"
}

func processCommandLineByPID(pid int) string {
	cmdlinePath := filepath.Join("/proc", strconv.Itoa(pid), "cmdline")

	data, err := os.ReadFile(cmdlinePath)
	if err != nil {
		return ""
	}

	raw := strings.TrimRight(string(data), "\x00")
	if raw == "" {
		return ""
	}

	parts := strings.Split(raw, "\x00")
	cleanParts := make([]string, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			cleanParts = append(cleanParts, part)
		}
	}

	return strings.Join(cleanParts, " ")
}
