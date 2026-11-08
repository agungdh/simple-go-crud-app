// Package repository provides data access helpers backed by Bun.
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/uptrace/bun"

	"simple-go-crud-app/internal/model"
)

var (
	// ErrDuplicateKode is returned when the bidang kode is already used.
	ErrDuplicateKode = errors.New("kode bidang sudah terdaftar")
	// ErrInUse is returned when a bidang is still referenced by a pegawai.
	ErrInUse = errors.New("bidang masih digunakan pegawai")
)

// BidangRepository persists bidang rows in SQLite.
type BidangRepository struct {
	db *bun.DB
}

// NewBidang returns a repository that uses db.
func NewBidang(db *bun.DB) *BidangRepository {
	return &BidangRepository{db: db}
}

// List returns every bidang ordered by name.
func (r *BidangRepository) List(ctx context.Context) ([]model.Bidang, error) {
	items := make([]model.Bidang, 0)
	if err := r.db.NewSelect().Model(&items).OrderExpr("b.nama ASC").Scan(ctx); err != nil {
		return nil, fmt.Errorf("list bidang: %w", err)
	}
	return items, nil
}

// BidangQuery describes a server-side DataTables request.
type BidangQuery struct {
	Search   string
	OrderBy  string
	OrderDir string
	Limit    int
	Offset   int
}

// Query returns a filtered, sorted page of bidang along with the total and
// filtered row counts expected by DataTables server-side processing.
func (r *BidangRepository) Query(ctx context.Context, q BidangQuery) ([]model.Bidang, int64, int64, error) {
	total, err := r.db.NewSelect().Model((*model.Bidang)(nil)).Count(ctx)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("count bidang: %w", err)
	}

	filtered, err := applyBidangSearch(r.db.NewSelect().Model((*model.Bidang)(nil)), q.Search).Count(ctx)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("count filtered bidang: %w", err)
	}

	items := make([]model.Bidang, 0)
	list := applyBidangSearch(r.db.NewSelect().Model(&items), q.Search).
		OrderExpr(q.OrderBy + " " + q.OrderDir).
		Limit(int64(q.Limit)).
		Offset(int64(q.Offset))
	if err := list.Scan(ctx); err != nil {
		return nil, 0, 0, fmt.Errorf("query bidang: %w", err)
	}

	return items, total, filtered, nil
}

// Search returns up to limit bidang matching keyword in kode or nama. It backs
// the Select2 autocomplete endpoint.
func (r *BidangRepository) Search(ctx context.Context, keyword string, limit int) ([]model.Bidang, error) {
	items := make([]model.Bidang, 0)
	if strings.TrimSpace(keyword) == "" {
		return items, nil
	}
	if err := applyBidangSearch(r.db.NewSelect().Model(&items), keyword).
		OrderExpr("b.nama ASC").
		Limit(int64(limit)).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("search bidang: %w", err)
	}
	return items, nil
}

func applyBidangSearch(q *bun.SelectQuery, search string) *bun.SelectQuery {
	search = strings.TrimSpace(search)
	if search == "" {
		return q
	}

	pattern := "%" + escapeLike(search) + "%"
	return q.WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
		return q.
			Where("b.kode LIKE ? ESCAPE '\\'", pattern).
			WhereOr("b.nama LIKE ? ESCAPE '\\'", pattern)
	})
}

// Get returns a single bidang by ID.
func (r *BidangRepository) Get(ctx context.Context, id int64) (*model.Bidang, error) {
	b := new(model.Bidang)
	err := r.db.NewSelect().Model(b).Where("b.id = ?", id).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get bidang: %w", err)
	}
	return b, nil
}

// Create inserts a new bidang and fills in its generated ID and timestamps.
func (r *BidangRepository) Create(ctx context.Context, b *model.Bidang) error {
	now := time.Now().UTC()
	b.CreatedAt = now
	b.UpdatedAt = now

	_, err := r.db.NewInsert().Model(b).Returning("id").Exec(ctx)
	if err != nil {
		return mapBidangWriteError(err, "create bidang")
	}
	return nil
}

// Update replaces the mutable fields of an existing bidang.
func (r *BidangRepository) Update(ctx context.Context, b *model.Bidang) error {
	b.UpdatedAt = time.Now().UTC()

	res, err := r.db.NewUpdate().
		Model(b).
		Column("kode", "nama", "updated_at").
		Where("b.id = ?", b.ID).
		Exec(ctx)
	if err != nil {
		return mapBidangWriteError(err, "update bidang")
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update bidang rows: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

// Delete removes a bidang by ID.
func (r *BidangRepository) Delete(ctx context.Context, id int64) error {
	res, err := r.db.NewDelete().Model((*model.Bidang)(nil)).Where("b.id = ?", id).Exec(ctx)
	if err != nil {
		return mapBidangWriteError(err, "delete bidang")
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete bidang rows: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

func mapBidangWriteError(err error, op string) error {
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "unique constraint") || strings.Contains(msg, "constraint failed: unique"):
		return ErrDuplicateKode
	case strings.Contains(msg, "foreign key constraint"):
		return ErrInUse
	}
	return fmt.Errorf("%s: %w", op, err)
}
