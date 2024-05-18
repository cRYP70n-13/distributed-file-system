package server

import (
	"distributed-file-system/p2p"
	"distributed-file-system/store"
	"log"
)

type FileServerOpts struct {
	StorageRoot       string
	PathTransformFunc store.TransaformFunc
	Transport         p2p.Transport
	BootstrapNodes    []string
}

type FileServer struct {
	FileServerOpts

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
	}
}

func (s *FileServer) Start() error {
	if err := s.Transport.ListenAndAccept(); err != nil {
		return err
	}

	if err := s.bootstrapNetwork(); err != nil {
		panic(err)
	}
	go s.loop()

	return nil
}

func (s *FileServer) Stop() {
	close(s.doneCh)
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
        log.Println("Bootstrapping the network...")
		addr := addr
		go func(addr string) {
			if err := s.Transport.Dial(addr); err != nil {
				log.Println("dial error: ", err)
                panic(err)
			}
		}(addr)
	}

	return nil
}
