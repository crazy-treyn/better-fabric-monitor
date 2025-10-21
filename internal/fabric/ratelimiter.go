package fabric

import (
	"sync"
	"time"
)

const (
	// Rate limiting defaults
	InitialRPS          = 50  // Start conservatively at 50 requests/second
	MinRPS              = 10  // Minimum when throttled
	MaxRPS              = 100 // Maximum during good conditions
	ThrottleCooldown    = 60 * time.Second
	RPSIncreaseInterval = 30 * time.Second
	RPSIncreaseRate     = 0.20 // 20% increase
	RPSDecreaseRate     = 0.50 // 50% decrease on throttle
)

// AdaptiveRateLimiter implements a token bucket rate limiter with adaptive throttling
type AdaptiveRateLimiter struct {
	mu               sync.Mutex
	currentRPS       int
	minRPS           int
	maxRPS           int
	tokens           chan struct{}
	throttleDetected bool
	lastThrottleTime time.Time
	lastIncreaseTime time.Time
	stopChan         chan struct{}
}

// NewAdaptiveRateLimiter creates a new adaptive rate limiter
func NewAdaptiveRateLimiter() *AdaptiveRateLimiter {
	rl := &AdaptiveRateLimiter{
		currentRPS:       InitialRPS,
		minRPS:           MinRPS,
		maxRPS:           MaxRPS,
		tokens:           make(chan struct{}, InitialRPS),
		lastIncreaseTime: time.Now(),
		stopChan:         make(chan struct{}),
	}

	// Start token refill goroutine
	go rl.refillTokens()

	// Start adaptive adjustment goroutine
	go rl.adaptiveAdjust()

	return rl
}

// Wait blocks until a token is available
func (rl *AdaptiveRateLimiter) Wait() {
	<-rl.tokens
}

// refillTokens continuously refills the token bucket
func (rl *AdaptiveRateLimiter) refillTokens() {
	ticker := time.NewTicker(time.Second / time.Duration(rl.currentRPS))
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.mu.Lock()
			currentRPS := rl.currentRPS
			rl.mu.Unlock()

			// Update ticker if RPS changed
			ticker.Reset(time.Second / time.Duration(currentRPS))

			// Try to add a token (non-blocking)
			select {
			case rl.tokens <- struct{}{}:
			default:
				// Token bucket is full, skip
			}
		case <-rl.stopChan:
			return
		}
	}
}

// adaptiveAdjust periodically adjusts the rate based on conditions
func (rl *AdaptiveRateLimiter) adaptiveAdjust() {
	ticker := time.NewTicker(RPSIncreaseInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.mu.Lock()

			// If we detected throttling recently, wait for cooldown
			if rl.throttleDetected && time.Since(rl.lastThrottleTime) < ThrottleCooldown {
				rl.mu.Unlock()
				continue
			}

			// Clear throttle flag after cooldown
			if rl.throttleDetected && time.Since(rl.lastThrottleTime) >= ThrottleCooldown {
				rl.throttleDetected = false
			}

			// Gradually increase RPS if no throttling and enough time passed
			if !rl.throttleDetected && time.Since(rl.lastIncreaseTime) >= RPSIncreaseInterval {
				newRPS := int(float64(rl.currentRPS) * (1 + RPSIncreaseRate))
				if newRPS > rl.maxRPS {
					newRPS = rl.maxRPS
				}
				if newRPS != rl.currentRPS {
					rl.currentRPS = newRPS
					rl.lastIncreaseTime = time.Now()
					// Recreate token channel with new capacity
					close(rl.tokens)
					rl.tokens = make(chan struct{}, newRPS)
				}
			}

			rl.mu.Unlock()

		case <-rl.stopChan:
			return
		}
	}
}

// OnThrottle should be called when a 429 response is detected
func (rl *AdaptiveRateLimiter) OnThrottle() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.throttleDetected = true
	rl.lastThrottleTime = time.Now()

	// Reduce RPS by 50%
	newRPS := int(float64(rl.currentRPS) * (1 - RPSDecreaseRate))
	if newRPS < rl.minRPS {
		newRPS = rl.minRPS
	}

	if newRPS != rl.currentRPS {
		rl.currentRPS = newRPS
		// Recreate token channel with new capacity
		oldTokens := rl.tokens
		rl.tokens = make(chan struct{}, newRPS)
		close(oldTokens)
	}
}

// GetCurrentRPS returns the current requests per second setting
func (rl *AdaptiveRateLimiter) GetCurrentRPS() int {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return rl.currentRPS
}

// Stop stops the rate limiter goroutines
func (rl *AdaptiveRateLimiter) Stop() {
	close(rl.stopChan)
}
