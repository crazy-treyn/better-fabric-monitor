package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"better-fabric-monitor/internal/logger"
)

// ExportTablesToParquet exports all tables to Parquet files
func (db *Database) ExportTablesToParquet(parquetPath string) ([]ParquetExportStats, error) {
	// Get absolute path for Parquet files
	absParquetPath, err := filepath.Abs(parquetPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute parquet path: %w", err)
	}

	// Ensure Parquet directory exists
	if err := os.MkdirAll(absParquetPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create parquet directory: %w", err)
	}

	tables := []string{"workspaces", "items", "job_instances", "notebook_sessions", "sync_metadata"}
	stats := make([]ParquetExportStats, 0, len(tables))

	for _, tableName := range tables {
		start := time.Now()
		stat := ParquetExportStats{
			TableName: tableName,
			Success:   false,
		}

		// Build Parquet file path
		parquetFile := filepath.Join(absParquetPath, fmt.Sprintf("%s.parquet", tableName))

		// Delete existing Parquet file if it exists
		if err := os.Remove(parquetFile); err != nil && !os.IsNotExist(err) {
			stat.ErrorMessage = fmt.Sprintf("failed to delete existing parquet file: %v", err)
			stat.DurationMs = time.Since(start).Milliseconds()
			stats = append(stats, stat)
			logger.Log("[PARQUET] ERROR: Failed to delete existing %s.parquet: %v\n", tableName, err)
			continue
		}

		// Get record count
		var count int
		err := db.conn.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)).Scan(&count)
		if err != nil {
			stat.ErrorMessage = fmt.Sprintf("failed to count records: %v", err)
			stat.DurationMs = time.Since(start).Milliseconds()
			stats = append(stats, stat)
			logger.Log("[PARQUET] ERROR: Failed to count records in %s: %v\n", tableName, err)
			continue
		}
		stat.RecordCount = count

		// Export to Parquet
		query := fmt.Sprintf("COPY (SELECT * FROM %s) TO '%s' (FORMAT PARQUET)", tableName, parquetFile)
		_, err = db.conn.Exec(query)
		if err != nil {
			stat.ErrorMessage = fmt.Sprintf("failed to export: %v", err)
			stat.DurationMs = time.Since(start).Milliseconds()
			stats = append(stats, stat)
			logger.Log("[PARQUET] ERROR: Failed to export %s: %v\n", tableName, err)
			continue
		}

		stat.Success = true
		stat.DurationMs = time.Since(start).Milliseconds()
		stats = append(stats, stat)
		logger.Log("[PARQUET] Exported %s: %d records in %dms\n", tableName, count, stat.DurationMs)
	}

	return stats, nil
}

// CreateReadOnlyDatabase creates a read-only replica database with views to Parquet files
func CreateReadOnlyDatabase(readOnlyPath, parquetPath string) error {
	// Get absolute paths
	absReadOnlyPath, err := filepath.Abs(readOnlyPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute readonly path: %w", err)
	}

	absParquetPath, err := filepath.Abs(parquetPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute parquet path: %w", err)
	}

	// Check if read-only database already exists
	if _, err := os.Stat(absReadOnlyPath); err == nil {
		// Database already exists, no need to recreate views
		logger.Log("[PARQUET] Read-only database already exists at: %s\n", absReadOnlyPath)
		return nil
	}

	// Ensure directory exists
	dir := filepath.Dir(absReadOnlyPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create readonly database directory: %w", err)
	}

	logger.Log("[PARQUET] Creating read-only database at: %s\n", absReadOnlyPath)

	// Open connection to create read-only database
	conn, err := sql.Open("duckdb", absReadOnlyPath)
	if err != nil {
		return fmt.Errorf("failed to create readonly database: %w", err)
	}
	defer conn.Close()

	// Create views for each table
	tables := []string{"workspaces", "items", "job_instances", "notebook_sessions", "sync_metadata"}

	for _, tableName := range tables {
		parquetFile := filepath.Join(absParquetPath, fmt.Sprintf("%s.parquet", tableName))

		// Verify Parquet file exists
		if _, err := os.Stat(parquetFile); os.IsNotExist(err) {
			return fmt.Errorf("parquet file not found for table %s: %s", tableName, parquetFile)
		}

		// Create view that reads from Parquet file
		query := fmt.Sprintf("CREATE VIEW %s AS SELECT * FROM read_parquet('%s')", tableName, parquetFile)
		_, err := conn.Exec(query)
		if err != nil {
			return fmt.Errorf("failed to create view for %s: %w", tableName, err)
		}

		logger.Log("[PARQUET] Created view for %s\n", tableName)
	}

	logger.Log("[PARQUET] Read-only database created successfully\n")
	return nil
}
