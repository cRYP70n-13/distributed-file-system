package p2p

import "net"

const (
	// MessageType is used to say that we are just sending a message.
	MessageType = 0x1
	// StreamType is used to say that we are about to stream something.
	StreamType = 0x2
)

// RPC represents/holds any data that's been sent over the network.
// (AKA each transport between two nodes).
type RPC struct {
	From    net.Addr
	Payload []byte
	Stream  bool
}
