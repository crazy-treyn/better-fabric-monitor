package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// SaveWorkspace saves or updates a workspace
func (db *Database) SaveWorkspace(workspace *Workspace) error {
	query := `
		INSERT INTO workspaces (id, display_name, type, description, updated_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
			display_name = EXCLUDED.display_name,
			type = EXCLUDED.type,
			description = EXCLUDED.description,
			updated_at = CURRENT_TIMESTAMP
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
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
			display_name = EXCLUDED.display_name,
			type = EXCLUDED.type,
			description = EXCLUDED.description,
			updated_at = CURRENT_TIMESTAMP
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
			end_time, duration_ms, failure_reason, invoker_type, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
			status = EXCLUDED.status,
			end_time = EXCLUDED.end_time,
			duration_ms = EXCLUDED.duration_ms,
			failure_reason = EXCLUDED.failure_reason,
			updated_at = CURRENT_TIMESTAMP
	`

	stmt, err := tx.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, job := range jobs {
		_, err = stmt.Exec(
			job.ID, job.WorkspaceID, job.ItemID, job.JobType, job.Status, job.StartTime,
			job.EndTime, job.DurationMs, job.FailureReason, job.InvokerType,
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
		conditions = append(conditions, "workspace_id = ?")
		args = append(args, *filter.WorkspaceID)
	}

	if filter.ItemID != nil {
		conditions = append(conditions, "item_id = ?")
		args = append(args, *filter.ItemID)
	}

	if filter.JobType != nil {
		conditions = append(conditions, "job_type = ?")
		args = append(args, *filter.JobType)
	}

	if filter.Status != nil {
		conditions = append(conditions, "status = ?")
		args = append(args, *filter.Status)
	}

	if filter.StartDateFrom != nil {
		conditions = append(conditions, "start_time >= ?")
		args = append(args, *filter.StartDateFrom)
	}

	if filter.StartDateTo != nil {
		conditions = append(conditions, "start_time <= ?")
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
		SELECT id, workspace_id, item_id, job_type, status, start_time,
			   end_time, duration_ms, failure_reason, invoker_type, created_at, updated_at
		FROM job_instances
		%s
		ORDER BY start_time DESC
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
		err := rows.Scan(
			&job.ID, &job.WorkspaceID, &job.ItemID, &job.JobType, &job.Status, &job.StartTime,
			&job.EndTime, &job.DurationMs, &job.FailureReason, &job.InvokerType, &job.CreatedAt, &job.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

// GetJobStats returns aggregated statistics
func (db *Database) GetJobStats(workspaceID string, from, to time.Time) (*JobStats, error) {
	query := `
		SELECT
			COUNT(*) as total_jobs,
			SUM(CASE WHEN status = 'Completed' THEN 1 ELSE 0 END) as successful,
			SUM(CASE WHEN status = 'Failed' THEN 1 ELSE 0 END) as failed,
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
		VALUES (CURRENT_TIMESTAMP, ?, ?, ?)
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
