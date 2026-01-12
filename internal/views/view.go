package views

import (
	"fmt"
	"io"
	"time"

	"euphio/internal/ansi"
	"euphio/internal/app"
	"euphio/internal/config"
	"euphio/internal/modules"
	"euphio/internal/nodes"
)

// View represents a screen or state in the BBS.
type View interface {
	Render(w io.Writer, node *nodes.Node) error
	HandleInput(input string, node *nodes.Node) (string, error) // Returns next view ID or empty
}

// Manager handles the navigation stack and current view.
type Manager struct {
	config   map[string]config.View
	registry *modules.Registry
	stack    []string
	current  string
}

func NewManager(viewConfig map[string]config.View, registry *modules.Registry, initialView string) *Manager {
	return &Manager{
		config:   viewConfig,
		registry: registry,
		stack:    []string{},
		current:  initialView,
	}
}

func (m *Manager) Current() string {
	return m.current
}

func (m *Manager) Push(viewID string) {
	app.Logger.Debug("View Manager: Push", "view", viewID, "prev", m.current)
	if m.current != "" {
		m.stack = append(m.stack, m.current)
	}
	m.current = viewID
}

func (m *Manager) Pop() string {
	if len(m.stack) == 0 {
		app.Logger.Debug("View Manager: Pop (empty stack)")
		return ""
	}
	prev := m.stack[len(m.stack)-1]
	m.stack = m.stack[:len(m.stack)-1]
	app.Logger.Debug("View Manager: Pop", "view", prev, "from", m.current)
	m.current = prev
	return prev
}

// RenderCurrent renders the current view to the writer.
func (m *Manager) RenderCurrent(w io.Writer, node *nodes.Node) error {
	app.Logger.Debug("View Manager: RenderCurrent", "view", m.current)
	viewConfig, ok := m.config[m.current]
	if !ok {
		return fmt.Errorf("view not found: %s", m.current)
	}

	// For now, we only support a simple "art" view type implicitly
	// In the future, we can use viewConfig.Type to instantiate different View implementations.

	if viewConfig.Ansi != "" {
		// Load and display art using the new ansi.RenderArt utility
		if err := ansi.RenderArt(w, viewConfig.Ansi, node.Conn.IsUTF8()); err != nil {
			return err
		}
	}

	// Handle automatic transition ("next")
	if viewConfig.Next != nil {
		app.Logger.Debug("View Manager: Auto-next configured", "next", viewConfig.Next.View, "delay", viewConfig.Next.Delay)
		go func(nextView string, delay int) {
			if delay > 0 {
				time.Sleep(time.Duration(delay) * time.Millisecond)
			}
			// We need a way to signal the session loop to update the view.
			// Since we don't have a channel or event bus yet, this is tricky.
			// For now, we can't easily push the next view from a goroutine without synchronization.
			// This part requires the Session loop to be event-driven or channel-based.
			// Let's leave a TODO or implement a basic channel if possible.

			// Ideally, the Session loop should select on input AND a "view change" channel.
		}(viewConfig.Next.View, viewConfig.Next.Delay)
	}

	return nil
}

// HandleInput processes input for the current view.
// Returns true if the input was handled (consumed), false otherwise.
func (m *Manager) HandleInput(w io.Writer, input string, node *nodes.Node) (bool, error) {
	app.Logger.Debug("View Manager: HandleInput", "input", input, "current", m.current)
	viewConfig, ok := m.config[m.current]
	if !ok {
		return false, fmt.Errorf("view not found: %s", m.current)
	}

	// 1. Check if the view uses a module
	if viewConfig.Module != "" {
		if mod := m.registry.Get(viewConfig.Module); mod != nil {
			// Check if the module implements CommandHandler
			if cmdHandler, ok := mod.(modules.CommandHandler); ok {
				app.Logger.Debug("View Manager: Delegating to module", "module", viewConfig.Module)
				handled, err := cmdHandler.HandleCommand(w, node, input, "") // TODO: Parse args properly
				if err != nil {
					return handled, err
				}
				if handled {
					return true, nil
				}
			} else {
				app.Logger.Debug("View Manager: Module does not handle commands", "module", viewConfig.Module)
			}
		} else {
			app.Logger.Warn("View Manager: Module not found", "module", viewConfig.Module)
		}
	}

	// 2. Check for explicit action mapping
	if nextView, ok := viewConfig.Actions[input]; ok {
		app.Logger.Debug("View Manager: Action matched", "input", input, "next", nextView)
		if nextView == "back" || nextView == "BACK" {
			m.Pop()
		} else {
			m.Push(nextView)
		}
		return true, nil
	}

	// 3. Check for "Press any key" behavior (Next without delay)
	if viewConfig.Next != nil && viewConfig.Next.Delay == 0 {
		// If there's a next view configured without a delay (or explicit 0),
		// treat any input as a trigger to move next.
		app.Logger.Debug("View Manager: Next triggered by input", "next", viewConfig.Next.View)
		m.Push(viewConfig.Next.View)
		return true, nil
	}

	app.Logger.Debug("View Manager: No action matched")
	return false, nil
}
