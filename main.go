package main

import (
	"bytes"
	"log"

	"distributed-file-system/p2p"
	"distributed-file-system/server"
	"distributed-file-system/store"
)

func makeServer(listenAddr string, nodes ...string) *server.FileServer {
	tcpTransportOps := p2p.TCPTransportOpts{
		ListenAddress: listenAddr,
		HandShakeFunc: p2p.NoOpHandshake,
		Decoder:       p2p.NoOpDecoder{},
	}

	tcpTransport := p2p.NewTCPTransport(tcpTransportOps)

	fileServerOpts := server.FileServerOpts{
		StorageRoot:       listenAddr + "_network",
		PathTransformFunc: store.CascadePathTransformFunc,
		Transport:         tcpTransport,
		BootstrapNodes:    nodes,
	}

	s := server.NewFileServer(fileServerOpts)

	tcpTransport.OnPeer = s.OnPeer

	return s

}

func main() {
	s1 := makeServer(":3000", "")
	s2 := makeServer(":4000", ":3000")

	go func() {
		log.Fatal(s1.Start())
	}()

	_ = s2.Start()

	content := bytes.NewReader([]byte("Hello from a big data file"))
	if err := s1.StoreFile("key", content); err != nil {
		panic(err)
	}
}
