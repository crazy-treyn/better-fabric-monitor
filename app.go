package main

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"better-fabric-monitor/internal/auth"
	"better-fabric-monitor/internal/config"
	"better-fabric-monitor/internal/db"
	"better-fabric-monitor/internal/fabric"
	"better-fabric-monitor/internal/logger"
	"better-fabric-monitor/internal/utils"
)

// App struct
type App struct {
	ctx                 context.Context
	config              *config.Config
	auth                *auth.AuthManager
	db                  *db.Database
	fabricClient        *fabric.Client
	currentToken        *auth.Token
	parquetExportMutex  sync.Mutex
	parquetExportActive bool
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Initialize log buffer
	logger.Init(2000)

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Log("Failed to load config: %v\n", err)
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
			App: config.AppConfig{
				Name:     "Better Fabric Monitor",
				Version:  "0.2.3",
				LogLevel: "info",
				Debug:    false,
			},
		}
	}
	a.config = cfg

	// Initialize database with proper path validation
	dbPath := cfg.Database.Path
	if dbPath == "" {
		dbPath = "data/fabric-monitor.db"
		logger.Log("Warning: database path not set, using default: %s\n", dbPath)
	}
	database, err := db.NewDatabase(dbPath, cfg.Database.EncryptionKey)
	if err != nil {
		logger.Log("Failed to initialize database: %v\n", err)
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
		logger.Log("Failed to initialize auth: %v\n", err)
	} else {
		a.auth = authManager

		// Try to restore existing session from cache
		if token, err := a.auth.GetToken(ctx); err == nil {
			logger.Log("Restored authentication from cache\n")
			a.currentToken = token
			a.fabricClient = fabric.NewClient(token.AccessToken)
		} else {
			logger.Log("No cached authentication found: %v\n", err)
		}
	}

	// Start Parquet export on startup
	a.StartParquetExport()
}

