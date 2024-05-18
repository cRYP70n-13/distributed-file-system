package p2p

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTCPTransport(t *testing.T) {
    testTcpOpts := TCPTransportOpts{
        ListenAddress: ":8000",
        HandShakeFunc: NoOpHandshake,
        Decoder: &NoOpDecoder{},
    }
    tr := NewTCPTransport(testTcpOpts)

    require.Equal(t, tr.ListenAddress, ":8000")

    err := tr.ListenAndAccept()
    require.NoError(t, err)
}
