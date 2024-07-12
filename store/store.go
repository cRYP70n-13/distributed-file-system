package store

import (
	"bytes"
	"crypto/sha1"
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

// TODO: Instead of reading this shit into memory
// put it directly where you need it.
func (s *Store) Read(key string) (int64, io.Reader, error) {
	n, f, err := s.readStream(key)
	if err != nil {
		return 0, nil, err
	}
	defer f.Close()

	// FIXME: Maybe just drop this shit ???
	buf := new(bytes.Buffer)

	_, err = io.Copy(buf, f)
	if err != nil {
		return 0, nil, err
	}

	return n, buf, nil
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

	// Anthony GG is just stupid enough to not notice this, I guess Tj or Prime would've noticed it right away.
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

// TODO: Here we need to also consider if they Gave us a folder that
// already exists.
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

	n, err := io.Copy(f, r)
	if err != nil {
		return 0, err
	}

	return n, nil
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
