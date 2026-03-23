package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestRedisRateLimitBlocksAfterThreshold(t *testing.T) {
	mini, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	defer mini.Close()

	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	defer client.Close()

	h := RedisRateLimit(client, 2, noopLogger())(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected status %d on request %d, got %d", http.StatusOK, i+1, rr.Code)
		}
	}

	blockedReq := httptest.NewRequest(http.MethodGet, "/health", nil)
	blockedReq.RemoteAddr = "127.0.0.1:12345"
	blockedRec := httptest.NewRecorder()
	h.ServeHTTP(blockedRec, blockedReq)
	if blockedRec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status %d, got %d", http.StatusTooManyRequests, blockedRec.Code)
	}
	if blockedRec.Header().Get("Retry-After") == "" {
		t.Fatal("expected Retry-After header for throttled response")
	}
}
