package p2p

import (
	"net"
)

// Message represents/holds any data that's been sent over
// each transport between two nodes in the network.
type RPC struct {
	From    net.Addr
	Payload []byte
}

// String is the method that we are using to print some pretty little messages
// in case we are doing some debugging fmt.Print... for this Message's Payload.
// func (m RPC) String() string {
// 	return fmt.Sprintf("%s Coming from %s", string(m.Payload), m.From.String())
// }
