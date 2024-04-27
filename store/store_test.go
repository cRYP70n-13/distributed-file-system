package Store

import (
	"bytes"
	"testing"
)

func TestStore(t *testing.T) {
    opts := StoreOpts{
        PathTransformFunc: DefaultPathTransformFunc,
    }
    s := NewStore(opts)

    data := bytes.NewReader([]byte("something"))
    if err := s.writeStream("test_1", data); err != nil {
        t.Error(err)
    }
}
