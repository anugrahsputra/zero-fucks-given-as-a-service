package main

import (
	"bufio"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type DontCare struct {
	Reason string `json:"reason"`
}

type RateLimiter struct {
	limiters        map[string]*rate.Limiter
	lastAccess      map[string]time.Time
	mu              sync.Mutex
	r               rate.Limit
	burst           int
	cleanupInterval time.Duration
	lastCleanup     time.Time
}

func NewRateLimiter(r rate.Limit, b int) *RateLimiter {
	return &RateLimiter{
		limiters:        make(map[string]*rate.Limiter),
		lastAccess:      make(map[string]time.Time),
		r:               r,
		burst:           b,
		cleanupInterval: 5 * time.Minute,
		lastCleanup:     time.Now(),
	}
}

func (rl *RateLimiter) getLimiter(key string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	if now.Sub(rl.lastCleanup) > rl.cleanupInterval {
		rl.cleanup(now)
		rl.lastCleanup = now
	}

	limiter, exists := rl.limiters[key]
	if !exists {
		limiter = rate.NewLimiter(rl.r, rl.burst)
		rl.limiters[key] = limiter
	}

	rl.lastAccess[key] = now
	return limiter
}

func (rl *RateLimiter) cleanup(now time.Time) {
	cutoff := now.Add(-rl.cleanupInterval * 2)
	for key, last := range rl.lastAccess {
		if last.Before(cutoff) {
			delete(rl.limiters, key)
			delete(rl.lastAccess, key)
		}
	}
}

func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := "ip:" + getClientIP(c.Request)

		limiter := rl.getLimiter(key)
		if !limiter.Allow() {
			c.Header("Retry-After", "1")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many requests, so maybe chill the fuck out",
			})
			return
		}

		c.Next()
	}
}

func getClientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return strings.Split(ip, ",")[0]
	}
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	return ip
}

var apologies []string

func readFile() []string {
	file, err := os.Open("zero-fucks.json")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	var lines []string
	sc := bufio.NewScanner(file)

	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		line = strings.Trim(line, `",`)
		if line != "" && line != "[" && line != "]" {
			lines = append(lines, line)
		}
	}

	return lines
}

func handler(c *gin.Context) {
	if len(apologies) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "empty dataset, just like your life",
		})
		return
	}

	i := rand.Intn(len(apologies))
	c.JSON(http.StatusOK, DontCare{
		Reason: apologies[i],
	})
}

func main() {
	rand.Seed(time.Now().UnixNano())

	apologies = readFile()

	r := gin.Default()

	rl := NewRateLimiter(1, 3)
	r.Use(rl.Middleware())

	r.GET("/sorry", handler)
	r.Run(":8080")
}
