package fabric

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

// FabricTime is a custom time type that can parse Microsoft Fabric's timestamp format
type FabricTime struct {
	time.Time
}

// UnmarshalJSON handles the custom timestamp format from Microsoft Fabric API
func (ft *FabricTime) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), "\"")
	if s == "null" || s == "" {
		ft.Time = time.Time{}
		return nil
	}

	// Microsoft Fabric returns timestamps like "2025-10-20T18:48:52.0917149" without timezone
	// We need to handle this format and assume UTC
	layouts := []string{
		time.RFC3339,                  // Standard format with timezone
		time.RFC3339Nano,              // Standard format with nanoseconds
		"2006-01-02T15:04:05.9999999", // Microsoft format without timezone
		"2006-01-02T15:04:05",         // Without fractional seconds
	}

	var err error
	for _, layout := range layouts {
		ft.Time, err = time.Parse(layout, s)
		if err == nil {
			// If no timezone was specified, assume UTC
			if ft.Time.Location() == time.UTC && !strings.Contains(s, "Z") && !strings.Contains(s, "+") && !strings.Contains(s, "-") {
				// Timestamp was parsed but has no timezone info, already in UTC
			}
			return nil
		}
	}

	return fmt.Errorf("unable to parse time %q: %w", s, err)
}

// MarshalJSON converts the time back to JSON
func (ft FabricTime) MarshalJSON() ([]byte, error) {
	if ft.Time.IsZero() {
		return []byte("null"), nil
	}
	return []byte(fmt.Sprintf("\"%s\"", ft.Time.Format(time.RFC3339))), nil
}

// Client handles Microsoft Fabric API requests
type Client struct {
	httpClient  *http.Client
	baseURL     string
	accessToken string
}

// NewClient creates a new Fabric API client
func NewClient(accessToken string) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL:     "https://api.fabric.microsoft.com/v1",
		accessToken: accessToken,
	}
}

// Workspace represents a Fabric workspace
type Workspace struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
	CapacityID  string `json:"capacityId,omitempty"`
}

// WorkspacesResponse represents the API response for workspaces
type WorkspacesResponse struct {
	Value             []Workspace `json:"value"`
	ContinuationURI   string      `json:"continuationUri,omitempty"`
	ContinuationToken string      `json:"continuationToken,omitempty"`
}

// Item represents a Fabric item (pipeline, notebook, etc.)
type Item struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
	WorkspaceID string `json:"workspaceId"`
}

// ItemsResponse represents the API response for items
type ItemsResponse struct {
	Value             []Item `json:"value"`
	ContinuationURI   string `json:"continuationUri,omitempty"`
	ContinuationToken string `json:"continuationToken,omitempty"`
}

// JobInstance represents a job run instance
type JobInstance struct {
	ID             string          `json:"id"`
	ItemID         string          `json:"itemId"`
	JobType        string          `json:"jobType"`
	InvokeType     string          `json:"invokeType"`
	Status         string          `json:"status"`
	StartTimeUtc   FabricTime      `json:"startTimeUtc"`
	EndTimeUtc     FabricTime      `json:"endTimeUtc,omitempty"`
	FailureReason  json.RawMessage `json:"failureReason,omitempty"` // Can be string or object
	RootActivityID string          `json:"rootActivityId,omitempty"`
}

// GetFailureReasonString extracts failure reason as a string
func (ji *JobInstance) GetFailureReasonString() string {
	if len(ji.FailureReason) == 0 {
		return ""
	}

	// Try to unmarshal as a string first
	var str string
	if err := json.Unmarshal(ji.FailureReason, &str); err == nil {
		return str
	}

	// If it's an object, try to extract a message field
	var obj map[string]interface{}
	if err := json.Unmarshal(ji.FailureReason, &obj); err == nil {
		if msg, ok := obj["message"].(string); ok {
			return msg
		}
		if msg, ok := obj["errorMessage"].(string); ok {
			return msg
		}
		// Return the whole object as JSON string
		return string(ji.FailureReason)
	}

	return string(ji.FailureReason)
}

