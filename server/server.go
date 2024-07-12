package server

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"distributed-file-system/p2p"
	"distributed-file-system/store"
)

func init() {
	gob.Register(MessageStoreFile{})
	gob.Register(MessageGetFile{})
}

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
	var (
		fileBuf = new(bytes.Buffer)
		tee     = io.TeeReader(r, fileBuf)
	)

	// This is just to get the size of how much bytes are in the file.
	size, err := s.store.Write(key, tee)
	if err != nil {
		return err
	}

	msg := Message{
		Payload: MessageStoreFile{
			Key:  key,
			Size: size,
		},
	}

	if err := s.broadcast(&msg); err != nil {
		return err
	}

	// TODO: Remove this hack from here, will figure it out later.
	time.Sleep(1 * time.Millisecond)

	// TODO: use Multiwriter here instead.
	for _, peer := range s.peers {
        if err := peer.Send([]byte{p2p.StreamType}); err != nil {
            return err
        }
		log.Println(peer.RemoteAddr().String())
		n, err := io.Copy(peer, fileBuf)
		if err != nil {
			return err
		}

		log.Println("received and written bytes to disk", n)
	}

	return nil
}

func (s *FileServer) Get(key string) (io.Reader, error) {
	if s.store.Has(key) {
		log.Printf("File %s is serving from local disk", key)
		return s.store.Read(key)
	}

	log.Printf("File %s not found locally, trying the network...", key)

	msg := Message{
		Payload: MessageGetFile{
			Key: key,
		},
	}

	if err := s.broadcast(&msg); err != nil {
		return nil, err
	}

    time.Sleep(time.Second * 2)

    // TODO: Here instead of assuming that all the peers have the file and creating
    // a buffer for each one of them we can ask them broadcast a msg if you have this
    // then in case of an ack then we open a stream to get it from it.
	for _, peer := range s.peers {
        buf := new(bytes.Buffer)
		n, err := io.Copy(buf, peer)
        if err != nil {
            return nil, err
        }
        log.Printf("Receved %d bytes over the network", n)
	}

	select {}

	return nil, nil
}

type MessageGetFile struct {
	Key string
}

type Message struct {
	Payload any
}

type MessageStoreFile struct {
	Key  string
	Size int64
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
	buf := new(bytes.Buffer)

	if err := gob.NewEncoder(buf).Encode(msg); err != nil {
		return err
	}

	for _, peer := range s.peers {
        if err := peer.Send([]byte{p2p.MessageType}); err != nil {
            return err
        }
		if err := peer.Send(buf.Bytes()); err != nil {
			return err
		}
	}

	return nil
}

func (s *FileServer) stream(msg *Message) error {
	peers := []io.Writer{}
	for _, peer := range s.peers {
		peers = append(peers, peer)
	}

	mw := io.MultiWriter(peers...)

	log.Println("Broadcasting the msg", msg.Payload)
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
				log.Println("Decoding error: ", err)
			}

			if err := s.handleMessage(rpc.From, &msg); err != nil {
				log.Println("s.handleMessage error: ", err)
			}

		case <-s.doneCh:
			return
		}
	}
}

func (s *FileServer) handleMessage(from net.Addr, msg *Message) error {
	switch v := msg.Payload.(type) {
	case MessageStoreFile:
		log.Printf("recieved data %+v\n", v)
		return s.handleMessageStoreFile(from, v)
	case MessageGetFile:
		return s.handleMessageGetFile(from, v)
	}

	return nil
}

func (s *FileServer) handleMessageGetFile(from net.Addr, msg MessageGetFile) error {
	if !s.store.Has(msg.Key) {
		return fmt.Errorf("sorry %s but the file (%s) is not found in my disk", from, msg.Key)
	}

	log.Printf("Got file (%s), serving over the network..", from)

	peer, ok := s.peers[from]
	if !ok {
		return fmt.Errorf("whoever (%s) is asking for this file (%s) should go fuck himself", from, msg.Key)
	}

	r, err := s.store.Read(msg.Key)
	if err != nil {
		return err
	}

	n, err := io.Copy(peer, r)
	if err != nil {
		return err
	}

	log.Printf("wrote %d byte over the network to %s\n", n, from)

	return nil
}

func (s *FileServer) handleMessageStoreFile(from net.Addr, msg MessageStoreFile) error {
	peer, ok := s.peers[from]
	if !ok {
		return fmt.Errorf("peer (%s) is not found in internal peer map", from)
	}

	n, err := s.store.Write(msg.Key, io.LimitReader(peer, msg.Size))
	if err != nil {
		log.Fatal(err)
		return err
	}

	log.Printf("[%s] Written (%d) bytes to disk: %s\n", s.Transport.Addr(), n, msg.Key)

	// FIXME: Quite of a hack in it
	peer.(*p2p.TCPPeer).Wg.Done()

	return nil
}

func (s *FileServer) bootstrapNetwork() error {
	for _, addr := range s.BootstrapNodes {
		if len(addr) == 0 {
			continue
		}

		go func(addr string) {
			log.Println("Bootstrapping the network... trying: ", addr)
			if err := s.Transport.Dial(addr); err != nil {
                // TODO: Here in case of a dialing error we can retry after some time
                // and then when we are done we can drop it.
				log.Println("dial error: ", err)
			}
		}(addr)
	}

	return nil
}
