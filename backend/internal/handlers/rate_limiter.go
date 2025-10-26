package handlers

import (
	"sync"
	"time"
)

const (
	maxAttempts     = 10
	rateLimitWindow = 5 * time.Minute
	cleanupInterval = 10 * time.Minute
)

type attemptRecord struct {
	count     int
	firstSeen time.Time
}

// RateLimiter implements in-memory rate limiting with automatic cleanup of old entries.
// It limits requests per key (e.g., email address) within a time window.
type RateLimiter struct {
	mu       sync.RWMutex
	attempts map[string]*attemptRecord
	stopChan chan struct{}
}

// NewRateLimiter creates a new rate limiter with automatic cleanup.
// The cleanup goroutine runs periodically to remove expired entries.
func NewRateLimiter() *RateLimiter {
	rl := &RateLimiter{
		attempts: make(map[string]*attemptRecord),
		stopChan: make(chan struct{}),
	}

	go rl.cleanupLoop()
	return rl
}

// Allow checks if the request from the given key is allowed.
// Returns true if allowed, false if rate limit exceeded.
// The rate limit is maxAttempts requests per rateLimitWindow.
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	record, exists := rl.attempts[key]
	if !exists {
		// First attempt for this key
		rl.attempts[key] = &attemptRecord{
			count:     1,
			firstSeen: now,
		}
		return true
	}

	// Check if the window has expired
	if now.Sub(record.firstSeen) > rateLimitWindow {
		// Reset the window
		record.count = 1
		record.firstSeen = now
		return true
	}

	// Within the window, check if limit exceeded
	if record.count >= maxAttempts {
		return false
	}

	// Increment and allow
	record.count++
	return true
}

// Reset clears the rate limit for a given key.
// Useful after successful login to prevent rate limit accumulation on valid accounts.
func (rl *RateLimiter) Reset(key string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.attempts, key)
}

// cleanupLoop periodically removes old entries to prevent memory leak
func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.cleanup()
		case <-rl.stopChan:
			return
		}
	}
}

// cleanup removes entries older than the rate limit window
func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for key, record := range rl.attempts {
		if now.Sub(record.firstSeen) > rateLimitWindow {
			delete(rl.attempts, key)
		}
	}
}

// Stop stops the cleanup loop.
// Should be called when the rate limiter is no longer needed to prevent goroutine leaks.
func (rl *RateLimiter) Stop() {
	close(rl.stopChan)
}
