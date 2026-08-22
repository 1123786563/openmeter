package payment

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileSecretProvider supplies startup-loaded secrets from mounted files. Values
// are cached so runtime reads neither depend on the filesystem nor expose file
// contents in read failures.
type FileSecretProvider struct {
	values map[string]string
}

// NewFileSecretProvider reads every configured secret file once and returns a
// provider that serves the cached values. Errors identify only the logical key
// and path; they never include secret contents.
func NewFileSecretProvider(paths map[string]string) (SecretProvider, error) {
	values := make(map[string]string, len(paths))
	for key, path := range paths {
		if strings.TrimSpace(key) == "" || strings.TrimSpace(path) == "" {
			return nil, errors.New("payment secret key and path are required")
		}

		contents, err := os.ReadFile(filepath.Clean(path))
		if err != nil {
			return nil, fmt.Errorf("read payment secret %q: %w", key, err)
		}
		if strings.TrimSpace(string(contents)) == "" {
			return nil, fmt.Errorf("payment secret %q is empty", key)
		}

		values[key] = string(contents)
	}

	return &FileSecretProvider{values: values}, nil
}

// Get returns the cached secret for key.
func (p *FileSecretProvider) Get(_ context.Context, key string) (string, error) {
	value, ok := p.values[key]
	if !ok {
		return "", &SecretNotFoundError{Key: key}
	}

	return value, nil
}
