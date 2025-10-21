package fabric

import (
	"fmt"
	"net/http"
	"strconv"
	"time"
)

const (
	// Retry policy defaults
	MaxRetries        = 5
	InitialBackoff    = 1 * time.Second
	MaxBackoff        = 32 * time.Second
	BackoffMultiplier = 2.0
)

// RetryPolicy defines retry behavior
type RetryPolicy struct {
	MaxRetries int
	BaseDelay  time.Duration
	MaxDelay   time.Duration
	Multiplier float64
}

// NewRetryPolicy creates a default retry policy
func NewRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxRetries: MaxRetries,
		BaseDelay:  InitialBackoff,
		MaxDelay:   MaxBackoff,
		Multiplier: BackoffMultiplier,
	}
}

// ShouldRetry determines if an error/status code should be retried
func (rp *RetryPolicy) ShouldRetry(statusCode int, attempt int) bool {
	if attempt >= rp.MaxRetries {
		return false
	}

	// Retry on these status codes
	switch statusCode {
	case http.StatusTooManyRequests, // 429
		http.StatusInternalServerError, // 500
		http.StatusBadGateway,          // 502
		http.StatusServiceUnavailable,  // 503
		http.StatusGatewayTimeout:      // 504
		return true
	default:
		return false
	}
}

// GetBackoffDuration calculates the backoff duration for a given attempt
// Respects Retry-After header if provided, otherwise uses exponential backoff
func (rp *RetryPolicy) GetBackoffDuration(attempt int, resp *http.Response) time.Duration {
	// Check for Retry-After header (takes precedence)
	if resp != nil {
		if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
			// Try to parse as seconds
			if seconds, err := strconv.Atoi(retryAfter); err == nil {
				duration := time.Duration(seconds) * time.Second
				if duration > rp.MaxDelay {
					return rp.MaxDelay
				}
				return duration
			}
			// Try to parse as HTTP date
			if retryTime, err := http.ParseTime(retryAfter); err == nil {
				duration := time.Until(retryTime)
				if duration < 0 {
					duration = rp.BaseDelay
				}
				if duration > rp.MaxDelay {
					return rp.MaxDelay
				}
				return duration
			}
		}
	}

	// Use exponential backoff
	backoff := float64(rp.BaseDelay)
	for i := 0; i < attempt; i++ {
		backoff *= rp.Multiplier
	}

	duration := time.Duration(backoff)
	if duration > rp.MaxDelay {
		duration = rp.MaxDelay
	}

	return duration
}

// ExecuteWithRetry executes a function with retry logic
func (rp *RetryPolicy) ExecuteWithRetry(fn func() (*http.Response, error), onThrottle func()) (*http.Response, error) {
	var resp *http.Response
	var err error

	for attempt := 0; attempt <= rp.MaxRetries; attempt++ {
		resp, err = fn()

		// Success case
		if err == nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return resp, nil
		}

		// Check if we should retry
		if resp != nil {
			if !rp.ShouldRetry(resp.StatusCode, attempt) {
				return resp, err
			}

			// Notify on throttle (429)
			if resp.StatusCode == http.StatusTooManyRequests && onThrottle != nil {
				onThrottle()
			}

			// Calculate backoff
			backoff := rp.GetBackoffDuration(attempt, resp)

			// Log retry attempt
			fmt.Printf("Retry attempt %d/%d after %v (status: %d)\n",
				attempt+1, rp.MaxRetries, backoff, resp.StatusCode)

			// Close the response body before retrying
			if resp.Body != nil {
				resp.Body.Close()
			}

			// Wait before retrying
			if attempt < rp.MaxRetries {
				time.Sleep(backoff)
			}
		} else if err != nil {
			// Network error or other error
			if attempt < rp.MaxRetries {
				backoff := rp.GetBackoffDuration(attempt, nil)
				fmt.Printf("Retry attempt %d/%d after %v (error: %v)\n",
					attempt+1, rp.MaxRetries, backoff, err)
				time.Sleep(backoff)
			}
		}
	}

	return resp, fmt.Errorf("max retries exceeded: %w", err)
}
