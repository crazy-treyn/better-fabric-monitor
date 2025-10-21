package db

import "time"

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

// JobInstance represents a job execution instance
type JobInstance struct {
	ID              string     `json:"id"`
	WorkspaceID     string     `json:"workspaceId"`
	ItemID          string     `json:"itemId"`
	JobType         string     `json:"jobType"`
	Status          string     `json:"status"`
	StartTime       time.Time  `json:"startTime"`
	EndTime         *time.Time `json:"endTime,omitempty"`
	DurationMs      *int64     `json:"durationMs,omitempty"`
	FailureReason   *string    `json:"failureReason,omitempty"`
	InvokerType     *string    `json:"invokerType,omitempty"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
	ItemDisplayName *string    `json:"itemDisplayName,omitempty"` // Joined from items table
	ItemType        *string    `json:"itemType,omitempty"`        // Joined from items table
	WorkspaceName   *string    `json:"workspaceName,omitempty"`   // Joined from workspaces table
}

// PipelineRun represents a pipeline run
type PipelineRun struct {
	RunID        string     `json:"runId"`
	WorkspaceID  string     `json:"workspaceId"`
	PipelineID   string     `json:"pipelineId"`
	PipelineName string     `json:"pipelineName"`
	Status       string     `json:"status"`
	StartTime    time.Time  `json:"startTime"`
	EndTime      *time.Time `json:"endTime,omitempty"`
	DurationMs   *int64     `json:"durationMs,omitempty"`
	ErrorMessage *string    `json:"errorMessage,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
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
}
