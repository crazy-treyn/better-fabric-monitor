package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Logger provides simple file logging for API operations
type Logger struct {
	filePath string
	mu       sync.Mutex
}

// APICallLog represents a single API call event
type APICallLog struct {
	Timestamp   time.Time
	Endpoint    string
	WorkspaceID string
	ItemID      string
	Duration    time.Duration
	StatusCode  int
	Throttled   bool
	RetryCount  int
	Error       string
}

// NewLogger creates a new logger that writes to the specified file
func NewLogger(filePath string) (*Logger, error) {
	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	return &Logger{
		filePath: filePath,
	}, nil
}

// LogAPICall logs an API call event
func (l *Logger) LogAPICall(log APICallLog) {
	l.mu.Lock()
	defer l.mu.Unlock()

	f, err := os.OpenFile(l.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Failed to open log file: %v\n", err)
		return
	}
	defer f.Close()

	logLine := fmt.Sprintf("%s | %s | WS:%s | Item:%s | %dms | Status:%d | Throttled:%v | Retries:%d",
		log.Timestamp.Format(time.RFC3339),
		log.Endpoint,
		log.WorkspaceID,
		log.ItemID,
		log.Duration.Milliseconds(),
		log.StatusCode,
		log.Throttled,
		log.RetryCount,
	)

	if log.Error != "" {
		logLine += fmt.Sprintf(" | Error:%s", log.Error)
	}

	logLine += "\n"

	if _, err := f.WriteString(logLine); err != nil {
		fmt.Printf("Failed to write to log file: %v\n", err)
	}
}

// LogError logs an error event
func (l *Logger) LogError(endpoint, workspaceID, itemID, errorMsg string) {
	l.LogAPICall(APICallLog{
		Timestamp:   time.Now(),
		Endpoint:    endpoint,
		WorkspaceID: workspaceID,
		ItemID:      itemID,
		Error:       errorMsg,
	})
}

// LogThrottle logs a throttle event
func (l *Logger) LogThrottle(endpoint, workspaceID string, retryAfter time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	f, err := os.OpenFile(l.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Failed to open log file: %v\n", err)
		return
	}
	defer f.Close()

	logLine := fmt.Sprintf("%s | THROTTLE | %s | WS:%s | RetryAfter:%v\n",
		time.Now().Format(time.RFC3339),
		endpoint,
		workspaceID,
		retryAfter,
	)

	if _, err := f.WriteString(logLine); err != nil {
		fmt.Printf("Failed to write to log file: %v\n", err)
	}
}
