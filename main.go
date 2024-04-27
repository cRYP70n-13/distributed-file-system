package main

import (
	"distributedStore/p2p"
)

func main() {
	tr := p2p.NewTCPTransport("localhost:3000")

	if err := tr.ListenAndAccept(); err != nil {
		panic(err)
	}

	select {}
}