// shutdown is called when the app is closing
func (a *App) shutdown(ctx context.Context) {
	logger.Log("Shutting down application...\n")

	// Close database connection
	if a.db != nil {
		if err := a.db.Close(); err != nil {
			logger.Log("Error closing database: %v\n", err)
		} else {
			logger.Log("Database connection closed successfully\n")
		}
	}

	// Clean up authentication if needed
	if a.auth != nil {
		// Auth cleanup is already handled by Logout if needed
		logger.Log("Authentication cleanup complete\n")
	}

	logger.Log("Shutdown complete\n")
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

// ensureValidToken checks if the current token is valid and refreshes if needed
// Returns error if token refresh fails (requires re-authentication)
func (a *App) ensureValidToken() error {
	// If no auth manager, cannot proceed
	if a.auth == nil {
		return fmt.Errorf("authentication not initialized")
	}

	// Check if token exists and is not expired (5-minute buffer)
	if a.currentToken != nil && time.Now().Before(a.currentToken.ExpiresAt.Add(-5*time.Minute)) {
		// Token is still valid
		return nil
	}

	logger.Log("Token expired or about to expire, refreshing...\n")

	// Try to refresh token silently
	token, err := a.auth.GetToken(a.ctx)
	if err != nil {
		logger.Log("ERROR: Token refresh failed: %v\n", err)
		return fmt.Errorf("token refresh failed: %w", err)
	}

	// Update token and recreate Fabric client
	a.currentToken = token
	a.fabricClient = fabric.NewClient(token.AccessToken)
	logger.Log("Token refreshed successfully, expires at: %s\n", token.ExpiresAt.Format(time.RFC3339))

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
	// Check and refresh token if needed
	if err := a.ensureValidToken(); err != nil {
		logger.Log("Authentication required: %v\n", err)
		// Check if we have cached data
		cachedWorkspaces := a.GetWorkspacesFromCache()
		hasCachedData := len(cachedWorkspaces) > 0

		if hasCachedData {
			logger.Log("Loaded %d workspaces from cache (authentication expired)\n", len(cachedWorkspaces))
			// Return cached data with error flag
			return append([]map[string]interface{}{
				{
					"error":                 "authentication_required",
					"message":               "Your session has expired. Please sign in again or continue with cached data.",
					"cached_data_available": true,
					"_is_error_marker":      true, // Special flag so frontend can filter this out
				},
			}, cachedWorkspaces...)
		}

		// No cached data, return error only
		return []map[string]interface{}{
			{
				"error":                 "authentication_required",
				"message":               "Your session has expired. Please sign in again.",
				"cached_data_available": false,
			},
		}
	}

	// Get real workspaces from Fabric API
	workspaces, err := a.fabricClient.GetWorkspaces(a.ctx)
	if err != nil {
		logger.Log("Failed to get workspaces from API: %v, checking cache...\n", err)
		// Try cache as fallback
		cachedWorkspaces := a.GetWorkspacesFromCache()
		if len(cachedWorkspaces) > 0 {
			logger.Log("Loaded %d workspaces from cache as fallback\n", len(cachedWorkspaces))
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
				logger.Log("Warning: failed to save workspace %s to database: %v\n", ws.ID, err)
			}
		}
		logger.Log("Persisted %d workspaces to database\n", len(workspaces))
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
	// Check and refresh token if needed
	if err := a.ensureValidToken(); err != nil {
		logger.Log("Authentication required: %v\n", err)
		// Check if we have cached data
		cachedJobs := a.GetJobsFromCache()
		hasCachedData := len(cachedJobs) > 0

		if hasCachedData {
			logger.Log("Loaded %d jobs from cache (authentication expired)\n", len(cachedJobs))
			// Return cached data with error flag
			return append([]map[string]interface{}{
				{
					"error":                 "authentication_required",
					"message":               "Your session has expired. Please sign in again or continue with cached data.",
					"cached_data_available": true,
					"_is_error_marker":      true, // Special flag so frontend can filter this out
				},
			}, cachedJobs...)
		}

		// No cached data, return error only
		return []map[string]interface{}{
			{
				"error":                 "authentication_required",
				"message":               "Your session has expired. Please sign in again.",
				"cached_data_available": false,
			},
		}
	}

	// Get real workspaces first
	workspaces, err := a.fabricClient.GetWorkspaces(a.ctx)
	if err != nil {
		logger.Log("Failed to get workspaces for jobs: %v\n", err)
		return []map[string]interface{}{}
	}

	// Persist workspaces to database first (needed for foreign key constraints)
	logger.Log("DEBUG: a.db=%v, len(workspaces)=%d\n", a.db != nil, len(workspaces))
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
				logger.Log("Warning: failed to save workspace %s to database: %v\n", ws.ID, err)
			}
		}
		logger.Log("Persisted %d workspaces to database\n", len(workspaces))
	} else {
		logger.Log("Skipping workspace persistence: db=%v, workspaces=%d\n", a.db != nil, len(workspaces))
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
			logger.Log("Incremental load starting from: %s\n", maxStartTime.Format(time.RFC3339))

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
					logger.Log("Loaded %d cached items for workspace %s\n", len(fabricItems), ws.DisplayName)
				}
			}
		} else {
			logger.Log("No previous jobs found, doing full load")
		}
	}
	// Get recent jobs across all workspaces (no limit - return all)
	// Pass startTimeFrom for incremental sync (will also fetch all in-progress jobs)
	// Pass cachedItemsByWorkspace to avoid fetching items from API during incremental syncs
	jobs, newItems, err := a.fabricClient.GetRecentJobs(a.ctx, workspaces, 0, startTimeFrom, cachedItemsByWorkspace)
	if err != nil {
		logger.Log("Failed to get jobs: %v\n", err)
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
					logger.Log("Warning: failed to save new item %s to database: %v\n", dbItem.ID, err)
				}
			}
			logger.Log("Persisted %d new items from API to database\n", len(newItems))
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
				logger.Log("Warning: failed to save item %s to database: %v\n", item.ID, err)
			}
		}
		logger.Log("Persisted %d unique items from jobs to database\n", len(itemsMap))

		// Now persist job instances
		dbJobs := make([]db.JobInstance, 0, len(jobs))
		for _, job := range jobs {
			// Parse start time
			startTime, err := time.Parse(time.RFC3339, job["startTime"].(string))
			if err != nil {
				logger.Log("Warning: failed to parse start time: %v\n", err)
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
				logger.Log("Warning: failed to save jobs to database: %v\n", err)
			} else {
				if startTimeFrom != nil {
					logger.Log("Persisted %d new/updated job instances to database (incremental)\n", len(dbJobs))
				} else {
					logger.Log("Persisted %d job instances to database (full sync)\n", len(dbJobs))
				}
				// Record sync metadata
				if err := a.db.UpdateSyncMetadata("job_instances", len(dbJobs), 0); err != nil {
					logger.Log("Warning: failed to update sync metadata: %v\n", err)
				}
			}
		}
	}

	// After all jobs are persisted, fetch activity runs for completed DataPipeline jobs
	// This blocks until enrichment completes to ensure child executions are available when UI loads
	// We do this AFTER the persistence block to ensure all jobs are committed to the database
	if a.db != nil {
		// Sync notebook sessions to get livyID for notebook deep links
		// This runs synchronously to ensure all livyIDs are available before UI loads
		// Run unconditionally during incremental refresh to backfill historical notebooks
		if len(jobs) > 0 || startTimeFrom != nil {
			if err := a.SyncNotebookSessions(); err != nil {
				logger.Log("Warning: failed to sync notebook sessions: %v\n", err)
			}
		}

		if len(jobs) > 0 {
			a.enrichPipelineJobsWithActivityRuns()
		}
	}

	// If doing incremental sync, get cached jobs AFTER enrichment to ensure fresh activity_runs data
	var cachedJobs []map[string]interface{}
	if startTimeFrom != nil && a.db != nil {
		cachedJobs = a.GetJobsFromCache()
	}

	// Add Fabric deep link URLs to jobs
	if a.db != nil {
		// Now get livyIDs from database and add Fabric deep link URLs to jobs
		jobIDs := make([]string, 0, len(jobs)+len(cachedJobs))
		for _, job := range jobs {
			if jobID, ok := job["id"].(string); ok {
				jobIDs = append(jobIDs, jobID)
			}
		}
		// Include cached notebook jobs so we can regenerate their URLs with fresh Livy data
		for _, cachedJob := range cachedJobs {
			if jobID, ok := cachedJob["id"].(string); ok {
				if itemType, ok := cachedJob["itemType"].(string); ok && itemType == "Notebook" {
					jobIDs = append(jobIDs, jobID)
				}
			}
		}

		livyIDMap := make(map[string]string)
		if len(jobIDs) > 0 {
			var err error
			livyIDMap, err = a.db.GetLivyIDsByJobInstanceIDs(jobIDs)
			if err != nil {
				logger.Log("Warning: failed to get livyIDs from database: %v\n", err)
			}
		}

		// Add Fabric deep link URLs to all jobs
		for i := range jobs {
			job := jobs[i]
			workspaceID, _ := job["workspaceId"].(string)
			itemID, _ := job["itemId"].(string)
			itemType, _ := job["itemType"].(string)
			jobID, _ := job["id"].(string)

			// Check if we have a livyID for this job
			var livyIDPtr *string
			if livyID, exists := livyIDMap[jobID]; exists && livyID != "" {
				livyIDPtr = &livyID
			}

			fabricURL := utils.GenerateFabricURL(workspaceID, itemID, itemType, jobID, livyIDPtr)
			if fabricURL != "" {
				jobs[i]["fabricUrl"] = fabricURL
			}
		}

		// Regenerate URLs for cached notebook jobs with fresh Livy data
		for i := range cachedJobs {
			cachedJob := cachedJobs[i]
			itemType, _ := cachedJob["itemType"].(string)
			if itemType != "Notebook" {
				continue
			}

			workspaceID, _ := cachedJob["workspaceId"].(string)
			itemID, _ := cachedJob["itemId"].(string)
			jobID, _ := cachedJob["id"].(string)

			// Check if we have a livyID for this cached job
			var livyIDPtr *string
			if livyID, exists := livyIDMap[jobID]; exists && livyID != "" {
				livyIDPtr = &livyID
			}

			fabricURL := utils.GenerateFabricURL(workspaceID, itemID, itemType, jobID, livyIDPtr)
			if fabricURL != "" {
				cachedJobs[i]["fabricUrl"] = fabricURL
			}
		}
	}

	// If doing incremental sync, merge with cached data to get complete view
	if startTimeFrom != nil && a.db != nil && len(cachedJobs) > 0 {
		logger.Log("Merging fresh jobs with cached historical data...")

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

		logger.Log("Total jobs after merge: %d (fresh: %d, cached: %d, replaced: %d)\n",
			len(mergedJobs), len(jobs), len(cachedJobs), len(freshJobMap))

		// Trigger Parquet export after data sync
		a.StartParquetExport()

		return mergedJobs
	}

	// Trigger Parquet export after data sync
	a.StartParquetExport()

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
		logger.Log("Failed to get jobs from cache: %v\n", err)
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

		var itemType string
		if job.ItemType != nil {
			jobMap["itemType"] = *job.ItemType
			itemType = *job.ItemType
		} else {
			jobMap["itemType"] = job.JobType // Fallback to job type
			itemType = job.JobType
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

		// Generate Fabric deep link URL
		fabricURL := utils.GenerateFabricURL(job.WorkspaceID, job.ItemID, itemType, job.ID, job.LivyID)
		if fabricURL != "" {
			jobMap["fabricUrl"] = fabricURL
		}

		result = append(result, jobMap)
	}

	logger.Log("Loaded %d jobs from cache\n", len(result))
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
		logger.Log("Failed to get workspaces from cache: %v\n", err)
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

	logger.Log("Loaded %d workspaces from cache\n", len(result))
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

