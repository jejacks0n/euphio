package nodes

import (
	"fmt"
	"net"

	"euphio/internal/store"
)

type TerminalInfo struct {
	Type   string
	Width  int
	Height int
}

type Connection interface {
	Send(msg string) error
	RemoteAddr() net.Addr
	GetTerminalInfo() TerminalInfo
	IsUTF8() bool
}

type Node struct {
	ID   int
	Conn Connection
	User *store.User
}

func (n *Node) String() string {
	if n.Conn == nil {
		return fmt.Sprintf("Node %d (Disconnected)", n.ID)
	}
	return fmt.Sprintf("Node %d (%s)", n.ID, n.Conn.RemoteAddr())
}
