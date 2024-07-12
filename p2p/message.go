package p2p

import "net"

const (
	StreamType  = 0x2
	MessageType = 0x1
)

// RPC represents/holds any data that's been sent over the network.
// (AKA each transport between two nodes).
type RPC struct {
	From    net.Addr
	Payload []byte
	Stream  bool
}
