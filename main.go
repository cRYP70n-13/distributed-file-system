package main

import (
	"distributedStore/p2p"
)

func main() {
    tcpOpts := p2p.TCPTransportOpts{
        ListenAddr: ":3000",
        HandShakeFunc: p2p.NOPHandshakeFunc,
    }

	tr := p2p.NewTCPTransport(tcpOpts)

	if err := tr.ListenAndAccept(); err != nil {
		panic(err)
	}

	select {}
}
