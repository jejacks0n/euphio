package telnet

import (
	"fmt"
	"net"
	"time"

	"euphio/internal/app"
	"euphio/internal/config"
	"euphio/internal/session"
)

type Server struct {
	config config.TelnetConfig
	ln     net.Listener
}

func NewServer() *Server {
	return &Server{
		config: app.Config.LoginServers.Telnet,
	}
}

func (s *Server) ListenAndServe() error {
	app.Logger.Info("Telnet server listening", "port", s.config.Port)

	var err error
	s.ln, err = net.Listen("tcp", fmt.Sprintf(":%d", s.config.Port))
	if err != nil {
		return err
	}
	defer s.ln.Close()

	for {
		conn, err := s.ln.Accept()
		if err != nil {
			// Check if the error is due to the listener being closed
			if opErr, ok := err.(*net.OpError); ok && opErr.Err.Error() == "use of closed network connection" {
				return nil
			}
			app.Logger.Error("Telnet accept error", "err", err)
			continue
		}
		go s.handleConnection(conn)
	}
}

func (s *Server) Stop() error {
	if s.ln != nil {
		return s.ln.Close()
	}
	return nil
}

func (s *Server) handleConnection(conn net.Conn) {
	node, err := app.Nodes.Acquire()
	if err != nil {
		app.Logger.Warn("Connection rejected: system full", "addr", conn.RemoteAddr())
		conn.Close()
		return
	}
	defer app.Nodes.Release(node.ID)

	logger := app.Logger.With("node", node.ID)

	// Wrap the connection with our Telnet Connection handler
	telnetConn := NewConnection(conn, logger)

	// Assign connection to node for cross-node comms
	node.Conn = telnetConn

	defer telnetConn.Close()
	defer logger.Info("Telnet connection closed", "addr", telnetConn.RemoteAddr())

	logger.Debug("Telnet connection from", "addr", telnetConn.RemoteAddr())

	// Initiate Negotiation
	telnetConn.SendWill(Echo)
	telnetConn.SendWill(SGA)
	telnetConn.SendDo(NAWS)
	telnetConn.SendDo(TType)

	// Start a background logger to report connection details once negotiation settles
	telnetConn.StartNegotiationLogger(2 * time.Second)

	// Hand off to the session manager
	// RunSession blocks until the user disconnects
	session.RunSession(telnetConn, node, s.config.InitialView)
}
