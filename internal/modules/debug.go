package modules

import (
	"fmt"
	"io"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"euphio/internal/app"
	"euphio/internal/nodes"
)

type DebugModule struct{}

func (m *DebugModule) Name() string {
	return "debug"
}

func (m *DebugModule) HandleCommand(w io.Writer, node *nodes.Node, cmd string, args string) (bool, error) {
	switch cmd {
	case "help":
		io.WriteString(w, "Debug commands: help, info, time, whoami, yell <msg>, box, tui\r\n")
		return true, nil
	case "info":
		info := node.Conn.GetTerminalInfo()
		fmt.Fprintf(w, "Terminal: %s (%dx%d)\r\n", info.Type, info.Width, info.Height)
		return true, nil
	case "whoami":
		username := "guest"
		if node.User != nil {
			username = node.User.Username
		}
		fmt.Fprintf(w, "You are %s on Node %d.\r\n", username, node.ID)
		return true, nil
	case "time":
		io.WriteString(w, "It is always time to code.\r\n")
		return true, nil
	case "yell":
		if args == "" {
			io.WriteString(w, "Usage: yell <message>\r\n")
			return true, nil
		}
		msg := fmt.Sprintf("\r\n[Node %d yells]: %s\r\n", node.ID, args)
		app.Nodes.BroadcastExcept(msg, node.ID)
		io.WriteString(w, "You yelled to everyone.\r\n")
		return true, nil
	case "box":
		// Define a safe ASCII border for legacy clients (CP437/ANSI)
		asciiBorder := lipgloss.Border{
			Top:         "-",
			Bottom:      "-",
			Left:        "|",
			Right:       "|",
			TopLeft:     "+",
			TopRight:    "+",
			BottomLeft:  "+",
			BottomRight: "+",
		}

		// Default to ASCII for safety in BBS context, unless we know it's a modern terminal
		border := asciiBorder
		if node.Conn.IsUTF8() {
			border = lipgloss.RoundedBorder()
		}

		// Example of using Lipgloss to draw a box
		style := lipgloss.NewStyle().
			BorderStyle(border).
			BorderForeground(lipgloss.Color("63")).
			Padding(1, 2).
			Render("Hello from Lipgloss!")

		io.WriteString(w, "\r\n"+style+"\r\n")
		return true, nil
	case "tui":
		// Example of running a Bubble Tea program
		// Note: We need a ReadWriter here, but w is just Writer.
		// We might need to change the interface if we want interactive modules.
		// For now, let's assume w is also a Reader (it is in Session), or cast it.
		if rw, ok := w.(io.ReadWriter); ok {
			p := tea.NewProgram(initialModel(), tea.WithInput(rw), tea.WithOutput(rw))
			if _, err := p.Run(); err != nil {
				fmt.Fprintf(w, "Error running TUI: %v\r\n", err)
			}
		} else {
			io.WriteString(w, "Error: IO does not support reading for TUI\r\n")
		}
		return true, nil
	case "exit", "quit":
		// We can handle exit here if we want the debug module to be able to close the session,
		// but usually session handles it. However, since we removed the session loop's explicit exit check,
		// we might want to re-add it or handle it via views.
		// For now, let's just return false so it falls through or does nothing,
		// as the user wants to focus on view navigation.
		return false, nil
	}
	return false, nil
}

// Simple Bubble Tea model for demonstration (copied from session)
type model struct {
	choices  []string
	cursor   int
	selected map[int]struct{}
}

func initialModel() model {
	return model{
		choices:  []string{"Buy carrots", "Buy celery", "Buy kohlrabi"},
		selected: make(map[int]struct{}),
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}
		case "enter", " ":
			_, ok := m.selected[m.cursor]
			if ok {
				delete(m.selected, m.cursor)
			} else {
				m.selected[m.cursor] = struct{}{}
			}
		}
	}
	return m, nil
}

func (m model) View() string {
	s := "What should we buy at the market?\n\n"

	for i, choice := range m.choices {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		checked := " "
		if _, ok := m.selected[i]; ok {
			checked = "x"
		}

		s += fmt.Sprintf("%s [%s] %s\n", cursor, checked, choice)
	}

	s += "\nPress q to quit.\n"
	return s
}
