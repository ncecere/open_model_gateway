package blob

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
)

type localStore struct {
	root string
	mu   sync.Mutex
}

type localMetadata struct {
	ContentType string            `json:"content_type"`
	Size        int64             `json:"size"`
	Metadata    map[string]string `json:"metadata"`
}

func newLocalStore(cfg config.FilesConfig) (*localStore, error) {
	dir := cfg.Local.Directory
	if strings.TrimSpace(dir) == "" {
		dir = "./data/files"
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("create local storage dir: %w", err)
	}
	return &localStore{root: dir}, nil
}

func (s *localStore) Put(ctx context.Context, key string, body io.Reader, opts PutOptions) (ObjectInfo, error) {
	select {
	case <-ctx.Done():
		return ObjectInfo{}, ctx.Err()
	default:
	}
	path, metaPath, err := s.pathsForKey(key)
	if err != nil {
		return ObjectInfo{}, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return ObjectInfo{}, err
	}
	tempFile, err := os.CreateTemp(filepath.Dir(path), "upload-*.tmp")
	if err != nil {
		return ObjectInfo{}, err
	}
	defer os.Remove(tempFile.Name())
	written, err := io.Copy(tempFile, body)
	if err != nil {
		tempFile.Close()
		return ObjectInfo{}, err
	}
	if err := tempFile.Sync(); err != nil {
		tempFile.Close()
		return ObjectInfo{}, err
	}
	if err := tempFile.Close(); err != nil {
		return ObjectInfo{}, err
	}
	if err := os.Rename(tempFile.Name(), path); err != nil {
		return ObjectInfo{}, err
	}
	meta := localMetadata{ContentType: opts.ContentType, Size: written, Metadata: opts.Metadata}
	if err := writeMetadata(metaPath, meta); err != nil {
		return ObjectInfo{}, err
	}
	return ObjectInfo{Key: key, Size: written, ContentType: opts.ContentType, Metadata: opts.Metadata}, nil
}

func (s *localStore) Get(ctx context.Context, key string) (io.ReadCloser, ObjectInfo, error) {
	path, metaPath, err := s.pathsForKey(key)
	if err != nil {
		return nil, ObjectInfo{}, err
	}
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, ObjectInfo{}, ErrNotFound
		}
		return nil, ObjectInfo{}, err
	}
	meta, err := readMetadata(metaPath)
	if err != nil {
		file.Close()
		return nil, ObjectInfo{}, err
	}
	info := ObjectInfo{Key: key, Size: meta.Size, ContentType: meta.ContentType, Metadata: meta.Metadata}
	return file, info, nil
}

func (s *localStore) Delete(ctx context.Context, key string) error {
	path, metaPath, err := s.pathsForKey(key)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if err := os.Remove(metaPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

func (s *localStore) pathsForKey(key string) (string, string, error) {
	cleaned := filepath.Clean(key)
	if cleaned == "." || strings.HasPrefix(cleaned, "..") {
		return "", "", fmt.Errorf("invalid key: %s", key)
	}
	path := filepath.Join(s.root, cleaned)
	return path, path + ".meta", nil
}

func writeMetadata(path string, meta localMetadata) error {
	data, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o640)
}

func readMetadata(path string) (localMetadata, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return localMetadata{}, ErrNotFound
		}
		return localMetadata{}, err
	}
	var meta localMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return localMetadata{}, err
	}
	return meta, nil
}
