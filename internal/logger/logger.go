package logger

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// LogEntry represents a single log entry
type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Message   string `json:"message"`
}

// LogBuffer stores recent log entries in a circular buffer
type LogBuffer struct {
	entries []LogEntry
	maxSize int
	index   int
	mutex   sync.RWMutex
}

// NewLogBuffer creates a new log buffer with specified size
func NewLogBuffer(maxSize int) *LogBuffer {
	return &LogBuffer{
		entries: make([]LogEntry, 0, maxSize),
		maxSize: maxSize,
		index:   0,
	}
}

// Add adds a log entry to the buffer
func (lb *LogBuffer) Add(level, message string) {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339Nano),
		Level:     level,
		Message:   message,
	}

	if len(lb.entries) < lb.maxSize {
		lb.entries = append(lb.entries, entry)
	} else {
		// Circular buffer - overwrite oldest entry
		lb.entries[lb.index] = entry
		lb.index = (lb.index + 1) % lb.maxSize
	}
}

// GetAll returns all log entries in chronological order
func (lb *LogBuffer) GetAll() []LogEntry {
	lb.mutex.RLock()
	defer lb.mutex.RUnlock()

	if len(lb.entries) < lb.maxSize {
		// Buffer not full yet, return in order
		result := make([]LogEntry, len(lb.entries))
		copy(result, lb.entries)
		return result
	}

	// Buffer is full, need to reorder starting from oldest
	result := make([]LogEntry, lb.maxSize)
	for i := 0; i < lb.maxSize; i++ {
		result[i] = lb.entries[(lb.index+i)%lb.maxSize]
	}
	return result
}

// Clear removes all log entries
func (lb *LogBuffer) Clear() {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()
	lb.entries = make([]LogEntry, 0, lb.maxSize)
	lb.index = 0
}

// Global log buffer
var globalBuffer *LogBuffer

// Init initializes the global log buffer
func Init(maxSize int) {
	globalBuffer = NewLogBuffer(maxSize)
}

// Log adds a log message to the buffer and prints to console
func Log(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)

	// Print to console as before
	fmt.Print(message)

	// Detect log level from content
	level := detectLogLevel(message)

	// Add to buffer
	if globalBuffer != nil {
		globalBuffer.Add(level, strings.TrimSpace(message))
	}
}

// detectLogLevel determines the log level based on message content
func detectLogLevel(message string) string {
	lower := strings.ToLower(message)
	if strings.Contains(lower, "error") || strings.Contains(lower, "failed") {
		return "ERROR"
	}
	if strings.Contains(lower, "warning") || strings.Contains(lower, "warn") {
		return "WARNING"
	}
	if strings.Contains(lower, "debug:") {
		return "DEBUG"
	}
	return "INFO"
}

// GetAll returns all log entries from the global buffer
func GetAll() []LogEntry {
	if globalBuffer == nil {
		return []LogEntry{}
	}
	return globalBuffer.GetAll()
}

// Clear clears all log entries from the global buffer
func Clear() {
	if globalBuffer != nil {
		globalBuffer.Clear()
	}
}