// StartParquetExport triggers an async export of tables to Parquet format
func (a *App) StartParquetExport() {
	// Skip if feature is disabled
	if !a.config.Database.EnableReadOnlyReplica {
		return
	}

	// Skip if database is not initialized
	if a.db == nil {
		return
	}

	// Check if an export is already running
	a.parquetExportMutex.Lock()
	if a.parquetExportActive {
		a.parquetExportMutex.Unlock()
		logger.Log("[PARQUET] Export already in progress, skipping\n")
		return
	}
	a.parquetExportActive = true
	a.parquetExportMutex.Unlock()

	// Run export in goroutine to avoid blocking
	go func() {
		defer func() {
			a.parquetExportMutex.Lock()
			a.parquetExportActive = false
			a.parquetExportMutex.Unlock()
		}()

		logger.Log("[PARQUET] Starting export to Parquet files...\n")
		startTime := time.Now()

		// Export all tables to Parquet
		stats, err := a.db.ExportTablesToParquet(a.config.Database.ParquetPath)
		if err != nil {
			logger.Log("[PARQUET] ERROR: Export failed: %v\n", err)
			return
		}

		// Log export statistics
		totalRecords := 0
		successCount := 0
		for _, stat := range stats {
			if stat.Success {
				successCount++
				totalRecords += stat.RecordCount
			}
		}

		logger.Log("[PARQUET] Export completed: %d/%d tables successful, %d total records in %dms\n",
			successCount, len(stats), totalRecords, time.Since(startTime).Milliseconds())

		// Create or verify read-only database
		if err := db.CreateReadOnlyDatabase(a.config.Database.ReadOnlyPath, a.config.Database.ParquetPath); err != nil {
			logger.Log("[PARQUET] ERROR: Failed to create read-only database: %v\n", err)
			return
		}

		logger.Log("[PARQUET] Read-only replica ready at: %s\n", a.config.Database.ReadOnlyPath)
	}()
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
		logger.Log("Failed to get daily stats: %v\n", err)
		result["dailyStatsError"] = err.Error()
	} else {
		result["dailyStats"] = dailyStats
	}

	// Get workspace stats
	workspaceStats, err := a.db.GetWorkspaceStats(days)
	if err != nil {
		logger.Log("Failed to get workspace stats: %v\n", err)
		result["workspaceStatsError"] = err.Error()
	} else {
		result["workspaceStats"] = workspaceStats
	}

	// Get item type stats
	itemTypeStats, err := a.db.GetItemTypeStats(days)
	if err != nil {
		logger.Log("Failed to get item type stats: %v\n", err)
		result["itemTypeStatsError"] = err.Error()
	} else {
		result["itemTypeStats"] = itemTypeStats
	}

	// Get recent failures (last 10 within the time period)
	recentFailures, err := a.db.GetRecentFailures(10, days)
	if err != nil {
		logger.Log("Failed to get recent failures: %v\n", err)
		result["recentFailuresError"] = err.Error()
	} else {
		// Add Fabric URLs to failures
		failuresWithURLs := make([]map[string]interface{}, 0, len(recentFailures))
		for _, failure := range recentFailures {
			failureMap := map[string]interface{}{
				"id":              failure.ID,
				"workspaceId":     failure.WorkspaceID,
				"workspaceName":   failure.WorkspaceName,
				"itemId":          failure.ItemID,
				"itemDisplayName": failure.ItemDisplayName,
				"itemType":        failure.ItemType,
				"jobType":         failure.JobType,
				"startTime":       failure.StartTime.Format(time.RFC3339),
				"endTime":         failure.EndTime.Format(time.RFC3339),
				"durationMs":      failure.DurationMs,
				"failureReason":   failure.FailureReason,
			}

			fabricURL := utils.GenerateFabricURL(failure.WorkspaceID, failure.ItemID, failure.ItemType, failure.ID, failure.LivyID)
			if fabricURL != "" {
				failureMap["fabricUrl"] = fabricURL
			}

			failuresWithURLs = append(failuresWithURLs, failureMap)
		}
		result["recentFailures"] = failuresWithURLs
	}

	// Get long-running jobs (50% or more above average, last 10)
	longRunningJobs, err := a.db.GetLongRunningJobs(days, 50.0, 10)
	if err != nil {
		logger.Log("Failed to get long-running jobs: %v\n", err)
		result["longRunningJobsError"] = err.Error()
	} else {
		// Add Fabric URLs to long-running jobs
		jobsWithURLs := make([]map[string]interface{}, 0, len(longRunningJobs))
		for _, job := range longRunningJobs {
			jobMap := map[string]interface{}{
				"id":              job.ID,
				"workspaceId":     job.WorkspaceID,
				"workspaceName":   job.WorkspaceName,
				"itemId":          job.ItemID,
				"itemDisplayName": job.ItemDisplayName,
				"itemType":        job.ItemType,
				"jobType":         job.JobType,
				"startTime":       job.StartTime.Format(time.RFC3339),
				"durationMs":      job.DurationMs,
				"avgDurationMs":   job.AvgDurationMs,
				"deviationPct":    job.DeviationPct,
			}

			fabricURL := utils.GenerateFabricURL(job.WorkspaceID, job.ItemID, job.ItemType, job.ID, job.LivyID)
			if fabricURL != "" {
				jobMap["fabricUrl"] = fabricURL
			}

			jobsWithURLs = append(jobsWithURLs, jobMap)
		}
		result["longRunningJobs"] = jobsWithURLs
	}

	// Get overall stats - calculated entirely in DuckDB for consistency
	overallStats, err := a.db.GetOverallStats(days)
	if err != nil {
		logger.Log("Failed to get overall stats: %v\n", err)
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
		logger.Log("Failed to get daily stats: %v\n", err)
		result["dailyStatsError"] = err.Error()
	} else {
		result["dailyStats"] = dailyStats
	}

	// Get workspace stats
	workspaceStats, err := a.db.GetWorkspaceStatsFiltered(days, workspaceIDs, itemTypes, itemNameSearch)
	if err != nil {
		logger.Log("Failed to get workspace stats: %v\n", err)
		result["workspaceStatsError"] = err.Error()
	} else {
		result["workspaceStats"] = workspaceStats
	}

	// Get item type stats
	itemTypeStats, err := a.db.GetItemTypeStatsFiltered(days, workspaceIDs, itemTypes, itemNameSearch)
	if err != nil {
		logger.Log("Failed to get item type stats: %v\n", err)
		result["itemTypeStatsError"] = err.Error()
	} else {
		result["itemTypeStats"] = itemTypeStats
	}

	// Get recent failures (last 10 within the time period)
	recentFailures, err := a.db.GetRecentFailuresFiltered(10, days, workspaceIDs, itemTypes, itemNameSearch)
	if err != nil {
		logger.Log("Failed to get recent failures: %v\n", err)
		result["recentFailuresError"] = err.Error()
	} else {
		// Add Fabric URLs to failures
		failuresWithURLs := make([]map[string]interface{}, 0, len(recentFailures))
		for _, failure := range recentFailures {
			failureMap := map[string]interface{}{
				"id":              failure.ID,
				"workspaceId":     failure.WorkspaceID,
				"workspaceName":   failure.WorkspaceName,
				"itemId":          failure.ItemID,
				"itemDisplayName": failure.ItemDisplayName,
				"itemType":        failure.ItemType,
				"jobType":         failure.JobType,
				"startTime":       failure.StartTime.Format(time.RFC3339),
				"endTime":         failure.EndTime.Format(time.RFC3339),
				"durationMs":      failure.DurationMs,
				"failureReason":   failure.FailureReason,
			}

			fabricURL := utils.GenerateFabricURL(failure.WorkspaceID, failure.ItemID, failure.ItemType, failure.ID, failure.LivyID)
			if fabricURL != "" {
				failureMap["fabricUrl"] = fabricURL
			}

			failuresWithURLs = append(failuresWithURLs, failureMap)
		}
		result["recentFailures"] = failuresWithURLs
	}

	// Get long-running jobs (50% or more above average, last 10)
	longRunningJobs, err := a.db.GetLongRunningJobsFiltered(days, 50.0, 10, workspaceIDs, itemTypes, itemNameSearch)
	if err != nil {
		logger.Log("Failed to get long-running jobs: %v\n", err)
		result["longRunningJobsError"] = err.Error()
	} else {
		// Add Fabric URLs to long-running jobs
		jobsWithURLs := make([]map[string]interface{}, 0, len(longRunningJobs))
		for _, job := range longRunningJobs {
			jobMap := map[string]interface{}{
				"id":              job.ID,
				"workspaceId":     job.WorkspaceID,
				"workspaceName":   job.WorkspaceName,
				"itemId":          job.ItemID,
				"itemDisplayName": job.ItemDisplayName,
				"itemType":        job.ItemType,
				"jobType":         job.JobType,
				"startTime":       job.StartTime.Format(time.RFC3339),
				"durationMs":      job.DurationMs,
				"avgDurationMs":   job.AvgDurationMs,
				"deviationPct":    job.DeviationPct,
			}

			fabricURL := utils.GenerateFabricURL(job.WorkspaceID, job.ItemID, job.ItemType, job.ID, job.LivyID)
			if fabricURL != "" {
				jobMap["fabricUrl"] = fabricURL
			}

			jobsWithURLs = append(jobsWithURLs, jobMap)
		}
		result["longRunningJobs"] = jobsWithURLs
	}

	// Get overall stats - calculated entirely in DuckDB for consistency
	overallStats, err := a.db.GetOverallStatsFiltered(days, workspaceIDs, itemTypes, itemNameSearch)
	if err != nil {
		logger.Log("Failed to get overall stats: %v\n", err)
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
		logger.Log("Failed to get available item types: %v\n", err)
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

// GetItemStatsByDate returns item-level statistics for a specific date with optional filters
func (a *App) GetItemStatsByDate(date string, workspaceIDs []string, itemTypes []string, itemNameSearch string) map[string]interface{} {
	if a.db == nil {
		return map[string]interface{}{
			"error": "Database not initialized",
		}
	}

	if date == "" {
		return map[string]interface{}{
			"error": "Date is required",
		}
	}

	itemStats, err := a.db.GetItemStatsByDate(date, workspaceIDs, itemTypes, itemNameSearch)
	if err != nil {
		return map[string]interface{}{
			"error": err.Error(),
		}
	}

	return map[string]interface{}{
		"items": itemStats,
		"date":  date,
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
		logger.Log("Failed to query pipeline jobs for activity runs: %v\n", err)
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
			logger.Log("Failed to scan pipeline job: %v\n", err)
			continue
		}
		jobs = append(jobs, job)
	}

	if len(jobs) == 0 {
		return
	}

	logger.Log("Fetching activity runs for %d pipeline jobs in parallel...\n", len(jobs))
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
			logger.Log("Failed to fetch activity runs for job %s: %v\n", result.jobID, result.err)
			errorCount++
			// Do NOT mark as processed - leave activity_runs as NULL so it can be retried
			// This allows the job to be re-enriched on the next sync
			continue
		}

		// Save activity runs (even if empty array - this is a valid result)
		if err := a.db.UpdateJobInstanceActivityRuns(result.jobID, result.activityRuns); err != nil {
			logger.Log("Failed to save activity runs for job %s: %v\n", result.jobID, err)
			errorCount++
			continue
		}

		successCount++
		totalActivities += result.activityCount
	}

	elapsed := time.Since(startTime)
	logger.Log("Activity runs sync completed in %v\n", elapsed)
	logger.Log("Successfully fetched activity runs for %d/%d pipeline jobs (%d activities, %d errors)\n",
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

	// Convert to map format and add Fabric URLs
	childrenMaps := make([]map[string]interface{}, 0, len(children))
	for _, child := range children {
		childMap := map[string]interface{}{
			"activityRunId": child.ActivityRunID,
			"activityName":  child.ActivityName,
			"activityType":  child.ActivityType,
			"status":        child.Status,
			"pipelineId":    child.PipelineID,
			"hasChildren":   child.HasChildren,
		}

		// Add optional time fields
		if child.StartTime != nil {
			childMap["activityRunStart"] = child.StartTime.Format(time.RFC3339)
		}
		if child.EndTime != nil {
			childMap["activityRunEnd"] = child.EndTime.Format(time.RFC3339)
		}
		if child.DurationMs != nil {
			childMap["durationMs"] = *child.DurationMs
		}
		if child.ErrorMessage != nil {
			childMap["error"] = *child.ErrorMessage
		}

		// Add child execution details
		if child.ChildJobInstanceID != nil {
			childMap["childJobInstanceId"] = *child.ChildJobInstanceID
		}
		if child.ChildPipelineName != nil {
			childMap["childPipelineName"] = *child.ChildPipelineName
		}
		if child.ChildItemDisplayName != nil {
			childMap["childNotebookName"] = *child.ChildItemDisplayName // For notebooks
		}
		if child.ChildWorkspaceID != nil {
			childMap["childWorkspaceId"] = *child.ChildWorkspaceID
		}
		if child.ChildItemID != nil {
			childMap["childItemId"] = *child.ChildItemID
		}
		if child.ChildItemType != nil {
			childMap["childItemType"] = *child.ChildItemType
		}

		// Generate Fabric deep link URL for child execution if we have the required info
		if child.ChildJobInstanceID != nil && child.ChildWorkspaceID != nil {
			var itemID string
			var itemType string

			if child.ChildItemID != nil {
				itemID = *child.ChildItemID
			}
			if child.ChildItemType != nil {
				itemType = *child.ChildItemType
			}

			fabricURL := utils.GenerateFabricURL(*child.ChildWorkspaceID, itemID, itemType, *child.ChildJobInstanceID, child.LivyID)
			if fabricURL != "" {
				childMap["fabricUrl"] = fabricURL
			}
		}

		childrenMaps = append(childrenMaps, childMap)
	}

	return map[string]interface{}{
		"children": childrenMaps,
		"count":    len(childrenMaps),
	}
}

