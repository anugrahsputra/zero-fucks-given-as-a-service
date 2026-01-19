package main

import (
	"bufio"
	"context"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/op/go-logging"
	"golang.org/x/time/rate"
)

var Logger = logging.MustGetLogger("github.com/anugrahsputra/github.com/anugrahsputra/go-rest")

func ConfigureLogger() {
	format := logging.MustStringFormatter(
		`%{color}[%{time:2006-01-02 15:04:05}] ▶ %{level}%{color:reset} %{message} ...[%{shortfile}@%{shortfunc}()]`,
	)

	backend := logging.NewLogBackend(os.Stderr, "", 0)
	backendFormatter := logging.NewBackendFormatter(backend, format)

	logging.SetBackend(backendFormatter)
	Logger.Info("Logger configured successfully")
}

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
			Logger.Warningf("Rate limit exceeded for %s", key)
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

func readFile() ([]string, error) {
	file, err := os.Open("zero-fucks.json")
	if err != nil {
		return nil, err
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

	return lines, sc.Err()
}

func sorryHandler(c *gin.Context) {
	if len(apologies) == 0 {
		Logger.Error("Empty apologies list requested")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "empty dataset, just like your life",
		})
		return
	}

	c.JSON(http.StatusOK, DontCare{
		Reason: apologies[rand.Intn(len(apologies))],
	})
}

func main() {
	ConfigureLogger()

	rand.Seed(time.Now().UnixNano())
	gin.SetMode(gin.ReleaseMode)

	var err error
	apologies, err = readFile()
	if err != nil {
		Logger.Errorf("Failed to load apologies set")
	}

	r := gin.New()
	r.Use(gin.Recovery())

	_ = r.SetTrustedProxies([]string{"127.0.0.1"})

	rl := NewRateLimiter(1, 3)
	r.Use(rl.Middleware())

	r.GET("/", func(c *gin.Context) {
		// CHECK OUT MY OTHER WORK
		c.Redirect(301, "https://downormal.dev")
	})

	r.GET("/sorry", sorryHandler)
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "OK"})
	})

	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	go func() {
		Logger.Infof("server running on :8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			Logger.Fatal("listen error:", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	Logger.Infof("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		Logger.Fatal("server forced to shutdown:", err)
	}

	Logger.Infof("server exited cleanly")

}
