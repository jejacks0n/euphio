package session

import (
	"fmt"
	"io"
	"strings"

	"golang.org/x/term"

	"euphio/internal/app"
)

// RunSession starts the REPL for an authenticated user.
func RunSession(rw io.ReadWriter, username string) {
	io.WriteString(rw, fmt.Sprintf("Welcome to %s\r\n", app.Config.General.BoardName))
	io.WriteString(rw, "Type 'help' for commands or 'exit' to quit.\r\n")
	io.WriteString(rw, "------------------------------------------\r\n")

	// We treat the connection as a terminal.
	// term.NewTerminal handles the prompt, line editing, and echo.
	t := term.NewTerminal(rw, fmt.Sprintf("[%s] > ", username))

	for {
		line, err := t.ReadLine()
		if err != nil {
			if err != io.EOF {
				app.Logger.Error("Error reading line", "err", err)
			}
			break
		}

		cmd := strings.TrimSpace(line)

		if cmd == "exit" || cmd == "quit" {
			t.Write([]byte("Goodbye!\r\n"))
			break
		}

		handleCommand(t, cmd)
	}
}

func handleCommand(w io.Writer, cmd string) {
	switch cmd {
	case "":
		// Ignore empty enter keys
		return
	case "help":
		io.WriteString(w, "Available commands: help, time, whoami, exit\r\n")
	case "whoami":
		io.WriteString(w, "You are a generic guest user.\r\n")
	case "time":
		io.WriteString(w, "It is always time to code.\r\n")
	default:
		fmt.Fprintf(w, "Unknown command: %s\r\n", cmd)
	}
}
