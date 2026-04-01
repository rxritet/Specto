package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/rxritet/Specto/internal/database"
	"github.com/rxritet/Specto/internal/domain"
)

type BankingService struct {
	repo    domain.AccountRepository
	txRepo  domain.TransferRepository
	db      database.DBTXProvider // For transactions
}

func NewBankingService(repo domain.AccountRepository, txRepo domain.TransferRepository, db database.DBTXProvider) *BankingService {
	return &BankingService{
		repo:   repo,
		txRepo: txRepo,
		db:     db,
	}
}

func (s *BankingService) GetUserAccounts(ctx context.Context, userID int64) ([]*domain.Account, error) {
	return s.repo.ListByUserID(ctx, userID)
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

func (s *BankingService) Transfer(ctx context.Context, senderUserID int64, req domain.CreateTransferRequest, senderAccountID int64) (*domain.Transfer, error) {
	// Execute within transaction
	var transfer *domain.Transfer
	
	err := database.WithTx(ctx, s.db, func(ctx context.Context) error {
		// 1. Get sender account and verify ownership
		senderAcc, err := s.repo.GetByID(ctx, senderAccountID)
		if err != nil {
			return fmt.Errorf("sender account not found: %w", err)
		}
		if senderAcc.UserID != senderUserID {
			return errors.New("sender does not own the account")
		}

		// 2. Check balance
		if senderAcc.Balance < req.Amount {
			return errors.New("insufficient funds")
		}

		// 3. Get receiver account
		receiverAcc, err := s.repo.GetByID(ctx, req.ReceiverAccountID)
		if err != nil {
			return fmt.Errorf("receiver account not found: %w", err)
		}

		// 4. Verify currencies match (Bank policy)
		if senderAcc.Currency != req.Currency || receiverAcc.Currency != req.Currency {
			return errors.New("currency mismatch")
		}

		// 5. Update balances
		if err := s.repo.UpdateBalance(ctx, senderAcc.ID, -req.Amount); err != nil {
			return err
		}
		if err := s.repo.UpdateBalance(ctx, receiverAcc.ID, req.Amount); err != nil {
			return err
		}

		// 6. Record transfer
		transfer = &domain.Transfer{
			SenderAccountID:   senderAcc.ID,
			ReceiverAccountID: receiverAcc.ID,
			Amount:            req.Amount,
			Currency:          req.Currency,
			Description:       req.Description,
		}
		return s.txRepo.Create(ctx, transfer)
	})

	if err != nil {
		return nil, err
	}
	return transfer, nil
}