// JobInstancesResponse represents the API response for job instances
type JobInstancesResponse struct {
	Value             []JobInstance `json:"value"`
	ContinuationURI   string        `json:"continuationUri,omitempty"`
	ContinuationToken string        `json:"continuationToken,omitempty"`
}

// GetWorkspaces retrieves all workspaces the user has access to
func (c *Client) GetWorkspaces(ctx context.Context) ([]Workspace, error) {
	url := fmt.Sprintf("%s/workspaces", c.baseURL)

	var allWorkspaces []Workspace

	for url != "" {
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+c.accessToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to execute request: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
		}

		var response WorkspacesResponse
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}

		allWorkspaces = append(allWorkspaces, response.Value...)

		// Handle pagination
		url = response.ContinuationURI
	}

	return allWorkspaces, nil
}

// GetWorkspaceItems retrieves all items in a workspace
func (c *Client) GetWorkspaceItems(ctx context.Context, workspaceID string) ([]Item, error) {
	url := fmt.Sprintf("%s/workspaces/%s/items", c.baseURL, workspaceID)

	var allItems []Item

	for url != "" {
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+c.accessToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to execute request: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
		}

		var response ItemsResponse
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}

		// Populate WorkspaceID for each item
		for i := range response.Value {
			response.Value[i].WorkspaceID = workspaceID
		}

		allItems = append(allItems, response.Value...)

		// Handle pagination
		url = response.ContinuationURI
	}

	return allItems, nil
}

// GetItemJobInstances retrieves job instances for a specific item
func (c *Client) GetItemJobInstances(ctx context.Context, workspaceID, itemID string) ([]JobInstance, error) {
	url := fmt.Sprintf("%s/workspaces/%s/items/%s/jobs/instances", c.baseURL, workspaceID, itemID)

	var allInstances []JobInstance

	for url != "" {
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+c.accessToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to execute request: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
		}

		var response JobInstancesResponse
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}

		allInstances = append(allInstances, response.Value...)

		// Handle pagination
		url = response.ContinuationURI
	}

	return allInstances, nil
}

