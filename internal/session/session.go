package session

import (
	"fmt"
	"io"
	"strings"

	"golang.org/x/term"

	"euphio/internal/app"
	"euphio/internal/nodes"
)

// RunSession starts the REPL for an authenticated user.
func RunSession(rw io.ReadWriter, node *nodes.Node, username string) {
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

		handleCommand(t, node, cmd)
	}
}

func handleCommand(w io.Writer, node *nodes.Node, line string) {
	parts := strings.SplitN(line, " ", 2)
	cmd := parts[0]
	args := ""
	if len(parts) > 1 {
		args = parts[1]
	}

	switch cmd {
	case "":
		// Ignore empty enter keys
		return
	case "help":
		io.WriteString(w, "Available commands: help, info, time, whoami, yell <msg>, exit\r\n")
	case "info":
		info := node.Conn.GetTerminalInfo()
		fmt.Fprintf(w, "Terminal: %s (%dx%d)\r\n", info.Type, info.Width, info.Height)
	case "whoami":
		fmt.Fprintf(w, "You are a generic guest user on Node %d.\r\n", node.ID)
	case "time":
		io.WriteString(w, "It is always time to code.\r\n")
	case "yell":
		if args == "" {
			io.WriteString(w, "Usage: yell <message>\r\n")
			return
		}
		msg := fmt.Sprintf("\r\n[Node %d yells]: %s\r\n", node.ID, args)
		app.Nodes.BroadcastExcept(msg, node.ID)
		io.WriteString(w, "You yelled to everyone.\r\n")
	default:
		fmt.Fprintf(w, "Unknown command: %s\r\n", cmd)
	}
}
