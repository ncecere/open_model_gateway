package auth

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

const (
	apiKeyPrefixLength = 10
	apiKeySecretLength = 48
	apiKeyPrefix       = "sk-"
	alphabet           = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

// GenerateAPIKey returns the random prefix, secret, and encoded token for a new API key.
func GenerateAPIKey() (string, string, string, error) {
	prefix, err := randomString(apiKeyPrefixLength)
	if err != nil {
		return "", "", "", err
	}
	secret, err := randomString(apiKeySecretLength)
	if err != nil {
		return "", "", "", err
	}
	token := fmt.Sprintf("%s%s.%s", apiKeyPrefix, prefix, secret)
	return prefix, secret, token, nil
}

func randomString(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("length must be positive")
	}
	out := make([]byte, length)
	max := big.NewInt(int64(len(alphabet)))
	for i := range out {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		out[i] = alphabet[n.Int64()]
	}
	return string(out), nil
}
