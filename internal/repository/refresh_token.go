package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/uptrace/bun"

	"simple-go-crud-app/internal/apperr"
	"simple-go-crud-app/internal/model"
)

// RefreshTokenRepository persists opaque refresh tokens. Rows are hard deleted
// on logout and rotation because this is a technical table, not a business
// entity with soft delete.
type RefreshTokenRepository struct {
	db *bun.DB
}

func NewRefreshTokenRepository(db *bun.DB) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

// Create inserts a refresh token record and sets its generated id.
func (r *RefreshTokenRepository) Create(ctx context.Context, token *model.RefreshToken) error {
	if _, err := r.db.NewInsert().
		Model(token).
		Returning("id").
		Exec(ctx); err != nil {
		return fmt.Errorf("insert refresh token: %w", err)
	}
	return nil
}

// GetActiveByHash returns a refresh token that belongs to user and has not
// expired. now is the current UTC time formatted as RFC 3339.
func (r *RefreshTokenRepository) GetActiveByHash(ctx context.Context, hash, now string) (*model.RefreshToken, error) {
	token := new(model.RefreshToken)

	err := r.db.NewSelect().
		Model(token).
		Column("id", "user_id", "token_hash", "expires_at").
		Where("rt.token_hash = ?", hash).
		Where("rt.expires_at > ?", now).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperr.ErrNotFound
		}
		return nil, fmt.Errorf("get refresh token: %w", err)
	}
	return token, nil
}

// RevokeByHash deletes the refresh token if it belongs to user. A missing row
// is reported as apperr.ErrNotFound so logout can stay idempotent.
func (r *RefreshTokenRepository) RevokeByHash(ctx context.Context, hash string, userID int64) error {
	res, err := r.db.NewDelete().
		Model((*model.RefreshToken)(nil)).
		Where("token_hash = ?", hash).
		Where("user_id = ?", userID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("revoke refresh token rows affected: %w", err)
	}
	if affected == 0 {
		return apperr.ErrNotFound
	}
	return nil
}

// Rotate deletes the old refresh token and inserts its replacement in one
// transaction. If the old token is gone, it returns apperr.ErrNotFound, which
// prevents a concurrent or replayed refresh from succeeding.
func (r *RefreshTokenRepository) Rotate(ctx context.Context, oldHash string, userID int64, next *model.RefreshToken) error {
	return r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		res, err := tx.NewDelete().
			Model((*model.RefreshToken)(nil)).
			Where("token_hash = ?", oldHash).
			Where("user_id = ?", userID).
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("consume refresh token: %w", err)
		}

		affected, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("consume refresh token rows affected: %w", err)
		}
		if affected == 0 {
			return apperr.ErrNotFound
		}

		if _, err := tx.NewInsert().
			Model(next).
			Returning("id").
			Exec(ctx); err != nil {
			return fmt.Errorf("insert rotated refresh token: %w", err)
		}
		return nil
	})
}
