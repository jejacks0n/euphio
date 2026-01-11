package nodes

import (
	"fmt"
	"net"
)

type Connection interface {
	Send(msg string) error
	RemoteAddr() net.Addr
}

type Node struct {
	ID   int
	Conn Connection
	// We can add more fields here like User, etc.
}

func (n *Node) String() string {
	if n.Conn == nil {
		return fmt.Sprintf("Node %d (Disconnected)", n.ID)
	}
	return fmt.Sprintf("Node %d (%s)", n.ID, n.Conn.RemoteAddr())
}
