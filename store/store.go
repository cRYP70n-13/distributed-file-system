package Store

import (
	"io"
	"log"
	"os"
)

type PathTransformFunc func(string) string

type StoreOpts struct {
    PathTransformFunc PathTransformFunc
}

var DefaultPathTransformFunc = func(key string) string {
    return key
}

type Store struct {
    StoreOpts
}

func NewStore(opts StoreOpts) *Store {
    return &Store{
        StoreOpts: opts,
    }
}

func (s *Store) writeStream(key string, r io.Reader) error {
    pathName := s.PathTransformFunc(key)
    fileName := "some-file-name"
    fullFilePath := pathName + "/" + fileName

    if err := os.MkdirAll(pathName, os.ModePerm); err != nil {
        return err
    }

    f, err := os.Create(fullFilePath)
    if err != nil {
        return err
    }

    n, err := io.Copy(f, r)
    if err != nil {
        return err
    }

    log.Printf("written (%d) bytes to %s\n", n, fullFilePath)
    return nil
}
