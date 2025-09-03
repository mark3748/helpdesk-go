package ratelimit

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// Limiter implements a token bucket rate limiter backed by Redis.
type Limiter struct {
	rdb    *redis.Client
	limit  int           // max tokens per window
	window time.Duration // window for limit
	prefix string
}

// New returns a new Limiter. limit is the maximum number of requests per window.
// window defines the period over which limit applies. prefix namespaces keys in
// Redis so multiple limiters can coexist without interfering with each other.
func New(rdb *redis.Client, limit int, window time.Duration, prefix string) *Limiter {
	if prefix == "" {
		prefix = "rl:"
	} else if !strings.HasPrefix(prefix, "rl:") {
		prefix = "rl:" + prefix
	}
	return &Limiter{rdb: rdb, limit: limit, window: window, prefix: prefix}
}

// Allow consumes a token for the given key if available.
func (l *Limiter) Allow(ctx context.Context, key string) (bool, error) {
	if l.rdb == nil || l.limit <= 0 {
		return true, nil
	}
	now := time.Now().UnixMilli()
	interval := l.window.Milliseconds() / int64(l.limit)
	res, err := l.rdb.Eval(ctx, luaScript, []string{l.prefix + key}, l.limit, interval, now).Int()
	if err != nil {
		return false, err
	}
	return res == 1, nil
}

// Middleware returns a Gin middleware that rate limits based on the provided
// keyFunc. keyFunc should return a unique key per client (e.g., IP or user ID).
func (l *Limiter) Middleware(keyFunc func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := keyFunc(c)
		ok, err := l.Allow(c.Request.Context(), key)
		if err != nil || !ok {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limited"})
			return
		}
		c.Next()
	}
}

// luaScript implements a token bucket in Redis. It stores the remaining tokens
// and last refill timestamp in a hash per key.
const luaScript = `
local key = KEYS[1]
local capacity = tonumber(ARGV[1])
local interval = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local data = redis.call('HMGET', key, 'tokens', 'ts')
local tokens = tonumber(data[1])
local ts = tonumber(data[2])
if tokens == nil then
  tokens = capacity
  ts = now
else
  local delta = now - ts
  local add = math.floor(delta / interval)
  if add > 0 then
    tokens = math.min(tokens + add, capacity)
    ts = ts + add * interval
  end
end
local allowed = 0
if tokens > 0 then
  tokens = tokens - 1
  allowed = 1
end
redis.call('HMSET', key, 'tokens', tokens, 'ts', ts)
redis.call('PEXPIRE', key, interval * capacity)
return allowed
`
