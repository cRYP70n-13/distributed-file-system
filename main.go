package main

import (
	"bytes"
	"io"
	"log"
	"time"

	"distributed-file-system/cryptographer"
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

	// BUG: This encryption key should not be generated each time
	// Because it will screw the decryption, so we will generate it once
	// and keep re-using it, I have a dirty way of doing it but let's see
	encKey, _ := cryptographer.NewEncryptionKey()

	fileServerOpts := server.FileServerOpts{
		StorageRoot:       listenAddr + "_network",
		PathTransformFunc: store.CascadePathTransformFunc,
		Transport:         tcpTransport,
		BootstrapNodes:    nodes,
		EncKey:            encKey,
	}

	s := server.NewFileServer(fileServerOpts)

	tcpTransport.OnPeer = s.OnPeer

	return s
}

func main() {
    // TODO: we have to think about a way to announce that a new server has joined
    // and then we will need to think of how to bring him up to speed with the current state
    // In other words automagic peer discovery.
	s1 := makeServer(":3000")
	s2 := makeServer(":6000", ":3000")
	s3 := makeServer(":4000", ":3000", ":6000")
	s4 := makeServer(":7001", ":3000", ":4000", ":6000")

	// TODO: Here this time.Sleep looks like a bit hacky so we can use either a channel
	// to signal that we are ready to go or a sync broadcaster, we gonna find out in the future.
	go func() {
		log.Fatal(s1.Start())
	}()
	time.Sleep(2 * time.Second)

	go func() {
		log.Fatal(s2.Start())
	}()
	time.Sleep(2 * time.Second)

	go func() {
		log.Fatal(s3.Start())
	}()
	time.Sleep(2 * time.Second)

	go func() {
		log.Fatal(s4.Start())
	}()
	time.Sleep(2 * time.Second)

	key := "myImage.jpeg"

	content := bytes.NewReader([]byte("Hey Hey encrypt/decrypt testing but this is fucking buggy"))
	if err := s1.Store(key, content); err != nil {
		log.Fatal(err)
	}

	if err := s1.Delete(key); err != nil {
        log.Fatal(err)
	}

	r, err := s1.Get(key)
	if err != nil {
		log.Fatal(err)
	}

	b, err := io.ReadAll(r)
	if err != nil {
		log.Fatal(err)
	}
    log.Println("Server1 GOT: =====>", string(b))

	r2, err := s2.Get(key)
	if err != nil {
		log.Fatal(err)
	}

	b2, err := io.ReadAll(r2)
	if err != nil {
		log.Fatal(err)
	}
    log.Println("Server2 GOT: =====>", string(b2))
}
