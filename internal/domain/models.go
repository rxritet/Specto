package domain

import "time"

// ---------- User ----------

// User represents an application user.
type User struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Password  string    `json:"-"` // never serialised
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ---------- Task ----------

// TaskStatus enumerates the allowed states of a task.
type TaskStatus string

const (
	TaskStatusTodo       TaskStatus = "todo"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusDone       TaskStatus = "done"
)

// Task represents a single unit of work owned by a user.
type Task struct {
	ID          int64      `json:"id"`
	UserID      int64      `json:"user_id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      TaskStatus `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// ---------- Repository interfaces ----------

// UserRepository describes persistence operations for users.
type UserRepository interface {
	Create(user *User) error
	GetByID(id int64) (*User, error)
	GetByEmail(email string) (*User, error)
	Update(user *User) error
	Delete(id int64) error
}

// TaskRepository describes persistence operations for tasks.
type TaskRepository interface {
	Create(task *Task) error
	GetByID(id int64) (*Task, error)
	ListByUser(userID int64) ([]Task, error)
	Update(task *Task) error
	Delete(id int64) error
}
