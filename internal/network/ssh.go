package network

import (
	"fmt"
	"io"

	"github.com/gliderlabs/ssh"
	"golang.org/x/term"

	"euphio/internal/app"
	"euphio/internal/config"
)

type SSH struct {
	config config.SSHConfig
	server *ssh.Server
}

func NewSSH() *SSH {
	return &SSH{
		config: app.Config.LoginServers.SSH,
	}
}

func (s *SSH) ListenAndServe() error {
	app.Logger.Info("SSH server listening", "port", s.config.Port)

	s.server = &ssh.Server{
		Addr:            fmt.Sprintf(":%d", s.config.Port),
		Handler:         s.HandleSession,
		PasswordHandler: s.PasswordHandler,
	}

	err := s.server.SetOption(ssh.HostKeyFile(s.config.KeyFile))
	if err != nil {
		return err
	}
	if err := s.server.ListenAndServe(); err != nil && err != ssh.ErrServerClosed {
		// gliderlabs/ssh returns "ssh: Server closed" (ssh.ErrServerClosed) on Close
		// We want to suppress that error as it is expected during shutdown
		return err
	}
	return nil
}

func (s *SSH) Stop() error {
	if s.server != nil {
		return s.server.Close()
	}
	return nil
}

func (s *SSH) PasswordHandler(ctx ssh.Context, password string) bool {
	user, err := app.Store.Authenticate(ctx.User(), password)
	if err != nil {
		// TODO: Make login failure limit, etc.
		app.Logger.Debug("Login failed", "user", ctx.User(), "err", err)
		return false
	}
	ctx.SetValue("user", user)
	return true
}

func (s *SSH) HandleSession(sess ssh.Session) {
	app.Logger.Debug("SSH connection", "user", sess.User())

	// Set the terminal to raw mode to get character-by-character input
	// Note: In a real SSH session, the client requests a PTY.
	// We need to ensure we are in a mode that sends raw characters.
	// The gliderlabs/ssh library handles the PTY allocation if requested.

	// We can just read from the session like a normal reader.
	// However, to get "raw" behavior similar to our Telnet implementation,
	// we just read byte by byte.

	term := term.NewTerminal(sess, "> ")
	_ = term

	// Output welcome message
	io.WriteString(sess, fmt.Sprintf("\r\nWelcome to %s (SSH)\r\n", app.Config.General.BoardName))

	buf := make([]byte, 1)
	for {
		n, err := sess.Read(buf)
		if err != nil {
			if err != io.EOF {
				app.Logger.Debug("SSH read error", "err", err)
			}
			return
		}
		if n == 0 {
			continue
		}

		b := buf[0]

		// Log the received byte for debugging
		app.Logger.Debug(fmt.Sprintf("SSH Received: byte=%d hex=0x%02X char=%q", b, b, b))

		// Handle Backspace (0x08) or Delete (0x7F)
		if b == 8 || b == 127 {
			// Send Backspace, Space, Backspace to erase the character visually
			sess.Write([]byte{8, 32, 8})
			continue
		}

		// Simple echo so we can see what we type
		sess.Write([]byte{b})
	}
}
