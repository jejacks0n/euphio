package ssh

import (
	"fmt"

	"github.com/gliderlabs/ssh"

	"euphio/internal/app"
	"euphio/internal/config"
	"euphio/internal/session"
	"euphio/internal/store"
)

type Server struct {
	config config.SSHConfig
	server *ssh.Server
}

func NewServer() *Server {
	return &Server{
		config: app.Config.Listeners.SSH,
	}
}

func (s *Server) ListenAndServe() error {
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

func (s *Server) Stop() error {
	if s.server != nil {
		return s.server.Close()
	}
	return nil
}

func (s *Server) PasswordHandler(ctx ssh.Context, password string) bool {
	user, err := app.Store.Authenticate(ctx.User(), password)
	if err != nil {
		// TODO: Make login failure limit, etc.
		app.Logger.Debug("Login failed", "user", ctx.User(), "err", err)
		return false
	}
	ctx.SetValue("user", user)
	return true
}

func (s *Server) HandleSession(sess ssh.Session) {
	node, err := app.Nodes.Acquire()
	if err != nil {
		app.Logger.Warn("SSH Connection rejected: system full", "addr", sess.RemoteAddr())
		sess.Close()
		return
	}
	defer app.Nodes.Release(node.ID)

	// Assign connection wrapper to node
	conn := NewConnection(sess)
	node.Conn = conn

	// Retrieve authenticated user from context
	if user, ok := sess.Context().Value("user").(*store.User); ok {
		node.User = user
	}

	logger := app.Logger.With("node", node.ID)
	info := conn.GetTerminalInfo()

	logger.Info("SSH connection established", "user", sess.User(), "term", info.Type, "width", info.Width, "height", info.Height)
	defer logger.Info("SSH connection closed", "addr", sess.RemoteAddr())

	// Hand off to the session manager
	session.RunSession(conn, node, s.config.InitialView)
}
