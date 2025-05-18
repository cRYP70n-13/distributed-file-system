package server

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"distributed-file-system/cryptographer"
	"distributed-file-system/p2p"
	"distributed-file-system/store"
)

const cipherBlockSize = 16

func init() {
	gob.Register(MessageStoreFile{})
	gob.Register(MessageGetFile{})
}

type FileServerOpts struct {
	EncKey            []byte
	StorageRoot       string
	PathTransformFunc store.TransaformFunc
	Transport         p2p.Transport
	BootstrapNodes    []string
}

type FileServer struct {
	peers  map[net.Addr]p2p.Peer
	store  *store.Store
	doneCh chan struct{}
	FileServerOpts
	mu sync.RWMutex
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

// Delete will delete the given key from the network.
func (s *FileServer) Delete(key string) error {
	return s.store.Delete(key)
}

// Store will store and broadcast the file over the network.
func (s *FileServer) Store(key string, r io.Reader) error {
	var (
		fileBuf = new(bytes.Buffer)
		tee     = io.TeeReader(r, fileBuf)
	)

	// This write the current buffer to our own folder and gets us how much the size of the file is.
	size, err := s.store.Write(key, tee)
	if err != nil {
		return err
	}

	msg := Message{
		Payload: MessageStoreFile{
			Key:  key,
			Size: size + cipherBlockSize,
		},
	}

	if err := s.broadcast(&msg); err != nil {
		return err
	}

	time.Sleep(5 * time.Millisecond)

	var writers []io.Writer
	for _, peer := range s.peers {
		writers = append(writers, peer)
	}

	mwr := io.MultiWriter(writers...)
	if _, err := mwr.Write([]byte{p2p.StreamType}); err != nil {
		return err
	}

	n, err := cryptographer.CopyEncrypt(s.EncKey, fileBuf, mwr)
	if err != nil {
		return err
	}

	log.Printf("Received and wrote %d bytes\n", n)

	return nil
}

// BUG: This GET will send you the file encrypted in case the guy who's asking about it
// has a copy of it in local, so we need to make it more cleaner.

// Get will get you the file you are asking for.
func (s *FileServer) Get(key string) (io.Reader, error) {
	if s.store.Has(key) {
		log.Printf("[%s] File %s is being served from local disk", s.Transport.Addr(), key)
		_, r, err := s.store.Read(key)
		return r, err
	}

	log.Printf("[%s] File %s not found locally, trying the network...", s.Transport.Addr(), key)

	msg := Message{
		Payload: MessageGetFile{
			Key: key,
		},
	}

	if err := s.broadcast(&msg); err != nil {
		return nil, err
	}

	time.Sleep(time.Millisecond * 2)

	// TODO: Here instead of assuming that all the peers have the file and creating
	// a buffer for each one of them we can ask them broadcast a msg if you have this
	// then in case of an ack then we open a stream to get it from it.
	for _, peer := range s.peers {
		// Read the file size so we can limit the amount that we
		// read from the connection to save ourselves from keep hanging
		var fileSize int64
		if err := binary.Read(peer, binary.LittleEndian, &fileSize); err != nil {
			return nil, err
		}

		n, err := s.store.WriteDecrypt(s.EncKey, key, io.LimitReader(peer, fileSize))
		if err != nil {
			return nil, err
		}

		log.Printf("[%s] received %d bytes over the network from (%s)", s.Transport.Addr(), n, peer.LocalAddr())

		peer.CloseStream()
	}

	_, r, err := s.store.Read(key)
	return r, err
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
		return s.handleMessageStoreFile(from, v)
	case MessageGetFile:
		return s.handleMessageGetFile(from, v)
	}

	return nil
}

func (s *FileServer) handleMessageGetFile(from net.Addr, msg MessageGetFile) error {
	log.Printf("Message get file recieved data %+v\n", msg)
	if !s.store.Has(msg.Key) {
		return fmt.Errorf("sorry %s but the file (%s) is not found in my disk", s.Transport.Addr(), msg.Key)
	}

	log.Printf("[%s] The file (%s) exists, serving it over the network..", s.Transport.Addr(), msg.Key)

	peer, ok := s.peers[from]
	if !ok {
		return fmt.Errorf("whoever (%s) is asking for this file (%s) should go fuck himself", from, msg.Key)
	}

	fileSize, r, err := s.store.Read(msg.Key)
	if err != nil {
		return err
	}

	// Yeah it's hacky but better than nothing.
	if rc, ok := r.(io.ReadCloser); ok {
		defer func() {
			log.Println("Closing the readCloser reader")
			rc.Close()
		}()
	}

	// Send that we are about to stream some shit.
	if err = peer.Send([]byte{p2p.StreamType}); err != nil {
		return err
	}

	// And then we need to send how long AKA the size of the file as an int64.
	if err := binary.Write(peer, binary.LittleEndian, fileSize); err != nil {
		return err
	}

	n, err := io.Copy(peer, r)
	if err != nil {
		return err
	}

	log.Printf("[%s] wrote %d byte over the network to %s\n", s.Transport.Addr(), n, from)

	return nil
}

func (s *FileServer) handleMessageStoreFile(from net.Addr, msg MessageStoreFile) error {
	log.Printf("Message store file recieved data %+v\n", msg)
	peer, ok := s.peers[from]
	if !ok {
		return fmt.Errorf("peer (%s) is not found in internal peer map", from)
	}

	n, err := s.store.Write(msg.Key, io.LimitReader(peer, msg.Size))
	if err != nil {
		log.Fatal(err)
		return err
	}

	peer.CloseStream()

	log.Printf("[%s] Written (%d) bytes to disk: %s\n", s.Transport.Addr(), n, msg.Key)

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
