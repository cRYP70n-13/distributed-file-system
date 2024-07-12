package cryptographer

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
)

// TODO: Add a NoopEncrypter

const DefaultEncryptionKeyLen int = 32

// NewEncryptionKey generate a random new encryption key
func NewEncryptionKey() ([]byte, error) {
	keyBuf := make([]byte, DefaultEncryptionKeyLen)

	if _, err := io.ReadFull(rand.Reader, keyBuf); err != nil {
		return nil, err
	}

	return keyBuf, nil
}

// CopyDecrypt is basically the same as io.Copy but with decryption of our own encryption.
func CopyDecrypt(key []byte, src io.Reader, dst io.Writer) (int, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return 0, err
	}

	// Read the IV from the given io.Reader, Basically just what we wrote when we encrypt
	// before starting the encryption itself.
	iv := make([]byte, block.BlockSize())
	if _, err := src.Read(iv); err != nil {
		return 0, err
	}

	return processStream(block, iv, src, dst)
}

// CopyEncrypt is basically the same as io.Copy but with encryption in place.
func CopyEncrypt(key []byte, src io.Reader, dst io.Writer) (int, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return 0, err
	}

	iv := make([]byte, block.BlockSize())
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return 0, err
	}

	// Prepend the iv to the beginning of the file because we need it for decryption
	if _, err := dst.Write(iv); err != nil {
		return 0, err
	}

	return processStream(block, iv, src, dst)
}

// processStream handles the common encryption/decryption logic.
func processStream(block cipher.Block, iv []byte, src io.Reader, dst io.Writer) (int, error) {
	var (
		stream        = cipher.NewCTR(block, iv)
		buf           = make([]byte, 32*1024) // 32*1024 because that the buffer size io.Copy is using behind the scenes as a default one
		totalWrittern = block.BlockSize()
	)

	for {
		n, err := src.Read(buf)
		if n > 0 {
			stream.XORKeyStream(buf[:n], buf[:n])
			written, writeErr := dst.Write(buf[:n])
			totalWrittern += written
			if writeErr != nil {
				return 0, writeErr
			}
		}
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return 0, err
		}
	}

	return totalWrittern, nil
}
