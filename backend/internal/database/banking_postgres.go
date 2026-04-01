package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/rxritet/Specto/internal/domain"
)

type PgAccountRepo struct {
	db *sql.DB
}

func NewPgAccountRepo(db *sql.DB) *PgAccountRepo {
	return &PgAccountRepo{db: db}
}

func (r *PgAccountRepo) Create(ctx context.Context, acc *domain.Account) error {
	q := Conn(ctx, r.db)
	return q.QueryRowContext(ctx,
		`INSERT INTO accounts (user_id, currency, balance, created_at, updated_at)
		 VALUES ($1, $2, $3, NOW(), NOW())
		 RETURNING id, created_at, updated_at`,
		acc.UserID, acc.Currency, acc.Balance,
	).Scan(&acc.ID, &acc.CreatedAt, &acc.UpdatedAt)
}

func (r *PgAccountRepo) GetByID(ctx context.Context, id int64) (*domain.Account, error) {
	q := Conn(ctx, r.db)
	acc := &domain.Account{}
	err := q.QueryRowContext(ctx,
		`SELECT id, user_id, currency, balance, created_at, updated_at
		 FROM accounts WHERE id = $1`, id,
	).Scan(&acc.ID, &acc.UserID, &acc.Currency, &acc.Balance, &acc.CreatedAt, &acc.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return acc, nil
}

func (r *PgAccountRepo) ListByUserID(ctx context.Context, userID int64) ([]*domain.Account, error) {
	q := Conn(ctx, r.db)
	rows, err := q.QueryContext(ctx,
		`SELECT id, user_id, currency, balance, created_at, updated_at
		 FROM accounts WHERE user_id = $1 ORDER BY created_at DESC`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []*domain.Account
	for rows.Next() {
		acc := &domain.Account{}
		if err := rows.Scan(&acc.ID, &acc.UserID, &acc.Currency, &acc.Balance, &acc.CreatedAt, &acc.UpdatedAt); err != nil {
			return nil, err
		}
		accounts = append(accounts, acc)
	}
	return accounts, nil
}

func (r *PgAccountRepo) UpdateBalance(ctx context.Context, id int64, amount int64) error {
	q := Conn(ctx, r.db)
	_, err := q.ExecContext(ctx,
		`UPDATE accounts SET balance = balance + $1, updated_at = NOW() WHERE id = $2`,
		amount, id,
	)
	return err
}

type PgTransferRepo struct {
	db *sql.DB
}

func NewPgTransferRepo(db *sql.DB) *PgTransferRepo {
	return &PgTransferRepo{db: db}
}

func (r *PgTransferRepo) Create(ctx context.Context, tr *domain.Transfer) error {
	q := Conn(ctx, r.db)
	return q.QueryRowContext(ctx,
		`INSERT INTO transfers (sender_account_id, receiver_account_id, amount, currency, description, created_at)
		 VALUES ($1, $2, $3, $4, $5, NOW())
		 RETURNING id, created_at`,
		tr.SenderAccountID, tr.ReceiverAccountID, tr.Amount, tr.Currency, tr.Description,
	).Scan(&tr.ID, &tr.CreatedAt)
}

func (r *PgTransferRepo) ListByAccountID(ctx context.Context, accountID int64) ([]*domain.Transfer, error) {
	q := Conn(ctx, r.db)
	rows, err := q.QueryContext(ctx,
		`SELECT id, sender_account_id, receiver_account_id, amount, currency, description, created_at
		 FROM transfers WHERE sender_account_id = $1 OR receiver_account_id = $1
		 ORDER BY created_at DESC`, accountID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transfers []*domain.Transfer
	for rows.Next() {
		tr := &domain.Transfer{}
		if err := rows.Scan(&tr.ID, &tr.SenderAccountID, &tr.ReceiverAccountID, &tr.Amount, &tr.Currency, &tr.Description, &tr.CreatedAt); err != nil {
			return nil, err
		}
		transfers = append(transfers, tr)
	}
	return transfers, nil
}
