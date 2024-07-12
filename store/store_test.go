package store

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStore(t *testing.T) {
	t.Parallel()

	s := newStore(t)

	defer func() {
		_ = tearDown(t, s)
	}()

	// Hmm table testing ... I'm not sure I don't care atm
	// And maybe a good place to try fuzz testing as well :)
	key := "test_file"
	testData := []byte("This is a test after the update")

	data := bytes.NewReader(testData)
	_, err := s.writeStream(key, data)
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

func newStore(t *testing.T) *Store {
	t.Helper()

	opts := StoreOpts{
		TransaformFunc: CascadePathTransformFunc,
	}
	return NewStore(opts)
}

func tearDown(t *testing.T, s *Store) error {
	t.Helper()
	return s.Cleanup()
}
