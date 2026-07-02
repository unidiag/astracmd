package main

import (
	"log"
	"os"

	"github.com/rivo/tview"
)

const (
	APPNAME = "astracmd"
	VERSION = "1.00"
	BUILD   = "01.07.2026 00:00:00"
)

func main() {
	configPath := getConfigPathFromArgs()

	cfg, err := LoadConfig(configPath)
	if err != nil {
		log.Fatal(err)
	}

	app := tview.NewApplication()
	app.EnableMouse(true)

	ui := NewUI(app, cfg)

	app.SetRoot(ui.pages, true)

	ui.ShowConnections()

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

func getConfigPathFromArgs() string {
	if len(os.Args) > 1 && os.Args[1] != "" {
		return os.Args[1]
	}

	return DefaultConfigPath
}
