import React, { useEffect, useRef } from "react";
import { Box } from "@mui/material";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";

import "@xterm/xterm/css/xterm.css";

const WS_PATH = "/ws/terminal";

function buildTerminalWebSocketUrl() {
    const baseUrl = process.env.REACT_APP_URL || window.location.origin;
    const url = new URL(WS_PATH, baseUrl);

    url.protocol = url.protocol === "https:" ? "wss:" : "ws:";

    return url.toString();
}

export default function MainPage() {
    const terminalRef = useRef(null);
    const termRef = useRef(null);
    const fitAddonRef = useRef(null);
    const socketRef = useRef(null);

    useEffect(() => {
        if (!terminalRef.current) {
            return;
        }

        const term = new Terminal({
            cursorBlink: true,
            fontFamily: "monospace",
            fontSize: 14,
            scrollback: 5000,
            smoothScrollDuration: 0,
            theme: {
                background: "#050505",
                foreground: "#d6d6d6",
                cursor: "#ffffff",
                selectionBackground: "#444444",
            },
        });

        const fitAddon = new FitAddon();

        term.loadAddon(fitAddon);
        term.open(terminalRef.current);

        termRef.current = term;
        fitAddonRef.current = fitAddon;

        const sendResize = () => {
            const currentTerm = termRef.current;
            const currentSocket = socketRef.current;

            if (!currentTerm || !currentSocket) {
                return;
            }

            if (currentSocket.readyState !== WebSocket.OPEN) {
                return;
            }

            currentSocket.send(JSON.stringify({
                type: "resize",
                cols: currentTerm.cols,
                rows: currentTerm.rows,
            }));
        };

        const fitAndResize = () => {
            if (!fitAddonRef.current) {
                return;
            }

            fitAddonRef.current.fit();
            sendResize();
        };

        requestAnimationFrame(fitAndResize);

        const socket = new WebSocket(buildTerminalWebSocketUrl());
        socket.binaryType = "arraybuffer";
        socketRef.current = socket;

        socket.onopen = () => {
            //term.writeln("\x1b[32mConnected\x1b[0m");
            sendResize();
            term.focus();
        };

        const decoder = new TextDecoder("utf-8", {
            fatal: false,
            ignoreBOM: true,
        });

        socket.onmessage = (event) => {
            if (typeof event.data === "string") {
                term.write(event.data);
                return;
            }

            if (event.data instanceof ArrayBuffer) {
                const text = decoder.decode(event.data, { stream: true });
                term.write(text);
                return;
            }

            if (event.data instanceof Blob) {
                event.data.arrayBuffer().then((buffer) => {
                    const text = decoder.decode(buffer, { stream: true });
                    term.write(text);
                });
            }
        };

        socket.onerror = () => {
            term.writeln("");
            term.writeln("\x1b[31mWebSocket error\x1b[0m");
        };

        socket.onclose = () => {
            term.writeln("");
            term.writeln("\x1b[33mDisconnected\x1b[0m");
        };

        const inputDisposable = term.onData((data) => {
            if (socket.readyState !== WebSocket.OPEN) {
                return;
            }

            socket.send(JSON.stringify({
                type: "input",
                data,
            }));
        });

        const resizeObserver = new ResizeObserver(fitAndResize);
        resizeObserver.observe(terminalRef.current);

        window.addEventListener("resize", fitAndResize);

        return () => {
            window.removeEventListener("resize", fitAndResize);
            resizeObserver.disconnect();

            inputDisposable.dispose();

            if (socketRef.current) {
                socketRef.current.close();
                socketRef.current = null;
            }

            term.dispose();

            termRef.current = null;
            fitAddonRef.current = null;
        };
    }, []);

    return (
        <Box
            sx={{
                position: "fixed",
                inset: 0,
                bgcolor: "#000000",
                overflow: "hidden",
                p: "1px",
                boxSizing: "border-box",
            }}
        >
            <Box
                ref={terminalRef}
                sx={{
                    width: "100%",
                    height: "100%",
                    overflow: "hidden",
                    bgcolor: "#050505",

                    "& .xterm": {
                        width: "calc(100% + 1px)",
                        height: "100%",
                        boxSizing: "border-box",
                    },

                    "& .xterm-viewport": {
                        overflow: "hidden !important",
                        scrollbarWidth: "none",
                        msOverflowStyle: "none",
                    },

                    "& .xterm-viewport::-webkit-scrollbar": {
                        width: "0px",
                        height: "0px",
                        display: "none",
                    },

                    "& .xterm-screen": {
                        width: "100% !important",
                        height: "100% !important",
                    },

                    "& .xterm-scroll-area": {
                        visibility: "hidden",
                    },

                    "& .xterm-helper-textarea": {
                        opacity: 0,
                        width: 0,
                        height: 0,
                    },

                }}
            />
        </Box>
    );
}