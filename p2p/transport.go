package p2p

import "net"

// Transport is anything that handles the communication layer
// between the nodes in the network, can be of form TCP/UDP/WebSockets...
type Transport interface {
	ListenAndAccept() error
	Consume() <-chan RPC
	Close() error
	Dial(string) error
}

// Peer is an interface that represents the remote node.
type Peer interface {
    net.Conn
    Send([]byte) error
}
