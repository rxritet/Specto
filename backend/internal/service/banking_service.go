package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/rxritet/Specto/internal/database"
	"github.com/rxritet/Specto/internal/domain"
)

type BankingService struct {
	repo   domain.AccountRepository
	txRepo domain.TransferRepository
	db     *sql.DB
}

func NewBankingService(repo domain.AccountRepository, txRepo domain.TransferRepository, db *sql.DB) *BankingService {
	return &BankingService{
		repo:   repo,
		txRepo: txRepo,
		db:     db,
	}
}

func (s *BankingService) GetUserAccounts(ctx context.Context, userID int64) ([]*domain.Account, error) {
	return s.repo.GetByUserID(ctx, userID)
}

func (s *BankingService) CreateAccount(ctx context.Context, userID int64, currency string) (*domain.Account, error) {
	acc := &domain.Account{
		UserID:   userID,
		Currency: currency,
		Balance:  0,
	}
	if err := s.repo.Create(ctx, acc); err != nil {
		return nil, err
	}
	return acc, nil
}

func (s *BankingService) loadAndValidateTransferAccounts(ctx context.Context, senderUserID, senderAccountID int64, req domain.CreateTransferRequest) (*domain.Account, *domain.Account, error) {
	senderAcc, err := s.repo.GetByID(ctx, senderAccountID)
	if err != nil {
		return nil, nil, fmt.Errorf("sender account not found: %w", err)
	}
	if senderAcc.UserID != senderUserID {
		return nil, nil, errors.New("sender does not own the account")
	}
	if senderAcc.Balance < req.Amount {
		return nil, nil, errors.New("insufficient funds")
	}

	receiverAcc, err := s.repo.GetByID(ctx, req.ReceiverAccountID)
	if err != nil {
		return nil, nil, fmt.Errorf("receiver account not found: %w", err)
	}
	if senderAcc.Currency != req.Currency || receiverAcc.Currency != req.Currency {
		return nil, nil, errors.New("currency mismatch")
	}

	return senderAcc, receiverAcc, nil
}

func (s *BankingService) applyTransferBalances(ctx context.Context, senderID, receiverID, amount int64) error {
	if err := s.repo.UpdateBalance(ctx, senderID, -amount); err != nil {
		return err
	}
	if err := s.repo.UpdateBalance(ctx, receiverID, amount); err != nil {
		return err
	}
	return nil
}

func (s *BankingService) Transfer(ctx context.Context, senderUserID int64, req domain.CreateTransferRequest, senderAccountID int64) (*domain.Transfer, error) {
	// Execute within transaction
	var transfer *domain.Transfer

	err := database.RunInTx(ctx, s.db, func(txCtx context.Context) error {
		senderAcc, receiverAcc, err := s.loadAndValidateTransferAccounts(txCtx, senderUserID, senderAccountID, req)
		if err != nil {
			return err
		}
		if err := s.applyTransferBalances(txCtx, senderAcc.ID, receiverAcc.ID, req.Amount); err != nil {
			return err
		}

		transfer = &domain.Transfer{
			SenderAccountID:   senderAcc.ID,
			ReceiverAccountID: receiverAcc.ID,
			Amount:            req.Amount,
			Currency:          req.Currency,
			Description:       req.Description,
		}
		return s.txRepo.Create(txCtx, transfer)
	})

	if err != nil {
		return nil, err
	}
	return transfer, nil
}
