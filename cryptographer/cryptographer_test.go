package cryptographer_test

import (
	"bytes"
	"testing"

	cryptographer "distributed-file-system/cryptographer"

	"github.com/stretchr/testify/require"
)

// TODO: Refactor this piece of guarbage there is something that we can re-use here.
func TestCopyEcnrypt(t *testing.T) {
	src := bytes.NewReader([]byte("String to encrypt"))
	dst := new(bytes.Buffer)
	key, err := cryptographer.NewEncryptionKey()
	require.NoError(t, err)

	_, err = cryptographer.CopyDecrypt(key, src, dst)
	require.NoError(t, err)
	require.NotEqual(t, src, dst)
}

func TestCopyDecrypt(t *testing.T) {
    payload := "String to encrypt"

	src := bytes.NewReader([]byte(payload))
	dst := new(bytes.Buffer)
	key, err := cryptographer.NewEncryptionKey()
	require.NoError(t, err)

	_, err = cryptographer.CopyEncrypt(key, src, dst)
	require.NoError(t, err)
	require.NotEqual(t, src, dst)

    out := new(bytes.Buffer)
    _, err = cryptographer.CopyDecrypt(key, dst, out)
	require.NoError(t, err)
    require.Equal(t, out.String(), payload)
}

func TestNewEncryptionKey(t *testing.T) {

	key, err := cryptographer.NewEncryptionKey()
	require.NoError(t, err)
	require.NotEmpty(t, key)
	require.Len(t, key, cryptographer.DefaultEncryptionKeyLen, "the given key is not 32 bytes something is shady going on")
}
