package main

import (
	"distributed-file-system/p2p"
	"distributed-file-system/server"
	"distributed-file-system/store"
	"fmt"
	"log"
	"time"
)

// TODO: I can even control this with an env var flag to put it in debug mode for example.
func OnPeer(p p2p.Peer) error {
	// p.Close()
	fmt.Println("Do some logic with the peer son of a mother")
	return nil
}

func main() {
	tcpTransportOps := p2p.TCPTransportOpts{
		ListenAddress: ":4000",
		HandShakeFunc: p2p.NoOpHandshake,
		Decoder:       p2p.NoOpDecoder{},
		OnPeer:        OnPeer,
	}

	tcpTransport := p2p.NewTCPTransport(tcpTransportOps)
	fileServerOpts := server.FileServerOpts{
		StorageRoot:       "4000_network",
		PathTransformFunc: store.CascadePathTransformFunc,
		Transport:         tcpTransport,
		BootstrapNodes:    []string{":3000"},
	}

	s := server.NewFileServer(fileServerOpts)

	if err := s.Start(); err != nil {
		log.Fatal(err)
	}

	go func() {
		// NOTE: This is just to test if we are closing and cleaning up correctly, and it's working fine so far
		time.Sleep(5 * time.Minute)
		s.Stop()
	}()

	select {}
}
