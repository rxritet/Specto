package web

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/rxritet/Specto/internal/domain"
)

const invalidTaskIDMessage = "invalid task id"
const authenticationRequiredMessage = "authentication required"

// ---------- Task Handlers ----------

// handleTaskList  GET /tasks
func (rt *Router) handleTaskList(w http.ResponseWriter, r *http.Request) {
	user, ok := currentUser(r)
	if !ok {
		rt.respondError(w, http.StatusUnauthorized, authenticationRequiredMessage)
		return
	}

	tasks, err := rt.Tasks.ListByUser(r.Context(), user.ID)
	if err != nil {
		rt.handleServiceError(w, err)
		return
	}

	rt.respondJSON(w, http.StatusOK, tasks)
}

// handleTaskGet  GET /tasks/{id}
func (rt *Router) handleTaskGet(w http.ResponseWriter, r *http.Request) {
	user, ok := currentUser(r)
	if !ok {
		rt.respondError(w, http.StatusUnauthorized, authenticationRequiredMessage)
		return
	}

	id, err := pathInt64(r, "id")
	if err != nil {
		rt.respondError(w, http.StatusBadRequest, invalidTaskIDMessage)
		return
	}

	task, err := rt.taskForUser(r.Context(), id, user.ID)
	if err != nil {
		rt.handleServiceError(w, err)
		return
	}

	rt.respondJSON(w, http.StatusOK, task)
}

// handleTaskCreate  POST /tasks
// Accepts JSON body or form fields: title, description, status.
func (rt *Router) handleTaskCreate(w http.ResponseWriter, r *http.Request) {
	user, ok := currentUser(r)
	if !ok {
		rt.respondError(w, http.StatusUnauthorized, authenticationRequiredMessage)
		return
	}

	task, err := decodeTask(r)
	if err != nil {
		rt.respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	task.UserID = user.ID

	if err := rt.Tasks.Create(r.Context(), task); err != nil {
		rt.handleServiceError(w, err)
		return
	}

	rt.respondJSON(w, http.StatusCreated, task)
}

// handleTaskUpdate  PUT /tasks/{id}
// Accepts JSON body or form fields: title, description, status.
func (rt *Router) handleTaskUpdate(w http.ResponseWriter, r *http.Request) {
	user, ok := currentUser(r)
	if !ok {
		rt.respondError(w, http.StatusUnauthorized, authenticationRequiredMessage)
		return
	}

	id, err := pathInt64(r, "id")
	if err != nil {
		rt.respondError(w, http.StatusBadRequest, invalidTaskIDMessage)
		return
	}

	if _, err := rt.taskForUser(r.Context(), id, user.ID); err != nil {
		rt.handleServiceError(w, err)
		return
	}

	task, err := decodeTask(r)
	if err != nil {
		rt.respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	task.ID = id
	task.UserID = user.ID

	if err := rt.Tasks.Update(r.Context(), task); err != nil {
		rt.handleServiceError(w, err)
		return
	}

	rt.respondJSON(w, http.StatusOK, task)
}

// handleTaskDelete  DELETE /tasks/{id}
func (rt *Router) handleTaskDelete(w http.ResponseWriter, r *http.Request) {
	user, ok := currentUser(r)
	if !ok {
		rt.respondError(w, http.StatusUnauthorized, authenticationRequiredMessage)
		return
	}

	id, err := pathInt64(r, "id")
	if err != nil {
		rt.respondError(w, http.StatusBadRequest, invalidTaskIDMessage)
		return
	}

	if _, err := rt.taskForUser(r.Context(), id, user.ID); err != nil {
		rt.handleServiceError(w, err)
		return
	}

	if err := rt.Tasks.Delete(r.Context(), id); err != nil {
		rt.handleServiceError(w, err)
		return
	}

	rt.respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// handleTaskStats  GET /tasks/stats
func (rt *Router) handleTaskStats(w http.ResponseWriter, r *http.Request) {
	user, ok := currentUser(r)
	if !ok {
		rt.respondError(w, http.StatusUnauthorized, authenticationRequiredMessage)
		return
	}

	stats, err := rt.Tasks.StatsByUser(r.Context(), user.ID)
	if err != nil {
		rt.handleServiceError(w, err)
		return
	}

	rt.respondJSON(w, http.StatusOK, stats)
}

// ---------- Decode helpers ----------

// decodeTask builds a domain.Task from a JSON body or form values.
func decodeTask(r *http.Request) (*domain.Task, error) {
	ct := r.Header.Get("Content-Type")

	// JSON path.
	if ct == "" || strings.HasPrefix(ct, "application/json") {
		var t domain.Task
		if err := decodeJSONBody(r, &t); err != nil {
			return nil, err
		}
		return &t, nil
	}

	// Form path (HTML form submissions).
	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	userID, _ := strconv.ParseInt(r.FormValue("user_id"), 10, 64)

	return &domain.Task{
		UserID:      userID,
		Title:       r.FormValue("title"),
		Description: r.FormValue("description"),
		Status:      domain.TaskStatus(r.FormValue("status")),
	}, nil
}

// ---------- Response helpers ----------

// respondJSON writes a JSON response with the given status code.
func (rt *Router) respondJSON(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		rt.Logger.Error("json encode failed", "error", err)
	}
}

// respondError writes a JSON error object.
func (rt *Router) respondError(w http.ResponseWriter, code int, msg string) {
	rt.respondJSON(w, code, map[string]string{"error": msg})
}

// handleServiceError maps domain error types to HTTP status codes.
func (rt *Router) handleServiceError(w http.ResponseWriter, err error) {
	if nf, ok := errors.AsType[*domain.NotFoundError](err); ok {
		rt.respondError(w, http.StatusNotFound, nf.Error())
		return
	}
	if ve, ok := errors.AsType[*domain.ValidationError](err); ok {
		rt.respondError(w, http.StatusUnprocessableEntity, ve.Error())
		return
	}
	if ce, ok := errors.AsType[*domain.ConflictError](err); ok {
		rt.respondError(w, http.StatusConflict, ce.Error())
		return
	}
	if ue, ok := errors.AsType[*domain.UnauthorizedError](err); ok {
		rt.respondError(w, http.StatusUnauthorized, ue.Error())
		return
	}

	rt.Logger.Error("unhandled service error", "error", err)
	rt.respondError(w, http.StatusInternalServerError, "internal server error")
}

func decodeJSONBody(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}

func (rt *Router) taskForUser(ctx context.Context, taskID, userID int64) (*domain.Task, error) {
	task, err := rt.Tasks.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if task.UserID != userID {
		return nil, domain.NewNotFoundError("task", fmt.Sprintf("id=%d", taskID))
	}
	return task, nil
}

// ---------- Path / query param helpers ----------

// pathInt64 extracts a named path parameter (Go 1.22+) and parses it as int64.
func pathInt64(r *http.Request, name string) (int64, error) {
	return strconv.ParseInt(r.PathValue(name), 10, 64)
}

// queryInt64 extracts a query-string parameter and parses it as int64.
func queryInt64(r *http.Request, name string) (int64, error) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return 0, strconv.ErrRange
	}
	return strconv.ParseInt(raw, 10, 64)
}
