package fixtures

import (
	"embed"
	"encoding/json"
	"fmt"
)

//go:embed testdata/*
var files embed.FS

// Load decodes the named JSON fixture file into dest.
func Load(name string, dest interface{}) error {
	data, err := Read(name)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("decode fixture %s: %w", name, err)
	}
	return nil
}

// Read returns the raw bytes for a fixture file.
func Read(name string) ([]byte, error) {
	data, err := files.ReadFile("testdata/" + name)
	if err != nil {
		return nil, fmt.Errorf("read fixture %s: %w", name, err)
	}
	return data, nil
}