// SyncNotebookSessions fetches and stores Livy session information for all notebooks
// This allows generating correct notebook deep links using livyID
func (a *App) SyncNotebookSessions() error {
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}
	if a.fabricClient == nil {
		return fmt.Errorf("fabric client not initialized")
	}

	logger.Log("Starting notebook sessions sync...\n")

	// Get all unique notebooks from job_instances
	notebooks, err := a.db.GetUniqueNotebooks()
	if err != nil {
		return fmt.Errorf("failed to get unique notebooks: %w", err)
	}

	logger.Log("Found %d unique notebooks to sync\n", len(notebooks))

	// Use worker pool to parallelize notebook session fetching
	numWorkers := 4 // Process 4 notebooks concurrently
	notebookChan := make(chan struct {
		WorkspaceID string
		NotebookID  string
	}, len(notebooks))
	resultsChan := make(chan int, len(notebooks))
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for notebook := range notebookChan {
				sessionsCount := a.syncNotebookSessions(notebook.WorkspaceID, notebook.NotebookID)
				resultsChan <- sessionsCount
			}
		}()
	}

	// Send notebooks to workers
	for _, notebook := range notebooks {
		notebookChan <- struct {
			WorkspaceID string
			NotebookID  string
		}{
			WorkspaceID: notebook.WorkspaceID,
			NotebookID:  notebook.NotebookID,
		}
	}
	close(notebookChan)

	// Wait for all workers to complete
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results
	totalSessions := 0
	for count := range resultsChan {
		totalSessions += count
	}

	logger.Log("Notebook sessions sync complete: %d total sessions synced\n", totalSessions)
	return nil
}

