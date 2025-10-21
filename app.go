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
		// Continue with default config but set essential defaults
		cfg = &config.Config{
			Database: config.DatabaseConfig{
				Path:          "data/fabric-monitor.db",
				RetentionDays: 90,
			},
			Auth: config.AuthConfig{
				RedirectURI: "http://localhost:8400",
			},
			UI: config.UIConfig{
				PrimaryColor: "#00BCF2",
			},
		}
	}
	a.config = cfg

	// Initialize database with proper path validation
	dbPath := cfg.Database.Path
	if dbPath == "" {
		dbPath = "data/fabric-monitor.db"
		fmt.Printf("Warning: database path not set, using default: %s\n", dbPath)
	}
	database, err := db.NewDatabase(dbPath, cfg.Database.EncryptionKey)
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

		// Try to restore existing session from cache
		if token, err := a.auth.GetToken(ctx); err == nil {
			fmt.Println("Restored authentication from cache")
			a.currentToken = token
			a.fabricClient = fabric.NewClient(token.AccessToken)
		} else {
			fmt.Printf("No cached authentication found: %v\n", err)
		}
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
		fmt.Println("Fabric client not initialized, checking cache...")
		// Try to load from cache first
		cachedWorkspaces := a.GetWorkspacesFromCache()
		if len(cachedWorkspaces) > 0 {
			fmt.Printf("Loaded %d workspaces from cache\n", len(cachedWorkspaces))
			return cachedWorkspaces
		}

		// No cache, return mock data
		fmt.Println("No cached workspaces, returning mock data")
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
		fmt.Printf("Failed to get workspaces from API: %v, checking cache...\n", err)
		// Try cache as fallback
		cachedWorkspaces := a.GetWorkspacesFromCache()
		if len(cachedWorkspaces) > 0 {
			fmt.Printf("Loaded %d workspaces from cache as fallback\n", len(cachedWorkspaces))
			return cachedWorkspaces
		}

		return []map[string]interface{}{
			{
				"id":          "error",
				"displayName": fmt.Sprintf("Error loading workspaces: %v", err),
				"type":        "Error",
			},
		}
	}

	// Persist workspaces to DuckDB
	if a.db != nil {
		for _, ws := range workspaces {
			dbWorkspace := &db.Workspace{
				ID:          ws.ID,
				DisplayName: ws.DisplayName,
				Type:        ws.Type,
			}
			if ws.Description != "" {
				dbWorkspace.Description = &ws.Description
			}
			if err := a.db.SaveWorkspace(dbWorkspace); err != nil {
				fmt.Printf("Warning: failed to save workspace %s to database: %v\n", ws.ID, err)
			}
		}
		fmt.Printf("Persisted %d workspaces to database\n", len(workspaces))
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

	// Persist workspaces to database first (needed for foreign key constraints)
	fmt.Printf("DEBUG: a.db=%v, len(workspaces)=%d\n", a.db != nil, len(workspaces))
	if a.db != nil && len(workspaces) > 0 {
		for _, ws := range workspaces {
			dbWorkspace := &db.Workspace{
				ID:          ws.ID,
				DisplayName: ws.DisplayName,
				Type:        ws.Type,
			}
			if ws.Description != "" {
				dbWorkspace.Description = &ws.Description
			}
			if err := a.db.SaveWorkspace(dbWorkspace); err != nil {
				fmt.Printf("Warning: failed to save workspace %s to database: %v\n", ws.ID, err)
			}
		}
		fmt.Printf("Persisted %d workspaces to database\n", len(workspaces))
	} else {
		fmt.Printf("Skipping workspace persistence: db=%v, workspaces=%d\n", a.db != nil, len(workspaces))
	}

	// Check for last sync time to enable incremental loading
	// Use the max start_time from completed jobs (jobs with end_time) to ensure we don't miss any jobs
	var startTimeFrom *time.Time
	var cachedItemsByWorkspace map[string][]fabric.Item
	if a.db != nil {
		maxStartTime, err := a.db.GetMaxJobStartTime()
		if err == nil && maxStartTime != nil {
			startTimeFrom = maxStartTime
			fmt.Printf("Max completed job start time: %s, doing incremental load\n", maxStartTime.Format(time.RFC3339))

			// For incremental syncs, load cached items from database to avoid API calls
			cachedItemsByWorkspace = make(map[string][]fabric.Item)
			for _, ws := range workspaces {
				dbItems, err := a.db.GetItemsByWorkspace(ws.ID)
				if err == nil && len(dbItems) > 0 {
					// Convert db.Item to fabric.Item
					fabricItems := make([]fabric.Item, 0, len(dbItems))
					for _, dbItem := range dbItems {
						fabricItem := fabric.Item{
							ID:          dbItem.ID,
							DisplayName: dbItem.DisplayName,
							Type:        dbItem.Type,
						}
						if dbItem.Description != nil {
							fabricItem.Description = *dbItem.Description
						}
						fabricItems = append(fabricItems, fabricItem)
					}
					cachedItemsByWorkspace[ws.ID] = fabricItems
					fmt.Printf("Loaded %d cached items for workspace %s\n", len(fabricItems), ws.DisplayName)
				}
			}
		} else {
			fmt.Println("No previous jobs found, doing full load")
		}
	} // Get recent jobs across all workspaces (no limit - return all)
	// Pass startTimeFrom for incremental sync (will also fetch all in-progress jobs)
	// Pass cachedItemsByWorkspace to avoid fetching items from API during incremental syncs
	jobs, newItems, err := a.fabricClient.GetRecentJobs(a.ctx, workspaces, 0, startTimeFrom, cachedItemsByWorkspace)
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

	// Persist jobs to DuckDB
	if a.db != nil && len(jobs) > 0 {
		// First, persist any new items from the API (for full syncs or new items discovered)
		if len(newItems) > 0 {
			for _, fabricItem := range newItems {
				dbItem := db.Item{
					ID:          fabricItem.ID,
					WorkspaceID: fabricItem.WorkspaceID,
					DisplayName: fabricItem.DisplayName,
					Type:        fabricItem.Type,
				}
				if fabricItem.Description != "" {
					dbItem.Description = &fabricItem.Description
				}
				if err := a.db.SaveItem(&dbItem); err != nil {
					fmt.Printf("Warning: failed to save new item %s to database: %v\n", dbItem.ID, err)
				}
			}
			fmt.Printf("Persisted %d new items from API to database\n", len(newItems))
		}

		// Also persist all unique items that these jobs reference (to satisfy foreign key constraints)
		itemsMap := make(map[string]db.Item)
		for _, job := range jobs {
			itemID := job["itemId"].(string)
			if _, exists := itemsMap[itemID]; !exists {
				item := db.Item{
					ID:          itemID,
					WorkspaceID: job["workspaceId"].(string),
					DisplayName: job["itemDisplayName"].(string),
					Type:        job["itemType"].(string),
				}
				itemsMap[itemID] = item
			}
		}

		// Save all items referenced by jobs
		for _, item := range itemsMap {
			if err := a.db.SaveItem(&item); err != nil {
				fmt.Printf("Warning: failed to save item %s to database: %v\n", item.ID, err)
			}
		}
		fmt.Printf("Persisted %d unique items from jobs to database\n", len(itemsMap))

		// Now persist job instances
		dbJobs := make([]db.JobInstance, 0, len(jobs))
		for _, job := range jobs {
			// Parse start time
			startTime, err := time.Parse(time.RFC3339, job["startTime"].(string))
			if err != nil {
				fmt.Printf("Warning: failed to parse start time: %v\n", err)
				continue
			}

			dbJob := db.JobInstance{
				ID:          job["id"].(string),
				WorkspaceID: job["workspaceId"].(string),
				ItemID:      job["itemId"].(string),
				JobType:     job["jobType"].(string),
				Status:      job["status"].(string),
				StartTime:   startTime,
			}

			// Parse end time if present
			if endTimeStr, ok := job["endTime"].(string); ok && endTimeStr != "" {
				if endTime, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
					dbJob.EndTime = &endTime
				}
			}

			// Duration
			if durationMs, ok := job["durationMs"].(int64); ok {
				dbJob.DurationMs = &durationMs
			}

			// Failure reason
			if failureReason, ok := job["failureReason"].(string); ok && failureReason != "" {
				dbJob.FailureReason = &failureReason
			}

			dbJobs = append(dbJobs, dbJob)
		}

		if len(dbJobs) > 0 {
			if err := a.db.SaveJobInstances(dbJobs); err != nil {
				fmt.Printf("Warning: failed to save jobs to database: %v\n", err)
			} else {
				if startTimeFrom != nil {
					fmt.Printf("Persisted %d new/updated job instances to database (incremental)\n", len(dbJobs))
				} else {
					fmt.Printf("Persisted %d job instances to database (full sync)\n", len(dbJobs))
				}
				// Record sync metadata
				if err := a.db.UpdateSyncMetadata("job_instances", len(dbJobs), 0); err != nil {
					fmt.Printf("Warning: failed to update sync metadata: %v\n", err)
				}
			}
		}
	}

	// If doing incremental sync, merge with cached data to get complete view
	if startTimeFrom != nil && a.db != nil {
		fmt.Println("Merging fresh jobs with cached historical data...")
		cachedJobs := a.GetJobsFromCache()

		// Create a map of fresh job IDs for deduplication
		freshJobIDs := make(map[string]bool)
		for _, job := range jobs {
			if id, ok := job["id"].(string); ok {
				freshJobIDs[id] = true
			}
		}

		// Add cached jobs that aren't in the fresh results
		for _, cachedJob := range cachedJobs {
			if id, ok := cachedJob["id"].(string); ok {
				if !freshJobIDs[id] {
					jobs = append(jobs, cachedJob)
				}
			}
		}

		fmt.Printf("Total jobs after merge: %d (fresh: %d, cached: %d)\n", len(jobs), len(freshJobIDs), len(cachedJobs))
	}

	return jobs
}

