package bidang

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/uptrace/bun"

	"simple-go-crud-app/internal/shared/apperr"
	"simple-go-crud-app/internal/shared/database"
	"simple-go-crud-app/internal/shared/entity"
	"simple-go-crud-app/internal/shared/search"
)

// Repository persists bidang records.
type Repository struct {
	db *bun.DB
}

func NewRepository(db *bun.DB) *Repository {
	return &Repository{db: db}
}

// List returns active bidang ordered by id. When query is not empty, only rows
// whose search_text matches are returned. Callers must enforce the
// three-character minimum required by the trigram tokenizer.
func (r *Repository) List(ctx context.Context, query string) ([]Entity, error) {
	items := make([]Entity, 0)

	q := r.db.NewSelect().
		Model(&items).
		Column(entity.PublicColumns(r.db, (*Entity)(nil))...).
		Order("b.id ASC")
	q = entity.Active(q)

	if query != "" {
		q = q.Join("JOIN bidang_fts AS f ON f.rowid = b.id").
			Where("f.search_text MATCH ?", search.Phrase(query))
	}

	if err := q.Scan(ctx); err != nil {
		return nil, fmt.Errorf("list bidang: %w", err)
	}
	return items, nil
}

// GetByID returns one active bidang or apperr.ErrNotFound.
func (r *Repository) GetByID(ctx context.Context, id int64) (*Entity, error) {
	bidang := new(Entity)

	q := r.db.NewSelect().
		Model(bidang).
		Column(entity.PublicColumns(r.db, (*Entity)(nil))...).
		Where("b.id = ?", id)
	q = entity.Active(q)

	if err := q.Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperr.ErrNotFound
		}
		return nil, fmt.Errorf("get bidang %d: %w", id, err)
	}
	return bidang, nil
}

// ExistsActive reports whether an active bidang with id exists. Other domains
// use it to validate references without loading the whole record.
func (r *Repository) ExistsActive(ctx context.Context, id int64) (bool, error) {
	q := r.db.NewSelect().
		Model((*Entity)(nil)).
		Where("b.id = ?", id)
	q = entity.Active(q)

	exists, err := q.Exists(ctx)
	if err != nil {
		return false, fmt.Errorf("check bidang %d: %w", id, err)
	}
	return exists, nil
}

// NameExists reports whether an active bidang already uses name. A positive
// excludeID is ignored, which lets updates keep their own name.
func (r *Repository) NameExists(ctx context.Context, name string, excludeID int64) (bool, error) {
	q := r.db.NewSelect().
		Model((*Entity)(nil)).
		Where("b.bidang = ?", name)
	q = entity.Active(q)
	if excludeID > 0 {
		q = q.Where("b.id <> ?", excludeID)
	}

	exists, err := q.Exists(ctx)
	if err != nil {
		return false, fmt.Errorf("check bidang name %q: %w", name, err)
	}
	return exists, nil
}

// Create inserts a new bidang and sets its generated id.
func (r *Repository) Create(ctx context.Context, bidang *Entity) error {
	_, err := r.db.NewInsert().
		Model(bidang).
		Returning("id").
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("insert bidang: %w", err)
	}
	return nil
}

// Update replaces the editable fields of an active bidang.
func (r *Repository) Update(ctx context.Context, bidang *Entity) error {
	res, err := r.db.NewUpdate().
		Model(bidang).
		Column("bidang", "updated_at", "updated_by").
		WherePK().
		Where("deleted_at IS NULL").
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("update bidang %d: %w", bidang.ID, err)
	}
	return database.AffectedOrNotFound(res, fmt.Sprintf("update bidang %d", bidang.ID))
}

// Delete soft deletes an active bidang.
func (r *Repository) Delete(ctx context.Context, id, actorID int64) error {
	bidang := &Entity{Base: entity.Base{ID: id}}
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
	return database.AffectedOrNotFound(res, fmt.Sprintf("delete bidang %d", id))
}