// syncNotebookSessions fetches and saves Livy sessions for a single notebook
func (a *App) syncNotebookSessions(workspaceID, notebookID string) int {
	continuationToken := ""
	totalSessions := 0

	// Paginate through all Livy sessions for this notebook
	for {
		response, err := a.fabricClient.GetLivySessions(a.ctx, workspaceID, notebookID, continuationToken)
		if err != nil {
			logger.Log("Warning: failed to get Livy sessions for notebook %s: %v\n", notebookID, err)
			break // Skip this notebook
		}

		if response == nil || len(response.Value) == 0 {
			break
		}

		// Convert fabric.LivySession to db.NotebookSession
		dbSessions := make([]db.NotebookSession, 0, len(response.Value))
		for _, livySession := range response.Value {
			dbSession := db.NotebookSession{
				LivyID:        livySession.LivyID,
				JobInstanceID: livySession.JobInstanceID,
				WorkspaceID:   workspaceID,
				NotebookID:    notebookID,
				State:         livySession.State, // Required non-pointer field
			}

			// Handle optional string fields
			if livySession.SparkApplicationID != "" {
				dbSession.SparkApplicationID = &livySession.SparkApplicationID
			}
			if livySession.Origin != "" {
				dbSession.Origin = &livySession.Origin
			}
			if livySession.AttemptNumber != 0 {
				dbSession.AttemptNumber = &livySession.AttemptNumber
			}
			if livySession.LivyName != "" {
				dbSession.LivyName = &livySession.LivyName
			}
			if livySession.CancellationReason != "" {
				dbSession.CancellationReason = &livySession.CancellationReason
			}
			if livySession.CapacityID != "" {
				dbSession.CapacityID = &livySession.CapacityID
			}
			if livySession.OperationName != "" {
				dbSession.OperationName = &livySession.OperationName
			}
			if livySession.RuntimeVersion != "" {
				dbSession.RuntimeVersion = &livySession.RuntimeVersion
			}
			dbSession.IsHighConcurrency = &livySession.IsHighConcurrency

			// Handle FabricTime fields
			if !livySession.SubmittedDateTime.Time.IsZero() {
				dbSession.SubmittedDateTime = &livySession.SubmittedDateTime.Time
			}
			if !livySession.StartDateTime.Time.IsZero() {
				dbSession.StartDateTime = &livySession.StartDateTime.Time
			}
			if !livySession.EndDateTime.Time.IsZero() {
				dbSession.EndDateTime = &livySession.EndDateTime.Time
			}

			// Extract submitter info
			if livySession.Submitter.ID != "" {
				dbSession.SubmitterID = &livySession.Submitter.ID
			}
			if livySession.Submitter.Type != "" {
				dbSession.SubmitterType = &livySession.Submitter.Type
			}

			// Extract item info from top-level fields (not nested Item struct)
			if livySession.ItemName != "" {
				dbSession.ItemName = &livySession.ItemName
			}
			if livySession.ItemType != "" {
				dbSession.ItemType = &livySession.ItemType
			}
			if livySession.JobType != "" {
				dbSession.JobType = &livySession.JobType
			}

			// Convert durations to milliseconds
			if livySession.QueuedDuration.Value > 0 {
				ms := convertToMs(livySession.QueuedDuration.Value, livySession.QueuedDuration.TimeUnit)
				dbSession.QueuedDurationMs = &ms
			}
			if livySession.RunningDuration.Value > 0 {
				ms := convertToMs(livySession.RunningDuration.Value, livySession.RunningDuration.TimeUnit)
				dbSession.RunningDurationMs = &ms
			}
			if livySession.TotalDuration.Value > 0 {
				ms := convertToMs(livySession.TotalDuration.Value, livySession.TotalDuration.TimeUnit)
				dbSession.TotalDurationMs = &ms
			}

			// Extract consumer identity ID
			if livySession.ConsumerIdentity.ID != "" {
				dbSession.ConsumerIdentityID = &livySession.ConsumerIdentity.ID
			}

			dbSessions = append(dbSessions, dbSession)
		}

		// Save sessions to database
		if len(dbSessions) > 0 {
			if err := a.db.SaveLivySessions(dbSessions); err != nil {
				logger.Log("Warning: failed to save Livy sessions for notebook %s: %v\n", notebookID, err)
				break
			}
			totalSessions += len(dbSessions)
		}

		// Check if there are more pages
		if response.ContinuationToken == "" {
			break
		}
		continuationToken = response.ContinuationToken
	}

	if totalSessions > 0 {
		logger.Log("Synced %d sessions for notebook %s\n", totalSessions, notebookID)
	}

	return totalSessions
}

