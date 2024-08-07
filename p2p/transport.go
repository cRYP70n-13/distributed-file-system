package p2p

import "net"

// Transport is anything that handles the communication layer
// between the nodes in the network, can be of form TCP/UDP/WebSockets...
type Transport interface {
	ListenAndAccept() error
	Consume() <-chan RPC
	Close() error
	Dial(string) error
	Addr() string
}

// Peer is an interface that represents the remote node.
type Peer interface {
	net.Conn

	// Send will send the []byte over the network
	Send([]byte) error

	// CloseStream will close the curent stream allowing us
	// to move forward.
	CloseStream()
}
