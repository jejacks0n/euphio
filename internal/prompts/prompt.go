package prompts

import (
	"euphio/internal/ansi"
	"euphio/internal/config"
	"euphio/internal/nodes"
	"io"
)

type Prompt interface {
	Render(w io.Writer, node *nodes.Node) error
	HandleInput(input string, node *nodes.Node) (bool, bool, error) // handled, done, error
}

type BasicPrompt struct {
	cfg config.Prompt
}

func NewBasic(cfg config.Prompt) *BasicPrompt {
	return &BasicPrompt{cfg: cfg}
}

func (p *BasicPrompt) Render(w io.Writer, node *nodes.Node) error {
	isUTF8 := false
	if node.Conn != nil {
		isUTF8 = node.Conn.IsUTF8()
	}
	if p.cfg.Ansi != "" {
		if err := ansi.RenderArt(w, p.cfg.Ansi, isUTF8); err != nil {
			return err
		}
	}
	if p.cfg.LineFeed {
		w.Write([]byte("\r\n"))
	}
	return nil
}

func (p *BasicPrompt) HandleInput(input string, node *nodes.Node) (bool, bool, error) {
	if len(input) == 0 {
		return false, false, nil
	}
	// Basic prompt (like pause) accepts any input and is done.
	return true, true, nil
}
