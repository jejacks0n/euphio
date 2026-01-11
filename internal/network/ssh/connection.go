package ssh

import (
	"io"
	"net"
	"sync"

	"github.com/gliderlabs/ssh"

	"euphio/internal/nodes"
)

// Connection adapts ssh.Session to nodes.Connection interface
type Connection struct {
	sess   ssh.Session
	mu     sync.RWMutex
	width  int
	height int
	term   string
}

func NewConnection(sess ssh.Session) *Connection {
	c := &Connection{sess: sess}

	pty, winCh, isPty := sess.Pty()
	if isPty {
		c.term = pty.Term
		c.width = pty.Window.Width
		c.height = pty.Window.Height

		go func() {
			for win := range winCh {
				c.mu.Lock()
				c.width = win.Width
				c.height = win.Height
				c.mu.Unlock()
			}
		}()
	}

	return c
}

func (c *Connection) Send(msg string) error {
	_, err := io.WriteString(c.sess, msg+"\r\n")
	return err
}

func (c *Connection) RemoteAddr() net.Addr {
	return c.sess.RemoteAddr()
}

func (c *Connection) GetTerminalInfo() nodes.TerminalInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return nodes.TerminalInfo{
		Type:   c.term,
		Width:  c.width,
		Height: c.height,
	}
}

func (c *Connection) Close() error {
	return c.sess.Close()
}

func (c *Connection) Read(p []byte) (n int, err error) {
	return c.sess.Read(p)
}

func (c *Connection) Write(p []byte) (n int, err error) {
	return c.sess.Write(p)
}
