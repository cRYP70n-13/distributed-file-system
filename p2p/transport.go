package p2p

// Peer is an interface that represents the remote node.
type Peer interface {
    Close() error
}

// Transport is anything that handles the communication layer
// between nodes in our network. This can be of the form TCP/UDP/Websockets ...
type Transport interface {
	ListenAndAccept() error
	Consume() <-chan RPC
}
