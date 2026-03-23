package web

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/rxritet/Specto/internal/config"
	"github.com/rxritet/Specto/internal/service"
)

const authMePath = "/auth/me"

func TestRedisSessionLifecycle(t *testing.T) {
	mini, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	defer mini.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	defer rdb.Close()

	userRepo := newAuthTestUserRepo()
	taskRepo := newAuthTestTaskRepo()
	logger := noopLogger()
	userSvc := service.NewUserService(userRepo, logger)
	taskSvc := service.NewTaskService(taskRepo, userRepo, logger)
	cfg := &config.Config{
		AuthSecret:         "test-secret",
		AuthSessionTTL:     time.Hour,
		AuthSecureCookies:  false,
		RateLimitPerMinute: 60,
	}

	router := NewRouter(cfg, logger, taskSvc, userSvc, rdb, nil)

	cookie := registerAndLogin(t, router, "redis-user@example.com", "Redis User", "password123")
	if cookie == nil || cookie.Value == "" {
		t.Fatal("expected session cookie")
	}

	key := sessionRedisKey(cookie.Value)
	if !mini.Exists(key) {
		t.Fatalf("expected redis session key %q to exist", key)
	}

	meReq := httptest.NewRequest(http.MethodGet, authMePath, nil)
	meReq.AddCookie(cookie)
	meRec := httptest.NewRecorder()
	router.ServeHTTP(meRec, meReq)
	if meRec.Code != http.StatusOK {
		t.Fatalf(errExpectedStatus, http.StatusOK, meRec.Code)
	}

	logoutReq := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	logoutReq.AddCookie(cookie)
	logoutRec := httptest.NewRecorder()
	router.ServeHTTP(logoutRec, logoutReq)
	if logoutRec.Code != http.StatusOK {
		t.Fatalf(errExpectedStatus, http.StatusOK, logoutRec.Code)
	}

	if mini.Exists(key) {
		t.Fatalf("expected redis session key %q to be deleted", key)
	}

	meAfterLogoutReq := httptest.NewRequest(http.MethodGet, authMePath, nil)
	meAfterLogoutReq.AddCookie(cookie)
	meAfterLogoutRec := httptest.NewRecorder()
	router.ServeHTTP(meAfterLogoutRec, meAfterLogoutReq)
	if meAfterLogoutRec.Code != http.StatusUnauthorized {
		t.Fatalf(errExpectedStatus, http.StatusUnauthorized, meAfterLogoutRec.Code)
	}
}

func TestRedisSessionMissingKeyUnauthorized(t *testing.T) {
	mini, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	defer mini.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	defer rdb.Close()

	userRepo := newAuthTestUserRepo()
	taskRepo := newAuthTestTaskRepo()
	logger := noopLogger()
	userSvc := service.NewUserService(userRepo, logger)
	taskSvc := service.NewTaskService(taskRepo, userRepo, logger)
	cfg := &config.Config{
		AuthSecret:         "test-secret",
		AuthSessionTTL:     time.Hour,
		AuthSecureCookies:  false,
		RateLimitPerMinute: 60,
	}

	router := NewRouter(cfg, logger, taskSvc, userSvc, rdb, nil)

	body, err := json.Marshal(map[string]string{
		"email":    "ghost@example.com",
		"name":     "Ghost",
		"password": "password123",
	})
	if err != nil {
		t.Fatalf("marshal register request: %v", err)
	}

	registerReq := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	registerReq.Header.Set(headerContentType, contentTypeJSON)
	registerRec := httptest.NewRecorder()
	router.ServeHTTP(registerRec, registerReq)
	if registerRec.Code != http.StatusCreated {
		t.Fatalf(errExpectedStatus, http.StatusCreated, registerRec.Code)
	}

	// Session cookie is issued at registration; remove backing Redis key to emulate expiration.
	cookies := registerRec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected registration to return a session cookie")
	}
	if err := rdb.Del(context.Background(), sessionRedisKey(cookies[0].Value)).Err(); err != nil {
		t.Fatalf("delete redis session key: %v", err)
	}

	meReq := httptest.NewRequest(http.MethodGet, authMePath, nil)
	meReq.AddCookie(cookies[0])
	meRec := httptest.NewRecorder()
	router.ServeHTTP(meRec, meReq)
	if meRec.Code != http.StatusUnauthorized {
		t.Fatalf(errExpectedStatus, http.StatusUnauthorized, meRec.Code)
	}
}
