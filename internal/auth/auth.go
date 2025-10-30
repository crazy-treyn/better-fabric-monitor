package auth

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"better-fabric-monitor/internal/logger"

	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/public"
)

// AuthManager handles Microsoft Entra ID authentication
type AuthManager struct {
	client            public.Client
	config            *AuthConfig
	tokenCache        *TokenCache
	httpClient        *http.Client
	pendingDeviceCode *public.DeviceCode
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	ClientID    string
	TenantID    string
	RedirectURI string
	Scopes      []string
}

// Token represents an access token with metadata
type Token struct {
	AccessToken  string    `json:"accessToken"`
	RefreshToken string    `json:"refreshToken,omitempty"`
	ExpiresAt    time.Time `json:"expiresAt"`
	TokenType    string    `json:"tokenType"`
}

// DeviceCodeInfo contains information to display to the user during device code flow
type DeviceCodeInfo struct {
	UserCode        string `json:"userCode"`
	DeviceCode      string `json:"deviceCode"`
	VerificationURL string `json:"verificationURL"`
	ExpiresIn       int    `json:"expiresIn"`
	Message         string `json:"message"`
}

// NewAuthManager creates a new authentication manager
func NewAuthManager(config *AuthConfig) (*AuthManager, error) {
	cache, err := NewTokenCache()
	if err != nil {
		return nil, fmt.Errorf("failed to create token cache: %w", err)
	}

	// Create MSAL client with persistent cache
	client, err := public.New(config.ClientID, public.WithCache(cache))
	if err != nil {
		return nil, fmt.Errorf("failed to create MSAL client: %w", err)
	}

	return &AuthManager{
		client:     client,
		config:     config,
		tokenCache: cache,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

// StartDeviceCodeFlow initiates the device code flow and returns information to display to the user
func (a *AuthManager) StartDeviceCodeFlow(ctx context.Context) (*DeviceCodeInfo, error) {
	// Initiate device code flow
	deviceCode, err := a.client.AcquireTokenByDeviceCode(ctx, a.config.Scopes)
	if err != nil {
		return nil, fmt.Errorf("failed to start device code flow: %w", err)
	}

	// Store the device code for later completion
	a.pendingDeviceCode = &deviceCode

	// Open browser to the verification URL
	if err := openBrowser(deviceCode.Result.VerificationURL); err != nil {
		// Don't fail if browser can't open, user can navigate manually
		logger.Log("Warning: failed to open browser: %v\n", err)
	}

	// Return the device code information to display in the UI
	return &DeviceCodeInfo{
		UserCode:        deviceCode.Result.UserCode,
		DeviceCode:      deviceCode.Result.DeviceCode,
		VerificationURL: deviceCode.Result.VerificationURL,
		ExpiresIn:       deviceCode.Result.Interval, // Polling interval in seconds
		Message:         deviceCode.Result.Message,
	}, nil
}

// CompleteDeviceCodeFlow waits for the user to complete authentication and returns the token
func (a *AuthManager) CompleteDeviceCodeFlow(ctx context.Context) (*Token, error) {
	if a.pendingDeviceCode == nil {
		return nil, fmt.Errorf("no device code flow in progress, call StartDeviceCodeFlow first")
	}

	// Wait for the user to complete authentication
	authResult, err := a.pendingDeviceCode.AuthenticationResult(ctx)
	if err != nil {
		a.pendingDeviceCode = nil
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	// Clear the pending device code
	a.pendingDeviceCode = nil

	token := &Token{
		AccessToken: authResult.AccessToken,
		TokenType:   "Bearer",
		ExpiresAt:   authResult.ExpiresOn,
	}

	return token, nil
}

// GetToken retrieves a valid access token, refreshing if necessary
func (a *AuthManager) GetToken(ctx context.Context) (*Token, error) {
	accounts, err := a.client.Accounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get accounts: %w", err)
	}

	if len(accounts) == 0 {
		return nil, fmt.Errorf("no accounts found, please login first")
	}

	// Use the first account
	result, err := a.client.AcquireTokenSilent(ctx, a.config.Scopes, public.WithSilentAccount(accounts[0]))
	if err != nil {
		return nil, fmt.Errorf("failed to acquire token silently: %w", err)
	}

	token := &Token{
		AccessToken: result.AccessToken,
		TokenType:   "Bearer",
		ExpiresAt:   result.ExpiresOn,
	}

	return token, nil
}

// IsAuthenticated checks if there's a valid cached token
func (a *AuthManager) IsAuthenticated() bool {
	ctx := context.Background()
	_, err := a.GetToken(ctx)
	return err == nil
}

// Logout clears the token cache
func (a *AuthManager) Logout() error {
	return a.tokenCache.Clear()
}
