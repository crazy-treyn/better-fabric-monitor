package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/marcboeker/go-duckdb"
)

// Database represents the DuckDB connection and operations
type Database struct {
	conn *sql.DB
	path string
}

// NewDatabase creates or opens a DuckDB database file
func NewDatabase(path string, encryptionKey string) (*Database, error) {
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
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Pipeline runs table (if needed separately)
	CREATE TABLE IF NOT EXISTS pipeline_runs (
		run_id VARCHAR PRIMARY KEY,
		workspace_id VARCHAR NOT NULL REFERENCES workspaces(id),
		pipeline_id VARCHAR NOT NULL REFERENCES items(id),
		pipeline_name VARCHAR NOT NULL,
		status VARCHAR NOT NULL,
		start_time TIMESTAMP NOT NULL,
		end_time TIMESTAMP,
		duration_ms BIGINT,
		error_message VARCHAR,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Sync metadata
	CREATE TABLE IF NOT EXISTS sync_metadata (
		id INTEGER PRIMARY KEY,
		last_sync_time TIMESTAMP NOT NULL,
		sync_type VARCHAR NOT NULL,
		records_synced INTEGER NOT NULL,
		errors INTEGER DEFAULT 0,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Indexes for performance
	CREATE INDEX IF NOT EXISTS idx_workspaces_type ON workspaces(type);
	CREATE INDEX IF NOT EXISTS idx_items_workspace ON items(workspace_id);
	CREATE INDEX IF NOT EXISTS idx_items_type ON items(type);
	CREATE INDEX IF NOT EXISTS idx_job_instances_workspace ON job_instances(workspace_id);
	CREATE INDEX IF NOT EXISTS idx_job_instances_item ON job_instances(item_id);
	CREATE INDEX IF NOT EXISTS idx_job_instances_status ON job_instances(status);
	CREATE INDEX IF NOT EXISTS idx_job_instances_start_time ON job_instances(start_time);
	CREATE INDEX IF NOT EXISTS idx_job_instances_composite ON job_instances(workspace_id, start_time, status);
	CREATE INDEX IF NOT EXISTS idx_pipeline_runs_workspace ON pipeline_runs(workspace_id);
	CREATE INDEX IF NOT EXISTS idx_pipeline_runs_pipeline ON pipeline_runs(pipeline_id);
	CREATE INDEX IF NOT EXISTS idx_pipeline_runs_start_time ON pipeline_runs(start_time);
	`

	_, err := db.conn.Exec(schema)
	return err
}

// GetConnection returns the underlying database connection
func (db *Database) GetConnection() *sql.DB {
	return db.conn
}
