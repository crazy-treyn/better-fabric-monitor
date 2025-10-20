package main

import (
	"context"
	"fmt"
	"time"

	"better-fabric-monitor/internal/auth"
	"better-fabric-monitor/internal/config"
	"better-fabric-monitor/internal/db"
	"better-fabric-monitor/internal/fabric"
)

// App struct
type App struct {
	ctx          context.Context
	config       *config.Config
	auth         *auth.AuthManager
	db           *db.Database
	fabricClient *fabric.Client
	currentToken *auth.Token
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		// Continue with default config
		cfg = &config.Config{}
	}
	a.config = cfg

	// Initialize database
	database, err := db.NewDatabase(cfg.Database.Path, cfg.Database.EncryptionKey)
	if err != nil {
		fmt.Printf("Failed to initialize database: %v\n", err)
	} else {
		a.db = database
	}

	// Use Microsoft PowerShell public client ID for user authentication (no app registration needed)
	// This client ID has http://localhost redirect URIs pre-registered
	if cfg.Auth.ClientID == "" || cfg.Auth.ClientID == "your-client-id-here" {
		cfg.Auth.ClientID = "1950a258-227b-4e31-a9cf-717495945fc2" // Microsoft PowerShell public client
	}

	// Initialize authentication
	authConfig := &auth.AuthConfig{
		ClientID:    cfg.Auth.ClientID,
		TenantID:    cfg.Auth.TenantID,
		RedirectURI: cfg.Auth.RedirectURI,
		Scopes:      []string{"https://analysis.windows.net/powerbi/api/.default"},
	}

	authManager, err := auth.NewAuthManager(authConfig)
	if err != nil {
		fmt.Printf("Failed to initialize auth: %v\n", err)
	} else {
		a.auth = authManager
	}
}

// Login initiates the authentication flow
func (a *App) Login(tenantID string) map[string]interface{} {
	if a.auth == nil {
		return map[string]interface{}{
			"success": false,
			"error":   "Authentication not initialized",
		}
	}

	// Update tenant ID in config
	a.config.Auth.TenantID = tenantID

	// Use Microsoft PowerShell public client ID for user authentication
	clientID := a.config.Auth.ClientID
	if clientID == "" || clientID == "your-client-id-here" {
		clientID = "1950a258-227b-4e31-a9cf-717495945fc2" // Microsoft PowerShell public client
	}

	// Re-initialize auth with new tenant
	authConfig := &auth.AuthConfig{
		ClientID:    clientID,
		TenantID:    tenantID,
		RedirectURI: a.config.Auth.RedirectURI,
		Scopes:      []string{"https://analysis.windows.net/powerbi/api/.default"},
	}

	authManager, err := auth.NewAuthManager(authConfig)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Failed to initialize auth: %v", err),
		}
	}
	a.auth = authManager

	// Start device code flow
	deviceCodeInfo, err := a.auth.StartDeviceCodeFlow(a.ctx)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Failed to start login: %v", err),
		}
	}

	// Return device code information to display in UI
	return map[string]interface{}{
		"success":         true,
		"requiresCode":    true,
		"userCode":        deviceCodeInfo.UserCode,
		"verificationURL": deviceCodeInfo.VerificationURL,
		"message":         deviceCodeInfo.Message,
	}
}

// CompleteLogin waits for the user to complete device code authentication
func (a *App) CompleteLogin() map[string]interface{} {
	if a.auth == nil {
		return map[string]interface{}{
			"success": false,
			"error":   "Authentication not initialized",
		}
	}

	// Complete the device code flow
	token, err := a.auth.CompleteDeviceCodeFlow(a.ctx)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Login failed: %v", err),
		}
	}

	// Store the token and initialize Fabric client
	a.currentToken = token
	a.fabricClient = fabric.NewClient(token.AccessToken)

	return map[string]interface{}{
		"success": true,
		"user": map[string]interface{}{
			"id":    "user-id",          // TODO: Extract from token
			"name":  "User",             // TODO: Extract from token
			"email": "user@example.com", // TODO: Extract from token
		},
		"token": token,
	}
}

