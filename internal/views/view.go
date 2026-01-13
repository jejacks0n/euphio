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
	"euphio/internal/prompts"
)

// View represents a screen or state in the BBS.
type View interface {
	Render(w io.Writer, node *nodes.Node) error
	HandleInput(input string, node *nodes.Node) (string, error) // Returns next view ID or empty
}

// Manager handles the navigation stack and current view.
type Manager struct {
	config        map[string]config.View
	registry      *modules.Registry
	stack         []string
	current       string
	events        chan interface{} // Channel to send events back to the session
	currentPrompt prompts.Prompt
}

func NewManager(viewConfig map[string]config.View, registry *modules.Registry, initialView string, events chan interface{}) *Manager {
	return &Manager{
		config:   viewConfig,
		registry: registry,
		stack:    []string{},
		current:  initialView,
		events:   events,
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
	m.currentPrompt = nil // Reset prompt on view change
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
	m.currentPrompt = nil // Reset prompt on view change
	return prev
}

// RenderCurrent renders the current view to the writer.
func (m *Manager) RenderCurrent(w io.Writer, node *nodes.Node) error {
	app.Logger.Debug("View Manager: RenderCurrent", "view", m.current, "stack", m.stack)
	viewConfig, ok := m.config[m.current]
	if !ok {
		return fmt.Errorf("view not found: %s", m.current)
	}

	// Handle screen clearing
	if viewConfig.ClearScreen {
		w.Write([]byte(ansi.ClearScreen))
	}

	// Handle cursor visibility
	if viewConfig.HideCursor {
		w.Write([]byte(ansi.HideCursor))
	} else {
		w.Write([]byte(ansi.ShowCursor))
	}

	// For now, we only support a simple "art" view type implicitly
	// In the future, we can use viewConfig.Type to instantiate different View implementations.

	isUTF8 := false
	if node.Conn != nil {
		isUTF8 = node.Conn.IsUTF8()
	}

	if viewConfig.Ansi != "" {
		// Load and display art using the new ansi.RenderArt utility
		if err := ansi.RenderArt(w, viewConfig.Ansi, isUTF8); err != nil {
			return err
		}
	}

	// Handle Prompt
	if viewConfig.Prompt != "" {
		if promptCfg, ok := app.Config.Prompts[viewConfig.Prompt]; ok {
			// Instantiate the prompt
			// For now, we only have BasicPrompt. Later we can use promptCfg.Type
			m.currentPrompt = prompts.NewBasic(promptCfg)
			if err := m.currentPrompt.Render(w, node); err != nil {
				return err
			}
		}
	}

	// Handle automatic transition ("next")
	// Only trigger if Delay is greater than 0.
	// If Delay is 0 (default), it implies "wait for key press" which is handled in HandleInput.
	if viewConfig.Next != nil && viewConfig.Next.Delay > 0 {
		app.Logger.Debug("View Manager: Auto-next configured", "next", viewConfig.Next.View, "delay", viewConfig.Next.Delay)
		go func(nextView string, delay int) {
			time.Sleep(time.Duration(delay) * time.Millisecond)
			// Send event to session loop instead of modifying state directly
			if m.events != nil {
				m.events <- ChangeViewEvent{ViewID: nextView}
			}
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

	// 0. Check if there is an active prompt
	if m.currentPrompt != nil {
		handled, done, err := m.currentPrompt.HandleInput(input, node)
		if err != nil {
			return handled, err
		}
		if handled {
			if done {
				// Prompt is done, move to next view if configured
				if viewConfig.Next != nil {
					app.Logger.Debug("View Manager: Prompt done, moving next", "next", viewConfig.Next.View)
					m.Push(viewConfig.Next.View)
					return true, nil
				}
				// If no next view, we just stay here
			}
			return true, nil
		}
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
	// Only if NO prompt is active (prompts handle their own input)
	if m.currentPrompt == nil && viewConfig.Next != nil && viewConfig.Next.Delay == 0 {
		// If there's a next view configured without a delay (or explicit 0),
		// treat any input as a trigger to move next.
		app.Logger.Debug("View Manager: Next triggered by input", "next", viewConfig.Next.View)
		m.Push(viewConfig.Next.View)
		return true, nil
	}

	app.Logger.Debug("View Manager: No action matched")
	return false, nil
}

// Events
type ChangeViewEvent struct {
	ViewID string
}

type InputEvent struct {
	Input string
}
