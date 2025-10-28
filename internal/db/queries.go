package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// SaveWorkspace saves or updates a workspace
func (db *Database) SaveWorkspace(workspace *Workspace) error {
	query := `
		INSERT INTO workspaces (id, display_name, type, description, updated_at)
		VALUES (?, ?, ?, ?, get_current_timestamp())
		ON CONFLICT(id) DO UPDATE SET
			display_name = EXCLUDED.display_name,
			type = EXCLUDED.type,
			description = EXCLUDED.description,
			updated_at = get_current_timestamp()
	`
	_, err := db.conn.Exec(query, workspace.ID, workspace.DisplayName, workspace.Type, workspace.Description)
	return err
}

// GetWorkspaces retrieves all workspaces
func (db *Database) GetWorkspaces() ([]Workspace, error) {
	query := `
		SELECT id, display_name, type, description, created_at, updated_at
		FROM workspaces
		ORDER BY display_name
	`
	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workspaces []Workspace
	for rows.Next() {
		var w Workspace
		err := rows.Scan(&w.ID, &w.DisplayName, &w.Type, &w.Description, &w.CreatedAt, &w.UpdatedAt)
		if err != nil {
			return nil, err
		}
		workspaces = append(workspaces, w)
	}
	return workspaces, rows.Err()
}

// SaveItem saves or updates an item
func (db *Database) SaveItem(item *Item) error {
	query := `
		INSERT INTO items (id, workspace_id, display_name, type, description, updated_at)
		VALUES (?, ?, ?, ?, ?, get_current_timestamp())
		ON CONFLICT(id) DO UPDATE SET
			display_name = EXCLUDED.display_name,
			type = EXCLUDED.type,
			description = EXCLUDED.description,
			updated_at = get_current_timestamp()
	`
	_, err := db.conn.Exec(query, item.ID, item.WorkspaceID, item.DisplayName, item.Type, item.Description)
	return err
}

