package server

import (
	"bytes"
	"encoding/gob"
	"fmt"
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

func (s *FileServer) Store(key string, r io.Reader) error {
	// buf := new(bytes.Buffer)
	// tee := io.TeeReader(r, buf)

    msg := Message{
        Payload: []byte("storagekey"),
    }
    buf := new(bytes.Buffer)
    if err := gob.NewEncoder(buf).Encode(&msg); err != nil {
        return err
    }

    for _, peer := range s.peers {
        if err := peer.Send(buf.Bytes()); err != nil {
            return err
        }
    }

    fmt.Println("****** DONE WITH SENDING THE FIRST MESSAGE NOW SENDING THE SECOND *******")
    payload := []byte("This is a large file")
    for _, peer := range s.peers {
        if err := peer.Send(payload); err != nil {
            return err
        }
    }

    return nil

	// if err := s.store.Write(key, tee); err != nil {
	// 	return err
	// }
	// _, err := io.Copy(buf, r)
	// if err != nil {
	// 	return err
	// }

	// p := &DataMessage{
	// 	Key:  key,
	// 	Data: buf.Bytes(),
	// }

	// return s.broadcast(&Message{
	// 	From:    "TODO",
	// 	Payload: p,
	// })
}

type Message struct {
	Payload any
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

func (s *FileServer) broadcast(msg *Message) error {
	peers := []io.Writer{}
	for _, peer := range s.peers {
		peers = append(peers, peer)
	}

	mw := io.MultiWriter(peers...)

	fmt.Println("Broadcasting the msg", msg.Payload)
	return gob.NewEncoder(mw).Encode(msg)
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
		// TODO: Maybe here I can even do this in another goroutine
		case rpc := <-s.Transport.Consume():
			var msg Message
			if err := gob.NewDecoder(bytes.NewReader(rpc.Payload)).Decode(&msg); err != nil {
				log.Println(err)
			}

            peer, ok := s.peers[rpc.From]
            if !ok {
                log.Panicln("Peer not found in the server's peer map")
            }

            fmt.Println("Going to read now")
            b := make([]byte, 1000)
            if _, err := peer.Read(b); err != nil {
                panic(err)
            }
            fmt.Println("After reading...")
            panic("Hey")

			fmt.Printf("recv %s %+v\n", string(msg.Payload.([]byte)), msg)
			// if err := s.handleMessage(&m); err != nil {
			// 	log.Println(err)
			// }

		case <-s.doneCh:
			return
		}
	}
}

// func (s *FileServer) handleMessage(msg *Message) error {
// 	switch v := msg.Payload.(type) {
// 	case *DataMessage:
// 		fmt.Printf("recieved data %+v\n", v)
// 	}
//
// 	return nil
// }

func (s *FileServer) bootstrapNetwork() error {
	for _, addr := range s.BootstrapNodes {
		if len(addr) == 0 {
			continue
		}

		go func(addr string) {
			log.Println("Bootstrapping the network... trying: ", addr)
			if err := s.Transport.Dial(addr); err != nil {
				log.Println("dial error: ", err)
			}
		}(addr)
	}

	return nil
}
