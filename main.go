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
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/go-redis/redis_rate/v10"
	"github.com/op/go-logging"
	"github.com/redis/go-redis/v9"
)

var Logger = logging.MustGetLogger("github.com/anugrahsputra/zero-fucks-given-as-a-service")

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

type RedisRateLimiter struct {
	limiter *redis_rate.Limiter
	limit   redis_rate.Limit
}

func NewRedisRateLimiter(rdb *redis.Client, rate int, burst int, period time.Duration) *RedisRateLimiter {
	return &RedisRateLimiter{
		limiter: redis_rate.NewLimiter(rdb),
		limit: redis_rate.Limit{
			Rate:   rate,
			Burst:  burst,
			Period: period,
		},
	}
}

func (rl *RedisRateLimiter) Middleware() gin.HandlerFunc {
	prefix := os.Getenv("REDIS_PREFIX")
	if prefix == "" {
		prefix = "zfgaas"
	}

	return func(c *gin.Context) {
		key := prefix + ":rate_limit:ip:" + getClientIP(c.Request)

		res, err := rl.limiter.Allow(c, key, rl.limit)
		if err != nil {
			Logger.Errorf("Redis rate limit error: %v", err)
			c.Next()
			return
		}

		if res.Allowed == 0 {
			Logger.Warningf("Rate limit exceeded for %s", key)
			c.Header("Retry-After", time.Duration(res.RetryAfter).String())
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many requests, so maybe chill the fuck out",
			})
			return
		}

		c.Next()
	}
}

var trustedProxyNets []*net.IPNet

func isTrustedProxy(ip net.IP) bool {
	for _, n := range trustedProxyNets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

func getClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	remoteIP := net.ParseIP(host)
	if remoteIP == nil {
		return host
	}

	if !isTrustedProxy(remoteIP) {
		return remoteIP.String()
	}

	if fwd := r.Header.Get("Forwarded"); fwd != "" {
		for part := range strings.SplitSeq(fwd, ";") {
			if strings.HasPrefix(strings.ToLower(part), "for=") {
				ip := strings.Trim(part[4:], `"`)
				return strings.Split(ip, ":")[0]
			}
		}
	}

	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	if rip := r.Header.Get("X-Real-IP"); rip != "" {
		return rip
	}

	return remoteIP.String()
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

	rand.New(rand.NewSource(time.Now().UnixNano()))
	gin.SetMode(gin.ReleaseMode)

	var err error
	apologies, err = readFile()
	if err != nil {
		Logger.Errorf("Failed to load apologies set")
	}

	// Redis connection setup
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "redis:6379"
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	// Check Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := rdb.Ping(ctx).Result(); err != nil {
		Logger.Errorf("Failed to connect to Redis at %s: %v", redisAddr, err)
	} else {
		Logger.Infof("Connected to Redis at %s", redisAddr)
	}

	rl := NewRedisRateLimiter(rdb, 2, 4, time.Second)

	r := gin.Default()
	r.Use(gin.Recovery())

	r.GET("/", func(c *gin.Context) {
		// CHECK OUT MY OTHER WORK
		c.Redirect(301, "https://downormal.dev")
	})

	r.GET("/sorry", rl.Middleware(), sorryHandler)
	r.GET("/health", rl.Middleware(), func(c *gin.Context) {
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

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		Logger.Fatal("server forced to shutdown:", err)
	}

	if err := rdb.Close(); err != nil {
		Logger.Errorf("error closing redis: %v", err)
	}

	Logger.Infof("server exited cleanly")

}
