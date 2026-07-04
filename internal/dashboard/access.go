package dashboard

import (
	"sync/atomic"

	"github.com/rivo/tview"
)

var dashboardRestrictedMode atomic.Bool

func SetRestrictedMode(restricted bool) {
	dashboardRestrictedMode.Store(restricted)
}

func IsRestrictedMode() bool {
	return dashboardRestrictedMode.Load()
}

func AccessModeLabel() string {
	if IsRestrictedMode() {
		return "RESTRICTED"
	}

	return "FULL ACCESS"
}

func CanChangeConnections() bool {
	return !IsRestrictedMode()
}

func CanChangeAstraConfig() bool {
	return !IsRestrictedMode()
}

func restrictedStatusText() string {
	return "[yellow]Restricted mode: run astracmd as root to change settings[-]"
}

func denyRestrictedStatus(status *tview.TextView) bool {
	if !IsRestrictedMode() {
		return false
	}

	if status != nil {
		status.SetText(restrictedStatusText())
	}

	return true
}
