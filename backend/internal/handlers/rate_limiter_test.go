package handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRateLimiter_Allow(t *testing.T) {
	rl := NewRateLimiter()
	defer rl.Stop()

	key := "test@example.com"

	// First 10 attempts should be allowed
	for i := 0; i < 10; i++ {
		assert.True(t, rl.Allow(key), "Attempt %d should be allowed", i+1)
	}

	// 11th attempt should be blocked
	assert.False(t, rl.Allow(key), "11th attempt should be blocked")
}

func TestRateLimiter_DifferentKeys(t *testing.T) {
	rl := NewRateLimiter()
	defer rl.Stop()

	key1 := "user1@example.com"
	key2 := "user2@example.com"

	// Use up attempts for key1
	for i := 0; i < 10; i++ {
		rl.Allow(key1)
	}

	// key1 should be blocked
	assert.False(t, rl.Allow(key1), "key1 should be blocked")

	// key2 should still be allowed
	assert.True(t, rl.Allow(key2), "key2 should be allowed")
}

func TestRateLimiter_Reset(t *testing.T) {
	rl := NewRateLimiter()
	defer rl.Stop()

	key := "test@example.com"

	// Use up some attempts
	for i := 0; i < 5; i++ {
		rl.Allow(key)
	}

	// Reset
	rl.Reset(key)

	// Should be able to make full 10 attempts again
	for i := 0; i < 10; i++ {
		assert.True(t, rl.Allow(key), "Attempt %d should be allowed after reset", i+1)
	}
}

func TestRateLimiter_Cleanup(t *testing.T) {
	rl := NewRateLimiter()
	defer rl.Stop()

	// Add some entries
	rl.Allow("user1@example.com")
	rl.Allow("user2@example.com")
	rl.Allow("user3@example.com")

	// Manually trigger cleanup (in real usage, this runs automatically)
	rl.cleanup()

	// Verify cleanup doesn't affect current entries
	// (They should still be there as they're recent)
	assert.True(t, rl.Allow("user1@example.com"), "Recent entry should not be cleaned up")
}

func TestRateLimiter_MultipleKeys(t *testing.T) {
	rl := NewRateLimiter()
	defer rl.Stop()

	keys := []string{
		"user1@example.com",
		"user2@example.com",
		"user3@example.com",
	}

	// Each key should get its own 10 attempts
	for _, key := range keys {
		for i := 0; i < 10; i++ {
			assert.True(t, rl.Allow(key), "Key %s attempt %d should be allowed", key, i+1)
		}
		assert.False(t, rl.Allow(key), "Key %s 11th attempt should be blocked", key)
	}
}

func TestRateLimiter_ResetSpecificKey(t *testing.T) {
	rl := NewRateLimiter()
	defer rl.Stop()

	key1 := "user1@example.com"
	key2 := "user2@example.com"

	// Use up attempts for both keys
	for i := 0; i < 10; i++ {
		rl.Allow(key1)
		rl.Allow(key2)
	}

	// Both should be blocked
	assert.False(t, rl.Allow(key1))
	assert.False(t, rl.Allow(key2))

	// Reset only key1
	rl.Reset(key1)

	// key1 should be allowed, key2 still blocked
	assert.True(t, rl.Allow(key1), "key1 should be allowed after reset")
	assert.False(t, rl.Allow(key2), "key2 should still be blocked")
}
