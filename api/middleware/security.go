package middleware

import (
	"net/http"
	"strings"
	"time"

	"webhook-processor/pkg/metrics"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type SecurityMiddleware struct {
	logger       *zap.Logger
	apiKeys      map[string]string // clientID -> apiKey
	apiKeyHeader string
}

func NewSecurityMiddleware(logger *zap.Logger, apiKeys map[string]string, apiKeyHeader string) *SecurityMiddleware {
	return &SecurityMiddleware{
		logger:       logger,
		apiKeys:      apiKeys,
		apiKeyHeader: apiKeyHeader,
	}
}

func (m *SecurityMiddleware) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader(m.apiKeyHeader)
		if apiKey == "" {
			m.logger.Warn("Missing API key", zap.String("ip", c.ClientIP()))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing API key"})
			c.Abort()
			return
		}

		clientID := m.validateAPIKey(apiKey)
		if clientID == "" {
			prefixLen := len(apiKey)
			if prefixLen > 8 {
				prefixLen = 8
			}
			m.logger.Warn("Invalid API key", zap.String("ip", c.ClientIP()), zap.String("api_key_prefix", apiKey[:prefixLen]))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key"})
			c.Abort()
			return
		}

		// Set client ID for later use
		c.Set("clientID", clientID)
		m.logger.Debug("Successfully authenticated client", zap.String("client_id", clientID))
		c.Next()
	}
}

func (m *SecurityMiddleware) CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, "+m.apiKeyHeader)
		c.Header("Access-Control-Max-Age", "3600")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func (m *SecurityMiddleware) RateLimit() gin.HandlerFunc {
	// Simple token bucket implementation
	type bucket struct {
		tokens     float64
		lastRefill time.Time
	}

	buckets := make(map[string]*bucket)

	return func(c *gin.Context) {
		clientID, exists := c.Get("clientID")
		if !exists {
			c.Next()
			return
		}

		id := clientID.(string)
		b, exists := buckets[id]
		if !exists {
			b = &bucket{
				tokens:     10, // Initial tokens
				lastRefill: time.Now(),
			}
			buckets[id] = b
		}

		// Refill tokens
		now := time.Now()
		duration := now.Sub(b.lastRefill).Seconds()
		maxTokens := 10.0
		if b.tokens+duration > maxTokens {
			b.tokens = maxTokens
		} else {
			b.tokens += duration
		}
		b.lastRefill = now

		if b.tokens < 1 {
			metrics.RateLimitExceeded.WithLabelValues(id, "request_rate").Inc()
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Rate limit exceeded"})
			c.Abort()
			return
		}

		b.tokens--
		c.Next()
	}
}

func (m *SecurityMiddleware) ValidatePayload() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Validate content type
		if !strings.HasPrefix(c.GetHeader("Content-Type"), "application/json") {
			c.JSON(http.StatusUnsupportedMediaType, gin.H{"error": "Content-Type must be application/json"})
			c.Abort()
			return
		}

		// Continue only if there's a body
		if c.Request.Body == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Empty request body"})
			c.Abort()
			return
		}

		c.Next()
	}
}

func (m *SecurityMiddleware) validateAPIKey(apiKey string) string {
	// Find client ID by API key
	for clientID, key := range m.apiKeys {
		if key == apiKey {
			return clientID
		}
	}
	return ""
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
