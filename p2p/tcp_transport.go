package p2p

import (
	"fmt"
	"net"
	"sync"
)

// TCPPeer represents the remote node over TCP established connection.
type TCPPeer struct {
	// conn is the underlying connectin of the peer
	conn net.Conn

	// If we dial and retreive a connection => outbound = true
	// If we accept and retreive a connection => outbound = false
	outbound bool
}

func NewTCPPeer(conn net.Conn, outbound bool) *TCPPeer {
	return &TCPPeer{
		conn:     conn,
		outbound: outbound,
	}
}

type TCPTransportOpts struct {
	ListenAddr    string
	HandShakeFunc HandShakeFunc
	Decoder       Decoder
}

type TCPTransport struct {
	TCPTransportOpts
	listener net.Listener

	mu    sync.RWMutex
	peers map[net.Addr]Peer
}

func NewTCPTransport(opts TCPTransportOpts) *TCPTransport {
	return &TCPTransport{
		TCPTransportOpts: opts,
	}
}

func (t *TCPTransport) ListenAndAccept() error {
	ln, err := net.Listen("tcp", t.ListenAddr)
	if err != nil {
		return err
	}

	t.listener = ln

	go t.acceptLoop()

	return nil
}

func (t *TCPTransport) acceptLoop() {
	for {
		conn, err := t.listener.Accept()
		if err != nil {
			fmt.Printf("TCP accept error: %s\n", err)
		}

		// TODO: Maybe I will create peers with a channel in the future
		peer := NewTCPPeer(conn, true)
		go t.handleConn(peer)
	}
}

type Temp struct{}

func (t *TCPTransport) handleConn(peer *TCPPeer) {
	if err := t.HandShakeFunc(peer); err != nil {
		// Drop the connection if the handshake failed.
		fmt.Printf("TCP handshake error: %s\n", err)
		peer.conn.Close()
		return
	}

	// Message read loop
	msg := &Temp{}
	for {
		if err := t.Decoder.Decode(peer.conn, msg); err != nil {
			fmt.Printf("TCP error: %s\n", err)
			continue
		}
	}
}
