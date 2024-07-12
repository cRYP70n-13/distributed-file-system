package main

import (
	"bytes"
	"log"
	"strconv"
	"time"

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
	s1 := makeServer(":3000")
	s2 := makeServer(":4000", ":3000")

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

	for i := 0; i < 10; i++ {
		content := bytes.NewReader([]byte("Hello Otmane kimdil is preparing for his new Senior Software engineer role" + strconv.Itoa(i)))
		if err := s1.Store("myprivatedata"+strconv.Itoa(i), content); err != nil {
			panic(err)
		}
		// time.Sleep(time.Millisecond)
	}

	// r, err := s1.Get("myprivatedata")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	//
	// b, err := io.ReadAll(r)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// fmt.Println(string(b))

	select {}
}
