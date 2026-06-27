package handler

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// Lua script for run Token Bucket algorithm in Redis
const tokenBucketScript = `
local key = KEYS[1]
local capacity = tonumber(ARGV[1])
local refill_rate = tonumber(ARGV[2]) -- tokens per millisecond
local now = tonumber(ARGV[3]) -- current time in ms
local cost = tonumber(ARGV[4] or 1)
-- Fetch current bucket state from Hash
local bucket = redis.call('HMGET', key, 'tokens', 'last_updated')
local tokens = tonumber(bucket[1])
local last_updated = tonumber(bucket[2])
if not tokens then
    -- First request, fill the bucket
    tokens = capacity
    last_updated = now
else
    -- Calculate how many tokens should be refilled
    local elapsed = now - last_updated
    if elapsed > 0 then
        local refill = elapsed * refill_rate
        tokens = math.min(capacity, tokens + refill)
        last_updated = now
    end
end
-- Allow request if there are enough tokens
if tokens >= cost then
    tokens = tokens - cost
    redis.call('HMSET', key, 'tokens', tokens, 'last_updated', last_updated)
    redis.call('EXPIRE', key, 3600) -- expire key after 1 hour of inactivity
    return 1 -- Allowed
else
    redis.call('HMSET', key, 'tokens', tokens, 'last_updated', last_updated)
    redis.call('EXPIRE', key, 3600)
    return 0 -- Blocked
end
`

func RateLimiter(rdb *redis.Client, keyPrefix string, capacity int, refillPeriod time.Duration) gin.HandlerFunc {
	refillRate := float64(capacity) / float64(refillPeriod.Milliseconds())
	return func(c *gin.Context) {
		ip := c.ClientIP()
		key := fmt.Sprintf("rate_limit:%s:%s", keyPrefix, ip)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		nowMs := time.Now().UnixNano() / int64(time.Millisecond)
		res, err := rdb.Eval(ctx, tokenBucketScript, []string{key}, capacity, refillRate, nowMs, 1).Result()
		if err != nil {
			log.Printf("Rate limiter error (fail-open): %v", err)
			c.Next()
			return
		}
		allowed, ok := res.(int64)
		if !ok || allowed != 1 {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "too many requests. Please try again later.",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

