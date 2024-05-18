package p2p

import (
	"errors"
	"fmt"
	"log"
	"net"
)

// TCPPeer represents the remote node over TCP established connection.
type TCPPeer struct {
	// conn is the underlying connection of the peer
	conn net.Conn

	// if we dial and retreive a connection => outbound = true
	// if we accept and retreive a connection => outbound = false
	outbound bool
}

func NewTCPPeer(conn net.Conn, outbound bool) *TCPPeer {
	return &TCPPeer{
		conn:     conn,
		outbound: outbound,
	}
}

type TCPTransportOpts struct {
	ListenAddress string
	HandShakeFunc HandShakeFunc
	Decoder       Decoder
	OnPeer        func(Peer) error
}

type TCPTransport struct {
	TCPTransportOpts
	listener net.Listener
	rpcStream  chan RPC
	peer     map[net.Addr]Peer
}

func NewTCPTransport(opts TCPTransportOpts) *TCPTransport {
	return &TCPTransport{
		TCPTransportOpts: opts,
		rpcStream:          make(chan RPC),
	}
}

func (t *TCPTransport) ListenAndAccept() error {
	ln, err := net.Listen("tcp", t.ListenAddress)
	if err != nil {
		return err
	}

	t.listener = ln

	go t.acceptLoop()

	return nil
}

// Consume implements the transport interface, which will return
// a read only channel for reading the incoming messages received from antoher peer.
func (t *TCPTransport) Consume() <-chan RPC {
	return t.rpcStream
}

// Close implements the peer interface.
func (p *TCPPeer) Close() error {
	return p.conn.Close()
}

func (t *TCPTransport) acceptLoop() {
	for {
		conn, err := t.listener.Accept()
		if err != nil {
			log.Printf("TCP accept error %s\n", err)
		}

		go t.handleConn(conn)
	}
}

// FIXME: I don't like this maybe I can do better in the future
func (t *TCPTransport) handleConn(conn net.Conn) {
	var err error

	defer func() {
        fmt.Printf("dropping peer connection for peer: %s, err => %s\n", conn.RemoteAddr(), err)
		conn.Close()
	}()

	peer := NewTCPPeer(conn, true)

	if err = t.HandShakeFunc(peer); err != nil {
		return
	}

	if t.OnPeer != nil {
		if err = t.OnPeer(peer); err != nil {
			return
		}
	}

	rpc := RPC{}
	for {
		err := t.Decoder.Decode(peer.conn, &rpc)
		if errors.Is(err, net.ErrClosed) {
            fmt.Printf("TCP read error: %s\n", err.Error())
			return
		}
		if err != nil {
			fmt.Printf("TCP read error: %s\n", err.Error())
			continue
		}

		rpc.From = peer.conn.RemoteAddr()
		t.rpcStream <- rpc
	}
}
