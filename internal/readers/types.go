package readers

import (
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const (
	serialByIDPath      = "/dev/serial/by-id"
	readersLogTailBytes = 16 * 1024
)

const (
	readersPaneList = iota
	readersPaneLogFilter
)

const (
	readersRefreshPeriod = time.Second
)

type Options struct {
	App              *tview.Application
	Pages            *tview.Pages
	PageName         string
	ConfigPath       string
	HandleGlobalKeys func(*tcell.EventKey) bool
}

type Device struct {
	Name        string
	Path        string
	DisplayName string
	Target      string
	Busy        bool
	ProcessPID  int
	ProcessName string
	ProcessCmd  string
}

type selectedReaderState struct {
	Device      Device
	ConfigPath  string
	ConfigText  string
	ConfigDirty bool
	LogPath     string
}

type ReaderSection struct {
	Section    string
	Enabled    bool
	Name       string
	SerialByID string
	OscamBin   string
	OscamDir   string
}
