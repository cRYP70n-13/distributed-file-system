package p2p

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTCPTransport(t *testing.T) {
    listenAddr := ":4000"
    tr := NewTCPTransport(listenAddr)
    require.Equal(t, tr.listernAddress, listenAddr)

    err := tr.ListenAndAccept()
    require.NoError(t, err)
}
