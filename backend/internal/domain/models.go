package domain

import (
	"context"
	"time"
)

// ---------- User ----------

// User represents an application user.
type User struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Password  string    `json:"-"` // never serialized
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

// Task is kept for backward compatibility while banking features are introduced.
type Task struct {
	ID          int64      `json:"id"`
	UserID      int64      `json:"user_id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      TaskStatus `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// ---------- Account ----------

type Account struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Currency  string    `json:"currency"`
	Balance   int64     `json:"balance"` // strictly defined in cents
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ---------- Card ----------

type Card struct {
	ID         int64     `json:"id"`
	AccountID  int64     `json:"account_id"`
	Number     string    `json:"number"`
	CVV        string    `json:"cvv"`
	ExpiryDate time.Time `json:"expiry_date"`
	IsActive   bool      `json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
}

// ---------- Transaction (aka Transfer) ----------

type TransactionType string

const (
	TxDeposit    TransactionType = "deposit"
	TxWithdrawal TransactionType = "withdrawal"
	TxTransfer   TransactionType = "transfer"
)

type Transfer struct {
	ID                int64     `json:"id"`
	SenderAccountID   int64     `json:"sender_account_id"`
	ReceiverAccountID int64     `json:"receiver_account_id"`
	Amount            int64     `json:"amount"` // in cents
	Currency          string    `json:"currency"`
	Description       string    `json:"description"`
	CreatedAt         time.Time `json:"created_at"`
}

type CreateTransferRequest struct {
	ReceiverAccountID int64  `json:"receiver_account_id"`
	Amount            int64  `json:"amount"`
	Currency          string `json:"currency"`
	Description       string `json:"description"`
}

type Transaction struct {
	ID        int64           `json:"id"`
	FromID    *int64          `json:"from_account_id,omitempty"` // Nullable for initial deposits
	ToID      int64           `json:"to_account_id"`
	Amount    int64           `json:"amount"` // in cents
	Type      TransactionType `json:"type"`
	Status    string          `json:"status"` // pending, success, failed
	CreatedAt time.Time       `json:"created_at"`
}

// ---------- Repository interfaces ----------

// UserRepository describes persistence operations for users.
type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id int64) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id int64) error
}

// TaskRepository is kept to preserve existing API during migration.
type TaskRepository interface {
	Create(ctx context.Context, task *Task) error
	GetByID(ctx context.Context, id int64) (*Task, error)
	ListByUser(ctx context.Context, userID int64) ([]Task, error)
	Update(ctx context.Context, task *Task) error
	Delete(ctx context.Context, id int64) error
}

type AccountRepository interface {
	Create(ctx context.Context, account *Account) error
	GetByID(ctx context.Context, id int64) (*Account, error)
	GetByUserID(ctx context.Context, userID int64) ([]*Account, error)
	UpdateBalance(ctx context.Context, accountID int64, amount int64) error
}

type TransferRepository interface {
	Create(ctx context.Context, transfer *Transfer) error
	GetByAccountID(ctx context.Context, accountID int64) ([]*Transfer, error)
}

type CardRepository interface {
	Create(ctx context.Context, card *Card) error
	GetByAccountID(ctx context.Context, accountID int64) ([]*Card, error)
	Block(ctx context.Context, id int64) error
}

type TransactionRepository interface {
	Create(ctx context.Context, tx *Transaction) error
	GetByAccountID(ctx context.Context, accountID int64) ([]*Transaction, error)
	Transfer(ctx context.Context, fromID, toID int64, amount int64) error
} // end of models
