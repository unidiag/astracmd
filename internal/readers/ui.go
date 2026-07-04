package readers

import (
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/rivo/tview"
)

func killProcess(pid int) error {
	if pid <= 0 {
		return fmt.Errorf("invalid pid")
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}

	return process.Signal(syscall.SIGKILL)
}

func displayDeviceName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.TrimPrefix(name, "usb-")

	for {
		cleaned := false

		for _, suffix := range []string{
			"-if00-port0",
			"-if00-port1",
			"-if01-port0",
			"-if01-port1",
		} {
			if strings.HasSuffix(name, suffix) {
				name = strings.TrimSuffix(name, suffix)
				cleaned = true
				break
			}
		}

		if !cleaned {
			break
		}
	}

	return name
}

func centerPrimitive(p tview.Primitive, width int, height int) tview.Primitive {
	row := tview.NewFlex()
	row.SetDirection(tview.FlexRow)
	row.AddItem(nil, 0, 1, false)
	row.AddItem(p, height, 1, true)
	row.AddItem(nil, 0, 1, false)

	root := tview.NewFlex()
	root.SetDirection(tview.FlexColumn)
	root.AddItem(nil, 0, 1, false)
	root.AddItem(row, width, 1, true)
	root.AddItem(nil, 0, 1, false)

	return root
}
