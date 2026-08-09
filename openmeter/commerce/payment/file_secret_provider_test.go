package payment

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFileSecretProvider(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "merchant.pem")
	require.NoError(t, os.WriteFile(keyPath, []byte("PRIVATE-KEY\n"), 0o600))

	p, err := NewFileSecretProvider(map[string]string{"merchant_private_key": keyPath})
	require.NoError(t, err)

	got, err := p.Get(context.Background(), "merchant_private_key")
	require.NoError(t, err)
	require.Equal(t, "PRIVATE-KEY\n", got)

	_, err = p.Get(context.Background(), "missing")
	var notFound *SecretNotFoundError
	require.ErrorAs(t, err, &notFound)
}

func TestFileSecretProviderCachesValuesAtConstruction(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "merchant.pem")
	require.NoError(t, os.WriteFile(keyPath, []byte("ORIGINAL-KEY"), 0o600))

	p, err := NewFileSecretProvider(map[string]string{"merchant_private_key": keyPath})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(keyPath, []byte("REPLACED-KEY"), 0o600))

	got, err := p.Get(t.Context(), "merchant_private_key")
	require.NoError(t, err)
	require.Equal(t, "ORIGINAL-KEY", got)
}

func TestFileSecretProviderRejectsMissingFile(t *testing.T) {
	missingPath := filepath.Join(t.TempDir(), "missing.pem")

	_, err := NewFileSecretProvider(map[string]string{"merchant_private_key": missingPath})
	require.Error(t, err)
	require.ErrorContains(t, err, "merchant_private_key")
	require.ErrorContains(t, err, missingPath)
}

func TestFileSecretProviderRejectsBlankFile(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "blank.pem")
	require.NoError(t, os.WriteFile(keyPath, []byte(" \n\t "), 0o600))

	_, err := NewFileSecretProvider(map[string]string{"merchant_private_key": keyPath})
	require.Error(t, err)
	require.ErrorContains(t, err, "merchant_private_key")
}

func TestFileSecretProviderReadErrorDoesNotExposeSecretContent(t *testing.T) {
	secretContent := "private-key-must-not-appear-in-errors"
	secretPath := filepath.Join(t.TempDir(), "unreadable-secret.pem")
	require.NoError(t, os.WriteFile(secretPath, []byte(secretContent), 0o600))
	require.NoError(t, os.Chmod(secretPath, 0o000))
	t.Cleanup(func() {
		require.NoError(t, os.Chmod(secretPath, 0o600))
	})

	_, err := NewFileSecretProvider(map[string]string{"merchant_private_key": secretPath})
	require.Error(t, err)
	require.False(t, strings.Contains(err.Error(), secretContent))
}
