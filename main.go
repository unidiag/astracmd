package main

import (
	"log"
	"os"
	"sync"
)

const (
	APPNAME = "astracmd"
	VERSION = "1.01"
	BUILD   = "02.07.2026 13:11:27"
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

	args, err := parseAppArgs(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}

	cfg, err := LoadConfig(args.ConfigPath)
	if err != nil {
		log.Fatal(err)
	}

	switch args.Mode {
	case RunModeWeb:
		webserver(args.Port)

	case RunModeTUI:
		runTUI(cfg)

	default:
		log.Fatal("unknown run mode")
	}
}
