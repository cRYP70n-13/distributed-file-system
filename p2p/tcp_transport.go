package p2p

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
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

// Close implements the Peer interface.
func (p *TCPPeer) Close() error {
	return p.conn.Close()
}

type TCPTransportOpts struct {
	ListenAddr    string
	HandShakeFunc HandShakeFunc
	Decoder       Decoder
	OnPeer        func(Peer) error
}

type TCPTransport struct {
	TCPTransportOpts
	listener  net.Listener
	rpcStream chan RPC
}

func NewTCPTransport(opts TCPTransportOpts) *TCPTransport {
	return &TCPTransport{
		TCPTransportOpts: opts,
		rpcStream:        make(chan RPC),
	}
}

// Consume implements the transport interface, will return just a read-only chan
// for reading received message from another peer in our network.
func (t *TCPTransport) Consume() <-chan RPC {
	return t.rpcStream
}

func (t *TCPTransport) ListenAndAccept() error {
	ln, err := net.Listen("tcp", t.ListenAddr)
	if err != nil {
		return err
	}

	slog.Info("Server is running on: ", "tcp_transport", ln.Addr().String())
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

func (t *TCPTransport) handleConn(peer *TCPPeer) {
	var err error
	defer func() {
		fmt.Printf("dropping peer conncection: %s\n", err)
		peer.Close()
	}()

	if err = t.HandShakeFunc(peer); err != nil {
		// Drop the connection if the handshake failed.
		fmt.Printf("TCP handshake error: %s\n", err)
		peer.conn.Close()
		return
	}

	if t.OnPeer != nil {
		if err = t.OnPeer(peer); err != nil {
			return
		}
	}

	// Message read loop
	rpc := RPC{}
	for {
		err := t.Decoder.Decode(peer.conn, &rpc)
        if errors.Is(err, net.ErrClosed) {
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
