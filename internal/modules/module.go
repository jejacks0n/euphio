package modules

import (
	"io"

	"euphio/internal/nodes"
)

// Module defines the base interface for pluggable functionality.
type Module interface {
	// Name returns the unique identifier for the module.
	Name() string
}

// CommandHandler is an optional interface for modules that process user commands.
type CommandHandler interface {
	Module
	// HandleCommand processes a command.
	// Returns true if the command was handled, false otherwise.
	HandleCommand(w io.Writer, node *nodes.Node, cmd string, args string) (bool, error)
}

// Registry holds all available modules.
type Registry struct {
	modules map[string]Module
}

func NewRegistry() *Registry {
	return &Registry{
		modules: make(map[string]Module),
	}
}

func (r *Registry) Register(m Module) {
	r.modules[m.Name()] = m
}

func (r *Registry) Get(name string) Module {
	return r.modules[name]
}
