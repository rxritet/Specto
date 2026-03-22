package web

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/rxritet/Specto/internal/config"
	"github.com/rxritet/Specto/internal/domain"
	"github.com/rxritet/Specto/internal/service"
)

const sessionCookieName = "specto_session"

type sessionManager struct {
	secret        []byte
	ttl           time.Duration
	secureCookies bool
}

type authContextKey struct{}

func newSessionManager(cfg *config.Config) *sessionManager {
	return &sessionManager{
		secret:        []byte(cfg.AuthSecret),
		ttl:           cfg.AuthSessionTTL,
		secureCookies: cfg.AuthSecureCookies,
	}
}

func (sm *sessionManager) issue(w http.ResponseWriter, userID int64) error {
	expiresAt := time.Now().Add(sm.ttl)
	token, err := sm.sign(userID, expiresAt)
	if err != nil {
		return err
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   sm.secureCookies,
		MaxAge:   int(sm.ttl.Seconds()),
		Expires:  expiresAt,
	})
	return nil
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

func (sm *sessionManager) authenticate(r *http.Request, users *service.UserService) (*domain.User, error) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return nil, fmt.Errorf("cookie: %w", err)
	}

	userID, err := sm.verify(cookie.Value)
	if err != nil {
		return nil, err
	}

	user, err := users.GetByID(r.Context(), userID)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (sm *sessionManager) sign(userID int64, expiresAt time.Time) (string, error) {
	payload := fmt.Sprintf("%d:%d", userID, expiresAt.Unix())
	payloadEncoded := base64.RawURLEncoding.EncodeToString([]byte(payload))
	sig := sm.signature(payload)
	sigEncoded := base64.RawURLEncoding.EncodeToString(sig)
	return payloadEncoded + "." + sigEncoded, nil
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
