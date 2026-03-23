package web

import (
	"net/http"

	"github.com/rxritet/Specto/internal/domain"
)

type registerRequest struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (rt *Router) handleAuthRegister(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := decodeJSONBody(r, &req); err != nil {
		rt.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	user := &domain.User{
		Email:    req.Email,
		Name:     req.Name,
		Password: req.Password,
	}

	if err := rt.Users.Register(r.Context(), user); err != nil {
		rt.handleServiceError(w, err)
		return
	}

	if err := rt.auth.issue(r.Context(), w, user.ID); err != nil {
		rt.Logger.Error("issue session failed", "error", err)
		rt.respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	rt.respondJSON(w, http.StatusCreated, user)
}

func (rt *Router) handleAuthLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decodeJSONBody(r, &req); err != nil {
		rt.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	user, err := rt.Users.Authenticate(r.Context(), req.Email, req.Password)
	if err != nil {
		rt.handleServiceError(w, err)
		return
	}

	if err := rt.auth.issue(r.Context(), w, user.ID); err != nil {
		rt.Logger.Error("issue session failed", "error", err)
		rt.respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	rt.respondJSON(w, http.StatusOK, user)
}

func (rt *Router) handleAuthLogout(w http.ResponseWriter, r *http.Request) {
	rt.auth.revoke(r.Context(), r)
	rt.auth.clear(w)
	rt.respondJSON(w, http.StatusOK, map[string]string{"status": "logged_out"})
}

func (rt *Router) handleAuthMe(w http.ResponseWriter, r *http.Request) {
	user, ok := currentUser(r)
	if !ok {
		rt.respondError(w, http.StatusUnauthorized, authenticationRequiredMessage)
		return
	}
	rt.respondJSON(w, http.StatusOK, user)
}
