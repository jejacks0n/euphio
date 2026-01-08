package telnet

import (
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"

	"euphio/internal/app"
)

// OptionState represents the state of a Telnet option
type OptionState int

const (
	OptionDisabled OptionState = iota
	OptionEnabled
)

type Connection struct {
	conn   net.Conn
	reader *Reader
	writer *Writer

	// State tracking
	localOptions  map[byte]OptionState // Options WE have agreed to (WILL)
	remoteOptions map[byte]OptionState // Options THE CLIENT has agreed to (DO)

	// Negotiation tracking (to avoid loops)
	sentWill map[byte]bool
	sentDo   map[byte]bool

	mu sync.RWMutex

	// Terminal Info
	TerminalType string
	WindowWidth  int
	WindowHeight int
}

func NewConnection(conn net.Conn) *Connection {
	c := &Connection{
		conn:          conn,
		localOptions:  make(map[byte]OptionState),
		remoteOptions: make(map[byte]OptionState),
		sentWill:      make(map[byte]bool),
		sentDo:        make(map[byte]bool),
	}
	c.reader = NewReader(conn, c)
	c.writer = NewWriter(conn)
	return c
}

func (c *Connection) Read(p []byte) (n int, err error) {
	return c.reader.Read(p)
}

func (c *Connection) Write(p []byte) (n int, err error) {
	return c.writer.Write(p)
}

func (c *Connection) Close() error {
	return c.conn.Close()
}

func (c *Connection) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

// HandleCommand implements the CommandHandler interface
func (c *Connection) HandleCommand(cmd, option byte) {
	cmdName := CommandNames[cmd]
	optName := OptionNames[option]
	if optName == "" && (cmd == WILL || cmd == WONT || cmd == DO || cmd == DONT) {
		optName = fmt.Sprintf("Unknown(%d)", option)
	}

	app.Logger.Debug("Telnet command [IN]", "cmd", cmdName, "opt", optName)

	switch cmd {
	case DO:
		// Client wants US to do something
		switch option {
		case Echo:
			if !c.IsLocalOptionEnabled(Echo) {
				c.EnableLocalOption(Echo)
				c.SendWill(Echo)
			}
		case SGA:
			if !c.IsLocalOptionEnabled(SGA) {
				c.EnableLocalOption(SGA)
				c.SendWill(SGA)
			}
		case TransmitBinary:
			if !c.IsLocalOptionEnabled(TransmitBinary) {
				c.EnableLocalOption(TransmitBinary)
				c.SendWill(TransmitBinary)
			}
		default:
			c.SendWont(option)
		}

	case DONT:
		// Client wants us NOT to do something
		switch option {
		case Echo:
			if c.IsLocalOptionEnabled(Echo) {
				c.DisableLocalOption(Echo)
				c.SendWont(Echo)
			}
		default:
			c.DisableLocalOption(option)
			c.SendWont(option)
		}

	case WILL:
		// Client wants to do something
		switch option {
		case SGA:
			if !c.IsRemoteOptionEnabled(SGA) {
				c.EnableRemoteOption(SGA)
				c.SendDo(SGA)
			}
		case TransmitBinary:
			if !c.IsRemoteOptionEnabled(TransmitBinary) {
				c.EnableRemoteOption(TransmitBinary)
				c.SendDo(TransmitBinary)
			}
		case NAWS:
			if !c.IsRemoteOptionEnabled(NAWS) {
				c.EnableRemoteOption(NAWS)
				c.SendDo(NAWS)
				// Client will send SB NAWS ... SE automatically after this
			}
		case TType:
			if !c.IsRemoteOptionEnabled(TType) {
				c.EnableRemoteOption(TType)
				c.SendDo(TType)
				// We must explicitly ask for the terminal type
				c.SendSubNegotiation(TType, []byte{SEND})
			}
		default:
			c.SendDont(option)
		}

	case WONT:
		// Client refuses to do something
		if c.IsRemoteOptionEnabled(option) {
			c.DisableRemoteOption(option)
			c.SendDont(option)
		}

	// Simple Commands (no option)
	case AYT:
		// Are You There?
		c.writer.Write([]byte("\r\n[Yes]\r\n"))
	case IP:
		// Interrupt Process
		app.Logger.Info("Telnet IP (Interrupt Process) received")
		// TODO: Signal application to interrupt
	case AO:
		// Abort Output
		app.Logger.Info("Telnet AO (Abort Output) received")
		// TODO: Signal application to flush output buffers
	case BRK:
		// Break
		app.Logger.Info("Telnet BRK (Break) received")
	}
}