// convertToMs converts duration from Fabric API to milliseconds
func convertToMs(value int, timeUnit string) int {
	switch timeUnit {
	case "Seconds":
		return value * 1000
	case "Minutes":
		return value * 60000
	case "Hours":
		return value * 3600000
	case "Milliseconds":
		return value
	default:
		return value // Assume milliseconds if unknown
	}
}

// GetLogs returns all log entries
func (a *App) GetLogs() []logger.LogEntry {
	return logger.GetAll()
}

// ClearLogs clears all log entries
func (a *App) ClearLogs() {
	logger.Clear()
	logger.Log("Logs cleared\n")
}

// GetAppVersion returns the application version from config
func (a *App) GetAppVersion() string {
	if a.config != nil && a.config.App.Version != "" {
		return a.config.App.Version
	}
	return "0.2.3" // Fallback version
}

// IsReadOnlyReplicaEnabled returns whether the read-only replica feature is enabled
func (a *App) IsReadOnlyReplicaEnabled() bool {
	if a.config == nil {
		return false
	}
	return a.config.Database.EnableReadOnlyReplica
}

// GetReadOnlyDatabasePath returns the absolute path to the read-only replica database wrapped in quotes
func (a *App) GetReadOnlyDatabasePath() string {
	if a.config == nil || !a.config.Database.EnableReadOnlyReplica {
		return ""
	}

	// Get absolute path
	absPath, err := filepath.Abs(a.config.Database.ReadOnlyPath)
	if err != nil {
		logger.Log("Warning: failed to get absolute path for read-only database: %v\n", err)
		return fmt.Sprintf(`"%s"`, a.config.Database.ReadOnlyPath)
	}

	return fmt.Sprintf(`"%s"`, absPath)
}
