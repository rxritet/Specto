package web

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rxritet/Specto/internal/config"
	"github.com/rxritet/Specto/internal/domain"
	"github.com/rxritet/Specto/internal/service"
)

const sessionCookieName = "specto_session"

type sessionManager struct {
	secret        []byte
	ttl           time.Duration
	secureCookies bool
	redis         *redis.Client
}

type authContextKey struct{}

func newSessionManager(cfg *config.Config, redisClient *redis.Client) *sessionManager {
	return &sessionManager{
		secret:        []byte(cfg.AuthSecret),
		ttl:           cfg.AuthSessionTTL,
		secureCookies: cfg.AuthSecureCookies,
		redis:         redisClient,
	}
}

func (sm *sessionManager) issue(ctx context.Context, w http.ResponseWriter, userID int64) error {
	expiresAt := time.Now().Add(sm.ttl)

	value, err := sm.buildCookieValue(ctx, userID, expiresAt)
	if err != nil {
		return fmt.Errorf("build session value: %w", err)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   sm.secureCookies,
		MaxAge:   int(sm.ttl.Seconds()),
		Expires:  expiresAt,
	})
	return nil
}

func (sm *sessionManager) buildCookieValue(ctx context.Context, userID int64, expiresAt time.Time) (string, error) {
	if sm.redis == nil {
		return sm.sign(userID, expiresAt), nil
	}

	sid, err := randomSessionID()
	if err != nil {
		return "", fmt.Errorf("generate session id: %w", err)
	}

	redisCtx, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
	defer cancel()

	key := sessionRedisKey(sid)
	if err := sm.redis.Set(redisCtx, key, strconv.FormatInt(userID, 10), sm.ttl).Err(); err != nil {
		return "", fmt.Errorf("store session in redis: %w", err)
	}

	return sid, nil
}

func (sm *sessionManager) clear(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   sm.secureCookies,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}

func (sm *sessionManager) revoke(ctx context.Context, r *http.Request) {
	if sm.redis == nil {
		return
	}

	cookie, err := r.Cookie(sessionCookieName)
	if err != nil || cookie.Value == "" {
		return
	}

	redisCtx, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
	defer cancel()

	_ = sm.redis.Del(redisCtx, sessionRedisKey(cookie.Value)).Err()
}

func (sm *sessionManager) authenticate(r *http.Request, users *service.UserService) (*domain.User, error) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return nil, fmt.Errorf("cookie: %w", err)
	}

	userID, err := sm.userIDFromCookieValue(r.Context(), cookie.Value)
	if err != nil {
		return nil, err
	}

	user, err := users.GetByID(r.Context(), userID)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (sm *sessionManager) userIDFromCookieValue(ctx context.Context, cookieValue string) (int64, error) {
	if sm.redis == nil {
		return sm.verify(cookieValue)
	}

	redisCtx, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
	defer cancel()

	raw, err := sm.redis.Get(redisCtx, sessionRedisKey(cookieValue)).Result()
	if err != nil {
		if err == redis.Nil {
			return 0, fmt.Errorf("session expired")
		}
		return 0, fmt.Errorf("redis get session: %w", err)
	}

	userID, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse redis user id: %w", err)
	}

	return userID, nil
}

func (sm *sessionManager) sign(userID int64, expiresAt time.Time) string {
	payload := fmt.Sprintf("%d:%d", userID, expiresAt.Unix())
	payloadEncoded := base64.RawURLEncoding.EncodeToString([]byte(payload))
	sig := sm.signature(payload)
	sigEncoded := base64.RawURLEncoding.EncodeToString(sig)
	return payloadEncoded + "." + sigEncoded
}

func (sm *sessionManager) verify(token string) (int64, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid token format")
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return 0, fmt.Errorf("decode payload: %w", err)
	}

	sigBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return 0, fmt.Errorf("decode signature: %w", err)
	}

	expectedSig := sm.signature(string(payloadBytes))
	if subtle.ConstantTimeCompare(sigBytes, expectedSig) != 1 {
		return 0, fmt.Errorf("invalid signature")
	}

	payload := strings.Split(string(payloadBytes), ":")
	if len(payload) != 2 {
		return 0, fmt.Errorf("invalid payload")
	}

	userID, err := strconv.ParseInt(payload[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse user id: %w", err)
	}

	expiresUnix, err := strconv.ParseInt(payload[1], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse expiry: %w", err)
	}
	if time.Now().After(time.Unix(expiresUnix, 0)) {
		return 0, fmt.Errorf("session expired")
	}

	return userID, nil
}

func (sm *sessionManager) signature(payload string) []byte {
	mac := hmac.New(sha256.New, sm.secret)
	_, _ = mac.Write([]byte(payload))
	return mac.Sum(nil)
}

func randomSessionID() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func sessionRedisKey(sessionID string) string {
	return "session:" + sessionID
}

func (rt *Router) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := rt.auth.authenticate(r, rt.Users)
		if err != nil {
			rt.auth.clear(w)
			rt.respondError(w, http.StatusUnauthorized, authenticationRequiredMessage)
			return
		}

		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), authContextKey{}, user)))
	})
}

func currentUser(r *http.Request) (*domain.User, bool) {
	user, ok := r.Context().Value(authContextKey{}).(*domain.User)
	return user, ok
}
