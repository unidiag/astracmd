package main

import (
	"embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/fatih/color"
	"github.com/rivo/tview"
)

//go:embed build/*
var staticFiles embed.FS

type FileInfo struct {
	Path  string
	IsDir bool
}

// это для файлов ./build вебсервера
func readDirRecursively(dirPath string) ([]FileInfo, error) {
	var result []FileInfo
	files, err := staticFiles.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		fullPath := dirPath + "/" + file.Name()

		info := FileInfo{
			Path:  fullPath,
			IsDir: file.IsDir(),
		}
		result = append(result, info)
		if file.IsDir() {
			subdirContents, err := readDirRecursively(fullPath)
			if err != nil {
				return nil, err
			}
			result = append(result, subdirContents...)
		}
	}
	return result, nil
}

func getFileExtension(filePath string) string {
	parts := strings.Split(filePath, "/")
	fileName := parts[len(parts)-1]
	fileParts := strings.Split(fileName, ".")
	if len(fileParts) > 1 {
		extension := fileParts[len(fileParts)-1]
		return extension
	}
	return ""
}

// определяет что запущено в дебаг-режиме
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

func debugSave(some any) {
	if !debug {
		return
	}

	text := "DEBUG " + time.Now().Format("2006-01-02 15:04:05") + "\n\n"
	text += spew.Sdump(some)

	_ = os.WriteFile("debug.txt", []byte(text), 0644)
}
