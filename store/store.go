package store

import (
	"crypto/sha1"
	"distributed-file-system/cryptographer"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"strings"
)

const (
	blockSize             = 5
	defaultRootFolderName = "otmaneNetwork"
)

// TransaformFunc is the transformation that we gonna make over the pathname to get the desired tree structure.
type TransaformFunc func(pathname string) PathKey

type PathKey struct {
	Pathname string
	filename string
}

type StoreOpts struct {
	// Root is the parent folder of our folder tree structure.
	Root string
	TransaformFunc
}

type Store struct {
	StoreOpts
}

func NewStore(opts StoreOpts) *Store {
	if opts.TransaformFunc == nil {
		opts.TransaformFunc = DefaultTransformFunc
	}
	if len(opts.Root) == 0 {
		opts.Root = defaultRootFolderName
	}
	return &Store{
		StoreOpts: opts,
	}
}

func CascadePathTransformFunc(key string) PathKey {
	hash := sha1.Sum([]byte(key))
	hashStr := hex.EncodeToString(hash[:])

	sliceLen := len(hashStr) / blockSize

	paths := make([]string, sliceLen)

	for i := 0; i < sliceLen; i++ {
		from, to := i*blockSize, (i*blockSize)+blockSize
		paths[i] = hashStr[from:to]
	}

	return PathKey{
		Pathname: strings.Join(paths, "/"),
		filename: hashStr,
	}
}

func (s *Store) Read(key string) (int64, io.Reader, error) {
	return s.readStream(key)
}

func (s *Store) Write(key string, r io.Reader) (int64, error) {
	return s.writeStream(key, r)
}

// Delete deletes a whole folder from the system.
func (s *Store) Delete(key string) error {
	pathKey := s.TransaformFunc(key)
	firstPathNameWithRoot := fmt.Sprintf("%s/%s", s.Root, pathKey.ParentFolderName())

	defer func() {
		log.Printf("Deleted everything inside %s\n", firstPathNameWithRoot)
	}()

	return os.RemoveAll(firstPathNameWithRoot)
}

// Has Checks if a file exits or not.
func (s *Store) Has(key string) bool {
	pathKey := s.TransaformFunc(key)
	fullPathWithRoot := fmt.Sprintf("%s/%s", s.Root, pathKey.FullPath())

	_, err := os.Stat(fullPathWithRoot)

	return !errors.Is(err, fs.ErrNotExist)
}

func (p PathKey) ParentFolderName() string {
	return strings.Split(p.FullPath(), "/")[0]
}

func DefaultTransformFunc(key string) PathKey {
	return PathKey{
		Pathname: key,
		filename: key,
	}
}

func (p PathKey) FullPath() string {
	return fmt.Sprintf("%s/%s", p.Pathname, p.filename)
}

// Cleanup just drops everything including the root folder.
func (s *Store) Cleanup() error {
	return os.RemoveAll(s.Root)
}

func (s *Store) WriteDecrypt(encKey []byte, key string, r io.Reader) (int64, error) {
	pathKey := s.TransaformFunc(key)
	pathWithParentName := fmt.Sprintf("%s/%s", s.Root, pathKey.Pathname)
	if err := os.MkdirAll(pathWithParentName, os.ModePerm); err != nil {
		return 0, err
	}

	fullFilePath := fmt.Sprintf("%s/%s", s.Root, pathKey.FullPath())

	f, err := os.Create(fullFilePath)
	if err != nil {
		return 0, err
	}

    return cryptographer.CopyDecrypt(encKey, r, f)
}

func (s *Store) writeStream(key string, r io.Reader) (int64, error) {
	pathKey := s.TransaformFunc(key)
	pathWithParentName := fmt.Sprintf("%s/%s", s.Root, pathKey.Pathname)
	if err := os.MkdirAll(pathWithParentName, os.ModePerm); err != nil {
		return 0, err
	}

	fullFilePath := fmt.Sprintf("%s/%s", s.Root, pathKey.FullPath())

	f, err := os.Create(fullFilePath)
	if err != nil {
		return 0, err
	}

	return io.Copy(f, r)
}

func (s *Store) readStream(key string) (int64, io.ReadCloser, error) {
	pathKey := s.TransaformFunc(key)
	fullpathWithRoot := fmt.Sprintf("%s/%s", s.Root, pathKey.FullPath())

	file, err := os.Open(fullpathWithRoot)
	if err != nil {
		return 0, nil, err
	}

	fi, err := file.Stat()
	if err != nil {
		return 0, nil, err
	}

	return fi.Size(), file, nil
}
