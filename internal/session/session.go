package session

import (
	"fmt"
	"io"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"

	"euphio/internal/app"
	"euphio/internal/nodes"
)

// RunSession starts the REPL for an authenticated user.
func RunSession(rw io.ReadWriter, node *nodes.Node) {
	username := "guest"
	if node.User != nil {
		username = node.User.Username
	}

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

		handleCommand(t, rw, node, cmd)
	}
}

func handleCommand(t *term.Terminal, rw io.ReadWriter, node *nodes.Node, line string) {
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
		io.WriteString(t, "Available commands: help, info, time, whoami, yell <msg>, box, tui, exit\r\n")
	case "info":
		info := node.Conn.GetTerminalInfo()
		fmt.Fprintf(t, "Terminal: %s (%dx%d)\r\n", info.Type, info.Width, info.Height)
	case "whoami":
		username := "guest"
		if node.User != nil {
			username = node.User.Username
		}
		fmt.Fprintf(t, "You are %s on Node %d.\r\n", username, node.ID)
	case "time":
		io.WriteString(t, "It is always time to code.\r\n")
	case "yell":
		if args == "" {
			io.WriteString(t, "Usage: yell <message>\r\n")
			return
		}
		msg := fmt.Sprintf("\r\n[Node %d yells]: %s\r\n", node.ID, args)
		app.Nodes.BroadcastExcept(msg, node.ID)
		io.WriteString(t, "You yelled to everyone.\r\n")
	case "box":
		// Example of using Lipgloss to draw a box
		style := lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(1, 2).
			Render("Hello from Lipgloss!")

		io.WriteString(t, "\r\n"+style+"\r\n")
	case "tui":
		// Example of running a Bubble Tea program
		p := tea.NewProgram(initialModel(), tea.WithInput(rw), tea.WithOutput(rw))
		if _, err := p.Run(); err != nil {
			fmt.Fprintf(t, "Error running TUI: %v\r\n", err)
		}
	default:
		fmt.Fprintf(t, "Unknown command: %s\r\n", cmd)
	}
}

// Simple Bubble Tea model for demonstration
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
