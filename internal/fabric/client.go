package fabric

import (
	"bytes"
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
	rateLimiter *AdaptiveRateLimiter
	retryPolicy *RetryPolicy
}

// NewClient creates a new Fabric API client
func NewClient(accessToken string) *Client {
	// Configure HTTP transport with proper connection management
	transport := &http.Transport{
		MaxIdleConns:        100,              // Maximum idle connections across all hosts
		MaxIdleConnsPerHost: 10,               // Maximum idle connections per host
		IdleConnTimeout:     90 * time.Second, // How long idle connections stay open
		DisableKeepAlives:   false,            // Keep connections alive for reuse
		ForceAttemptHTTP2:   true,             // Prefer HTTP/2 when available
	}

	return &Client{
		httpClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		},
		baseURL:     "https://api.fabric.microsoft.com/v1",
		accessToken: accessToken,
		rateLimiter: NewAdaptiveRateLimiter(),
		retryPolicy: NewRetryPolicy(),
	}
}

// doRequestWithRetry performs an HTTP request with rate limiting and retry logic
func (c *Client) doRequestWithRetry(ctx context.Context, req *http.Request) (*http.Response, error) {
	// Wait for rate limiter token
	c.rateLimiter.Wait()

	// Execute with retry logic
	return c.retryPolicy.ExecuteWithRetry(
		func() (*http.Response, error) {
			return c.httpClient.Do(req)
		},
		func() {
			// On throttle detected
			c.rateLimiter.OnThrottle()
		},
	)
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

		resp, err := c.doRequestWithRetry(ctx, req)
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

		resp, err := c.doRequestWithRetry(ctx, req)
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

		resp, err := c.doRequestWithRetry(ctx, req)
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

// QueryActivityRunsResponse represents the response from the QueryActivityRuns API
type QueryActivityRunsResponse struct {
	Value             []ActivityRun `json:"value"`
	ContinuationToken *string       `json:"continuationToken"`
}

// QueryActivityRuns retrieves all activity runs for a pipeline job instance with pagination support
func (c *Client) QueryActivityRuns(ctx context.Context, workspaceID, jobInstanceID string, startTime, endTime time.Time) ([]ActivityRun, error) {
	url := fmt.Sprintf("%s/workspaces/%s/datapipelines/pipelineruns/%s/queryactivityruns",
		c.baseURL, workspaceID, jobInstanceID)

	var allActivityRuns []ActivityRun
	var continuationToken *string
	pageCount := 0

	for {
		pageCount++

		requestBody := map[string]interface{}{
			"filters": []interface{}{},
			"orderBy": []map[string]string{
				{"orderBy": "ActivityRunStart", "order": "DESC"},
			},
			"lastUpdatedAfter":  startTime.Format(time.RFC3339),
			"lastUpdatedBefore": endTime.Format(time.RFC3339),
		}

		// Add continuation token if we have one
		if continuationToken != nil && *continuationToken != "" {
			requestBody["continuationToken"] = *continuationToken
		}

		bodyBytes, err := json.Marshal(requestBody)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+c.accessToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.doRequestWithRetry(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("failed to execute request: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
		}

		// Read the response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}

		// Parse response with proper structure
		var response QueryActivityRunsResponse
		err = json.Unmarshal(body, &response)
		if err != nil {
			return nil, fmt.Errorf("failed to decode response: %w, body: %s", err, string(body[:min(500, len(body))]))
		}

		// Append results from this page
		allActivityRuns = append(allActivityRuns, response.Value...)

		if len(response.Value) > 0 {
			fmt.Printf("Fetched %d activity runs for job %s (page %d, total so far: %d)\n",
				len(response.Value), jobInstanceID, pageCount, len(allActivityRuns))
		}

		// Check if we need to fetch more pages
		if response.ContinuationToken == nil || *response.ContinuationToken == "" {
			break
		}

		continuationToken = response.ContinuationToken
	}

	if len(allActivityRuns) > 0 {
		fmt.Printf("Total activity runs fetched for job %s: %d (across %d pages)\n", jobInstanceID, len(allActivityRuns), pageCount)
	}

	return allActivityRuns, nil
}

// ActivityRun represents a single activity execution within a pipeline
type ActivityRun struct {
	PipelineID              string                 `json:"pipelineId"`
	PipelineRunID           string                 `json:"pipelineRunId"`
	ActivityName            string                 `json:"activityName"`
	ActivityType            string                 `json:"activityType"`
	ActivityRunID           string                 `json:"activityRunId"`
	Status                  string                 `json:"status"`
	ActivityRunStart        string                 `json:"activityRunStart"`
	ActivityRunEnd          string                 `json:"activityRunEnd"`
	DurationInMs            int64                  `json:"durationInMs"`
	Input                   map[string]interface{} `json:"input"`
	Output                  map[string]interface{} `json:"output"`
	Error                   ActivityError          `json:"error"`
	RetryAttempt            *int                   `json:"retryAttempt"`
	IterationHash           string                 `json:"iterationHash"`
	UserProperties          map[string]interface{} `json:"userProperties"`
	RecoveryStatus          string                 `json:"recoveryStatus"`
	IntegrationRuntimeNames []string               `json:"integrationRuntimeNames"`
	ExecutionDetails        map[string]interface{} `json:"executionDetails"`
}

// ActivityError represents error information from an activity run
type ActivityError struct {
	ErrorCode   string          `json:"errorCode"`
	Message     string          `json:"message"`
	FailureType string          `json:"failureType"`
	Target      string          `json:"target"`
	Details     json.RawMessage `json:"details"` // Can be string or array depending on error type
}

// LivySession represents a Spark Livy session for a notebook run
type LivySession struct {
	LivyID             string `json:"livyId"`
	JobInstanceID      string `json:"jobInstanceId"`
	SparkApplicationID string `json:"sparkApplicationId"`
	State              string `json:"state"`
	Origin             string `json:"origin"`
	AttemptNumber      int    `json:"attemptNumber"`
	MaxNumberAttempts  int    `json:"maxNumberOfAttempts"`
	LivyName           string `json:"livyName"`
	Submitter          struct {
		ID   string `json:"id"`
		Type string `json:"type"`
	} `json:"submitter"`
	Item struct {
		WorkspaceID   string `json:"workspaceId"`
		ItemID        string `json:"itemId"`
		ReferenceType string `json:"referenceType"`
	} `json:"item"`
	ItemName          string     `json:"itemName"`
	ItemType          string     `json:"itemType"`
	JobType           string     `json:"jobType"`
	SubmittedDateTime FabricTime `json:"submittedDateTime"`
	StartDateTime     FabricTime `json:"startDateTime"`
	EndDateTime       FabricTime `json:"endDateTime"`
	QueuedDuration    struct {
		Value    int    `json:"value"`
		TimeUnit string `json:"timeUnit"`
	} `json:"queuedDuration"`
	RunningDuration struct {
		Value    int    `json:"value"`
		TimeUnit string `json:"timeUnit"`
	} `json:"runningDuration"`
	TotalDuration struct {
		Value    int    `json:"value"`
		TimeUnit string `json:"timeUnit"`
	} `json:"totalDuration"`
	CancellationReason string `json:"cancellationReason"`
	CapacityID         string `json:"capacityId"`
	OperationName      string `json:"operationName"`
	ConsumerIdentity   struct {
		ID   string `json:"id"`
		Type string `json:"type"`
	} `json:"consumerIdentity"`
	RuntimeVersion    string `json:"runtimeVersion"`
	IsHighConcurrency bool   `json:"isHighConcurrency"`
	CreatorItem       struct {
		WorkspaceID   string `json:"workspaceId"`
		ItemID        string `json:"itemId"`
		ReferenceType string `json:"referenceType"`
	} `json:"creatorItem"`
}

// LivySessionsResponse represents the API response for Livy sessions
type LivySessionsResponse struct {
	Value             []LivySession `json:"value"`
	ContinuationToken string        `json:"continuationToken"`
	ContinuationURI   string        `json:"continuationUri"`
}

// GetRecentJobs retrieves recent job instances across all workspaces in Fabric with parallel processing
// If startTimeFrom is provided, only fetches jobs with start_time > startTimeFrom
// Always fetches jobs with end_time IS NULL (in progress) regardless of start time
// cachedItems can be provided to avoid fetching items from API (optimization for incremental syncs)
func (c *Client) GetRecentJobs(ctx context.Context, workspaces []Workspace, limit int, startTimeFrom *time.Time, cachedItems map[string][]Item) ([]map[string]interface{}, []Item, error) {
	// Item types that support job instances
	supportedTypes := map[string]bool{
		"DataPipeline":       true,
		"Notebook":           true,
		"SparkJobDefinition": true,
		"Dataflow":           true,
		"ApacheAirflowJob":   true,
	}

	if startTimeFrom != nil {
		fmt.Printf("Fetching jobs from %d workspaces (incremental sync from %s)...\n", len(workspaces), startTimeFrom.Format(time.RFC3339))
		fmt.Printf("Rate limiter: %d RPS\n", c.rateLimiter.GetCurrentRPS())
	} else {
		fmt.Printf("Fetching jobs from %d workspaces (full sync)...\n", len(workspaces))
		fmt.Printf("Rate limiter: %d RPS\n", c.rateLimiter.GetCurrentRPS())
	}

	startTime := time.Now()

	// Create workspace worker pool
	workspacePool := NewWorkerPool(MaxWorkspaceConcurrency)

	// Channel to collect results
	workspaceResults := make(chan WorkspaceResult, len(workspaces))

	// Process each workspace in parallel
	for _, workspace := range workspaces {
		workspace := workspace // Capture for goroutine

		workspacePool.Submit(ctx, func() error {
			result := WorkspaceResult{
				WorkspaceID:   workspace.ID,
				WorkspaceName: workspace.DisplayName,
				Jobs:          []map[string]interface{}{},
				Items:         []Item{},
			}

			// Get items for this workspace
			items, err := c.GetWorkspaceItems(ctx, workspace.ID)
			if err != nil {
				result.Error = fmt.Errorf("failed to get items: %w", err)
				workspaceResults <- result
				return nil // Continue with other workspaces
			}

			result.Items = items

			// Filter to supported items
			var supportedItems []Item
			for _, item := range items {
				if supportedTypes[item.Type] {
					supportedItems = append(supportedItems, item)
				}
			}

			fmt.Printf("[%s] Found %d items, %d with job support\n",
				workspace.DisplayName, len(items), len(supportedItems))

			if len(supportedItems) == 0 {
				workspaceResults <- result
				return nil
			}

			// Create item worker pool for this workspace
			itemPool := NewWorkerPool(MaxItemConcurrency)
			itemResults := make(chan ItemResult, len(supportedItems))

			// Process each item in parallel
			for _, item := range supportedItems {
				item := item // Capture for goroutine

				itemPool.Submit(ctx, func() error {
					itemResult := ItemResult{
						WorkspaceID:   workspace.ID,
						WorkspaceName: workspace.DisplayName,
						Item:          item,
						Jobs:          []map[string]interface{}{},
					}

					instances, err := c.GetItemJobInstances(ctx, workspace.ID, item.ID)
					if err != nil {
						itemResult.Error = fmt.Errorf("failed to get job instances: %w", err)
						itemResults <- itemResult
						return nil
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

						if instance.RootActivityID != "" {
							job["rootActivityId"] = instance.RootActivityID
						}

						itemResult.Jobs = append(itemResult.Jobs, job)
					}

					itemResults <- itemResult
					return nil
				})
			}

			// Wait for all items to complete
			itemPool.Wait()
			close(itemResults)

			// Collect item results
			for itemResult := range itemResults {
				if itemResult.Error != nil {
					fmt.Printf("  [%s] Warning: %v\n", itemResult.Item.DisplayName, itemResult.Error)
					continue
				}
				result.Jobs = append(result.Jobs, itemResult.Jobs...)
			}

			workspaceResults <- result
			return nil
		})
	}

	// Wait for all workspaces to complete
	workspacePool.Wait()
	close(workspaceResults)

	// Collect all results
	var allJobs []map[string]interface{}
	var allItems []Item
	var errors []string

	for result := range workspaceResults {
		if result.Error != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", result.WorkspaceName, result.Error))
			continue
		}
		allJobs = append(allJobs, result.Jobs...)
		allItems = append(allItems, result.Items...)
	}

	elapsed := time.Since(startTime)
	fmt.Printf("\nCompleted in %v\n", elapsed)
	fmt.Printf("Total jobs found: %d across %d workspaces\n", len(allJobs), len(workspaces))
	fmt.Printf("Final rate limiter: %d RPS\n", c.rateLimiter.GetCurrentRPS())

	if len(errors) > 0 {
		fmt.Printf("Errors encountered: %d\n", len(errors))
		for _, err := range errors {
			fmt.Printf("  - %s\n", err)
		}
	}

	// Sort by start time (most recent first)
	sort.Slice(allJobs, func(i, j int) bool {
		timeI, _ := time.Parse(time.RFC3339, allJobs[i]["startTime"].(string))
		timeJ, _ := time.Parse(time.RFC3339, allJobs[j]["startTime"].(string))
		return timeI.After(timeJ)
	})

	// Limit results (0 means no limit)
	if limit > 0 && len(allJobs) > limit {
		allJobs = allJobs[:limit]
	}

	return allJobs, allItems, nil
}

// GetLivySessions retrieves Livy sessions for a specific notebook with pagination support
func (c *Client) GetLivySessions(ctx context.Context, workspaceID, notebookID string, continuationToken string) (*LivySessionsResponse, error) {
	url := fmt.Sprintf("%s/workspaces/%s/notebooks/%s/livySessions", c.baseURL, workspaceID, notebookID)
	if continuationToken != "" {
		url += "?continuationToken=" + continuationToken
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.doRequestWithRetry(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var response LivySessionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}
