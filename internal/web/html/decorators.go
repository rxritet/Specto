package html

import (
	"fmt"
	"time"

	"github.com/rxritet/Specto/internal/domain"
	"github.com/rxritet/Specto/internal/service"
)

// ---------- User Decorator ----------

// User wraps domain.User with UI-specific helper methods.
// The embedded struct is accessible in templates via its field names directly.
type User struct {
	domain.User
}

// NewUser creates a UI decorator from a domain user.
func NewUser(u *domain.User) User {
	return User{User: *u}
}

// CreatedAtFmt returns CreatedAt formatted for display.
func (u User) CreatedAtFmt() string {
	return u.CreatedAt.Format("02 Jan 2006, 15:04")
}

// UpdatedAtFmt returns UpdatedAt formatted for display.
func (u User) UpdatedAtFmt() string {
	return u.UpdatedAt.Format("02 Jan 2006, 15:04")
}

// Initials returns the first letter of the user's name (for avatars).
func (u User) Initials() string {
	if u.Name == "" {
		return "?"
	}
	return string([]rune(u.Name)[0])
}

// ---------- Task Decorator ----------

// Task wraps domain.Task with UI-specific helper methods.
type Task struct {
	domain.Task
}

// NewTask creates a UI decorator from a domain task.
func NewTask(t *domain.Task) Task {
	return Task{Task: *t}
}

// NewTaskList converts a slice of domain tasks into decorated tasks.
func NewTaskList(tasks []domain.Task) []Task {
	out := make([]Task, len(tasks))
	for i := range tasks {
		out[i] = NewTask(&tasks[i])
	}
	return out
}

// CreatedAtFmt returns CreatedAt formatted for display.
func (t Task) CreatedAtFmt() string {
	return t.CreatedAt.Format("02 Jan 2006, 15:04")
}

// UpdatedAtFmt returns UpdatedAt formatted for display.
func (t Task) UpdatedAtFmt() string {
	return t.UpdatedAt.Format("02 Jan 2006, 15:04")
}

// CreatedAgo returns a human-readable relative time string.
func (t Task) CreatedAgo() string {
	return relativeTime(t.CreatedAt)
}

// StatusLabel returns a human-friendly label for the task status.
func (t Task) StatusLabel() string {
	switch t.Status {
	case domain.TaskStatusTodo:
		return "To Do"
	case domain.TaskStatusInProgress:
		return "In Progress"
	case domain.TaskStatusDone:
		return "Done"
	default:
		return string(t.Status)
	}
}

// StatusClass returns a CSS class name suitable for styling the status badge.
func (t Task) StatusClass() string {
	switch t.Status {
	case domain.TaskStatusTodo:
		return "badge-todo"
	case domain.TaskStatusInProgress:
		return "badge-inprogress"
	case domain.TaskStatusDone:
		return "badge-done"
	default:
		return "badge-default"
	}
}

// IsDone reports whether the task is completed.
func (t Task) IsDone() bool {
	return t.Status == domain.TaskStatusDone
}

// ---------- TaskStats Decorator ----------

// TaskStats wraps service.TaskStats with display helpers.
type TaskStats struct {
	service.TaskStats
}

// NewTaskStats creates a UI decorator from service stats.
func NewTaskStats(s *service.TaskStats) TaskStats {
	return TaskStats{TaskStats: *s}
}

// TodoPctFmt returns the "todo" percentage formatted to one decimal.
func (s TaskStats) TodoPctFmt() string {
	return fmt.Sprintf("%.1f%%", s.TodoPct)
}

// InProgPctFmt returns the "in_progress" percentage formatted.
func (s TaskStats) InProgPctFmt() string {
	return fmt.Sprintf("%.1f%%", s.InProgPct)
}

// DonePctFmt returns the "done" percentage formatted.
func (s TaskStats) DonePctFmt() string {
	return fmt.Sprintf("%.1f%%", s.DonePct)
}

// ---------- Helpers ----------

// relativeTime produces a human-readable "time ago" string.
func relativeTime(t time.Time) string {
	d := time.Since(t)

	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		return fmt.Sprintf("%d min ago", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		return fmt.Sprintf("%d hr ago", h)
	default:
		days := int(d.Hours() / 24)
		return fmt.Sprintf("%d days ago", days)
	}
}
