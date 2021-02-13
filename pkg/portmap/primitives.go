package portmap

// IANA protocol number.
//
// See https://www.iana.org/assignments/protocol-numbers/protocol-numbers.xhtml
type Protocol uint8

const (
	ProtocolAny  = Protocol(0)
	ProtocolTCP  = Protocol(6)
	ProtocolUDP  = Protocol(17)
	ProtocolSCTP = Protocol(132)
)

// Port number (for protocols that have ports).
type Port uint16

const (
	// Any port.
	//
	// Usage:
	//
	// - when used in the port mapping request as the node port -
	//   indicates that the port mapping has to apply for all protocols;
	// - when used in the port mapping request as the gateway port -
	//   indicates that any available port can be chosen by the NAT server.
	//
	// This is designed to match the PCP semantics, so
	// see https://tools.ietf.org/html/rfc6887#section-11.1
	PortAny = Port(0)
)

// Lifetime in seconds.
type Lifetime uint32

const (
	// Indicate that the removal of the mapping is requested.
	LifetimeDelete = Lifetime(0)
)
