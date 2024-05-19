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
	listener  net.Listener
	rpcStream chan RPC
	peer      map[net.Addr]Peer
}

func NewTCPTransport(opts TCPTransportOpts) *TCPTransport {
	return &TCPTransport{
		TCPTransportOpts: opts,
		rpcStream:        make(chan RPC),
	}
}

func (t *TCPTransport) ListenAndAccept() error {
	ln, err := net.Listen("tcp", t.ListenAddress)
	if err != nil {
		return err
	}

	t.listener = ln

	go t.acceptLoop()

	log.Printf("TCP transport listening on PORT: %s\n", t.ListenAddress)

	return nil
}

// Consume implements the transport interface, which will return
// a read only channel for reading the incoming messages received from antoher peer.
func (t *TCPTransport) Consume() <-chan RPC {
	return t.rpcStream
}

// Close closes the listener in case we got a signal to stop this.
func (t *TCPTransport) Close() error {
	return t.listener.Close()
}

// Dial implements the transport interface.
func (t *TCPTransport) Dial(addr string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}

	isOutBound := true
	go t.handleConn(conn, isOutBound)

	return nil
}

// Close implements the Peer interface.
func (p *TCPPeer) Close() error {
	return p.conn.Close()
}

// RemoteAddr implements the Peer interface
// And will return the remote addr of its underlying connection.
func (p *TCPPeer) RemoteAddr() net.Addr {
    return p.conn.RemoteAddr()
}

// Send implements the Peer interface
// and it will send a slice of bytes over the network.
func (p *TCPPeer) Send(data []byte) error {
    _, err := p.conn.Write(data)
    return err
}

func (t *TCPTransport) acceptLoop() {
	isOutbound := false
	for {
		conn, err := t.listener.Accept()
		if errors.Is(err, net.ErrClosed) {
			log.Println("Something went wrong with accepting the connection")
			return
		}

		if err != nil {
			log.Printf("TCP accept error %s\n", err)
		}

		log.Printf("We've got a new incomming connection %s\n", conn.RemoteAddr())

		go t.handleConn(conn, isOutbound)
	}
}

// FIXME: I don't like this maybe I can do better in the future, like creating peers with a channel or something.
func (t *TCPTransport) handleConn(conn net.Conn, isOutbound bool) {
	var err error

	defer func() {
		fmt.Printf("dropping peer connection for peer: %s, err => %s\n", conn.RemoteAddr(), err)
		conn.Close()
	}()

	peer := NewTCPPeer(conn, isOutbound)

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