// Logout clears authentication
func (a *App) Logout() error {
	a.currentToken = nil
	a.fabricClient = nil
	if a.auth != nil {
		return a.auth.Logout()
	}
	return nil
}

// IsAuthenticated checks if user is authenticated
func (a *App) IsAuthenticated() bool {
	if a.auth != nil {
		return a.auth.IsAuthenticated()
	}
	return false
}

// GetUserInfo returns current user information
func (a *App) GetUserInfo() map[string]interface{} {
	return map[string]interface{}{
		"id":    "user-id",
		"name":  "User",
		"email": "user@example.com",
	}
}

// GetWorkspaces returns available workspaces
func (a *App) GetWorkspaces() []map[string]interface{} {
	if a.fabricClient == nil {
		fmt.Println("Fabric client not initialized, returning mock data")
		return []map[string]interface{}{
			{
				"id":          "workspace-1",
				"displayName": "Production Workspace",
				"type":        "Workspace",
			},
			{
				"id":          "workspace-2",
				"displayName": "Development Workspace",
				"type":        "Workspace",
			},
		}
	}

	// Get real workspaces from Fabric API
	workspaces, err := a.fabricClient.GetWorkspaces(a.ctx)
	if err != nil {
		fmt.Printf("Failed to get workspaces: %v\n", err)
		return []map[string]interface{}{
			{
				"id":          "error",
				"displayName": fmt.Sprintf("Error loading workspaces: %v", err),
				"type":        "Error",
			},
		}
	}

	// Convert to map format for frontend
	result := make([]map[string]interface{}, 0, len(workspaces))
	for _, ws := range workspaces {
		result = append(result, map[string]interface{}{
			"id":          ws.ID,
			"displayName": ws.DisplayName,
			"type":        ws.Type,
			"description": ws.Description,
		})
	}

	return result
}

// GetJobs returns recent jobs
func (a *App) GetJobs() []map[string]interface{} {
	if a.fabricClient == nil {
		fmt.Println("Fabric client not initialized, returning mock data")
		return []map[string]interface{}{
			{
				"id":              "job-1",
				"workspaceId":     "workspace-1",
				"itemId":          "pipeline-1",
				"itemDisplayName": "Daily ETL Pipeline",
				"jobType":         "Pipeline",
				"status":          "Completed",
				"startTime":       time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
				"endTime":         time.Now().Add(-50 * time.Minute).Format(time.RFC3339),
				"durationMs":      600000,
			},
			{
				"id":              "job-2",
				"workspaceId":     "workspace-1",
				"itemId":          "notebook-1",
				"itemDisplayName": "Data Analysis Notebook",
				"jobType":         "Notebook",
				"status":          "Failed",
				"startTime":       time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
				"endTime":         time.Now().Add(-1*time.Hour - 30*time.Minute).Format(time.RFC3339),
				"durationMs":      5400000,
				"failureReason":   "Connection timeout",
			},
			{
				"id":              "job-3",
				"workspaceId":     "workspace-2",
				"itemId":          "pipeline-2",
				"itemDisplayName": "Test Pipeline",
				"jobType":         "Pipeline",
				"status":          "Running",
				"startTime":       time.Now().Add(-10 * time.Minute).Format(time.RFC3339),
			},
		}
	}

	// Get real workspaces first
	workspaces, err := a.fabricClient.GetWorkspaces(a.ctx)
	if err != nil {
		fmt.Printf("Failed to get workspaces for jobs: %v\n", err)
		return []map[string]interface{}{}
	}

	// Get recent jobs across all workspaces
	jobs, err := a.fabricClient.GetRecentJobs(a.ctx, workspaces, 50)
	if err != nil {
		fmt.Printf("Failed to get jobs: %v\n", err)
		return []map[string]interface{}{
			{
				"id":              "error",
				"itemDisplayName": fmt.Sprintf("Error loading jobs: %v", err),
				"status":          "Error",
			},
		}
	}

	return jobs
}

// Greet returns a greeting for the given name (legacy method)
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}
