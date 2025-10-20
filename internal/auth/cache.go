package auth

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/cache"
)

// TokenCache handles secure token storage
type TokenCache struct {
	filePath string
	contract []byte
}

// NewTokenCache creates a new token cache
func NewTokenCache() (*TokenCache, error) {
	// Get user's config directory
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get config directory: %w", err)
	}

	// Create app directory
	appDir := filepath.Join(configDir, "better-fabric-monitor")
	if err := os.MkdirAll(appDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create app directory: %w", err)
	}

	cachePath := filepath.Join(appDir, "msal_cache.bin")

	tc := &TokenCache{
		filePath: cachePath,
	}

	// Load existing cache if it exists
	if data, err := os.ReadFile(cachePath); err == nil {
		tc.contract = data
	}

	return tc, nil
}

// Export exports the cache data - this is called when MSAL wants to read the cache
func (tc *TokenCache) Export(ctx context.Context, m cache.Marshaler, hints cache.ExportHints) error {
	// Get the serialized cache from MSAL
	data, err := m.Marshal()
	if err != nil {
		return err
	}

	// Save to our contract
	tc.contract = data

	// Persist to disk
	if err := os.WriteFile(tc.filePath, data, 0600); err != nil {
		fmt.Printf("Warning: failed to persist cache: %v\n", err)
	}

	return nil
}

// Replace replaces the cache data - this is called when MSAL wants to save the cache
func (tc *TokenCache) Replace(ctx context.Context, u cache.Unmarshaler, hints cache.ReplaceHints) error {
	// Give MSAL our cached contract to unmarshal
	if len(tc.contract) > 0 {
		return u.Unmarshal(tc.contract)
	}
	return nil
}

// Clear clears all cached tokens
func (tc *TokenCache) Clear() error {
	tc.contract = nil

	// Remove cache file
	if err := os.Remove(tc.filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove cache file: %w", err)
	}

	return nil
}