// GetItemsByWorkspace retrieves items for a specific workspace
func (db *Database) GetItemsByWorkspace(workspaceID string) ([]Item, error) {
	query := `
		SELECT id, workspace_id, display_name, type, description, created_at, updated_at
		FROM items
		WHERE workspace_id = ?
		ORDER BY type, display_name
	`
	rows, err := db.conn.Query(query, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Item
	for rows.Next() {
		var item Item
		err := rows.Scan(&item.ID, &item.WorkspaceID, &item.DisplayName, &item.Type, &item.Description, &item.CreatedAt, &item.UpdatedAt)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// SaveJobInstances bulk inserts job instances
func (db *Database) SaveJobInstances(jobs []JobInstance) error {
	if len(jobs) == 0 {
		return nil
	}

	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO job_instances (
			id, workspace_id, item_id, job_type, status, start_time,
			end_time, duration_ms, failure_reason, invoker_type, root_activity_id, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, get_current_timestamp())
		ON CONFLICT(id) DO UPDATE SET
			status = EXCLUDED.status,
			end_time = EXCLUDED.end_time,
			duration_ms = EXCLUDED.duration_ms,
			failure_reason = EXCLUDED.failure_reason,
			root_activity_id = EXCLUDED.root_activity_id,
			updated_at = get_current_timestamp()
	`

	stmt, err := tx.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, job := range jobs {
		_, err = stmt.Exec(
			job.ID, job.WorkspaceID, job.ItemID, job.JobType, job.Status, job.StartTime,
			job.EndTime, job.DurationMs, job.FailureReason, job.InvokerType, job.RootActivityID,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetJobInstances retrieves job instances with filtering
func (db *Database) GetJobInstances(filter JobFilter) ([]JobInstance, error) {
	var conditions []string
	var args []interface{}

	if filter.WorkspaceID != nil {
		conditions = append(conditions, "j.workspace_id = ?")
		args = append(args, *filter.WorkspaceID)
	}

	if filter.ItemID != nil {
		conditions = append(conditions, "j.item_id = ?")
		args = append(args, *filter.ItemID)
	}

	if filter.JobType != nil {
		conditions = append(conditions, "j.job_type = ?")
		args = append(args, *filter.JobType)
	}

	if filter.Status != nil {
		conditions = append(conditions, "j.status = ?")
		args = append(args, *filter.Status)
	}

	if filter.StartDateFrom != nil {
		conditions = append(conditions, "j.start_time >= ?")
		args = append(args, *filter.StartDateFrom)
	}

	if filter.StartDateTo != nil {
		conditions = append(conditions, "j.start_time <= ?")
		args = append(args, *filter.StartDateTo)
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	limitClause := ""
	if filter.Limit != nil {
		limitClause = fmt.Sprintf("LIMIT %d", *filter.Limit)
		if filter.Offset != nil {
			limitClause += fmt.Sprintf(" OFFSET %d", *filter.Offset)
		}
	}

	query := fmt.Sprintf(`
		SELECT j.id, j.workspace_id, j.item_id, j.job_type, j.status, j.start_time,
			   j.end_time, j.duration_ms, j.failure_reason, j.invoker_type, j.root_activity_id, j.created_at, j.updated_at,
			   i.display_name as item_display_name, i.type as item_type,
			   w.display_name as workspace_display_name
		FROM job_instances j
		LEFT JOIN items i ON j.item_id = i.id
		LEFT JOIN workspaces w ON j.workspace_id = w.id
		%s
		ORDER BY j.start_time DESC
		%s
	`, whereClause, limitClause)

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []JobInstance
	for rows.Next() {
		var job JobInstance
		var itemDisplayName sql.NullString
		var itemType sql.NullString
		var workspaceDisplayName sql.NullString
		var rootActivityID sql.NullString

		err := rows.Scan(
			&job.ID, &job.WorkspaceID, &job.ItemID, &job.JobType, &job.Status, &job.StartTime,
			&job.EndTime, &job.DurationMs, &job.FailureReason, &job.InvokerType, &rootActivityID, &job.CreatedAt, &job.UpdatedAt,
			&itemDisplayName, &itemType, &workspaceDisplayName,
		)
		if err != nil {
			return nil, err
		}

		// Add item details to the job instance
		if itemDisplayName.Valid {
			job.ItemDisplayName = &itemDisplayName.String
		}
		if itemType.Valid {
			job.ItemType = &itemType.String
		}
		if workspaceDisplayName.Valid {
			job.WorkspaceName = &workspaceDisplayName.String
		}
		if rootActivityID.Valid {
			job.RootActivityID = &rootActivityID.String
		}

		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

// UpdateJobInstanceActivityRuns updates the activity runs for a job instance
func (db *Database) UpdateJobInstanceActivityRuns(jobID string, activityRuns []ActivityRun) error {
	activityRunsJSON, err := json.Marshal(activityRuns)
	if err != nil {
		return fmt.Errorf("failed to marshal activity runs: %w", err)
	}

	query := `
		UPDATE job_instances 
		SET activity_runs = ?,
			updated_at = get_current_timestamp()
		WHERE id = ?
	`

	_, err = db.conn.Exec(query, string(activityRunsJSON), jobID)
	return err
}

// GetJobInstanceWithActivities retrieves a job instance with its activity runs
func (db *Database) GetJobInstanceWithActivities(jobID string) (*JobInstance, error) {
	query := `
		SELECT 
			j.id, j.workspace_id, j.item_id, j.job_type, j.status, 
			j.start_time, j.end_time, j.duration_ms, j.failure_reason, 
			j.invoker_type, j.root_activity_id, j.activity_runs,
			j.created_at, j.updated_at,
			i.display_name as item_display_name, i.type as item_type,
			w.display_name as workspace_display_name
		FROM job_instances j
		LEFT JOIN items i ON j.item_id = i.id
		LEFT JOIN workspaces w ON j.workspace_id = w.id
		WHERE j.id = ?
	`

	var job JobInstance
	var activityRunsJSON sql.NullString
	var itemDisplayName sql.NullString
	var itemType sql.NullString
	var workspaceDisplayName sql.NullString
	var rootActivityID sql.NullString

	err := db.conn.QueryRow(query, jobID).Scan(
		&job.ID, &job.WorkspaceID, &job.ItemID, &job.JobType, &job.Status,
		&job.StartTime, &job.EndTime, &job.DurationMs, &job.FailureReason,
		&job.InvokerType, &rootActivityID, &activityRunsJSON,
		&job.CreatedAt, &job.UpdatedAt,
		&itemDisplayName, &itemType, &workspaceDisplayName,
	)

	if err != nil {
		return nil, err
	}

	// Set optional fields
	if itemDisplayName.Valid {
		job.ItemDisplayName = &itemDisplayName.String
	}
	if itemType.Valid {
		job.ItemType = &itemType.String
	}
	if workspaceDisplayName.Valid {
		job.WorkspaceName = &workspaceDisplayName.String
	}
	if rootActivityID.Valid {
		job.RootActivityID = &rootActivityID.String
	}

	// Unmarshal activity runs if present
	if activityRunsJSON.Valid && activityRunsJSON.String != "" {
		if err := json.Unmarshal([]byte(activityRunsJSON.String), &job.ActivityRuns); err != nil {
			return nil, fmt.Errorf("failed to unmarshal activity runs: %w", err)
		}
		count := len(job.ActivityRuns)
		job.ActivityCount = &count
	}

	return &job, nil
}

// GetChildExecutions extracts child pipeline and notebook executions from activity runs
func (db *Database) GetChildExecutions(jobID string) ([]ChildExecution, error) {
	query := `
		WITH child_activities AS (
			SELECT 
				json_extract_string(activity, '$.activityRunId') as activity_run_id,
				json_extract_string(activity, '$.activityName') as activity_name,
				json_extract_string(activity, '$.activityType') as activity_type,
				json_extract_string(activity, '$.status') as status,
				json_extract_string(activity, '$.activityRunStart') as start_time,
				json_extract_string(activity, '$.activityRunEnd') as end_time,
				CAST(json_extract(activity, '$.durationInMs') AS BIGINT) as duration_ms,
				json_extract_string(activity, '$.error.message') as error_message,
				json_extract_string(activity, '$.pipelineId') as pipeline_id,
				-- Extract child job instance ID from output
				COALESCE(
					json_extract_string(activity, '$.output.pipelineRunId'),
					json_extract_string(activity, '$.output.runId')
				) as child_job_instance_id,
				-- Extract child pipeline name from output
				json_extract_string(activity, '$.output.properties.pipelineName') as child_pipeline_name,
				-- Parent job's workspace
				j.workspace_id as parent_workspace_id
			FROM job_instances j
			CROSS JOIN unnest(
				CASE 
					WHEN j.activity_runs IS NOT NULL 
					THEN CAST(j.activity_runs AS JSON[])
					ELSE []::JSON[]
				END
			) as t(activity)
			WHERE j.id = ?
				AND json_extract_string(activity, '$.activityType') IN ('ExecutePipeline', 'TridentNotebook')
		)
		SELECT 
			ca.activity_run_id,
			ca.activity_name,
			ca.activity_type,
			ca.status,
			ca.start_time,
			ca.end_time,
			ca.duration_ms,
			ca.error_message,
			ca.pipeline_id,
			ca.child_job_instance_id,
			ca.child_pipeline_name,
			ca.parent_workspace_id,
			-- Join with job_instances to get child job details if exists
			child_job.item_id as child_item_id,
			child_item.type as child_item_type,
			child_item.display_name as child_item_display_name
		FROM child_activities ca
		LEFT JOIN job_instances child_job ON child_job.id = ca.child_job_instance_id
		LEFT JOIN items child_item ON child_job.item_id = child_item.id
		ORDER BY ca.start_time ASC
	`

	rows, err := db.conn.Query(query, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var children []ChildExecution
	for rows.Next() {
		var child ChildExecution
		var startTimeStr sql.NullString
		var endTimeStr sql.NullString
		var durationMs sql.NullInt64
		var errorMsg sql.NullString
		var childJobInstanceID sql.NullString
		var childPipelineName sql.NullString
		var parentWorkspaceID sql.NullString
		var childItemID sql.NullString
		var childItemType sql.NullString
		var childItemDisplayName sql.NullString

		err := rows.Scan(
			&child.ActivityRunID,
			&child.ActivityName,
			&child.ActivityType,
			&child.Status,
			&startTimeStr,
			&endTimeStr,
			&durationMs,
			&errorMsg,
			&child.PipelineID,
			&childJobInstanceID,
			&childPipelineName,
			&parentWorkspaceID,
			&childItemID,
			&childItemType,
			&childItemDisplayName,
		)
		if err != nil {
			return nil, err
		}

		// Parse start time if present
		if startTimeStr.Valid && startTimeStr.String != "" {
			startTime, err := time.Parse(time.RFC3339, startTimeStr.String)
			if err == nil {
				child.StartTime = &startTime
			}
		}

		// Parse end time if present
		if endTimeStr.Valid && endTimeStr.String != "" {
			endTime, err := time.Parse(time.RFC3339, endTimeStr.String)
			if err == nil {
				child.EndTime = &endTime
			}
		}

		// Set duration if valid
		if durationMs.Valid {
			child.DurationMs = &durationMs.Int64
		}

		// Set error message if present
		if errorMsg.Valid && errorMsg.String != "" {
			child.ErrorMessage = &errorMsg.String
		}

		// Set child execution details for deep linking
		if childJobInstanceID.Valid && childJobInstanceID.String != "" {
			child.ChildJobInstanceID = &childJobInstanceID.String
		}
		if childPipelineName.Valid && childPipelineName.String != "" {
			child.ChildPipelineName = &childPipelineName.String
		}
		if parentWorkspaceID.Valid && parentWorkspaceID.String != "" {
			child.ChildWorkspaceID = &parentWorkspaceID.String
		}
		if childItemID.Valid && childItemID.String != "" {
			child.ChildItemID = &childItemID.String
		}
		if childItemType.Valid && childItemType.String != "" {
			child.ChildItemType = &childItemType.String
		}
		if childItemDisplayName.Valid && childItemDisplayName.String != "" {
			child.ChildItemDisplayName = &childItemDisplayName.String
		}

		// For future recursive expansion - check if this is an ExecutePipeline
		child.HasChildren = child.ActivityType == "ExecutePipeline"

		children = append(children, child)
	}

	return children, rows.Err()
}

// GetOverallStats returns aggregated statistics for the specified time period
func (db *Database) GetOverallStats(days int) (*JobStats, error) {
	query := `
		SELECT
			COUNT(*) as total_jobs,
			COALESCE(SUM(CASE WHEN status = 'Completed' THEN 1 ELSE 0 END), 0) as successful,
			COALESCE(SUM(CASE WHEN status = 'Failed' THEN 1 ELSE 0 END), 0) as failed,
			COALESCE(SUM(CASE WHEN status IN ('InProgress', 'Running', 'NotStarted') THEN 1 ELSE 0 END), 0) as running,
			AVG(CASE WHEN status = 'Completed' AND duration_ms IS NOT NULL THEN duration_ms ELSE NULL END) as avg_duration_ms
		FROM job_instances
		WHERE start_time >= CURRENT_TIMESTAMP - INTERVAL (? || ' days')
	`

	var stats JobStats
	var avgDuration sql.NullFloat64

	err := db.conn.QueryRow(query, fmt.Sprintf("%d", days)).Scan(
		&stats.TotalJobs, &stats.Successful, &stats.Failed, &stats.Running, &avgDuration,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return &JobStats{}, nil
		}
		return nil, err
	}

	if avgDuration.Valid {
		stats.AvgDurationMs = avgDuration.Float64
	}

	if stats.TotalJobs > 0 {
		stats.SuccessRate = float64(stats.Successful) / float64(stats.TotalJobs) * 100
	}

	return &stats, nil
}

// GetJobStats returns aggregated statistics
func (db *Database) GetJobStats(workspaceID string, from, to time.Time) (*JobStats, error) {
	query := `
		SELECT
			COUNT(*) as total_jobs,
			COALESCE(SUM(CASE WHEN status = 'Completed' THEN 1 ELSE 0 END), 0) as successful,
			COALESCE(SUM(CASE WHEN status = 'Failed' THEN 1 ELSE 0 END), 0) as failed,
			AVG(duration_ms) as avg_duration_ms
		FROM job_instances
		WHERE workspace_id = ? AND start_time >= ? AND start_time <= ?
	`

	var stats JobStats
	err := db.conn.QueryRow(query, workspaceID, from, to).Scan(
		&stats.TotalJobs, &stats.Successful, &stats.Failed, &stats.AvgDurationMs,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return &JobStats{}, nil
		}
		return nil, err
	}

	if stats.TotalJobs > 0 {
		stats.SuccessRate = float64(stats.Successful) / float64(stats.TotalJobs) * 100
	}

	return &stats, nil
}

// UpdateSyncMetadata records a sync operation
func (db *Database) UpdateSyncMetadata(syncType string, recordsSynced, errors int) error {
	query := `
		INSERT INTO sync_metadata (last_sync_time, sync_type, records_synced, errors)
		VALUES (get_current_timestamp(), ?, ?, ?)
	`
	_, err := db.conn.Exec(query, syncType, recordsSynced, errors)
	return err
}

// GetLastSyncTime returns the last sync time for a given sync type
func (db *Database) GetLastSyncTime(syncType string) (*time.Time, error) {
	query := `
		SELECT last_sync_time
		FROM sync_metadata
		WHERE sync_type = ?
		ORDER BY last_sync_time DESC
		LIMIT 1
	`

	var lastSync time.Time
	err := db.conn.QueryRow(query, syncType).Scan(&lastSync)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &lastSync, nil
}

// GetMaxJobStartTime returns the start time to use for incremental sync
// If there are any in-progress jobs (no end_time), returns the MINIMUM start_time of those jobs
// Otherwise, returns the MAXIMUM start_time of completed jobs
// This ensures we always re-check in-progress jobs for status updates
func (db *Database) GetMaxJobStartTime() (*time.Time, error) {
	// First check if there are any in-progress jobs
	queryInProgress := `
		SELECT MIN(start_time)
		FROM job_instances
		WHERE end_time IS NULL
	`

	var minInProgressStartTime sql.NullTime
	err := db.conn.QueryRow(queryInProgress).Scan(&minInProgressStartTime)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// If we found in-progress jobs, return the earliest one
	if minInProgressStartTime.Valid {
		return &minInProgressStartTime.Time, nil
	}

	// No in-progress jobs, use max start time of completed jobs
	queryCompleted := `
		SELECT MAX(start_time)
		FROM job_instances
		WHERE end_time IS NOT NULL
	`

	var maxStartTime sql.NullTime
	err = db.conn.QueryRow(queryCompleted).Scan(&maxStartTime)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	if maxStartTime.Valid {
		return &maxStartTime.Time, nil
	}

	// No jobs at all
	return nil, nil
}

// GetInProgressJobIDs returns IDs of jobs that don't have an end time (still in progress)
func (db *Database) GetInProgressJobIDs() ([]string, error) {
	query := `
		SELECT id
		FROM job_instances
		WHERE end_time IS NULL
	`

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// GetDailyStats returns job statistics grouped by day for the last N days
func (db *Database) GetDailyStats(days int) ([]DailyStats, error) {
	query := `
		SELECT
			DATE_TRUNC('day', start_time)::DATE as date,
			COUNT(*) as total_jobs,
			COALESCE(SUM(CASE WHEN status = 'Completed' THEN 1 ELSE 0 END), 0) as successful,
			COALESCE(SUM(CASE WHEN status = 'Failed' THEN 1 ELSE 0 END), 0) as failed,
			COALESCE(SUM(CASE WHEN status IN ('InProgress', 'Running', 'NotStarted') THEN 1 ELSE 0 END), 0) as running,
			AVG(CASE WHEN duration_ms IS NOT NULL THEN duration_ms ELSE NULL END) as avg_duration_ms
		FROM job_instances
		WHERE start_time >= CURRENT_TIMESTAMP - INTERVAL (? || ' days')
		GROUP BY DATE_TRUNC('day', start_time)::DATE
		ORDER BY date ASC
	`

	rows, err := db.conn.Query(query, fmt.Sprintf("%d", days))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []DailyStats
	for rows.Next() {
		var s DailyStats
		var avgDuration sql.NullFloat64

		err := rows.Scan(&s.Date, &s.TotalJobs, &s.Successful, &s.Failed, &s.Running, &avgDuration)
		if err != nil {
			return nil, err
		}

		if avgDuration.Valid {
			s.AvgDurationMs = avgDuration.Float64
		}

		if s.TotalJobs > 0 {
			s.SuccessRate = float64(s.Successful) / float64(s.TotalJobs) * 100
		}

		stats = append(stats, s)
	}
	return stats, rows.Err()
}

// GetWorkspaceStats returns job statistics by workspace
func (db *Database) GetWorkspaceStats(days int) ([]WorkspaceStats, error) {
	query := `
		SELECT
			j.workspace_id,
			w.display_name as workspace_name,
			COUNT(*) as total_jobs,
			COALESCE(SUM(CASE WHEN j.status = 'Completed' THEN 1 ELSE 0 END), 0) as successful,
			COALESCE(SUM(CASE WHEN j.status = 'Failed' THEN 1 ELSE 0 END), 0) as failed,
			COALESCE(SUM(CASE WHEN j.status IN ('InProgress', 'Running', 'NotStarted') THEN 1 ELSE 0 END), 0) as running,
			AVG(CASE WHEN j.duration_ms IS NOT NULL THEN j.duration_ms ELSE NULL END) as avg_duration_ms
		FROM job_instances j
		LEFT JOIN workspaces w ON j.workspace_id = w.id
		WHERE j.start_time >= CURRENT_TIMESTAMP - INTERVAL (? || ' days')
		GROUP BY j.workspace_id, w.display_name
		ORDER BY total_jobs DESC
	`

	rows, err := db.conn.Query(query, fmt.Sprintf("%d", days))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []WorkspaceStats
	for rows.Next() {
		var s WorkspaceStats
		var avgDuration sql.NullFloat64

		err := rows.Scan(&s.WorkspaceID, &s.WorkspaceName, &s.TotalJobs, &s.Successful, &s.Failed, &s.Running, &avgDuration)
		if err != nil {
			return nil, err
		}

		if avgDuration.Valid {
			s.AvgDurationMs = avgDuration.Float64
		}

		if s.TotalJobs > 0 {
			s.SuccessRate = float64(s.Successful) / float64(s.TotalJobs) * 100
		}

		stats = append(stats, s)
	}
	return stats, rows.Err()
}

// GetItemTypeStats returns job statistics by item type
func (db *Database) GetItemTypeStats(days int) ([]ItemTypeStats, error) {
	query := `
		SELECT
			i.type as item_type,
			COUNT(*) as total_jobs,
			COALESCE(SUM(CASE WHEN j.status = 'Completed' THEN 1 ELSE 0 END), 0) as successful,
			COALESCE(SUM(CASE WHEN j.status = 'Failed' THEN 1 ELSE 0 END), 0) as failed,
			COALESCE(SUM(CASE WHEN j.status IN ('InProgress', 'Running', 'NotStarted') THEN 1 ELSE 0 END), 0) as running,
			AVG(CASE WHEN j.duration_ms IS NOT NULL THEN j.duration_ms ELSE NULL END) as avg_duration_ms
		FROM job_instances j
		LEFT JOIN items i ON j.item_id = i.id
		WHERE j.start_time >= CURRENT_TIMESTAMP - INTERVAL (? || ' days')
		GROUP BY i.type
		ORDER BY total_jobs DESC
	`

	rows, err := db.conn.Query(query, fmt.Sprintf("%d", days))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []ItemTypeStats
	for rows.Next() {
		var s ItemTypeStats
		var avgDuration sql.NullFloat64

		err := rows.Scan(&s.ItemType, &s.TotalJobs, &s.Successful, &s.Failed, &s.Running, &avgDuration)
		if err != nil {
			return nil, err
		}

		if avgDuration.Valid {
			s.AvgDurationMs = avgDuration.Float64
		}

		if s.TotalJobs > 0 {
			s.SuccessRate = float64(s.Successful) / float64(s.TotalJobs) * 100
		}

		stats = append(stats, s)
	}
	return stats, rows.Err()
}

// GetRecentFailures returns the most recent job failures within the specified days
func (db *Database) GetRecentFailures(limit int, days int) ([]RecentFailure, error) {
	query := `
		SELECT
			j.id, j.workspace_id, w.display_name as workspace_name,
			j.item_id, i.display_name as item_display_name, i.type as item_type,
			j.job_type, j.start_time, j.end_time, j.duration_ms, j.failure_reason
		FROM job_instances j
		LEFT JOIN items i ON j.item_id = i.id
		LEFT JOIN workspaces w ON j.workspace_id = w.id
		WHERE j.status = 'Failed' 
			AND j.end_time IS NOT NULL
			AND j.start_time >= CURRENT_TIMESTAMP - INTERVAL (? || ' days')
		ORDER BY j.start_time DESC
		LIMIT ?
	`

	rows, err := db.conn.Query(query, fmt.Sprintf("%d", days), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var failures []RecentFailure
	for rows.Next() {
		var f RecentFailure
		var endTime sql.NullTime
		var durationMs sql.NullInt64
		var failureReason sql.NullString

		err := rows.Scan(
			&f.ID, &f.WorkspaceID, &f.WorkspaceName,
			&f.ItemID, &f.ItemDisplayName, &f.ItemType,
			&f.JobType, &f.StartTime, &endTime, &durationMs, &failureReason,
		)
		if err != nil {
			return nil, err
		}

		if endTime.Valid {
			f.EndTime = endTime.Time
		}
		if durationMs.Valid {
			f.DurationMs = durationMs.Int64
		}
		if failureReason.Valid {
			f.FailureReason = failureReason.String
		}

		failures = append(failures, f)
	}
	return failures, rows.Err()
}

// GetLongRunningJobs returns jobs that took significantly longer than average
func (db *Database) GetLongRunningJobs(days int, minDeviationPct float64, limit int) ([]LongRunningJob, error) {
	query := `
		WITH item_averages AS (
			SELECT
				item_id,
				AVG(duration_ms) as avg_duration_ms
			FROM job_instances
			WHERE status = 'Completed'
				AND duration_ms IS NOT NULL
				AND start_time >= CURRENT_TIMESTAMP - INTERVAL (? || ' days')
			GROUP BY item_id
			HAVING COUNT(*) >= 3
		)
		SELECT
			j.id, j.workspace_id, w.display_name as workspace_name,
			j.item_id, i.display_name as item_display_name, i.type as item_type,
			j.job_type, j.start_time, j.duration_ms,
			a.avg_duration_ms,
			((j.duration_ms - a.avg_duration_ms) / a.avg_duration_ms * 100) as deviation_pct
		FROM job_instances j
		INNER JOIN item_averages a ON j.item_id = a.item_id
		LEFT JOIN items i ON j.item_id = i.id
		LEFT JOIN workspaces w ON j.workspace_id = w.id
		WHERE j.status = 'Completed'
			AND j.duration_ms IS NOT NULL
			AND j.start_time >= CURRENT_TIMESTAMP - INTERVAL (? || ' days')
			AND ((j.duration_ms - a.avg_duration_ms) / a.avg_duration_ms * 100) > ?
		ORDER BY deviation_pct DESC
		LIMIT ?
	`

	rows, err := db.conn.Query(query, fmt.Sprintf("%d", days), fmt.Sprintf("%d", days), minDeviationPct, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []LongRunningJob
	for rows.Next() {
		var j LongRunningJob

		err := rows.Scan(
			&j.ID, &j.WorkspaceID, &j.WorkspaceName,
			&j.ItemID, &j.ItemDisplayName, &j.ItemType,
			&j.JobType, &j.StartTime, &j.DurationMs,
			&j.AvgDurationMs, &j.DeviationPct,
		)
		if err != nil {
			return nil, err
		}

		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

// GetItemStatsByWorkspace returns job statistics for each item in a workspace
func (db *Database) GetItemStatsByWorkspace(workspaceID string, days int) ([]ItemStats, error) {
	query := `
		SELECT
			j.item_id,
			i.display_name as item_name,
			i.type as item_type,
			j.workspace_id,
			w.display_name as workspace_name,
			COUNT(*) as total_jobs,
			COALESCE(SUM(CASE WHEN j.status = 'Completed' THEN 1 ELSE 0 END), 0) as successful,
			COALESCE(SUM(CASE WHEN j.status = 'Failed' THEN 1 ELSE 0 END), 0) as failed,
			COALESCE(SUM(CASE WHEN j.status IN ('InProgress', 'Running', 'NotStarted') THEN 1 ELSE 0 END), 0) as running,
			AVG(CASE WHEN j.duration_ms IS NOT NULL THEN j.duration_ms ELSE NULL END) as avg_duration_ms
		FROM job_instances j
		LEFT JOIN items i ON j.item_id = i.id
		LEFT JOIN workspaces w ON j.workspace_id = w.id
		WHERE j.workspace_id = ?
			AND j.start_time >= CURRENT_TIMESTAMP - INTERVAL (? || ' days')
		GROUP BY j.item_id, i.display_name, i.type, j.workspace_id, w.display_name
		ORDER BY total_jobs DESC
	`

	rows, err := db.conn.Query(query, workspaceID, fmt.Sprintf("%d", days))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []ItemStats
	for rows.Next() {
		var s ItemStats
		var avgDuration sql.NullFloat64

		err := rows.Scan(&s.ItemID, &s.ItemName, &s.ItemType, &s.WorkspaceID, &s.WorkspaceName, &s.TotalJobs, &s.Successful, &s.Failed, &s.Running, &avgDuration)
		if err != nil {
			return nil, err
		}

		if avgDuration.Valid {
			s.AvgDurationMs = avgDuration.Float64
		}

		if s.TotalJobs > 0 {
			s.SuccessRate = float64(s.Successful) / float64(s.TotalJobs) * 100
		}

		stats = append(stats, s)
	}
	return stats, rows.Err()
}

// GetItemStatsByJobType returns job statistics for each item of a specific type
func (db *Database) GetItemStatsByJobType(itemType string, days int) ([]ItemStats, error) {
	query := `
		SELECT
			j.item_id,
			i.display_name as item_name,
			i.type as item_type,
			j.workspace_id,
			w.display_name as workspace_name,
			COUNT(*) as total_jobs,
			COALESCE(SUM(CASE WHEN j.status = 'Completed' THEN 1 ELSE 0 END), 0) as successful,
			COALESCE(SUM(CASE WHEN j.status = 'Failed' THEN 1 ELSE 0 END), 0) as failed,
			COALESCE(SUM(CASE WHEN j.status IN ('InProgress', 'Running', 'NotStarted') THEN 1 ELSE 0 END), 0) as running,
			AVG(CASE WHEN j.duration_ms IS NOT NULL THEN j.duration_ms ELSE NULL END) as avg_duration_ms
		FROM job_instances j
		LEFT JOIN items i ON j.item_id = i.id
		LEFT JOIN workspaces w ON j.workspace_id = w.id
		WHERE i.type = ?
			AND j.start_time >= CURRENT_TIMESTAMP - INTERVAL (? || ' days')
		GROUP BY j.item_id, i.display_name, i.type, j.workspace_id, w.display_name
		ORDER BY total_jobs DESC
	`

	rows, err := db.conn.Query(query, itemType, fmt.Sprintf("%d", days))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []ItemStats
	for rows.Next() {
		var s ItemStats
		var avgDuration sql.NullFloat64

		err := rows.Scan(&s.ItemID, &s.ItemName, &s.ItemType, &s.WorkspaceID, &s.WorkspaceName, &s.TotalJobs, &s.Successful, &s.Failed, &s.Running, &avgDuration)
		if err != nil {
			return nil, err
		}

		if avgDuration.Valid {
			s.AvgDurationMs = avgDuration.Float64
		}

		if s.TotalJobs > 0 {
			s.SuccessRate = float64(s.Successful) / float64(s.TotalJobs) * 100
		}

		stats = append(stats, s)
	}
	return stats, rows.Err()
}

// GetItemStatsByDate returns job statistics for each item on a specific date with optional filters
func (db *Database) GetItemStatsByDate(date string, workspaceIDs []string, itemTypes []string, itemNameSearch string) ([]DailyItemStats, error) {
	filterClause, filterArgs := buildFilterConditions(workspaceIDs, itemTypes, itemNameSearch)

	query := fmt.Sprintf(`
		SELECT
			j.item_id,
			i.display_name as item_name,
			i.type as item_type,
			j.workspace_id,
			w.display_name as workspace_name,
			COUNT(*) as total_jobs,
			COALESCE(SUM(CASE WHEN j.status = 'Completed' THEN 1 ELSE 0 END), 0) as successful,
			COALESCE(SUM(CASE WHEN j.status = 'Failed' THEN 1 ELSE 0 END), 0) as failed,
			MIN(CASE WHEN j.duration_ms IS NOT NULL THEN j.duration_ms END) as min_duration_ms,
			MAX(CASE WHEN j.duration_ms IS NOT NULL THEN j.duration_ms END) as max_duration_ms,
			AVG(CASE WHEN j.duration_ms IS NOT NULL THEN j.duration_ms END) as avg_duration_ms
		FROM job_instances j
		LEFT JOIN items i ON j.item_id = i.id
		LEFT JOIN workspaces w ON j.workspace_id = w.id
		WHERE DATE_TRUNC('day', j.start_time)::DATE = ?
		%s
		GROUP BY j.item_id, i.display_name, i.type, j.workspace_id, w.display_name
		ORDER BY total_jobs DESC
	`, filterClause)

	args := []interface{}{date}
	args = append(args, filterArgs...)

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []DailyItemStats
	for rows.Next() {
		var s DailyItemStats
		var minDuration sql.NullInt64
		var maxDuration sql.NullInt64
		var avgDuration sql.NullFloat64

		err := rows.Scan(&s.ItemID, &s.ItemName, &s.ItemType, &s.WorkspaceID, &s.WorkspaceName, &s.TotalJobs, &s.Successful, &s.Failed, &minDuration, &maxDuration, &avgDuration)
		if err != nil {
			return nil, err
		}

		if minDuration.Valid {
			s.MinDurationMs = minDuration.Int64
		}
		if maxDuration.Valid {
			s.MaxDurationMs = maxDuration.Int64
		}
		if avgDuration.Valid {
			s.AvgDurationMs = avgDuration.Float64
		}

		if s.TotalJobs > 0 {
			s.SuccessRate = float64(s.Successful) / float64(s.TotalJobs) * 100
		}

		stats = append(stats, s)
	}
	return stats, rows.Err()
}

// GetInProgressJobsByWorkspaceAndItem returns job instances that are in progress for a specific workspace/item
func (db *Database) GetInProgressJobsByWorkspaceAndItem(workspaceID, itemID string) ([]JobInstance, error) {
	query := `
		SELECT id, workspace_id, item_id, job_type, status, start_time,
			   end_time, duration_ms, failure_reason, invoker_type, root_activity_id, created_at, updated_at
		FROM job_instances
		WHERE workspace_id = ? AND item_id = ? AND end_time IS NULL
		ORDER BY start_time DESC
	`

	rows, err := db.conn.Query(query, workspaceID, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []JobInstance
	for rows.Next() {
		var job JobInstance
		err := rows.Scan(
			&job.ID, &job.WorkspaceID, &job.ItemID, &job.JobType, &job.Status, &job.StartTime,
			&job.EndTime, &job.DurationMs, &job.FailureReason, &job.InvokerType, &job.RootActivityID, &job.CreatedAt, &job.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

// buildFilterConditions builds WHERE clause conditions for analytics queries
func buildFilterConditions(workspaceIDs []string, itemTypes []string, itemNameSearch string) (string, []interface{}) {
	var conditions []string
	var args []interface{}

	// Workspace filter
	if len(workspaceIDs) > 0 {
		placeholders := make([]string, len(workspaceIDs))
		for i, id := range workspaceIDs {
			placeholders[i] = "?"
			args = append(args, id)
		}
		conditions = append(conditions, fmt.Sprintf("j.workspace_id IN (%s)", strings.Join(placeholders, ",")))
	}

	// Item type filter
	if len(itemTypes) > 0 {
		placeholders := make([]string, len(itemTypes))
		for i, t := range itemTypes {
			placeholders[i] = "?"
			args = append(args, t)
		}
		conditions = append(conditions, fmt.Sprintf("i.type IN (%s)", strings.Join(placeholders, ",")))
	}

	// Item name search (case-insensitive partial match)
	if itemNameSearch != "" {
		conditions = append(conditions, "LOWER(i.display_name) LIKE LOWER(?)")
		args = append(args, "%"+itemNameSearch+"%")
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " AND " + strings.Join(conditions, " AND ")
	}

	return whereClause, args
}

// GetAvailableItemTypes returns distinct item types that have job data in the specified time period
func (db *Database) GetAvailableItemTypes(days int, workspaceIDs []string) ([]string, error) {
	baseQuery := `
		SELECT DISTINCT i.type
		FROM job_instances j
		LEFT JOIN items i ON j.item_id = i.id
		WHERE j.start_time >= CURRENT_TIMESTAMP - INTERVAL (? || ' days')
			AND i.type IS NOT NULL
	`

	var conditions []string
	var args []interface{}
	args = append(args, fmt.Sprintf("%d", days))

	// Workspace filter
	if len(workspaceIDs) > 0 {
		placeholders := make([]string, len(workspaceIDs))
		for i, id := range workspaceIDs {
			placeholders[i] = "?"
			args = append(args, id)
		}
		conditions = append(conditions, fmt.Sprintf("j.workspace_id IN (%s)", strings.Join(placeholders, ",")))
	}

	if len(conditions) > 0 {
		baseQuery += " AND " + strings.Join(conditions, " AND ")
	}

	baseQuery += " ORDER BY i.type"

	rows, err := db.conn.Query(baseQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var types []string
	for rows.Next() {
		var itemType string
		if err := rows.Scan(&itemType); err != nil {
			return nil, err
		}
		types = append(types, itemType)
	}
	return types, rows.Err()
}

// GetOverallStatsFiltered returns aggregated statistics with optional filters
func (db *Database) GetOverallStatsFiltered(days int, workspaceIDs []string, itemTypes []string, itemNameSearch string) (*JobStats, error) {
	filterClause, filterArgs := buildFilterConditions(workspaceIDs, itemTypes, itemNameSearch)

	query := fmt.Sprintf(`
		SELECT
			COUNT(*) as total_jobs,
			COALESCE(SUM(CASE WHEN j.status = 'Completed' THEN 1 ELSE 0 END), 0) as successful,
			COALESCE(SUM(CASE WHEN j.status = 'Failed' THEN 1 ELSE 0 END), 0) as failed,
			COALESCE(SUM(CASE WHEN j.status IN ('InProgress', 'Running', 'NotStarted') THEN 1 ELSE 0 END), 0) as running,
			AVG(CASE WHEN j.status = 'Completed' AND j.duration_ms IS NOT NULL THEN j.duration_ms ELSE NULL END) as avg_duration_ms
		FROM job_instances j
		LEFT JOIN items i ON j.item_id = i.id
		WHERE j.start_time >= CURRENT_TIMESTAMP - INTERVAL (? || ' days')
		%s
	`, filterClause)

	args := []interface{}{fmt.Sprintf("%d", days)}
	args = append(args, filterArgs...)

	var stats JobStats
	var avgDuration sql.NullFloat64

	err := db.conn.QueryRow(query, args...).Scan(
		&stats.TotalJobs, &stats.Successful, &stats.Failed, &stats.Running, &avgDuration,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return &JobStats{}, nil
		}
		return nil, err
	}

	if avgDuration.Valid {
		stats.AvgDurationMs = avgDuration.Float64
	}

	if stats.TotalJobs > 0 {
		stats.SuccessRate = float64(stats.Successful) / float64(stats.TotalJobs) * 100
	}

	return &stats, nil
}

// GetDailyStatsFiltered returns daily statistics with optional filters
func (db *Database) GetDailyStatsFiltered(days int, workspaceIDs []string, itemTypes []string, itemNameSearch string) ([]DailyStats, error) {
	filterClause, filterArgs := buildFilterConditions(workspaceIDs, itemTypes, itemNameSearch)

	query := fmt.Sprintf(`
		SELECT
			DATE_TRUNC('day', j.start_time)::DATE as date,
			COUNT(*) as total_jobs,
			COALESCE(SUM(CASE WHEN j.status = 'Completed' THEN 1 ELSE 0 END), 0) as successful,
			COALESCE(SUM(CASE WHEN j.status = 'Failed' THEN 1 ELSE 0 END), 0) as failed,
			COALESCE(SUM(CASE WHEN j.status IN ('InProgress', 'Running', 'NotStarted') THEN 1 ELSE 0 END), 0) as running,
			AVG(CASE WHEN j.duration_ms IS NOT NULL THEN j.duration_ms ELSE NULL END) as avg_duration_ms
		FROM job_instances j
		LEFT JOIN items i ON j.item_id = i.id
		WHERE j.start_time >= CURRENT_TIMESTAMP - INTERVAL (? || ' days')
		%s
		GROUP BY DATE_TRUNC('day', j.start_time)::DATE
		ORDER BY date ASC
	`, filterClause)

	args := []interface{}{fmt.Sprintf("%d", days)}
	args = append(args, filterArgs...)

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []DailyStats
	for rows.Next() {
		var s DailyStats
		var avgDuration sql.NullFloat64

		err := rows.Scan(&s.Date, &s.TotalJobs, &s.Successful, &s.Failed, &s.Running, &avgDuration)
		if err != nil {
			return nil, err
		}

		if avgDuration.Valid {
			s.AvgDurationMs = avgDuration.Float64
		}

		if s.TotalJobs > 0 {
			s.SuccessRate = float64(s.Successful) / float64(s.TotalJobs) * 100
		}

		stats = append(stats, s)
	}
	return stats, rows.Err()
}

// GetWorkspaceStatsFiltered returns workspace statistics with optional filters
func (db *Database) GetWorkspaceStatsFiltered(days int, workspaceIDs []string, itemTypes []string, itemNameSearch string) ([]WorkspaceStats, error) {
	filterClause, filterArgs := buildFilterConditions(workspaceIDs, itemTypes, itemNameSearch)

	query := fmt.Sprintf(`
		SELECT
			j.workspace_id,
			w.display_name as workspace_name,
			COUNT(*) as total_jobs,
			COALESCE(SUM(CASE WHEN j.status = 'Completed' THEN 1 ELSE 0 END), 0) as successful,
			COALESCE(SUM(CASE WHEN j.status = 'Failed' THEN 1 ELSE 0 END), 0) as failed,
			COALESCE(SUM(CASE WHEN j.status IN ('InProgress', 'Running', 'NotStarted') THEN 1 ELSE 0 END), 0) as running,
			AVG(CASE WHEN j.duration_ms IS NOT NULL THEN j.duration_ms ELSE NULL END) as avg_duration_ms
		FROM job_instances j
		LEFT JOIN workspaces w ON j.workspace_id = w.id
		LEFT JOIN items i ON j.item_id = i.id
		WHERE j.start_time >= CURRENT_TIMESTAMP - INTERVAL (? || ' days')
		%s
		GROUP BY j.workspace_id, w.display_name
		ORDER BY total_jobs DESC
	`, filterClause)

	args := []interface{}{fmt.Sprintf("%d", days)}
	args = append(args, filterArgs...)

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []WorkspaceStats
	for rows.Next() {
		var s WorkspaceStats
		var avgDuration sql.NullFloat64

		err := rows.Scan(&s.WorkspaceID, &s.WorkspaceName, &s.TotalJobs, &s.Successful, &s.Failed, &s.Running, &avgDuration)
		if err != nil {
			return nil, err
		}

		if avgDuration.Valid {
			s.AvgDurationMs = avgDuration.Float64
		}

		if s.TotalJobs > 0 {
			s.SuccessRate = float64(s.Successful) / float64(s.TotalJobs) * 100
		}

		stats = append(stats, s)
	}
	return stats, rows.Err()
}

// GetItemTypeStatsFiltered returns item type statistics with optional filters
func (db *Database) GetItemTypeStatsFiltered(days int, workspaceIDs []string, itemTypes []string, itemNameSearch string) ([]ItemTypeStats, error) {
	filterClause, filterArgs := buildFilterConditions(workspaceIDs, itemTypes, itemNameSearch)

	query := fmt.Sprintf(`
		SELECT
			i.type as item_type,
			COUNT(*) as total_jobs,
			COALESCE(SUM(CASE WHEN j.status = 'Completed' THEN 1 ELSE 0 END), 0) as successful,
			COALESCE(SUM(CASE WHEN j.status = 'Failed' THEN 1 ELSE 0 END), 0) as failed,
			COALESCE(SUM(CASE WHEN j.status IN ('InProgress', 'Running', 'NotStarted') THEN 1 ELSE 0 END), 0) as running,
			AVG(CASE WHEN j.duration_ms IS NOT NULL THEN j.duration_ms ELSE NULL END) as avg_duration_ms
		FROM job_instances j
		LEFT JOIN items i ON j.item_id = i.id
		WHERE j.start_time >= CURRENT_TIMESTAMP - INTERVAL (? || ' days')
		%s
		GROUP BY i.type
		ORDER BY total_jobs DESC
	`, filterClause)

	args := []interface{}{fmt.Sprintf("%d", days)}
	args = append(args, filterArgs...)

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []ItemTypeStats
	for rows.Next() {
		var s ItemTypeStats
		var avgDuration sql.NullFloat64

		err := rows.Scan(&s.ItemType, &s.TotalJobs, &s.Successful, &s.Failed, &s.Running, &avgDuration)
		if err != nil {
			return nil, err
		}

		if avgDuration.Valid {
			s.AvgDurationMs = avgDuration.Float64
		}

		if s.TotalJobs > 0 {
			s.SuccessRate = float64(s.Successful) / float64(s.TotalJobs) * 100
		}

		stats = append(stats, s)
	}
	return stats, rows.Err()
}

// GetRecentFailuresFiltered returns recent failures with optional filters
func (db *Database) GetRecentFailuresFiltered(limit int, days int, workspaceIDs []string, itemTypes []string, itemNameSearch string) ([]RecentFailure, error) {
	filterClause, filterArgs := buildFilterConditions(workspaceIDs, itemTypes, itemNameSearch)

	query := fmt.Sprintf(`
		SELECT
			j.id, j.workspace_id, w.display_name as workspace_name,
			j.item_id, i.display_name as item_display_name, i.type as item_type,
			j.job_type, j.start_time, j.end_time, j.duration_ms, j.failure_reason
		FROM job_instances j
		LEFT JOIN items i ON j.item_id = i.id
		LEFT JOIN workspaces w ON j.workspace_id = w.id
		WHERE j.status = 'Failed' 
			AND j.end_time IS NOT NULL
			AND j.start_time >= CURRENT_TIMESTAMP - INTERVAL (? || ' days')
		%s
		ORDER BY j.start_time DESC
		LIMIT ?
	`, filterClause)

	args := []interface{}{fmt.Sprintf("%d", days)}
	args = append(args, filterArgs...)
	args = append(args, limit)

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var failures []RecentFailure
	for rows.Next() {
		var f RecentFailure
		var endTime sql.NullTime
		var durationMs sql.NullInt64
		var failureReason sql.NullString

		err := rows.Scan(
			&f.ID, &f.WorkspaceID, &f.WorkspaceName,
			&f.ItemID, &f.ItemDisplayName, &f.ItemType,
			&f.JobType, &f.StartTime, &endTime, &durationMs, &failureReason,
		)
		if err != nil {
			return nil, err
		}

		if endTime.Valid {
			f.EndTime = endTime.Time
		}
		if durationMs.Valid {
			f.DurationMs = durationMs.Int64
		}
		if failureReason.Valid {
			f.FailureReason = failureReason.String
		}

		failures = append(failures, f)
	}
	return failures, rows.Err()
}

// GetLongRunningJobsFiltered returns long-running jobs with optional filters
func (db *Database) GetLongRunningJobsFiltered(days int, minDeviationPct float64, limit int, workspaceIDs []string, itemTypes []string, itemNameSearch string) ([]LongRunningJob, error) {
	filterClause, filterArgs := buildFilterConditions(workspaceIDs, itemTypes, itemNameSearch)

	query := fmt.Sprintf(`
		WITH item_averages AS (
			SELECT
				j.item_id,
				AVG(j.duration_ms) as avg_duration_ms
			FROM job_instances j
			LEFT JOIN items i ON j.item_id = i.id
			WHERE j.status = 'Completed'
				AND j.duration_ms IS NOT NULL
				AND j.start_time >= CURRENT_TIMESTAMP - INTERVAL (? || ' days')
			%s
			GROUP BY j.item_id
			HAVING COUNT(*) >= 3
		)
		SELECT
			j.id, j.workspace_id, w.display_name as workspace_name,
			j.item_id, i.display_name as item_display_name, i.type as item_type,
			j.job_type, j.start_time, j.duration_ms,
			a.avg_duration_ms,
			((j.duration_ms - a.avg_duration_ms) / a.avg_duration_ms * 100) as deviation_pct
		FROM job_instances j
		INNER JOIN item_averages a ON j.item_id = a.item_id
		LEFT JOIN items i ON j.item_id = i.id
		LEFT JOIN workspaces w ON j.workspace_id = w.id
		WHERE j.status = 'Completed'
			AND j.duration_ms IS NOT NULL
			AND j.start_time >= CURRENT_TIMESTAMP - INTERVAL (? || ' days')
			AND ((j.duration_ms - a.avg_duration_ms) / a.avg_duration_ms * 100) > ?
		%s
		ORDER BY deviation_pct DESC
		LIMIT ?
	`, filterClause, filterClause)

	args := []interface{}{fmt.Sprintf("%d", days)}
	args = append(args, filterArgs...)
	args = append(args, fmt.Sprintf("%d", days))
	args = append(args, minDeviationPct)
	args = append(args, filterArgs...)
	args = append(args, limit)

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []LongRunningJob
	for rows.Next() {
		var j LongRunningJob

		err := rows.Scan(
			&j.ID, &j.WorkspaceID, &j.WorkspaceName,
			&j.ItemID, &j.ItemDisplayName, &j.ItemType,
			&j.JobType, &j.StartTime, &j.DurationMs,
			&j.AvgDurationMs, &j.DeviationPct,
		)
		if err != nil {
			return nil, err
		}

		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}
