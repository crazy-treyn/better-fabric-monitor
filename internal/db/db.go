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

	-- Create sequence for sync_metadata id
	CREATE SEQUENCE IF NOT EXISTS sync_metadata_id_seq START 1;

	-- Sync metadata
	CREATE TABLE IF NOT EXISTS sync_metadata (
		id BIGINT PRIMARY KEY DEFAULT nextval('sync_metadata_id_seq'),
		last_sync_time TIMESTAMP NOT NULL,
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
