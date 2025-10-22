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

// shutdown is called when the app is closing
func (a *App) shutdown(ctx context.Context) {
	fmt.Println("Shutting down application...")

	// Close database connection
	if a.db != nil {
		if err := a.db.Close(); err != nil {
			fmt.Printf("Error closing database: %v\n", err)
		} else {
			fmt.Println("Database connection closed successfully")
		}
	}

	// Clean up authentication if needed
	if a.auth != nil {
		// Auth cleanup is already handled by Logout if needed
		fmt.Println("Authentication cleanup complete")
	}

	fmt.Println("Shutdown complete")
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
	// GetMaxJobStartTime returns either:
	// - The MIN start_time of in-progress jobs (to re-check them for completion), OR
	// - The MAX start_time of completed jobs (if no in-progress jobs exist)
	var startTimeFrom *time.Time
	var cachedItemsByWorkspace map[string][]fabric.Item
	if a.db != nil {
		maxStartTime, err := a.db.GetMaxJobStartTime()
		if err == nil && maxStartTime != nil {
			startTimeFrom = maxStartTime
			fmt.Printf("Incremental load starting from: %s\n", maxStartTime.Format(time.RFC3339))

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
	}
	// Get recent jobs across all workspaces (no limit - return all)
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

	// If doing incremental sync, get cached jobs BEFORE persisting to database
	var cachedJobs []map[string]interface{}
	if startTimeFrom != nil && a.db != nil {
		cachedJobs = a.GetJobsFromCache()
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

			// Root activity ID
			if rootActivityId, ok := job["rootActivityId"].(string); ok && rootActivityId != "" {
				dbJob.RootActivityID = &rootActivityId
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

	// After all jobs are persisted, fetch activity runs for completed DataPipeline jobs
	// This runs in a background goroutine to avoid blocking the UI response
	// We do this AFTER the persistence block to ensure all jobs are committed to the database
	if a.db != nil && len(jobs) > 0 {
		go a.enrichPipelineJobsWithActivityRuns()
	}

	// If doing incremental sync, merge with cached data to get complete view
	if startTimeFrom != nil && a.db != nil && len(cachedJobs) > 0 {
		fmt.Println("Merging fresh jobs with cached historical data...")

		// Create a map of fresh jobs by ID for quick lookup
		freshJobMap := make(map[string]map[string]interface{})
		for _, job := range jobs {
			if id, ok := job["id"].(string); ok {
				freshJobMap[id] = job
			}
		}

		// Start with fresh jobs (these have the latest data)
		mergedJobs := make([]map[string]interface{}, 0, len(cachedJobs))
		mergedJobs = append(mergedJobs, jobs...)

		// Add cached jobs that aren't in the fresh results
		for _, cachedJob := range cachedJobs {
			if id, ok := cachedJob["id"].(string); ok {
				if _, exists := freshJobMap[id]; !exists {
					mergedJobs = append(mergedJobs, cachedJob)
				}
			}
		}

		fmt.Printf("Total jobs after merge: %d (fresh: %d, cached: %d, replaced: %d)\n",
			len(mergedJobs), len(jobs), len(cachedJobs), len(freshJobMap))

		return mergedJobs
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
		if job.RootActivityID != nil {
			jobMap["rootActivityId"] = *job.RootActivityID
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

// GetAnalytics returns comprehensive analytics data for the dashboard
func (a *App) GetAnalytics(days int) map[string]interface{} {
	if a.db == nil {
		return map[string]interface{}{
			"error": "Database not initialized",
		}
	}

	if days <= 0 {
		days = 7 // Default to 7 days
	}

	result := make(map[string]interface{})

	// Get daily stats
	dailyStats, err := a.db.GetDailyStats(days)
	if err != nil {
		fmt.Printf("Failed to get daily stats: %v\n", err)
		result["dailyStatsError"] = err.Error()
	} else {
		result["dailyStats"] = dailyStats
	}

	// Get workspace stats
	workspaceStats, err := a.db.GetWorkspaceStats(days)
	if err != nil {
		fmt.Printf("Failed to get workspace stats: %v\n", err)
		result["workspaceStatsError"] = err.Error()
	} else {
		result["workspaceStats"] = workspaceStats
	}

	// Get item type stats
	itemTypeStats, err := a.db.GetItemTypeStats(days)
	if err != nil {
		fmt.Printf("Failed to get item type stats: %v\n", err)
		result["itemTypeStatsError"] = err.Error()
	} else {
		result["itemTypeStats"] = itemTypeStats
	}

	// Get recent failures (last 10 within the time period)
	recentFailures, err := a.db.GetRecentFailures(10, days)
	if err != nil {
		fmt.Printf("Failed to get recent failures: %v\n", err)
		result["recentFailuresError"] = err.Error()
	} else {
		result["recentFailures"] = recentFailures
	}

	// Get long-running jobs (50% or more above average, last 10)
	longRunningJobs, err := a.db.GetLongRunningJobs(days, 50.0, 10)
	if err != nil {
		fmt.Printf("Failed to get long-running jobs: %v\n", err)
		result["longRunningJobsError"] = err.Error()
	} else {
		result["longRunningJobs"] = longRunningJobs
	}

	// Get overall stats - calculated entirely in DuckDB for consistency
	overallStats, err := a.db.GetOverallStats(days)
	if err != nil {
		fmt.Printf("Failed to get overall stats: %v\n", err)
		result["overallStatsError"] = err.Error()
	} else {
		result["overallStats"] = map[string]interface{}{
			"totalJobs":     overallStats.TotalJobs,
			"successful":    overallStats.Successful,
			"failed":        overallStats.Failed,
			"running":       overallStats.Running,
			"successRate":   overallStats.SuccessRate,
			"avgDurationMs": overallStats.AvgDurationMs,
		}
	}

	result["days"] = days

	return result
}

// GetAnalyticsFiltered returns comprehensive analytics data with optional filters
func (a *App) GetAnalyticsFiltered(days int, workspaceIDs []string, itemTypes []string, itemNameSearch string) map[string]interface{} {
	if a.db == nil {
		return map[string]interface{}{
			"error": "Database not initialized",
		}
	}

	if days <= 0 {
		days = 7 // Default to 7 days
	}

	result := make(map[string]interface{})

	// Get daily stats
	dailyStats, err := a.db.GetDailyStatsFiltered(days, workspaceIDs, itemTypes, itemNameSearch)
	if err != nil {
		fmt.Printf("Failed to get daily stats: %v\n", err)
		result["dailyStatsError"] = err.Error()
	} else {
		result["dailyStats"] = dailyStats
	}

	// Get workspace stats
	workspaceStats, err := a.db.GetWorkspaceStatsFiltered(days, workspaceIDs, itemTypes, itemNameSearch)
	if err != nil {
		fmt.Printf("Failed to get workspace stats: %v\n", err)
		result["workspaceStatsError"] = err.Error()
	} else {
		result["workspaceStats"] = workspaceStats
	}

	// Get item type stats
	itemTypeStats, err := a.db.GetItemTypeStatsFiltered(days, workspaceIDs, itemTypes, itemNameSearch)
	if err != nil {
		fmt.Printf("Failed to get item type stats: %v\n", err)
		result["itemTypeStatsError"] = err.Error()
	} else {
		result["itemTypeStats"] = itemTypeStats
	}

	// Get recent failures (last 10 within the time period)
	recentFailures, err := a.db.GetRecentFailuresFiltered(10, days, workspaceIDs, itemTypes, itemNameSearch)
	if err != nil {
		fmt.Printf("Failed to get recent failures: %v\n", err)
		result["recentFailuresError"] = err.Error()
	} else {
		result["recentFailures"] = recentFailures
	}

	// Get long-running jobs (50% or more above average, last 10)
	longRunningJobs, err := a.db.GetLongRunningJobsFiltered(days, 50.0, 10, workspaceIDs, itemTypes, itemNameSearch)
	if err != nil {
		fmt.Printf("Failed to get long-running jobs: %v\n", err)
		result["longRunningJobsError"] = err.Error()
	} else {
		result["longRunningJobs"] = longRunningJobs
	}

	// Get overall stats - calculated entirely in DuckDB for consistency
	overallStats, err := a.db.GetOverallStatsFiltered(days, workspaceIDs, itemTypes, itemNameSearch)
	if err != nil {
		fmt.Printf("Failed to get overall stats: %v\n", err)
		result["overallStatsError"] = err.Error()
	} else {
		result["overallStats"] = map[string]interface{}{
			"totalJobs":     overallStats.TotalJobs,
			"successful":    overallStats.Successful,
			"failed":        overallStats.Failed,
			"running":       overallStats.Running,
			"successRate":   overallStats.SuccessRate,
			"avgDurationMs": overallStats.AvgDurationMs,
		}
	}

	result["days"] = days

	return result
}

// GetAvailableItemTypes returns distinct item types that have job data
func (a *App) GetAvailableItemTypes(days int, workspaceIDs []string) []string {
	if a.db == nil {
		return []string{}
	}

	if days <= 0 {
		days = 7
	}

	itemTypes, err := a.db.GetAvailableItemTypes(days, workspaceIDs)
	if err != nil {
		fmt.Printf("Failed to get available item types: %v\n", err)
		return []string{}
	}

	return itemTypes
}

// GetItemStatsByWorkspace returns item-level statistics for a specific workspace
func (a *App) GetItemStatsByWorkspace(workspaceID string, days int) map[string]interface{} {
	if a.db == nil {
		return map[string]interface{}{
			"error": "Database not initialized",
		}
	}

	if days <= 0 {
		days = 7
	}

	itemStats, err := a.db.GetItemStatsByWorkspace(workspaceID, days)
	if err != nil {
		return map[string]interface{}{
			"error": err.Error(),
		}
	}

	return map[string]interface{}{
		"items": itemStats,
		"days":  days,
	}
}

// GetItemStatsByJobType returns item-level statistics for a specific job type
func (a *App) GetItemStatsByJobType(itemType string, days int) map[string]interface{} {
	if a.db == nil {
		return map[string]interface{}{
			"error": "Database not initialized",
		}
	}

	if days <= 0 {
		days = 7
	}

	itemStats, err := a.db.GetItemStatsByJobType(itemType, days)
	if err != nil {
		return map[string]interface{}{
			"error": err.Error(),
		}
	}

	return map[string]interface{}{
		"items": itemStats,
		"days":  days,
	}
}

// enrichPipelineJobsWithActivityRuns fetches activity runs for completed pipeline jobs
// This runs in the background to avoid blocking the main sync process
// Uses parallel processing with worker pools for scalability
func (a *App) enrichPipelineJobsWithActivityRuns() {
	if a.db == nil {
		return
	}

	// Get all completed pipeline jobs without activity runs (removed LIMIT)
	query := `
		SELECT j.id, j.workspace_id, j.start_time, j.end_time
		FROM job_instances j
		LEFT JOIN items i ON j.item_id = i.id
		WHERE i.type = 'DataPipeline'
			AND j.end_time IS NOT NULL
			AND j.activity_runs IS NULL
		ORDER BY j.start_time DESC
	`

	rows, err := a.db.GetConnection().Query(query)
	if err != nil {
		fmt.Printf("Failed to query pipeline jobs for activity runs: %v\n", err)
		return
	}
	defer rows.Close()

	type pipelineJob struct {
		ID          string
		WorkspaceID string
		StartTime   time.Time
		EndTime     time.Time
	}

	var jobs []pipelineJob
	for rows.Next() {
		var job pipelineJob
		if err := rows.Scan(&job.ID, &job.WorkspaceID, &job.StartTime, &job.EndTime); err != nil {
			fmt.Printf("Failed to scan pipeline job: %v\n", err)
			continue
		}
		jobs = append(jobs, job)
	}

	if len(jobs) == 0 {
		return
	}

	fmt.Printf("Fetching activity runs for %d pipeline jobs in parallel...\n", len(jobs))
	startTime := time.Now()

	// Create worker pool for parallel processing (limit to 20 concurrent requests)
	pool := fabric.NewWorkerPool(20)

	// Channel to collect results
	type jobResult struct {
		jobID         string
		activityRuns  []db.ActivityRun
		err           error
		activityCount int
	}
	results := make(chan jobResult, len(jobs))

	// Process each job in parallel
	for _, job := range jobs {
		job := job // Capture for goroutine

		pool.Submit(a.ctx, func() error {
			result := jobResult{jobID: job.ID}

			// Add some buffer time before and after the job run
			startTime := job.StartTime.Add(-1 * time.Minute)
			endTime := job.EndTime.Add(1 * time.Minute)

			activityRuns, err := a.fabricClient.QueryActivityRuns(a.ctx, job.WorkspaceID, job.ID, startTime, endTime)
			if err != nil {
				result.err = err
				results <- result
				return nil
			}

			result.activityCount = len(activityRuns)

			// Convert fabric.ActivityRun to db.ActivityRun
			dbActivityRuns := make([]db.ActivityRun, len(activityRuns))
			for i, ar := range activityRuns {
				dbActivityRuns[i] = db.ActivityRun{
					PipelineID:              ar.PipelineID,
					PipelineRunID:           ar.PipelineRunID,
					ActivityName:            ar.ActivityName,
					ActivityType:            ar.ActivityType,
					ActivityRunID:           ar.ActivityRunID,
					Status:                  ar.Status,
					ActivityRunStart:        ar.ActivityRunStart,
					ActivityRunEnd:          ar.ActivityRunEnd,
					DurationInMs:            ar.DurationInMs,
					Input:                   ar.Input,
					Output:                  ar.Output,
					Error:                   db.ActivityError(ar.Error),
					RetryAttempt:            ar.RetryAttempt,
					IterationHash:           ar.IterationHash,
					UserProperties:          ar.UserProperties,
					RecoveryStatus:          ar.RecoveryStatus,
					IntegrationRuntimeNames: ar.IntegrationRuntimeNames,
					ExecutionDetails:        ar.ExecutionDetails,
				}
			}

			result.activityRuns = dbActivityRuns
			results <- result
			return nil
		})
	}

	// Wait for all jobs to complete
	pool.Wait()
	close(results)

	// Process results and save to database
	successCount := 0
	errorCount := 0
	totalActivities := 0

	for result := range results {
		if result.err != nil {
			fmt.Printf("Failed to fetch activity runs for job %s: %v\n", result.jobID, result.err)
			errorCount++
			// Still mark as processed with empty array to avoid retrying
			if err := a.db.UpdateJobInstanceActivityRuns(result.jobID, []db.ActivityRun{}); err != nil {
				fmt.Printf("Failed to save empty activity runs for job %s: %v\n", result.jobID, err)
			}
			continue
		}

		if err := a.db.UpdateJobInstanceActivityRuns(result.jobID, result.activityRuns); err != nil {
			fmt.Printf("Failed to save activity runs for job %s: %v\n", result.jobID, err)
			errorCount++
			continue
		}

		successCount++
		totalActivities += result.activityCount
	}

	elapsed := time.Since(startTime)
	fmt.Printf("Activity runs sync completed in %v\n", elapsed)
	fmt.Printf("Successfully fetched activity runs for %d/%d pipeline jobs (%d activities, %d errors)\n",
		successCount, len(jobs), totalActivities, errorCount)
}

// GetJobInstanceWithActivities retrieves a job instance with its activity runs
func (a *App) GetJobInstanceWithActivities(jobID string) map[string]interface{} {
	if a.db == nil {
		return map[string]interface{}{
			"error": "Database not initialized",
		}
	}

	job, err := a.db.GetJobInstanceWithActivities(jobID)
	if err != nil {
		return map[string]interface{}{
			"error": fmt.Sprintf("Failed to get job: %v", err),
		}
	}

	return map[string]interface{}{
		"job": job,
	}
}

// GetChildExecutions retrieves child pipeline and notebook executions for a job
func (a *App) GetChildExecutions(jobID string) map[string]interface{} {
	if a.db == nil {
		return map[string]interface{}{
			"error": "Database not initialized",
		}
	}

	children, err := a.db.GetChildExecutions(jobID)
	if err != nil {
		return map[string]interface{}{
			"error": fmt.Sprintf("Failed to get child executions: %v", err),
		}
	}

	return map[string]interface{}{
		"children": children,
		"count":    len(children),
	}
}

// GetActivityRunsSample returns a sample of activity runs for debugging (first DataPipeline with activity runs)
func (a *App) GetActivityRunsSample() map[string]interface{} {
	if a.db == nil {
		return map[string]interface{}{
			"error": "Database not initialized",
		}
	}

	query := `
		SELECT j.id, i.display_name, j.activity_runs
		FROM job_instances j
		LEFT JOIN items i ON j.item_id = i.id
		WHERE i.type = 'DataPipeline'
			AND j.activity_runs IS NOT NULL
			AND json_array_length(CAST(j.activity_runs AS JSON[])) > 0
		LIMIT 1
	`

	var jobID string
	var displayName string
	var activityRunsJSON string

	err := a.db.GetConnection().QueryRow(query).Scan(&jobID, &displayName, &activityRunsJSON)
	if err != nil {
		return map[string]interface{}{
			"error": fmt.Sprintf("No pipeline jobs with activity runs found: %v", err),
		}
	}

	return map[string]interface{}{
		"jobID":        jobID,
		"displayName":  displayName,
		"activityRuns": activityRunsJSON,
	}
}

// Greet returns a greeting for the given name (legacy method)
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}
