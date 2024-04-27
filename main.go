package main

import (
	"distributedStore/p2p"
	"fmt"
)

func OnPeer(peer p2p.Peer) error {
    peer.Close()
    // fmt.Println("we are now allowing you to connect")
    return nil
}

func main() {
	tcpOpts := p2p.TCPTransportOpts{
		ListenAddr:    ":3000",
		HandShakeFunc: p2p.NOPHandshakeFunc,
		Decoder:       p2p.NOPDecoder{},
        OnPeer: OnPeer,
	}

	tr := p2p.NewTCPTransport(tcpOpts)

    go func() {
        for {
            msg := <-tr.Consume()
            fmt.Printf("======= %+v\n", msg)
        }
    }()

	if err := tr.ListenAndAccept(); err != nil {
		panic(err)
	}

	select {}
}
