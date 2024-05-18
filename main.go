package main

import (
	"distributed-file-system/p2p"
	"fmt"
	"log"
)

// TODO: I can even control this with an env var flag to put it in debug mode for example.
func OnPeer(p p2p.Peer) error {
    p.Close()
	// fmt.Println("Do some logic with the peer son of a mother")
	return nil
}

func main() {
	opts := p2p.TCPTransportOpts{
		ListenAddress: ":4000",
		HandShakeFunc: p2p.NoOpHandshake,
		Decoder:       &p2p.NoOpDecoder{},
		OnPeer:        OnPeer,
	}

	tr := p2p.NewTCPTransport(opts)

	go func() {
		for {
			msg := <-tr.Consume()
			fmt.Printf("%+v\n", msg)
		}
	}()

	if err := tr.ListenAndAccept(); err != nil {
		panic(err)
	}
	log.Println("Server is up and running")

	select {}
}
