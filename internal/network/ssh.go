package network

import (
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/gliderlabs/ssh"

	"euphio/internal/app"
	"euphio/internal/config"
	"euphio/internal/nodes"
	"euphio/internal/session"
)

type SSH struct {
	config config.SSHConfig
	server *ssh.Server
}

// sshConnectionWrapper adapts ssh.Session to nodes.Connection interface
type sshConnectionWrapper struct {
	sess   ssh.Session
	mu     sync.RWMutex
	width  int
	height int
	term   string
}

func (w *sshConnectionWrapper) Send(msg string) error {
	_, err := io.WriteString(w.sess, msg+"\r\n")
	return err
}

func (w *sshConnectionWrapper) RemoteAddr() net.Addr {
	return w.sess.RemoteAddr()
}

func (w *sshConnectionWrapper) GetTerminalInfo() nodes.TerminalInfo {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return nodes.TerminalInfo{
		Type:   w.term,
		Width:  w.width,
		Height: w.height,
	}
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
	node, err := app.Nodes.Acquire()
	if err != nil {
		app.Logger.Warn("SSH Connection rejected: system full", "addr", sess.RemoteAddr())
		sess.Close()
		return
	}
	defer app.Nodes.Release(node.ID)

	// Assign connection wrapper to node
	wrapper := &sshConnectionWrapper{sess: sess}

	pty, winCh, isPty := sess.Pty()
	if isPty {
		wrapper.term = pty.Term
		wrapper.width = pty.Window.Width
		wrapper.height = pty.Window.Height

		go func() {
			for win := range winCh {
				wrapper.mu.Lock()
				wrapper.width = win.Width
				wrapper.height = win.Height
				wrapper.mu.Unlock()
			}
		}()
	}

	node.Conn = wrapper

	logger := app.Logger.With("node", node.ID)

	logger.Info("SSH connection established", "user", sess.User(), "term", wrapper.term, "width", wrapper.width, "height", wrapper.height)
	defer logger.Info("SSH connection closed", "addr", sess.RemoteAddr())

	// Hand off to the session manager
	session.RunSession(sess, node, "guest")
}
