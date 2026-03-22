package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/rxritet/Specto/internal/domain"
)

// ---------- Fuzz: JSON task decoding ----------

// FuzzDecodeTaskJSON feeds arbitrary bytes as a JSON body into decodeTask.
// The function must never panic regardless of input.
func FuzzDecodeTaskJSON(f *testing.F) {
	// Seed corpus — valid and edge-case JSON payloads.
	f.Add([]byte(`{"user_id":1,"title":"Buy milk","status":"todo"}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"title":""}`))
	f.Add([]byte(`{"status":"unknown_status_value"}`))
	f.Add([]byte(`{"user_id":-1,"title":"neg"}`))
	f.Add([]byte(`{"user_id":99999999999999999,"title":"big"}`))
	f.Add([]byte(`not json at all`))
	f.Add([]byte{})
	f.Add([]byte(`null`))
	f.Add([]byte(`[1,2,3]`))

	f.Fuzz(func(t *testing.T, data []byte) {
		r := httptest.NewRequest(http.MethodPost, "/tasks",
			bytes.NewReader(data))
		r.Header.Set("Content-Type", "application/json")

		// Must not panic. Errors are acceptable.
		task, err := decodeTask(r)
		if err != nil {
			return // decode failure is fine
		}

		// If decoding succeeded, re-encode to verify round-trip stability.
		out, err := json.Marshal(task)
		if err != nil {
			t.Fatalf("re-encode failed: %v", err)
		}
		_ = out
	})
}

// ---------- Fuzz: Form task decoding ----------

// FuzzDecodeTaskForm feeds arbitrary form field values into decodeTask.
// The function must never panic regardless of input.
func FuzzDecodeTaskForm(f *testing.F) {
	f.Add("1", "Buy groceries", "Milk and bread", "todo")
	f.Add("", "", "", "")
	f.Add("abc", "Title", "Desc", "invalid")
	f.Add("-1", "Neg", "", "done")
	f.Add("99999999999999999999", "Big", "", "in_progress")
	f.Add("0", strings.Repeat("A", 10000), "", "todo")

	f.Fuzz(func(t *testing.T, userID, title, description, status string) {
		form := url.Values{}
		form.Set("user_id", userID)
		form.Set("title", title)
		form.Set("description", description)
		form.Set("status", status)

		r := httptest.NewRequest(http.MethodPost, "/tasks",
			strings.NewReader(form.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		// Must not panic.
		task, err := decodeTask(r)
		if err != nil {
			return
		}

		// Basic sanity: title should match input.
		if task.Title != title {
			t.Fatalf("title mismatch: got %q, want %q", task.Title, title)
		}
		if task.Status != domain.TaskStatus(status) {
			t.Fatalf("status mismatch: got %q, want %q", task.Status, status)
		}
	})
}

// ---------- Fuzz: queryInt64 parsing ----------

// FuzzQueryInt64 verifies that queryInt64 never panics on arbitrary input.
func FuzzQueryInt64(f *testing.F) {
	f.Add("1")
	f.Add("0")
	f.Add("-1")
	f.Add("")
	f.Add("abc")
	f.Add("99999999999999999999")
	f.Add("  42  ")
	f.Add("1.5")

	f.Fuzz(func(t *testing.T, val string) {
		r := httptest.NewRequest(http.MethodGet,
			"/tasks?user_id="+url.QueryEscape(val), nil)

		// Must not panic.
		_, _ = queryInt64(r, "user_id")
	})
}

// ---------- Fuzz: handleServiceError mapping ----------

// FuzzHandleServiceError verifies the error-to-HTTP mapping never panics
// on arbitrary error messages.
func FuzzHandleServiceError(f *testing.F) {
	f.Add("user", "id=1")
	f.Add("", "")
	f.Add("task", strings.Repeat("x", 10000))

	f.Fuzz(func(t *testing.T, entity, key string) {
		rt := &Router{
			Mux:    http.NewServeMux(),
			Logger: noopLogger(),
		}

		rec := httptest.NewRecorder()

		// NotFoundError
		rt.handleServiceError(rec, domain.NewNotFoundError(entity, key))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rec.Code)
		}

		// ValidationError
		rec = httptest.NewRecorder()
		rt.handleServiceError(rec, domain.NewValidationError(entity, key))
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("expected 422, got %d", rec.Code)
		}

		// ConflictError
		rec = httptest.NewRecorder()
		rt.handleServiceError(rec, domain.NewConflictError(entity, key))
		if rec.Code != http.StatusConflict {
			t.Fatalf("expected 409, got %d", rec.Code)
		}
	})
}
