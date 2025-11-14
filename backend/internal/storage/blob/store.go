package blob

import (
	"context"
	"io"
	"strings"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
)

type PutOptions struct {
	ContentType string
	Metadata    map[string]string
}

type ObjectInfo struct {
	Key         string
	Size        int64
	ContentType string
	Metadata    map[string]string
	Encrypted   bool
}

type Store interface {
	Put(ctx context.Context, key string, body io.Reader, opts PutOptions) (ObjectInfo, error)
	Get(ctx context.Context, key string) (io.ReadCloser, ObjectInfo, error)
	Delete(ctx context.Context, key string) error
}

type backendStore interface {
	Put(ctx context.Context, key string, body io.Reader, opts PutOptions) (ObjectInfo, error)
	Get(ctx context.Context, key string) (io.ReadCloser, ObjectInfo, error)
	Delete(ctx context.Context, key string) error
}

type store struct {
	backend   backendStore
	encryptor *encryptor
}

func New(ctx context.Context, cfg config.FilesConfig) (Store, error) {
	backend, err := buildBackend(ctx, cfg)
	if err != nil {
		return nil, err
	}
	enc, err := newEncryptor(cfg.EncryptionKey)
	if err != nil {
		return nil, err
	}
	return &store{backend: backend, encryptor: enc}, nil
}

func buildBackend(ctx context.Context, cfg config.FilesConfig) (backendStore, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.Storage)) {
	case "s3":
		awsCfg, err := loadS3Config(ctx, cfg)
		if err != nil {
			return nil, err
		}
		return newS3Store(cfg, awsCfg)
	default:
		return newLocalStore(cfg)
	}
}

func (s *store) Put(ctx context.Context, key string, body io.Reader, opts PutOptions) (ObjectInfo, error) {
	if s.encryptor == nil {
		return s.backend.Put(ctx, key, body, opts)
	}
	encReader, size, metadata, err := s.encryptor.encrypt(body)
	if err != nil {
		return ObjectInfo{}, err
	}
	merged := mergeMetadata(opts.Metadata, metadata)
	info, err := s.backend.Put(ctx, key, encReader, PutOptions{
		ContentType: opts.ContentType,
		Metadata:    merged,
	})
	if err != nil {
		return ObjectInfo{}, err
	}
	info.Size = size
	info.Metadata = mergeMetadata(info.Metadata, metadata)
	info.Encrypted = s.encryptor != nil
	return info, nil
}

func (s *store) Get(ctx context.Context, key string) (io.ReadCloser, ObjectInfo, error) {
	reader, info, err := s.backend.Get(ctx, key)
	if err != nil {
		return nil, ObjectInfo{}, err
	}
	if s.encryptor == nil || !isEncrypted(info.Metadata) {
		info.Encrypted = s.encryptor != nil && isEncrypted(info.Metadata)
		return reader, info, nil
	}
	decReader, size, err := s.encryptor.decrypt(reader)
	if err != nil {
		return nil, ObjectInfo{}, err
	}
	info.Size = size
	info.Encrypted = true
	return decReader, info, nil
}

func (s *store) Delete(ctx context.Context, key string) error {
	return s.backend.Delete(ctx, key)
}

func mergeMetadata(a, b map[string]string) map[string]string {
	if len(a) == 0 && len(b) == 0 {
		return nil
	}
	merged := make(map[string]string, len(a)+len(b))
	for k, v := range a {
		merged[k] = v
	}
	for k, v := range b {
		merged[k] = v
	}
	return merged
}

func isEncrypted(meta map[string]string) bool {
	if meta == nil {
		return false
	}
	_, ok := meta[encryptionMetadataKey]
	return ok
}
