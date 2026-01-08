package telnet

// A good place to start with the Telnet protocol is Wikipedia:
// https://en.wikipedia.org/wiki/Telnet
//
// This Telnet implementation attempts to be "complete enough" to cover what is
// generally seen out in the wild in relation to Bulletin Board System software
// and modern terms and MUD clients.
//
// This was adapted from https://github.com/NuSkooler/telnet-socket
// Copyright (c) 2019-2022, Bryan D. Ashby
// All rights reserved.
//
// RFCs of particular interest:
// - RFC 854  : Telnet Protocol Specification
// - RFC 856  : Telnet Binary Transmission
// - RFC 857  : Telnet Echo Option
// - RFC 858  : Telnet Suppress Go Ahead Option
// - RFC 859  : Telnet Status Option
// - RFC 860  : Telnet Timing Mark Option
// - RFC 861  : Telnet Extended Options: List Option
// - RFC 856  : Telnet End of Record Option
// - RFC 1073 : Telnet Window Size Option
// - RFC 1572 : Telnet Environment Option (replaces RFC 1404)

const (
	// RFC 854: Telnet Protocol Specification
	SE   byte = 240 // Sub negotiation End
	NOP  byte = 241 // No Operation
	DM   byte = 242 // Data Mark
	BRK  byte = 243 // Break
	IP   byte = 244 // Interrupt Process
	AO   byte = 245 // Abort Output
	AYT  byte = 246 // Are You There?
	EC   byte = 247 // Erase Character
	EL   byte = 248 // Erase Line
	GA   byte = 249 // Go Ahead
	SB   byte = 250 // Sub negotiation Begin
	WILL byte = 251 // Will
	WONT byte = 252 // Won't
	DO   byte = 253 // Do
	DONT byte = 254 // Don't
	IAC  byte = 255 // Interpret As Command

	// Sub-negotiation Commands
	IS      byte = 0
	SEND    byte = 1
	INFO    byte = 2
	VAR     byte = 0
	VALUE   byte = 1
	ESC     byte = 2
	USERVAR byte = 3
	MSSPVAR byte = 1
	MSSPVAL byte = 2

	// Telnet Options
	TransmitBinary byte = 0   // RFC 854
	Echo           byte = 1   // RFC 857
	SGA            byte = 3   // RFC 858 - Suppress Go Ahead
	Status         byte = 5   // RFC 859
	TimingMark     byte = 6   // RFC 860
	TType          byte = 24  // RFC 930 - Terminal Type
	EOR            byte = 25  // RFC 885 - End of Record
	TacacsUserID   byte = 26  // RFC 927
	OutputMarking  byte = 27  // RFC 933
	NAWS           byte = 31  // RFC 1073 - Negotiate About Window Size
	TerminalSpeed  byte = 32  // RFC 1079
	Linemode       byte = 34  // RFC 1148
	NewEnvironOld  byte = 36  // Deprecated RFC 1408 'NEW-ENVIRON'
	Encrypt        byte = 38  // RFC 2496
	NewEnviron     byte = 39  // RFC 1572 'NEW-ENVIRON'
	MSSP           byte = 70  // MUD Server Status Protocol
	GMCP           byte = 201 // Generic MUD Communication Protocol
	Exopl          byte = 255 // RFC 860 - Extended Options List
)

// CommandNames maps Telnet command bytes to their string representation.
var CommandNames = map[byte]string{
	SE:   "SE",
	NOP:  "NOP",
	DM:   "DM",
	BRK:  "BRK",
	IP:   "IP",
	AO:   "AO",
	AYT:  "AYT",
	EC:   "EC",
	EL:   "EL",
	GA:   "GA",
	SB:   "SB",
	WILL: "WILL",
	WONT: "WONT",
	DO:   "DO",
	DONT: "DONT",
	IAC:  "IAC",
}

// OptionNames maps Telnet option bytes to their string representation.
var OptionNames = map[byte]string{
	TransmitBinary: "TransmitBinary",
	Echo:           "Echo",
	SGA:            "SGA",
	Status:         "Status",
	TimingMark:     "TimingMark",
	TType:          "TType",
	EOR:            "EOR",
	TacacsUserID:   "TacacsUserID",
	OutputMarking:  "OutputMarking",
	NAWS:           "NAWS",
	TerminalSpeed:  "TerminalSpeed",
	Linemode:       "Linemode",
	NewEnvironOld:  "NewEnvironOld",
	Encrypt:        "Encrypt",
	NewEnviron:     "NewEnviron",
	MSSP:           "MSSP",
	GMCP:           "GMCP",
	Exopl:          "Exopl",
}
