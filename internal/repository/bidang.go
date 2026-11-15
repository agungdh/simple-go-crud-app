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

// BidangRepository persists bidang records.
type BidangRepository struct {
	db *bun.DB
}

func NewBidangRepository(db *bun.DB) *BidangRepository {
	return &BidangRepository{db: db}
}

// List returns active bidang ordered by id. When query is not empty, only rows
// whose search_text matches are returned. Callers must enforce the
// three-character minimum required by the trigram tokenizer.
func (r *BidangRepository) List(ctx context.Context, query string) ([]model.Bidang, error) {
	items := make([]model.Bidang, 0)

	q := r.db.NewSelect().
		Model(&items).
		Column(model.PublicColumns(r.db, (*model.Bidang)(nil))...).
		Order("b.id ASC")
	q = model.Active(q)

	if query != "" {
		q = q.Join("JOIN bidang_fts AS f ON f.rowid = b.id").
			Where("f.search_text MATCH ?", ftsPhrase(query))
	}

	if err := q.Scan(ctx); err != nil {
		return nil, fmt.Errorf("list bidang: %w", err)
	}
	return items, nil
}

// GetByID returns one active bidang or apperr.ErrNotFound.
func (r *BidangRepository) GetByID(ctx context.Context, id int64) (*model.Bidang, error) {
	bidang := new(model.Bidang)

	q := r.db.NewSelect().
		Model(bidang).
		Column(model.PublicColumns(r.db, (*model.Bidang)(nil))...).
		Where("b.id = ?", id)
	q = model.Active(q)

	if err := q.Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperr.ErrNotFound
		}
		return nil, fmt.Errorf("get bidang %d: %w", id, err)
	}
	return bidang, nil
}

// Create inserts a new bidang and sets its generated id.
func (r *BidangRepository) Create(ctx context.Context, bidang *model.Bidang) error {
	_, err := r.db.NewInsert().
		Model(bidang).
		Returning("id").
		Exec(ctx)
	if err != nil {
		if isUniqueViolation(err) {
			return apperr.ErrConflict
		}
		return fmt.Errorf("create bidang: %w", err)
	}
	return nil
}

// Update replaces the editable fields of an active bidang.
func (r *BidangRepository) Update(ctx context.Context, bidang *model.Bidang) error {
	res, err := r.db.NewUpdate().
		Model(bidang).
		Column("bidang", "updated_at", "updated_by").
		WherePK().
		Where("deleted_at IS NULL").
		Exec(ctx)
	if err != nil {
		if isUniqueViolation(err) {
			return apperr.ErrConflict
		}
		return fmt.Errorf("update bidang %d: %w", bidang.ID, err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update bidang %d rows affected: %w", bidang.ID, err)
	}
	if affected == 0 {
		return apperr.ErrNotFound
	}
	return nil
}

// Delete soft deletes an active bidang.
func (r *BidangRepository) Delete(ctx context.Context, id, actorID int64) error {
	bidang := &model.Bidang{Base: model.Base{ID: id}}
	bidang.MarkDeleted(actorID)

	res, err := r.db.NewUpdate().
		Model(bidang).
		Column("deleted_at", "deleted_by").
		WherePK().
		Where("deleted_at IS NULL").
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete bidang %d: %w", id, err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete bidang %d rows affected: %w", id, err)
	}
	if affected == 0 {
		return apperr.ErrNotFound
	}
	return nil
}
