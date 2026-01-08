package network

import (
	"euphio/internal/network/telnet"
)

func NewTelnet() *telnet.Server {
	return telnet.NewServer()
}
