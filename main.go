package main

import (
	"log"
	"main/internal/dashboard"
	"os"
	"sync"
)

const (
	APPNAME     = "astracmd"
	APPNAMEFULL = "Astra Commander"
	VERSION     = "1.01"
	BUILD       = "05.07.2026 18:55:28"
)

type RunMode int

const (
	RunModeTUI RunMode = iota
	RunModeWeb
)

var (
	mu    sync.Mutex
	debug = false
	err   error
)

type AppArgs struct {
	Mode       RunMode
	ConfigPath string
	Port       int
}

func main() {

	debug = isRunThroughGoRun()
	dashboard.SetRestrictedMode(os.Geteuid() != 0)

	args, err := parseAppArgs(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}

	cfg, err := LoadConfig(args.ConfigPath)
	if err != nil {
		log.Fatal(err)
	}

	switch args.Mode {
	// disable web-version
	// case RunModeWeb:
	// 	webserver(args.Port)

	case RunModeTUI:
		runTUI(cfg)

	default:
		log.Fatal("unknown run mode")
	}
}
