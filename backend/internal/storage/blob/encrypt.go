package blob

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

const (
	encryptionMetadataKey = "blob-encryption"
	encryptionNonceKey    = "blob-nonce"
	encryptionMethod      = "aes-gcm"
)

type encryptor struct {
	key []byte
}

func newEncryptor(raw string) (*encryptor, error) {
	trimmed := bytes.TrimSpace([]byte(raw))
	if len(trimmed) == 0 {
		return nil, nil
	}
	decoded, err := base64.StdEncoding.DecodeString(string(trimmed))
	if err != nil {
		return nil, fmt.Errorf("files.encryption_key must be base64: %w", err)
	}
	switch len(decoded) {
	case 16, 24, 32:
	default:
		return nil, fmt.Errorf("files.encryption_key must be 16/24/32 bytes after decoding")
	}
	return &encryptor{key: decoded}, nil
}

func (e *encryptor) encrypt(r io.Reader) (io.ReadCloser, int64, map[string]string, error) {
	plain, err := io.ReadAll(r)
	if err != nil {
		return nil, 0, nil, err
	}
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, 0, nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, 0, nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, 0, nil, err
	}
	cipherText := gcm.Seal(nil, nonce, plain, nil)
	payload := append(nonce, cipherText...)
	meta := map[string]string{
		encryptionMetadataKey: encryptionMethod,
		encryptionNonceKey:    base64.StdEncoding.EncodeToString(nonce),
	}
	return nopCloser{bytes.NewReader(payload)}, int64(len(payload)), meta, nil
}

func (e *encryptor) decrypt(r io.Reader) (io.ReadCloser, int64, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, 0, err
	}
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, 0, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, 0, err
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, 0, errors.New("encrypted payload too short")
	}
	nonce := data[:nonceSize]
	cipherText := data[nonceSize:]
	plain, err := gcm.Open(nil, nonce, cipherText, nil)
	if err != nil {
		return nil, 0, err
	}
	return nopCloser{bytes.NewReader(plain)}, int64(len(plain)), nil
}

type nopCloser struct {
	io.Reader
}

func (nopCloser) Close() error { return nil }

// Helper for future streaming decryption if metadata is available without reading entire payload.
func nonceFromMetadata(meta map[string]string) ([]byte, error) {
	if meta == nil {
		return nil, fmt.Errorf("missing encryption metadata")
	}
	base64Nonce, ok := meta[encryptionNonceKey]
	if !ok {
		return nil, fmt.Errorf("missing nonce metadata")
	}
	return base64.StdEncoding.DecodeString(base64Nonce)
}
