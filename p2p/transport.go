package p2p

// Transport is anything that handles the communication layer
// between the nodes in the network, can be of form TCP/UDP/WebSockets...
type Transport interface {
	ListenAndAccept() error
	Consume() <-chan RPC
}

// Peer is an interface that represents the remote node.
type Peer interface {
	Close() error
}
