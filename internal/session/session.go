package session

import (
	"fmt"
	"io"
	"strings"

	"golang.org/x/term"

	"euphio/internal/app"
	"euphio/internal/modules"
	"euphio/internal/nodes"
	"euphio/internal/views"
)

// Session represents an active user session.
type Session struct {
	rw   io.ReadWriter
	node *nodes.Node
	vm   *views.Manager
	term *term.Terminal
}

// RunSession starts the REPL for an authenticated user.
func RunSession(rw io.ReadWriter, node *nodes.Node, initialView string) {
	// Initialize Module Registry
	registry := modules.NewRegistry()
	registry.Register(&modules.DebugModule{})

	s := &Session{
		rw:   rw,
		node: node,
		vm:   views.NewManager(app.Config.Views, registry, initialView),
	}
	s.Run()
}

func (s *Session) Run() {
	username := "guest"
	if s.node.User != nil {
		username = s.node.User.Username
	}

	// If we have an initial view, try to render it
	if s.vm.Current() != "" {
		if err := s.vm.RenderCurrent(s.rw, s.node); err != nil {
			app.Logger.Error("Failed to render initial view", "view", s.vm.Current(), "err", err)
		}
	}

	// We treat the connection as a terminal.
	// term.NewTerminal handles the prompt, line editing, and echo.
	s.term = term.NewTerminal(s.rw, fmt.Sprintf("[%s] > ", username))

	for {
		line, err := s.term.ReadLine()
		if err != nil {
			if err != io.EOF {
				app.Logger.Error("Error reading line", "err", err)
			}
			break
		}

		cmd := strings.TrimSpace(line)

		if cmd == "exit" || cmd == "quit" {
			s.term.Write([]byte("Goodbye!\r\n"))
			break
		}

		// Check if the view manager wants to handle the input first
		// This allows views to override standard commands or handle navigation
		if s.vm.Current() != "" {
			handled, err := s.vm.HandleInput(s.rw, cmd, s.node)
			if err == nil && handled {
				// If input was handled (no error), re-render the current view (which might have changed)
				s.vm.RenderCurrent(s.rw, s.node)
				continue
			}
			// If not handled, fall through to standard commands
		}

		// Fallback to debug module if no view handled it (for now, until we have a proper command shell)
		// Or we can just remove this if we want views to be the only way to interact.
		// For now, let's keep the debug module accessible globally for testing if needed,
		// or just rely on views.

		// Let's rely on views. If you want debug commands, configure a view to use the 'debug' module.
		// But wait, the user said "move it out of the session, where it's mostly just code for debugging now".
		// So we should remove handleCommand entirely.

		fmt.Fprintf(s.term, "Unknown command: %s\r\n", cmd)
	}
}
