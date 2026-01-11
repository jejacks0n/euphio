package network

import (
	"euphio/internal/network/ssh"
)

func NewSSH() *ssh.Server {
	return ssh.NewServer()
}