// GetJobsFromCache retrieves jobs from the local DuckDB cache
func (a *App) GetJobsFromCache() []map[string]interface{} {
	if a.db == nil {
		return []map[string]interface{}{}
	}

	// Get all jobs from database
	filter := db.JobFilter{}
	jobs, err := a.db.GetJobInstances(filter)
	if err != nil {
		fmt.Printf("Failed to get jobs from cache: %v\n", err)
		return []map[string]interface{}{}
	}

	// Convert to map format for frontend
	result := make([]map[string]interface{}, 0, len(jobs))
	for _, job := range jobs {
		jobMap := map[string]interface{}{
			"id":          job.ID,
			"workspaceId": job.WorkspaceID,
			"itemId":      job.ItemID,
			"jobType":     job.JobType,
			"status":      job.Status,
			"startTime":   job.StartTime.Format(time.RFC3339),
		}

		// Add item display name and type from the joined data
		if job.ItemDisplayName != nil {
			jobMap["itemDisplayName"] = *job.ItemDisplayName
		} else {
			jobMap["itemDisplayName"] = job.ItemID // Fallback to ID if name not available
		}

		if job.ItemType != nil {
			jobMap["itemType"] = *job.ItemType
		} else {
			jobMap["itemType"] = job.JobType // Fallback to job type
		}

		// Add workspace name from the joined data
		if job.WorkspaceName != nil {
			jobMap["workspaceName"] = *job.WorkspaceName
		}

		if job.EndTime != nil {
			jobMap["endTime"] = job.EndTime.Format(time.RFC3339)
		}
		if job.DurationMs != nil {
			jobMap["durationMs"] = *job.DurationMs
		}
		if job.FailureReason != nil {
			jobMap["failureReason"] = *job.FailureReason
		}

		result = append(result, jobMap)
	}

	fmt.Printf("Loaded %d jobs from cache\n", len(result))
	return result
}