// GetRecentJobs retrieves recent job instances across all workspaces in Fabric
// If startTimeFrom is provided, only fetches jobs with start_time > startTimeFrom
// Always fetches jobs with end_time IS NULL (in progress) regardless of start time
// cachedItems can be provided to avoid fetching items from API (optimization for incremental syncs)
func (c *Client) GetRecentJobs(ctx context.Context, workspaces []Workspace, limit int, startTimeFrom *time.Time, cachedItems map[string][]Item) ([]map[string]interface{}, []Item, error) {
	var allJobs []map[string]interface{}
	itemCache := make(map[string]Item) // Cache item details
	var allItems []Item                // Track all items for persistence

	// Item types that support job instances based on Microsoft Fabric documentation
	// https://learn.microsoft.com/en-us/rest/api/fabric/core/job-scheduler/list-item-job-instances
	supportedTypes := map[string]bool{
		"DataPipeline":       true,
		"Notebook":           true,
		"SparkJobDefinition": true,
		"Dataflow":           true,
		"ApacheAirflowJob":   true,
		"SemanticModel":      true,
	}

	if startTimeFrom != nil {
		fmt.Printf("Fetching jobs from %d workspaces (incremental sync from %s)...\n", len(workspaces), startTimeFrom.Format(time.RFC3339))
	} else {
		fmt.Printf("Fetching jobs from %d workspaces (full sync)...\n", len(workspaces))
	}

	apiCallCount := 0
	for _, workspace := range workspaces {
		fmt.Printf("Checking workspace: %s (%s)\n", workspace.DisplayName, workspace.ID)

		// Always get all items in the workspace from API to discover new items
		// This ensures we don't miss any newly created items since the last sync
		items, err := c.GetWorkspaceItems(ctx, workspace.ID)
		apiCallCount++
		if err != nil {
			fmt.Printf("Warning: failed to get items for workspace %s: %v\n", workspace.ID, err)
			continue
		}
		allItems = append(allItems, items...) // Track for persistence

		// Add delay after API call to avoid rate limiting
		time.Sleep(200 * time.Millisecond) // Filter to only items that support job instances
		var supportedItems []Item
		for _, item := range items {
			if supportedTypes[item.Type] {
				supportedItems = append(supportedItems, item)
			}
		}

		fmt.Printf("Found %d total items, %d with job support in workspace %s\n",
			len(items), len(supportedItems), workspace.DisplayName)

		// Cache items
		for _, item := range supportedItems {
			itemCache[item.ID] = item
		}

		// Get job instances for each supported item
		for i, item := range supportedItems {
			fmt.Printf("[%d/%d] Getting job instances for %s (%s)...\n", i+1, len(supportedItems), item.DisplayName, item.Type)

			instances, err := c.GetItemJobInstances(ctx, workspace.ID, item.ID)
			apiCallCount++
			if err != nil {
				fmt.Printf("Warning: failed to get job instances for %s: %v\n", item.DisplayName, err)
				// Add delay even on error to respect rate limits
				time.Sleep(300 * time.Millisecond)
				continue
			}

			// Filter jobs based on incremental sync criteria
			var filteredInstances []JobInstance
			for _, instance := range instances {
				// Always include jobs with no end time (in progress)
				if instance.EndTimeUtc.Time.IsZero() {
					filteredInstances = append(filteredInstances, instance)
					continue
				}

				// If doing incremental sync, only include jobs newer than last sync
				if startTimeFrom != nil {
					if instance.StartTimeUtc.Time.After(*startTimeFrom) {
						filteredInstances = append(filteredInstances, instance)
					}
				} else {
					// Full sync - include all jobs
					filteredInstances = append(filteredInstances, instance)
				}
			}

			if len(filteredInstances) > 0 {
				fmt.Printf("Found %d job instances for %s", len(filteredInstances), item.DisplayName)
				if startTimeFrom != nil {
					fmt.Printf(" (%d total, %d new/in-progress)\n", len(instances), len(filteredInstances))
				} else {
					fmt.Println()
				}
			}

			// Add delay to avoid rate limiting (200-300ms between requests)
			time.Sleep(200 * time.Millisecond)

			// Convert to map format for frontend
			for _, instance := range filteredInstances {
				job := map[string]interface{}{
					"id":              instance.ID,
					"workspaceId":     workspace.ID,
					"workspaceName":   workspace.DisplayName,
					"itemId":          item.ID,
					"itemDisplayName": item.DisplayName,
					"itemType":        item.Type,
					"jobType":         instance.JobType,
					"status":          instance.Status,
					"startTime":       instance.StartTimeUtc.Time.Format(time.RFC3339),
				}

				if !instance.EndTimeUtc.Time.IsZero() {
					job["endTime"] = instance.EndTimeUtc.Time.Format(time.RFC3339)
					duration := instance.EndTimeUtc.Time.Sub(instance.StartTimeUtc.Time)
					job["durationMs"] = int64(duration / time.Millisecond)
				}

				failureReason := instance.GetFailureReasonString()
				if failureReason != "" {
					job["failureReason"] = failureReason
				}

				allJobs = append(allJobs, job)
			}
		}
	}

	fmt.Printf("Total jobs found: %d (made %d API calls)\n", len(allJobs), apiCallCount)

	// Sort by start time (most recent first)
	sort.Slice(allJobs, func(i, j int) bool {
		timeI, _ := time.Parse(time.RFC3339, allJobs[i]["startTime"].(string))
		timeJ, _ := time.Parse(time.RFC3339, allJobs[j]["startTime"].(string))
		return timeI.After(timeJ) // Most recent first
	})

	// Limit results (0 means no limit)
	if limit > 0 && len(allJobs) > limit {
		allJobs = allJobs[:limit]
	}

	return allJobs, allItems, nil
}
