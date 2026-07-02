package main

import (
	"log"
	"os"
	"sync"
)

const (
	APPNAME     = "astracmd"
	APPNAMEFULL = "Astra Commander"
	VERSION     = "1.01"
	BUILD       = "02.07.2026 14:57:14"
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
