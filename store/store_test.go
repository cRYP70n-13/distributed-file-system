package store

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

// TODO: Actually this test is breaking the single door principale I have to make some helper functions
// to create a file and add some dummy data there
// then use it in our tests but it is what it is.

func TestStore(t *testing.T) {
	t.Parallel()

	opts := StoreOpts{
		TransaformFunc: CascadePathTransformFunc,
	}
	s := NewStore(opts)
	key := "test_file"
	testData := []byte("This is a test after the update")

	data := bytes.NewReader(testData)
	err := s.writeStream(key, data)
	require.NoError(t, err)

	r, err := s.Read(key)
	require.NoError(t, err)

	b, err := io.ReadAll(r)
	require.NoError(t, err)

	require.Equal(t, b, testData)

    exists := s.Has(key)
    require.True(t, exists, "Expected to have the key but found Naada")

	err = s.Delete(key)
	require.NoError(t, err)

    require.NoDirExists(t, key, "Meeeeh motherfucker")
}
