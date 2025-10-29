package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/duckdb/duckdb-go/v2"
)

// Database represents the DuckDB connection and operations
type Database struct {
	conn *sql.DB
	path string
}

// NewDatabase creates or opens a DuckDB database file
func NewDatabase(path string, encryptionKey string) (*Database, error) {
	// Validate path - empty path creates in-memory database!
	if path == "" {
		return nil, fmt.Errorf("database path cannot be empty (empty path creates in-memory database)")
	}

	// Get absolute path for logging
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path // fallback to relative path
	}
	fmt.Printf("Initializing DuckDB database at: %s\n", absPath)

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Build connection string
	connStr := path
	if encryptionKey != "" {
		connStr = fmt.Sprintf("%s?access_mode=READ_WRITE&motherduck_token=%s", path, encryptionKey)
	}

	conn, err := sql.Open("duckdb", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := conn.Ping(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db := &Database{
		conn: conn,
		path: path,
	}

	// Initialize schema
	if err := db.initSchema(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return db, nil
}

// Close closes the database connection
func (db *Database) Close() error {
	if db.conn != nil {
		// Force a checkpoint to merge WAL into main database file
		// This ensures all pending writes are flushed and the .wal file is cleaned up
		_, err := db.conn.Exec("CHECKPOINT")
		if err != nil {
			// Log but don't fail - still try to close the connection
			fmt.Printf("Warning: failed to checkpoint database before close: %v\n", err)
		}
		return db.conn.Close()
	}
	return nil
}

// initSchema creates the database tables and indexes
func (db *Database) initSchema() error {
	schema := `
	-- Workspaces table
	CREATE TABLE IF NOT EXISTS workspaces (
		id VARCHAR PRIMARY KEY,
		display_name VARCHAR NOT NULL,
		type VARCHAR NOT NULL,
		description VARCHAR,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Items table (pipelines, notebooks, etc.)
	CREATE TABLE IF NOT EXISTS items (
		id VARCHAR PRIMARY KEY,
		workspace_id VARCHAR NOT NULL REFERENCES workspaces(id),
		display_name VARCHAR NOT NULL,
		type VARCHAR NOT NULL,
		description VARCHAR,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Job instances table
	CREATE TABLE IF NOT EXISTS job_instances (
		id VARCHAR PRIMARY KEY,
		workspace_id VARCHAR NOT NULL REFERENCES workspaces(id),
		item_id VARCHAR NOT NULL REFERENCES items(id),
		job_type VARCHAR NOT NULL,
		status VARCHAR NOT NULL,
		start_time TIMESTAMP NOT NULL,
		end_time TIMESTAMP,
		duration_ms BIGINT,
		failure_reason VARCHAR,
		invoker_type VARCHAR,
		root_activity_id VARCHAR,
		activity_runs JSON,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Create sequence for sync_metadata id
	CREATE SEQUENCE IF NOT EXISTS sync_metadata_id_seq START 1;

	-- Notebook sessions table (Livy sessions)
	CREATE TABLE IF NOT EXISTS notebook_sessions (
		livy_id VARCHAR PRIMARY KEY,
		job_instance_id VARCHAR NOT NULL,
		workspace_id VARCHAR NOT NULL,
		notebook_id VARCHAR NOT NULL,
		spark_application_id VARCHAR,
		state VARCHAR NOT NULL,
		origin VARCHAR,
		attempt_number INTEGER,
		livy_name VARCHAR,
		submitter_id VARCHAR,
		submitter_type VARCHAR,
		item_name VARCHAR,
		item_type VARCHAR,
		job_type VARCHAR,
		submitted_datetime TIMESTAMP,
		start_datetime TIMESTAMP,
		end_datetime TIMESTAMP,
		queued_duration_ms INTEGER,
		running_duration_ms INTEGER,
		total_duration_ms INTEGER,
		cancellation_reason VARCHAR,
		capacity_id VARCHAR,
		operation_name VARCHAR,
		consumer_identity_id VARCHAR,
		runtime_version VARCHAR,
		is_high_concurrency BOOLEAN,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Sync metadata
	CREATE TABLE IF NOT EXISTS sync_metadata (
		id BIGINT PRIMARY KEY DEFAULT nextval('sync_metadata_id_seq'),
		last_sync_time TIMESTAMPTZ NOT NULL,
		sync_type VARCHAR NOT NULL,
		records_synced INTEGER NOT NULL,
		errors INTEGER DEFAULT 0,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	`

	_, err := db.conn.Exec(schema)
	return err
}

// GetConnection returns the underlying database connection
func (db *Database) GetConnection() *sql.DB {
	return db.conn
}
