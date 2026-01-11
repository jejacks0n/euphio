package nodes

import (
	"fmt"
	"sync"
)

type Manager struct {
	mu       sync.RWMutex
	maxNodes int
	nodes    []*Node
}

func NewManager(maxNodes int) *Manager {
	if maxNodes <= 0 {
		maxNodes = 10
	}
	return &Manager{
		maxNodes: maxNodes,
		nodes:    make([]*Node, maxNodes),
	}
}

func (m *Manager) Acquire() (*Node, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, n := range m.nodes {
		if n == nil {
			node := &Node{
				ID: i + 1,
			}
			m.nodes[i] = node
			return node, nil
		}
	}
	return nil, fmt.Errorf("system full")
}

func (m *Manager) Release(id int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if id < 1 || id > m.maxNodes {
		return
	}
	m.nodes[id-1] = nil
}

func (m *Manager) Get(id int) *Node {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if id < 1 || id > m.maxNodes {
		return nil
	}
	return m.nodes[id-1]
}

func (m *Manager) Broadcast(msg string) {
	m.BroadcastExcept(msg, -1)
}

func (m *Manager) BroadcastExcept(msg string, exceptID int) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, n := range m.nodes {
		if n != nil && n.Conn != nil && n.ID != exceptID {
			// Ignore errors for broadcast
			n.Conn.Send(msg)
		}
	}
}
