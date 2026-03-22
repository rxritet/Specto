package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver

	"github.com/rxritet/Specto/internal/domain"
)

// ---------- Connection ----------

// OpenPostgres opens a connection pool to PostgreSQL and verifies it with a ping.
func OpenPostgres(dsn string, logger *slog.Logger) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres open: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("postgres ping: %w", err)
	}

	logger.Info("connected to postgresql")
	return db, nil
}

// ---------- User Repository ----------

// PgUserRepo implements domain.UserRepository backed by PostgreSQL.
type PgUserRepo struct {
	db *sql.DB
}

func NewPgUserRepo(db *sql.DB) *PgUserRepo {
	return &PgUserRepo{db: db}
}

func (r *PgUserRepo) Create(ctx context.Context, user *domain.User) error {
	q := Conn(ctx, r.db)

	return q.QueryRowContext(ctx,
		`INSERT INTO users (email, name, password, created_at, updated_at)
		 VALUES ($1, $2, $3, NOW(), NOW())
		 RETURNING id, created_at, updated_at`,
		user.Email, user.Name, user.Password,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
}

func (r *PgUserRepo) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	q := Conn(ctx, r.db)

	u := &domain.User{}
	err := q.QueryRowContext(ctx,
		`SELECT id, email, name, password, created_at, updated_at
		 FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Email, &u.Name, &u.Password, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *PgUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	q := Conn(ctx, r.db)

	u := &domain.User{}
	err := q.QueryRowContext(ctx,
		`SELECT id, email, name, password, created_at, updated_at
		 FROM users WHERE email = $1`, email,
	).Scan(&u.ID, &u.Email, &u.Name, &u.Password, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *PgUserRepo) Update(ctx context.Context, user *domain.User) error {
	q := Conn(ctx, r.db)

	_, err := q.ExecContext(ctx,
		`UPDATE users SET email = $1, name = $2, password = $3, updated_at = NOW()
		 WHERE id = $4`,
		user.Email, user.Name, user.Password, user.ID,
	)
	return err
}

func (r *PgUserRepo) Delete(ctx context.Context, id int64) error {
	q := Conn(ctx, r.db)

	_, err := q.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, id)
	return err
}

// ---------- Task Repository ----------

// PgTaskRepo implements domain.TaskRepository backed by PostgreSQL.
type PgTaskRepo struct {
	db *sql.DB
}

func NewPgTaskRepo(db *sql.DB) *PgTaskRepo {
	return &PgTaskRepo{db: db}
}

func (r *PgTaskRepo) Create(ctx context.Context, task *domain.Task) error {
	q := Conn(ctx, r.db)

	return q.QueryRowContext(ctx,
		`INSERT INTO tasks (user_id, title, description, status, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, NOW(), NOW())
		 RETURNING id, created_at, updated_at`,
		task.UserID, task.Title, task.Description, task.Status,
	).Scan(&task.ID, &task.CreatedAt, &task.UpdatedAt)
}

func (r *PgTaskRepo) GetByID(ctx context.Context, id int64) (*domain.Task, error) {
	q := Conn(ctx, r.db)

	t := &domain.Task{}
	err := q.QueryRowContext(ctx,
		`SELECT id, user_id, title, description, status, created_at, updated_at
		 FROM tasks WHERE id = $1`, id,
	).Scan(&t.ID, &t.UserID, &t.Title, &t.Description, &t.Status, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (r *PgTaskRepo) ListByUser(ctx context.Context, userID int64) ([]domain.Task, error) {
	q := Conn(ctx, r.db)

	rows, err := q.QueryContext(ctx,
		`SELECT id, user_id, title, description, status, created_at, updated_at
		 FROM tasks WHERE user_id = $1 ORDER BY created_at DESC`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []domain.Task
	for rows.Next() {
		var t domain.Task
		if err := rows.Scan(&t.ID, &t.UserID, &t.Title, &t.Description, &t.Status, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

func (r *PgTaskRepo) Update(ctx context.Context, task *domain.Task) error {
	q := Conn(ctx, r.db)

	_, err := q.ExecContext(ctx,
		`UPDATE tasks SET title = $1, description = $2, status = $3, updated_at = NOW()
		 WHERE id = $4`,
		task.Title, task.Description, task.Status, task.ID,
	)
	return err
}

func (r *PgTaskRepo) Delete(ctx context.Context, id int64) error {
	q := Conn(ctx, r.db)

	_, err := q.ExecContext(ctx, `DELETE FROM tasks WHERE id = $1`, id)
	return err
}
