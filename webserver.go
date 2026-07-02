package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
	"github.com/spf13/cast"
)

var terminalUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow same-origin and LAN access.
		return true
	},
}

type TerminalMessage struct {
	Type string `json:"type"`
	Data string `json:"data,omitempty"`
	Cols uint16 `json:"cols,omitempty"`
	Rows uint16 `json:"rows,omitempty"`
}

// ██╗    ██╗███████╗██████╗ ███████╗███████╗██████╗ ██╗   ██╗███████╗██████╗
// ██║    ██║██╔════╝██╔══██╗██╔════╝██╔════╝██╔══██╗██║   ██║██╔════╝██╔══██╗
// ██║ █╗ ██║█████╗  ██████╔╝███████╗█████╗  ██████╔╝██║   ██║█████╗  ██████╔╝
// ██║███╗██║██╔══╝  ██╔══██╗╚════██║██╔══╝  ██╔══██╗╚██╗ ██╔╝██╔══╝  ██╔══██╗
// ╚███╔███╔╝███████╗██████╔╝███████║███████╗██║  ██║ ╚████╔╝ ███████╗██║  ██║
//  ╚══╝╚══╝ ╚══════╝╚═════╝ ╚══════╝╚══════╝╚═╝  ╚═╝  ╚═══╝  ╚══════╝╚═╝  ╚═╝

func webserver(port int) {
	http.HandleFunc("/ws/terminal", terminalWebSocketHandler)

	http.Handle("/build/", http.FileServer(http.FS(staticFiles)))

	ext := make(map[string]string)
	ext["html"] = "text/html"
	ext["json"] = "application/json"
	ext["css"] = "text/css"
	ext["js"] = "application/javascript"
	ext["gif"] = "image/gif"
	ext["svg"] = "image/svg+xml"
	ext["png"] = "image/png"
	ext["jpg"] = "image/jpeg"
	ext["jpeg"] = "image/jpeg"
	ext["ico"] = "image/x-icon"
	ext["woff"] = "font/woff"
	ext["woff2"] = "font/woff2"
	ext["ttf"] = "font/ttf"
	ext["eot"] = "application/vnd.ms-fontobject"

	// Получаем содержимое корневой директории
	contents, err := readDirRecursively("build")
	if err != nil {
		log.Fatal(err)
	}

	for _, item := range contents {
		ff := item.Path

		http.HandleFunc(strings.ReplaceAll(ff, "build", ""), func(w http.ResponseWriter, r *http.Request) {
			file, err := staticFiles.ReadFile(ff)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", ext[getFileExtension(ff)])

			_, err = w.Write(file)
			if err != nil {
				log.Println(err)
			}
		})
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		file, err := staticFiles.ReadFile("build/index.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html")

		_, err = w.Write(file)
		if err != nil {
			log.Println(err)
		}
	})

	// Запуск сервера на порту
	webport := ":" + cast.ToString(port)
	log.Println("WebSSH was run on the " + webport)
	log.Fatal(http.ListenAndServe(webport, nil))
}

// ████████╗███████╗██████╗ ███╗   ███╗██╗███╗   ██╗ █████╗ ██╗
// ╚══██╔══╝██╔════╝██╔══██╗████╗ ████║██║████╗  ██║██╔══██╗██║
//    ██║   █████╗  ██████╔╝██╔████╔██║██║██╔██╗ ██║███████║██║
//    ██║   ██╔══╝  ██╔══██╗██║╚██╔╝██║██║██║╚██╗██║██╔══██║██║
//    ██║   ███████╗██║  ██║██║ ╚═╝ ██║██║██║ ╚████║██║  ██║███████╗
//    ╚═╝   ╚══════╝╚═╝  ╚═╝╚═╝     ╚═╝╚═╝╚═╝  ╚═══╝╚═╝  ╚═╝╚══════╝

func terminalWebSocketHandler(w http.ResponseWriter, r *http.Request) {

	conn, err := terminalUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("terminal websocket upgrade:", err)
		return
	}
	defer conn.Close()

	cmd := buildTerminalLoginCommand()

	ptmx, err := pty.Start(cmd)
	if err != nil {
		log.Println("terminal pty start:", err)
		_ = conn.WriteMessage(websocket.TextMessage, []byte("Failed to start terminal\r\n"))
		return
	}

	defer func() {
		_ = ptmx.Close()

		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
		}
	}()

	done := make(chan struct{})

	go func() {
		defer close(done)

		buf := make([]byte, 4096)

		for {
			n, err := ptmx.Read(buf)
			if err != nil {
				if err != io.EOF {
					log.Println("terminal pty read:", err)
				}
				return
			}

			if n > 0 {
				if err := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
					log.Println("terminal websocket write:", err)
					return
				}
			}
		}
	}()

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var msg TerminalMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}

		switch msg.Type {
		case "input":
			if msg.Data == "" {
				continue
			}

			if _, err := ptmx.Write([]byte(msg.Data)); err != nil {
				log.Println("terminal pty write:", err)
				return
			}

		case "resize":
			resizeTerminal(ptmx, msg.Cols, msg.Rows)
		}
	}

	<-done
}

func resizeTerminal(file *os.File, cols uint16, rows uint16) {
	if cols == 0 || rows == 0 {
		return
	}

	err := pty.Setsize(file, &pty.Winsize{
		Cols: cols,
		Rows: rows,
	})

	if err != nil {
		log.Println("terminal resize:", err)
	}
}

func buildTerminalLoginCommand() *exec.Cmd {
	loginPath := "/bin/login"

	if _, err := os.Stat(loginPath); err == nil {
		cmd := exec.Command(loginPath)

		cmd.Env = append(os.Environ(),
			"TERM=xterm-256color",
			"LANG=C.UTF-8",
			"LC_ALL=C.UTF-8",
			"ASTRACMD_WEB_TERMINAL=1",
		)

		return cmd
	}

	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}

	cmd := exec.Command(shell)

	cmd.Env = append(os.Environ(),
		"TERM=xterm-256color",
		"LANG=C.UTF-8",
		"LC_ALL=C.UTF-8",
		"ASTRACMD_WEB_TERMINAL=1",
	)

	return cmd
}
