package web

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rxritet/Specto/internal/domain"
)

// Middleware is a standard HTTP middleware signature.
type Middleware func(http.Handler) http.Handler

// Chain applies a sequence of middleware to a handler.
// Middleware are executed in the order they are provided (first wraps outermost).
func Chain(h http.Handler, mw ...Middleware) http.Handler {
	// Apply in reverse so the first middleware in the slice
	// is the outermost wrapper.
	for i := len(mw) - 1; i >= 0; i-- {
		h = mw[i](h)
	}
	return h
}

// ---------- Logging ----------

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.wroteHeader {
		rw.status = code
		rw.wroteHeader = true
	}
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}

// Logging returns middleware that logs every request with slog.
func Logging(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rw, r)

			logger.Info("http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rw.status,
				"duration", time.Since(start).String(),
				"remote", r.RemoteAddr,
			)
		})
	}
}

// ---------- Recovery ----------

// Recovery returns middleware that catches panics, logs them and responds
// with 500 Internal Server Error.
func Recovery(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					stack := debug.Stack()
					logger.Error("panic recovered",
						"error", fmt.Sprint(rec),
						"method", r.Method,
						"path", r.URL.Path,
						"stack", string(stack),
					)
					http.Error(w, http.StatusText(http.StatusInternalServerError),
						http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// ---------- Security Headers ----------

// SecureHeaders returns middleware that sets common security-related
// HTTP response headers.
func SecureHeaders() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("X-XSS-Protection", "0")
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			h.Set("Content-Security-Policy",
				"default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'")
			next.ServeHTTP(w, r)
		})
	}
}

// RedisRateLimit limits requests per IP in a rolling 1-minute bucket.
// If Redis is unavailable, requests are allowed (fail-open) to avoid hard outages.
func RedisRateLimit(client *redis.Client, perMinute int, logger *slog.Logger) Middleware {
	if client == nil || perMinute <= 0 {
		return func(next http.Handler) http.Handler { return next }
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := rateLimitKey(r.RemoteAddr, time.Now())
			count, err := incrementRateCounter(r.Context(), client, key, logger)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			if isRateLimited(count, perMinute) {
				writeRateLimitResponse(w, time.Now())
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func rateLimitKey(remoteAddr string, now time.Time) string {
	ip := clientIP(remoteAddr)
	bucket := now.Unix() / 60
	return fmt.Sprintf("rate:%s:%d", ip, bucket)
}

func incrementRateCounter(ctx context.Context, client *redis.Client, key string, logger *slog.Logger) (int64, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()

	count, err := client.Incr(timeoutCtx, key).Result()
	if err != nil {
		logger.Warn("rate limiter redis error", "error", err)
		return 0, err
	}

	if count == 1 {
		if expireErr := client.Expire(timeoutCtx, key, 2*time.Minute).Err(); expireErr != nil {
			logger.Warn("rate limiter expire error", "error", expireErr)
		}
	}

	return count, nil
}

func isRateLimited(count int64, perMinute int) bool {
	return count > int64(perMinute)
}

func writeRateLimitResponse(w http.ResponseWriter, now time.Time) {
	retryAfter := 60 - now.Second()
	if retryAfter <= 0 {
		retryAfter = 1
	}

	w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
	http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
}

func clientIP(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return remoteAddr
	}
	if host == "" {
		return remoteAddr
	}
	return host
}

// AuditTrail appends request metadata to the audit store.
func AuditTrail(audit domain.AuditLogger, logger *slog.Logger) Middleware {
	if audit == nil {
		return func(next http.Handler) http.Handler { return next }
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rw, r)

			rec := domain.AuditRecord{
				Timestamp: time.Now().UTC(),
				Method:    r.Method,
				Path:      r.URL.Path,
				Remote:    clientIP(r.RemoteAddr),
				Status:    rw.status,
			}

			ctx, cancel := context.WithTimeout(r.Context(), 300*time.Millisecond)
			defer cancel()

			if err := audit.Append(ctx, rec); err != nil {
				logger.Warn("audit append failed", "error", err)
			}
		})
	}
}
