package handlers

import (
	"sync"
	"time"
)

type RateLimiter struct {
	mu       sync.RWMutex
	limits   map[string]*clientLimit
	freePlan struct {
		dailyLimit   int
		webhookLimit int
	}
	premiumPlan struct {
		webhookLimit int
	}
}

type clientLimit struct {
	dailyCount   int
	lastReset    time.Time
	webhookCount int
	isPremium    bool
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		limits: make(map[string]*clientLimit),
		freePlan: struct {
			dailyLimit   int
			webhookLimit int
		}{
			dailyLimit:   10000, // 10k events per day
			webhookLimit: 20,    // 20 webhooks
		},
		premiumPlan: struct {
			webhookLimit int
		}{
			webhookLimit: 50, // 50 webhooks
		},
	}
}

func (rl *RateLimiter) AllowRequest(clientID string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limit, exists := rl.limits[clientID]
	if !exists {
		limit = &clientLimit{
			lastReset: time.Now().UTC(),
		}
		rl.limits[clientID] = limit
	}

	// Reset daily count if it's a new day
	now := time.Now().UTC()
	if now.Sub(limit.lastReset) >= 24*time.Hour {
		limit.dailyCount = 0
		limit.lastReset = now
	}

	// Check limits based on plan
	if limit.isPremium {
		if limit.webhookCount >= rl.premiumPlan.webhookLimit {
			return false
		}
		// Premium has unlimited daily events
		limit.dailyCount++
		return true
	}

	// Free plan limits
	if limit.webhookCount >= rl.freePlan.webhookLimit {
		return false
	}
	if limit.dailyCount >= rl.freePlan.dailyLimit {
		return false
	}

	limit.dailyCount++
	return true
}
