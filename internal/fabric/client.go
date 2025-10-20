package fabric

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

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
	ID             string    `json:"id"`
	ItemID         string    `json:"itemId"`
	JobType        string    `json:"jobType"`
	InvokeType     string    `json:"invokeType"`
	Status         string    `json:"status"`
	StartTimeUtc   time.Time `json:"startTimeUtc"`
	EndTimeUtc     time.Time `json:"endTimeUtc,omitempty"`
	FailureReason  string    `json:"failureReason,omitempty"`
	RootActivityID string    `json:"rootActivityId,omitempty"`
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

// GetRecentJobs retrieves recent job instances across all workspaces
func (c *Client) GetRecentJobs(ctx context.Context, workspaces []Workspace, limit int) ([]map[string]interface{}, error) {
	var allJobs []map[string]interface{}
	itemCache := make(map[string]Item) // Cache item details

	fmt.Printf("Fetching jobs from %d workspaces...\n", len(workspaces))

	for _, workspace := range workspaces {
		fmt.Printf("Checking workspace: %s (%s)\n", workspace.DisplayName, workspace.ID)

		// Get all items in the workspace
		items, err := c.GetWorkspaceItems(ctx, workspace.ID)
		if err != nil {
			fmt.Printf("Warning: failed to get items for workspace %s: %v\n", workspace.ID, err)
			continue
		}

		fmt.Printf("Found %d items in workspace %s\n", len(items), workspace.DisplayName)

		// Cache items
		for _, item := range items {
			itemCache[item.ID] = item
			fmt.Printf("  - %s (%s)\n", item.DisplayName, item.Type)
		}

		// Get job instances for each item
		for _, item := range items {
			// Check for items that might have jobs
			// Common types: DataPipeline, Notebook, Dataflow
			if item.Type != "DataPipeline" && item.Type != "Notebook" && item.Type != "Dataflow" {
				continue
			}

			fmt.Printf("Getting job instances for %s (%s)...\n", item.DisplayName, item.Type)

			instances, err := c.GetItemJobInstances(ctx, workspace.ID, item.ID)
			if err != nil {
				fmt.Printf("Warning: failed to get job instances for item %s: %v\n", item.ID, err)
				continue
			}

			fmt.Printf("Found %d job instances for %s\n", len(instances), item.DisplayName)

			// Convert to map format for frontend
			for _, instance := range instances {
				job := map[string]interface{}{
					"id":              instance.ID,
					"workspaceId":     workspace.ID,
					"workspaceName":   workspace.DisplayName,
					"itemId":          item.ID,
					"itemDisplayName": item.DisplayName,
					"jobType":         instance.JobType,
					"status":          instance.Status,
					"startTime":       instance.StartTimeUtc.Format(time.RFC3339),
				}

				if !instance.EndTimeUtc.IsZero() {
					job["endTime"] = instance.EndTimeUtc.Format(time.RFC3339)
					duration := instance.EndTimeUtc.Sub(instance.StartTimeUtc)
					job["durationMs"] = int64(duration / time.Millisecond)
				}

				if instance.FailureReason != "" {
					job["failureReason"] = instance.FailureReason
				}

				allJobs = append(allJobs, job)
			}
		}
	}

	fmt.Printf("Total jobs found: %d\n", len(allJobs))

	// Sort by start time (most recent first) and limit
	// Note: This is a simple implementation. For better performance with large datasets,
	// consider implementing pagination at the API level
	if len(allJobs) > limit {
		allJobs = allJobs[:limit]
	}

	return allJobs, nil
}