// HandleSubNegotiation implements the CommandHandler interface
func (c *Connection) HandleSubNegotiation(option byte, data []byte) {
	optName := OptionNames[option]
	app.Logger.Debug("Telnet sub-negotiation [IN]", "opt", optName, "len", len(data))

	switch option {
	case NAWS:
		// RFC 1073: IAC SB NAWS <16-bit width> <16-bit height> IAC SE
		if len(data) >= 4 {
			width := int(binary.BigEndian.Uint16(data[0:2]))
			height := int(binary.BigEndian.Uint16(data[2:4]))

			c.mu.Lock()
			c.WindowWidth = width
			c.WindowHeight = height
			c.mu.Unlock()

			app.Logger.Debug("Telnet window size", "dims", fmt.Sprintf("%dx%d", width, height))
		}
	case TType:
		// RFC 1091: IAC SB TTYPE IS <terminal-type-string> IAC SE
		if len(data) > 1 && data[0] == IS {
			ttype := string(data[1:])

			c.mu.Lock()
			c.TerminalType = ttype
			c.mu.Unlock()

			app.Logger.Debug("Telnet terminal type", "type", ttype)
		}
	}
}

// EnableLocalOption marks an option as enabled for the server side
func (c *Connection) EnableLocalOption(option byte) {
	c.localOptions[option] = OptionEnabled
}

// DisableLocalOption marks an option as disabled for the server side
func (c *Connection) DisableLocalOption(option byte) {
	c.localOptions[option] = OptionDisabled
}

// EnableRemoteOption marks an option as enabled for the client side
func (c *Connection) EnableRemoteOption(option byte) {
	c.remoteOptions[option] = OptionEnabled
}

// DisableRemoteOption marks an option as disabled for the client side
func (c *Connection) DisableRemoteOption(option byte) {
	c.remoteOptions[option] = OptionDisabled
}

// IsLocalOptionEnabled checks if we have enabled a specific option
func (c *Connection) IsLocalOptionEnabled(option byte) bool {
	return c.localOptions[option] == OptionEnabled
}

// IsRemoteOptionEnabled checks if the client has enabled a specific option
func (c *Connection) IsRemoteOptionEnabled(option byte) bool {
	return c.remoteOptions[option] == OptionEnabled
}

func (c *Connection) logCommand(direction string, cmd, option byte) {
	cmdName := CommandNames[cmd]
	optName := OptionNames[option]
	if optName == "" {
		optName = fmt.Sprintf("Unknown(%d)", option)
	}
	app.Logger.Debug(fmt.Sprintf("Telnet command [%s]", direction), "cmd", cmdName, "opt", optName)
}

// SendCommand sends a raw Telnet command
func (c *Connection) SendCommand(cmd, option byte) error {
	c.logCommand("OUT", cmd, option)
	return c.writer.WriteCommand(cmd, option)
}

// SendWill sends IAC WILL <option>
func (c *Connection) SendWill(option byte) error {
	// If we already sent WILL for this option, don't send it again
	if c.sentWill[option] {
		return nil
	}
	c.sentWill[option] = true
	c.logCommand("OUT", WILL, option)
	return c.writer.WriteCommand(WILL, option)
}

// SendWont sends IAC WONT <option>
func (c *Connection) SendWont(option byte) error {
	// We can always send WONT to be safe, or track it too.
	// Usually WONT is final, so tracking isn't as critical for loops, but good for noise.
	c.sentWill[option] = false // Reset WILL state
	c.logCommand("OUT", WONT, option)
	return c.writer.WriteCommand(WONT, option)
}

// SendDo sends IAC DO <option>
func (c *Connection) SendDo(option byte) error {
	if c.sentDo[option] {
		return nil
	}
	c.sentDo[option] = true
	c.logCommand("OUT", DO, option)
	return c.writer.WriteCommand(DO, option)
}

// SendDont sends IAC DONT <option>
func (c *Connection) SendDont(option byte) error {
	c.sentDo[option] = false // Reset DO state
	c.logCommand("OUT", DONT, option)
	return c.writer.WriteCommand(DONT, option)
}

// SendSubNegotiation sends a sub-negotiation sequence
func (c *Connection) SendSubNegotiation(option byte, data []byte) error {
	optName := OptionNames[option]
	app.Logger.Debug("Telnet sub-negotiation [OUT]", "opt", optName, "len", len(data))
	return c.writer.WriteSubNegotiation(option, data)
}

// StartNegotiationLogger starts a goroutine that waits for negotiation to complete
// (or timeout) and then logs the connection details.
func (c *Connection) StartNegotiationLogger(timeout time.Duration) {
	go func() {
		deadline := time.Now().Add(timeout)
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			if time.Now().After(deadline) {
				break
			}

			c.mu.RLock()
			done := c.TerminalType != "" && c.WindowWidth > 0
			c.mu.RUnlock()

			if done {
				break
			}

			<-ticker.C
		}

		c.LogConnectionInfo()
	}()
}

// LogConnectionInfo logs the summary of the connection info
func (c *Connection) LogConnectionInfo() {
	c.mu.RLock()
	defer c.mu.RUnlock()

	ttype := c.TerminalType
	if ttype == "" {
		ttype = "UNKNOWN"
	}

	dims := fmt.Sprintf("%dx%d", c.WindowWidth, c.WindowHeight)
	if c.WindowWidth == 0 || c.WindowHeight == 0 {
		dims = "UNKNOWN"
	}

	app.Logger.Info("Telnet connection established",
		"addr", c.RemoteAddr(),
		"terminal", ttype,
		"window", dims,
	)
}