// GetWorkspacesFromCache retrieves workspaces from the local DuckDB cache
func (a *App) GetWorkspacesFromCache() []map[string]interface{} {
	if a.db == nil {
		return []map[string]interface{}{}
	}

	// Get all workspaces from database
	workspaces, err := a.db.GetWorkspaces()
	if err != nil {
		fmt.Printf("Failed to get workspaces from cache: %v\n", err)
		return []map[string]interface{}{}
	}

	// Convert to map format for frontend
	result := make([]map[string]interface{}, 0, len(workspaces))
	for _, ws := range workspaces {
		wsMap := map[string]interface{}{
			"id":          ws.ID,
			"displayName": ws.DisplayName,
			"type":        ws.Type,
		}

		if ws.Description != nil {
			wsMap["description"] = *ws.Description
		}

		result = append(result, wsMap)
	}

	fmt.Printf("Loaded %d workspaces from cache\n", len(result))
	return result
}

// GetLastSyncTime returns the last time data was synced from the API
func (a *App) GetLastSyncTime() string {
	if a.db == nil {
		return ""
	}

	lastSync, err := a.db.GetLastSyncTime("job_instances")
	if err != nil || lastSync == nil {
		return ""
	}

	return lastSync.Format(time.RFC3339)
}

// Greet returns a greeting for the given name (legacy method)
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}
