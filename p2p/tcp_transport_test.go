package p2p

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTCPTransport(t *testing.T) {
    tcpOpts := TCPTransportOpts{
        ListenAddr: ":3000",
        HandShakeFunc: NOPHandshakeFunc,
    }
    tr := NewTCPTransport(tcpOpts)

    require.Equal(t, tr.ListenAddr, tcpOpts.ListenAddr)

    err := tr.ListenAndAccept()
    require.NoError(t, err)
}
