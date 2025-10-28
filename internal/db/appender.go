package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"strings"

	"github.com/duckdb/duckdb-go/v2"
)

// getDriverConn extracts the raw driver.Conn from the database connection
// This is required to create a DuckDB appender
func getDriverConn(db *sql.DB) (driver.Conn, error) {
	conn, err := db.Conn(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}

	var driverConn driver.Conn
	err = conn.Raw(func(dc interface{}) error {
		var ok bool
		driverConn, ok = dc.(driver.Conn)
		if !ok {
			return fmt.Errorf("failed to cast to driver.Conn")
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to extract driver connection: %w", err)
	}

	return driverConn, nil
}

// executeInTransaction wraps a function in a BEGIN/COMMIT transaction using raw SQL
// This is used to ensure the appender operations are transactional
func executeInTransaction(db *sql.DB, fn func(driverConn driver.Conn) error) error {
	// Get driver connection
	driverConn, err := getDriverConn(db)
	if err != nil {
		return err
	}

	// Cast to execer context to execute raw SQL
	execer, ok := driverConn.(driver.ExecerContext)
	if !ok {
		return fmt.Errorf("connection does not support ExecerContext interface")
	}

	ctx := context.Background()

	// Begin transaction
	_, err = execer.ExecContext(ctx, "BEGIN TRANSACTION", nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Execute the function
	err = fn(driverConn)
	if err != nil {
		// Rollback on error
		_, rollbackErr := execer.ExecContext(ctx, "ROLLBACK", nil)
		if rollbackErr != nil {
			return fmt.Errorf("operation failed: %w, rollback failed: %v", err, rollbackErr)
		}
		return err
	}

	// Commit transaction
	_, err = execer.ExecContext(ctx, "COMMIT", nil)
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// bulkDeleteByIDsWithConn performs a bulk DELETE operation using a driver connection
func bulkDeleteByIDsWithConn(driverConn driver.Conn, tableName string, ids []string) error {
	return bulkDeleteByColumnWithConn(driverConn, tableName, "id", ids)
}

// bulkDeleteByColumnWithConn performs a bulk DELETE operation using a driver connection with a custom column name
func bulkDeleteByColumnWithConn(driverConn driver.Conn, tableName string, columnName string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	execer, ok := driverConn.(driver.ExecerContext)
	if !ok {
		return fmt.Errorf("connection does not support ExecerContext interface")
	}

	// Build placeholders for the IN clause
	placeholders := make([]string, len(ids))
	args := make([]driver.NamedValue, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = driver.NamedValue{Ordinal: i + 1, Value: id}
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE %s IN (%s)", tableName, columnName, strings.Join(placeholders, ","))
	_, err := execer.ExecContext(context.Background(), query, args)
	if err != nil {
		return fmt.Errorf("failed to delete existing records from %s: %w", tableName, err)
	}

	return nil
}

// bulkDeleteByIDs performs a bulk DELETE operation for records with the given IDs (deprecated - use bulkDeleteByIDsWithConn)
func bulkDeleteByIDs(tx *sql.Tx, tableName string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	// Build placeholders for the IN clause
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE id IN (%s)", tableName, strings.Join(placeholders, ","))
	_, err := tx.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to delete existing records from %s: %w", tableName, err)
	}

	return nil
}

// extractJobInstanceIDs extracts IDs from a slice of JobInstance
func extractJobInstanceIDs(jobs []JobInstance) []string {
	ids := make([]string, len(jobs))
	for i, job := range jobs {
		ids[i] = job.ID
	}
	return ids
}

// extractNotebookSessionIDs extracts IDs from a slice of NotebookSession
func extractNotebookSessionIDs(sessions []NotebookSession) []string {
	ids := make([]string, len(sessions))
	for i, session := range sessions {
		ids[i] = session.LivyID
	}
	return ids
}

// extractWorkspaceIDs extracts IDs from a slice of Workspace
func extractWorkspaceIDs(workspaces []Workspace) []string {
	ids := make([]string, len(workspaces))
	for i, ws := range workspaces {
		ids[i] = ws.ID
	}
	return ids
}

// extractItemIDs extracts IDs from a slice of Item
func extractItemIDs(items []Item) []string {
	ids := make([]string, len(items))
	for i, item := range items {
		ids[i] = item.ID
	}
	return ids
}

// appendJobInstances uses DuckDB appender for bulk insert of job instances
func appendJobInstances(driverConn driver.Conn, jobs []JobInstance) error {
	if len(jobs) == 0 {
		return nil
	}

	appender, err := duckdb.NewAppenderFromConn(driverConn, "", "job_instances")
	if err != nil {
		return fmt.Errorf("failed to create appender for job_instances: %w", err)
	}
	defer appender.Close()

	for _, job := range jobs {
		// Dereference pointers for appender - use nil for NULL values
		var endTime interface{} = nil
		if job.EndTime != nil {
			endTime = *job.EndTime
		}
		var durationMs interface{} = nil
		if job.DurationMs != nil {
			durationMs = *job.DurationMs
		}
		var failureReason interface{} = nil
		if job.FailureReason != nil {
			failureReason = *job.FailureReason
		}
		var invokerType interface{} = nil
		if job.InvokerType != nil {
			invokerType = *job.InvokerType
		}
		var rootActivityID interface{} = nil
		if job.RootActivityID != nil {
			rootActivityID = *job.RootActivityID
		}

		// Handle ActivityRuns - convert empty slice to NULL for proper enrichment later
		var activityRuns interface{} = nil
		if len(job.ActivityRuns) > 0 {
			activityRuns = job.ActivityRuns
		}

		err = appender.AppendRow(
			job.ID,
			job.WorkspaceID,
			job.ItemID,
			job.JobType,
			job.Status,
			job.StartTime,
			endTime,
			durationMs,
			failureReason,
			invokerType,
			rootActivityID,
			activityRuns,
			nil, // created_at - let DuckDB use DEFAULT CURRENT_TIMESTAMP
			nil, // updated_at - let DuckDB use DEFAULT CURRENT_TIMESTAMP
		)
		if err != nil {
			return fmt.Errorf("failed to append job instance %s: %w", job.ID, err)
		}
	}

	// Flush to ensure all rows are written
	if err := appender.Flush(); err != nil {
		return fmt.Errorf("failed to flush job instances: %w", err)
	}

	return nil
}

// appendNotebookSessions uses DuckDB appender for bulk insert of notebook sessions
func appendNotebookSessions(driverConn driver.Conn, sessions []NotebookSession) error {
	if len(sessions) == 0 {
		return nil
	}

	appender, err := duckdb.NewAppenderFromConn(driverConn, "", "notebook_sessions")
	if err != nil {
		return fmt.Errorf("failed to create appender for notebook_sessions: %w", err)
	}
	defer appender.Close()

	for _, s := range sessions {
		// Dereference all pointer fields for appender
		var sparkAppID interface{} = nil
		if s.SparkApplicationID != nil {
			sparkAppID = *s.SparkApplicationID
		}
		var origin interface{} = nil
		if s.Origin != nil {
			origin = *s.Origin
		}
		var attemptNumber interface{} = nil
		if s.AttemptNumber != nil {
			attemptNumber = *s.AttemptNumber
		}
		var livyName interface{} = nil
		if s.LivyName != nil {
			livyName = *s.LivyName
		}
		var submitterID interface{} = nil
		if s.SubmitterID != nil {
			submitterID = *s.SubmitterID
		}
		var submitterType interface{} = nil
		if s.SubmitterType != nil {
			submitterType = *s.SubmitterType
		}
		var itemName interface{} = nil
		if s.ItemName != nil {
			itemName = *s.ItemName
		}
		var itemType interface{} = nil
		if s.ItemType != nil {
			itemType = *s.ItemType
		}
		var jobType interface{} = nil
		if s.JobType != nil {
			jobType = *s.JobType
		}
		var submittedDateTime interface{} = nil
		if s.SubmittedDateTime != nil {
			submittedDateTime = *s.SubmittedDateTime
		}
		var startDateTime interface{} = nil
		if s.StartDateTime != nil {
			startDateTime = *s.StartDateTime
		}
		var endDateTime interface{} = nil
		if s.EndDateTime != nil {
			endDateTime = *s.EndDateTime
		}
		var queuedDurationMs interface{} = nil
		if s.QueuedDurationMs != nil {
			queuedDurationMs = *s.QueuedDurationMs
		}
		var runningDurationMs interface{} = nil
		if s.RunningDurationMs != nil {
			runningDurationMs = *s.RunningDurationMs
		}
		var totalDurationMs interface{} = nil
		if s.TotalDurationMs != nil {
			totalDurationMs = *s.TotalDurationMs
		}
		var cancellationReason interface{} = nil
		if s.CancellationReason != nil {
			cancellationReason = *s.CancellationReason
		}
		var capacityID interface{} = nil
		if s.CapacityID != nil {
			capacityID = *s.CapacityID
		}
		var operationName interface{} = nil
		if s.OperationName != nil {
			operationName = *s.OperationName
		}
		var consumerIdentityID interface{} = nil
		if s.ConsumerIdentityID != nil {
			consumerIdentityID = *s.ConsumerIdentityID
		}
		var runtimeVersion interface{} = nil
		if s.RuntimeVersion != nil {
			runtimeVersion = *s.RuntimeVersion
		}
		var isHighConcurrency interface{} = nil
		if s.IsHighConcurrency != nil {
			isHighConcurrency = *s.IsHighConcurrency
		}

		err = appender.AppendRow(
			s.LivyID,
			s.JobInstanceID,
			s.WorkspaceID,
			s.NotebookID,
			sparkAppID,
			s.State,
			origin,
			attemptNumber,
			livyName,
			submitterID,
			submitterType,
			itemName,
			itemType,
			jobType,
			submittedDateTime,
			startDateTime,
			endDateTime,
			queuedDurationMs,
			runningDurationMs,
			totalDurationMs,
			cancellationReason,
			capacityID,
			operationName,
			consumerIdentityID,
			runtimeVersion,
			isHighConcurrency,
			nil, // created_at - let DuckDB use DEFAULT CURRENT_TIMESTAMP
			nil, // updated_at - let DuckDB use DEFAULT CURRENT_TIMESTAMP
		)
		if err != nil {
			return fmt.Errorf("failed to append notebook session %s: %w", s.LivyID, err)
		}
	}

	// Flush to ensure all rows are written
	if err := appender.Flush(); err != nil {
		return fmt.Errorf("failed to flush notebook sessions: %w", err)
	}

	return nil
}

// appendWorkspaces uses DuckDB appender for bulk insert of workspaces
func appendWorkspaces(driverConn driver.Conn, workspaces []Workspace) error {
	if len(workspaces) == 0 {
		return nil
	}

	appender, err := duckdb.NewAppenderFromConn(driverConn, "", "workspaces")
	if err != nil {
		return fmt.Errorf("failed to create appender for workspaces: %w", err)
	}
	defer appender.Close()

	for _, ws := range workspaces {
		var description interface{} = nil
		if ws.Description != nil {
			description = *ws.Description
		}

		err = appender.AppendRow(
			ws.ID,
			ws.DisplayName,
			ws.Type,
			description,
			nil, // created_at - let DuckDB use DEFAULT CURRENT_TIMESTAMP
			nil, // updated_at - let DuckDB use DEFAULT CURRENT_TIMESTAMP
		)
		if err != nil {
			return fmt.Errorf("failed to append workspace %s: %w", ws.ID, err)
		}
	}

	// Flush to ensure all rows are written
	if err := appender.Flush(); err != nil {
		return fmt.Errorf("failed to flush workspaces: %w", err)
	}

	return nil
}

// appendItems uses DuckDB appender for bulk insert of items
func appendItems(driverConn driver.Conn, items []Item) error {
	if len(items) == 0 {
		return nil
	}

	appender, err := duckdb.NewAppenderFromConn(driverConn, "", "items")
	if err != nil {
		return fmt.Errorf("failed to create appender for items: %w", err)
	}
	defer appender.Close()

	for _, item := range items {
		var description interface{} = nil
		if item.Description != nil {
			description = *item.Description
		}

		err = appender.AppendRow(
			item.ID,
			item.WorkspaceID,
			item.DisplayName,
			item.Type,
			description,
			nil, // created_at - let DuckDB use DEFAULT CURRENT_TIMESTAMP
			nil, // updated_at - let DuckDB use DEFAULT CURRENT_TIMESTAMP
		)
		if err != nil {
			return fmt.Errorf("failed to append item %s: %w", item.ID, err)
		}
	}

	// Flush to ensure all rows are written
	if err := appender.Flush(); err != nil {
		return fmt.Errorf("failed to flush items: %w", err)
	}

	return nil
}
