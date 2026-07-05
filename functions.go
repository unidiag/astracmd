package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/fatih/color"
	"github.com/rivo/tview"
)

func isRunThroughGoRun() bool {
	exePath, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exeDir := filepath.Dir(exePath)
	mainGoPath := filepath.Join(exeDir, "main")
	if _, err := os.Stat(mainGoPath); err == nil {
		color.Yellow("DEBUG MODE")
		return true
	}
	return false
}

func runTUI(cfg *Config) {
	app := tview.NewApplication()
	app.EnableMouse(true)

	ui := NewUI(app, cfg)

	app.SetRoot(ui.pages, true)

	ui.ShowConnections()

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

func parseAppArgs(argv []string) (AppArgs, error) {
	args := AppArgs{
		Mode:       RunModeTUI,
		ConfigPath: DefaultConfigPath,
		Port:       0,
	}

	if len(argv) == 0 {
		return args, nil
	}

	first := argv[0]

	if port, ok := parseWebPort(first); ok {
		args.Mode = RunModeWeb
		args.Port = port

		if os.Geteuid() != 0 {
			return args, fmt.Errorf("WebSSH mode requires root privileges")
		}

		return args, nil
	}

	args.Mode = RunModeTUI
	args.ConfigPath = first

	return args, nil
}

func parseWebPort(value string) (int, bool) {
	port, err := strconv.Atoi(value)
	if err != nil {
		return 0, false
	}

	if port < 80 || port > 65535 {
		return 0, false
	}

	return port, true
}
