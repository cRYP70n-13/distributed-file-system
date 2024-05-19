package server

import (
	"bytes"
	"encoding/gob"
	"io"
	"log"
	"net"
	"sync"

	"distributed-file-system/p2p"
	"distributed-file-system/store"
)

type FileServerOpts struct {
	StorageRoot       string
	PathTransformFunc store.TransaformFunc
	Transport         p2p.Transport
	BootstrapNodes    []string
}

type FileServer struct {
	FileServerOpts

	mu    sync.RWMutex
	peers map[net.Addr]p2p.Peer

	store  *store.Store
	doneCh chan struct{}
}

func NewFileServer(opts FileServerOpts) *FileServer {
	storeOpts := store.StoreOpts{
		Root:           opts.StorageRoot,
		TransaformFunc: opts.PathTransformFunc,
	}
	return &FileServer{
		FileServerOpts: opts,
		store:          store.NewStore(storeOpts),
		doneCh:         make(chan struct{}),
		// Maybe add and remove peers with channels, and get rid of the mutex :)
		peers: make(map[net.Addr]p2p.Peer),
	}
}

func (s *FileServer) Start() error {
	if err := s.Transport.ListenAndAccept(); err != nil {
		return err
	}

	if err := s.bootstrapNetwork(); err != nil {
		panic(err)
	}

	s.loop()

	return nil
}

func (s *FileServer) StoreFile(key string, r io.Reader) error {
	// 1. Store the file in disk first
	// 2. Broadcast the file to all the known peers in the network.
	// 3. Return a good response to the client :)
	if err := s.store.Write(key, r); err != nil {
		return err
	}

    // payload := Payload{
    //     Key: key,
    //     Data: ,
    // }
    // if err := s.broadcast(); err != nil {
    //     return err
    // }

	return nil
}

type Payload struct {
	Key  string
	Data []byte
}

func (s *FileServer) OnPeer(peer p2p.Peer) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.peers[peer.RemoteAddr()] = peer
	log.Printf("New peer connected successfully: %s\n", peer.RemoteAddr())

	return nil
}

func (s *FileServer) Stop() {
	close(s.doneCh)
}

func (s *FileServer) broadcast(p Payload) error {
    buf := new(bytes.Buffer)

	for _, p := range s.peers {
        if err := gob.NewEncoder(buf).Encode(p); err != nil {
            return err
        }

        if err := p.Send(buf.Bytes()); err != nil {
            return err
        }
	}
	return nil
}

func (s *FileServer) loop() {
	defer func() {
		log.Println("Sorry to tell you but we are done here sir!!")
		err := s.Transport.Close()
		if err != nil {
			log.Fatal("something went wrong while closing the connection", err)
		}
	}()

	for {
		select {
		case msg := <-s.Transport.Consume():
			log.Println("We've got a RPC => ", msg.From, string(msg.Payload))
		case <-s.doneCh:
			return
		}
	}
}

func (s *FileServer) bootstrapNetwork() error {
	for _, addr := range s.BootstrapNodes {
		if len(addr) == 0 {
			continue
		}

		address := addr
		go func(addr string) {
			log.Println("Bootstrapping the network... trying: ", addr)
			if err := s.Transport.Dial(addr); err != nil {
				log.Println("dial error: ", err)
			}
		}(address)
	}

	return nil
}
