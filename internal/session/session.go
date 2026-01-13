package session

import (
	"io"
	"time"

	"euphio/internal/ansi"
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
	// Event channel for session-wide events
	events chan interface{}
}

// RunSession starts the REPL for an authenticated user.
func RunSession(rw io.ReadWriter, node *nodes.Node, initialView string) {
	// Initialize Module Registry
	registry := modules.NewRegistry()
	registry.Register(&modules.DebugModule{})

	events := make(chan interface{}, 10)

	s := &Session{
		rw:     rw,
		node:   node,
		vm:     views.NewManager(app.Config.Views, registry, initialView, events),
		events: events,
	}
	s.Run()
}

func (s *Session) Run() {
	// Hide the cursor
	s.rw.Write([]byte(ansi.HideCursor))

	// Render initial view
	if err := s.vm.RenderCurrent(s.rw, s.node); err != nil {
		app.Logger.Error("Failed to render initial view", "view", s.vm.Current(), "err", err)
	}

	// Start Input Listener
	go s.readInput()

	// Main Event Loop
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case event := <-s.events:
			// Handle session events (e.g., view changes, messages)
			s.handleEvent(event)
		case <-ticker.C:
			// Periodic updates if needed
		}
	}
}

func (s *Session) readInput() {
	buf := make([]byte, 1024)
	for {
		n, err := s.rw.Read(buf)
		if err != nil {
			// TODO: Handle disconnect
			return
		}
		if n > 0 {
			s.events <- views.InputEvent{Input: string(buf[:n])}
		}
	}
}

func (s *Session) handleEvent(event interface{}) {
	switch e := event.(type) {
	case string:
		app.Logger.Debug("Session event", "msg", e)
	case views.ChangeViewEvent:
		app.Logger.Debug("Handling ChangeViewEvent", "view", e.ViewID)
		s.vm.Push(e.ViewID)
		if err := s.vm.RenderCurrent(s.rw, s.node); err != nil {
			app.Logger.Error("Failed to render view", "view", e.ViewID, "err", err)
		}
	case views.InputEvent:
		handled, err := s.vm.HandleInput(s.rw, e.Input, s.node)
		if err != nil {
			app.Logger.Error("Error handling input", "err", err)
		}
		if handled {
			if err := s.vm.RenderCurrent(s.rw, s.node); err != nil {
				app.Logger.Error("Failed to render view after input", "err", err)
			}
		}
	}
}
