package testutil

import (
	"fmt"
	"testing"

	miniredis "github.com/alicebob/miniredis/v2"
)

// StartTestRedis starts a miniredis instance for testing.
// Returns a cleanup function and the Redis URL.
func StartTestRedis(t *testing.T) (cleanup func(), url string) {
	t.Helper()

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}

	return func() { mr.Close() }, fmt.Sprintf("redis://%s", mr.Addr())
}

// MiniRedis represents a miniredis instance with helper methods.
type MiniRedis struct {
	*miniredis.Miniredis
}

// StartMiniRedis starts a miniredis instance with direct access.
func StartMiniRedis(t *testing.T) *MiniRedis {
	t.Helper()

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	t.Cleanup(func() { mr.Close() })

	return &MiniRedis{Miniredis: mr}
}

// URL returns the Redis connection URL.
func (m *MiniRedis) URL() string {
	return fmt.Sprintf("redis://%s", m.Addr())
}
