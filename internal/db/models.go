package db

import (
	"encoding/json"
	"time"
)

// Workspace represents a Fabric workspace
type Workspace struct {
	ID          string    `json:"id"`
	DisplayName string    `json:"displayName"`
	Type        string    `json:"type"`
	Description *string   `json:"description,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// Item represents a Fabric item (pipeline, notebook, etc.)
type Item struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspaceId"`
	DisplayName string    `json:"displayName"`
	Type        string    `json:"type"`
	Description *string   `json:"description,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
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

// ActivityError represents an error in an activity run
type ActivityError struct {
	ErrorCode   string          `json:"errorCode"`
	Message     string          `json:"message"`
	FailureType string          `json:"failureType"`
	Target      string          `json:"target"`
	Details     json.RawMessage `json:"details"` // Can be string or array depending on error type
}

// JobInstance represents a job execution instance
type JobInstance struct {
	ID              string        `json:"id"`
	WorkspaceID     string        `json:"workspaceId"`
	ItemID          string        `json:"itemId"`
	JobType         string        `json:"jobType"`
	Status          string        `json:"status"`
	StartTime       time.Time     `json:"startTime"`
	EndTime         *time.Time    `json:"endTime,omitempty"`
	DurationMs      *int64        `json:"durationMs,omitempty"`
	FailureReason   *string       `json:"failureReason,omitempty"`
	InvokerType     *string       `json:"invokerType,omitempty"`
	RootActivityID  *string       `json:"rootActivityId,omitempty"` // Root activity id to trace requests across services
	ActivityRuns    []ActivityRun `json:"activityRuns,omitempty"`   // Activity runs data for pipelines
	ActivityCount   *int          `json:"activityCount,omitempty"`  // Count of activities
	LivyID          *string       `json:"livyId,omitempty"`         // Livy session ID for notebooks
	CreatedAt       time.Time     `json:"createdAt"`
	UpdatedAt       time.Time     `json:"updatedAt"`
	ItemDisplayName *string       `json:"itemDisplayName,omitempty"` // Joined from items table
	ItemType        *string       `json:"itemType,omitempty"`        // Joined from items table
	WorkspaceName   *string       `json:"workspaceName,omitempty"`   // Joined from workspaces table
}

// NotebookSession represents a Livy session for a notebook execution
type NotebookSession struct {
	LivyID             string     `json:"livyId"`
	JobInstanceID      string     `json:"jobInstanceId"`
	WorkspaceID        string     `json:"workspaceId"`
	NotebookID         string     `json:"notebookId"`
	SparkApplicationID *string    `json:"sparkApplicationId,omitempty"`
	State              string     `json:"state"`
	Origin             *string    `json:"origin,omitempty"`
	AttemptNumber      *int       `json:"attemptNumber,omitempty"`
	LivyName           *string    `json:"livyName,omitempty"`
	SubmitterID        *string    `json:"submitterId,omitempty"`
	SubmitterType      *string    `json:"submitterType,omitempty"`
	ItemName           *string    `json:"itemName,omitempty"`
	ItemType           *string    `json:"itemType,omitempty"`
	JobType            *string    `json:"jobType,omitempty"`
	SubmittedDateTime  *time.Time `json:"submittedDatetime,omitempty"`
	StartDateTime      *time.Time `json:"startDatetime,omitempty"`
	EndDateTime        *time.Time `json:"endDatetime,omitempty"`
	QueuedDurationMs   *int       `json:"queuedDurationMs,omitempty"`
	RunningDurationMs  *int       `json:"runningDurationMs,omitempty"`
	TotalDurationMs    *int       `json:"totalDurationMs,omitempty"`
	CancellationReason *string    `json:"cancellationReason,omitempty"`
	CapacityID         *string    `json:"capacityId,omitempty"`
	OperationName      *string    `json:"operationName,omitempty"`
	ConsumerIdentityID *string    `json:"consumerIdentityId,omitempty"`
	RuntimeVersion     *string    `json:"runtimeVersion,omitempty"`
	IsHighConcurrency  *bool      `json:"isHighConcurrency,omitempty"`
	CreatedAt          time.Time  `json:"createdAt"`
	UpdatedAt          time.Time  `json:"updatedAt"`
}

// ChildExecution represents a child pipeline or notebook execution
type ChildExecution struct {
	ActivityRunID        string     `json:"activityRunId"`
	ActivityName         string     `json:"activityName"`
	ActivityType         string     `json:"activityType"`
	Status               string     `json:"status"`
	StartTime            *time.Time `json:"activityRunStart"` // Match API field name
	EndTime              *time.Time `json:"activityRunEnd"`   // Match API field name
	DurationMs           *int64     `json:"durationMs"`
	ErrorMessage         *string    `json:"errorMessage"`
	PipelineID           string     `json:"pipelineId"`
	HasChildren          bool       `json:"hasChildren"` // For future recursive expansion
	ChildJobInstanceID   *string    `json:"childJobInstanceId,omitempty"`
	ChildPipelineName    *string    `json:"childPipelineName,omitempty"`
	ChildNotebookName    *string    `json:"childNotebookName,omitempty"` // Alias for display name
	ChildWorkspaceID     *string    `json:"childWorkspaceId,omitempty"`
	ChildItemID          *string    `json:"childItemId,omitempty"`
	ChildItemType        *string    `json:"childItemType,omitempty"`
	ChildItemDisplayName *string    `json:"childItemDisplayName,omitempty"`
	LivyID               *string    `json:"livyId,omitempty"`
}

// SyncMetadata tracks sync operations
type SyncMetadata struct {
	ID            int64     `json:"id"`
	LastSyncTime  time.Time `json:"lastSyncTime"`
	SyncType      string    `json:"syncType"`
	RecordsSynced int       `json:"recordsSynced"`
	Errors        int       `json:"errors"`
	CreatedAt     time.Time `json:"createdAt"`
}

// JobFilter represents filtering options for job queries
type JobFilter struct {
	WorkspaceID   *string    `json:"workspaceId,omitempty"`
	ItemID        *string    `json:"itemId,omitempty"`
	JobType       *string    `json:"jobType,omitempty"`
	Status        *string    `json:"status,omitempty"`
	StartDateFrom *time.Time `json:"startDateFrom,omitempty"`
	StartDateTo   *time.Time `json:"startDateTo,omitempty"`
	Limit         *int       `json:"limit,omitempty"`
	Offset        *int       `json:"offset,omitempty"`
}

// JobStats represents aggregated job statistics
type JobStats struct {
	TotalJobs     int     `json:"totalJobs"`
	Successful    int     `json:"successful"`
	Failed        int     `json:"failed"`
	Running       int     `json:"running"`
	SuccessRate   float64 `json:"successRate"`
	AvgDurationMs float64 `json:"avgDurationMs"`
}

// DailyStats represents job statistics aggregated by day
type DailyStats struct {
	Date          string  `json:"date"`
	TotalJobs     int     `json:"totalJobs"`
	Successful    int     `json:"successful"`
	Failed        int     `json:"failed"`
	Running       int     `json:"running"`
	SuccessRate   float64 `json:"successRate"`
	AvgDurationMs float64 `json:"avgDurationMs"`
}

// WorkspaceStats represents job statistics by workspace
type WorkspaceStats struct {
	WorkspaceID   string  `json:"workspaceId"`
	WorkspaceName string  `json:"workspaceName"`
	TotalJobs     int     `json:"totalJobs"`
	Successful    int     `json:"successful"`
	Failed        int     `json:"failed"`
	Running       int     `json:"running"`
	SuccessRate   float64 `json:"successRate"`
	AvgDurationMs float64 `json:"avgDurationMs"`
}

// ItemTypeStats represents job statistics by item type
type ItemTypeStats struct {
	ItemType      string  `json:"itemType"`
	TotalJobs     int     `json:"totalJobs"`
	Successful    int     `json:"successful"`
	Failed        int     `json:"failed"`
	Running       int     `json:"running"`
	SuccessRate   float64 `json:"successRate"`
	AvgDurationMs float64 `json:"avgDurationMs"`
}

// RecentFailures represents recent failed jobs
type RecentFailure struct {
	ID              string    `json:"id"`
	WorkspaceID     string    `json:"workspaceId"`
	WorkspaceName   string    `json:"workspaceName"`
	ItemID          string    `json:"itemId"`
	ItemDisplayName string    `json:"itemDisplayName"`
	ItemType        string    `json:"itemType"`
	JobType         string    `json:"jobType"`
	StartTime       time.Time `json:"startTime"`
	EndTime         time.Time `json:"endTime"`
	DurationMs      int64     `json:"durationMs"`
	FailureReason   string    `json:"failureReason"`
	LivyID          *string   `json:"livyId,omitempty"`
}

// LongRunningJob represents jobs with unusually long durations
type LongRunningJob struct {
	ID              string    `json:"id"`
	WorkspaceID     string    `json:"workspaceId"`
	WorkspaceName   string    `json:"workspaceName"`
	ItemID          string    `json:"itemId"`
	ItemDisplayName string    `json:"itemDisplayName"`
	ItemType        string    `json:"itemType"`
	JobType         string    `json:"jobType"`
	StartTime       time.Time `json:"startTime"`
	DurationMs      int64     `json:"durationMs"`
	AvgDurationMs   float64   `json:"avgDurationMs"`
	DeviationPct    float64   `json:"deviationPct"`
	LivyID          *string   `json:"livyId,omitempty"`
}

// ItemStats represents job statistics by individual item
type ItemStats struct {
	ItemID        string  `json:"itemId"`
	ItemName      string  `json:"itemName"`
	ItemType      string  `json:"itemType"`
	WorkspaceID   string  `json:"workspaceId"`
	WorkspaceName string  `json:"workspaceName"`
	TotalJobs     int     `json:"totalJobs"`
	Successful    int     `json:"successful"`
	Failed        int     `json:"failed"`
	Running       int     `json:"running"`
	SuccessRate   float64 `json:"successRate"`
	AvgDurationMs float64 `json:"avgDurationMs"`
}

// DailyItemStats represents job statistics for items on a specific date
type DailyItemStats struct {
	ItemID        string  `json:"itemId"`
	ItemName      string  `json:"itemName"`
	ItemType      string  `json:"itemType"`
	WorkspaceID   string  `json:"workspaceId"`
	WorkspaceName string  `json:"workspaceName"`
	TotalJobs     int     `json:"totalJobs"`
	Successful    int     `json:"successful"`
	Failed        int     `json:"failed"`
	SuccessRate   float64 `json:"successRate"`
	MinDurationMs int64   `json:"minDurationMs"`
	MaxDurationMs int64   `json:"maxDurationMs"`
	AvgDurationMs float64 `json:"avgDurationMs"`
}
