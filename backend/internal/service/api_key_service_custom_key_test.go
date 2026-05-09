//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateCustomKeyRejectsWeakValues(t *testing.T) {
	service := &APIKeyService{}

	weakKeys := []string{
		"aaaaaaaaaaaaaaaa",
		"1234567890123456",
		"abcdefghijklmnop",
		"aaaaabaaaaaaaaaa",
		"abcabcabcabcabca",
	}

	for _, key := range weakKeys {
		err := service.ValidateCustomKey(key)
		require.ErrorIsf(t, err, ErrAPIKeyTooWeak, "key=%s", key)
	}
}

func TestValidateCustomKeyAcceptsHighEntropyValue(t *testing.T) {
	service := &APIKeyService{}

	err := service.ValidateCustomKey("aB3fK9pQ2vR7xY1z")
	require.NoError(t, err)
}
