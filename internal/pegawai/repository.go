package pegawai

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

// Repository persists pegawai records.
type Repository struct {
	db *bun.DB
}

func NewRepository(db *bun.DB) *Repository {
	return &Repository{db: db}
}

// List returns active pegawai ordered by id. When query is not empty, only rows
// whose search_text matches are returned. Callers must enforce the
// three-character minimum required by the trigram tokenizer.
func (r *Repository) List(ctx context.Context, query string) ([]Entity, error) {
	items := make([]Entity, 0)

	q := r.db.NewSelect().
		Model(&items).
		Column(entity.PublicColumns(r.db, (*Entity)(nil))...).
		Order("p.id ASC")
	q = entity.Active(q)

	if query != "" {
		q = q.Join("JOIN pegawai_fts AS f ON f.rowid = p.id").
			Where("f.search_text MATCH ?", search.Phrase(query))
	}

	if err := q.Scan(ctx); err != nil {
		return nil, fmt.Errorf("list pegawai: %w", err)
	}
	return items, nil
}

// GetByID returns one active pegawai or apperr.ErrNotFound.
func (r *Repository) GetByID(ctx context.Context, id int64) (*Entity, error) {
	pegawai := new(Entity)

	q := r.db.NewSelect().
		Model(pegawai).
		Column(entity.PublicColumns(r.db, (*Entity)(nil))...).
		Where("p.id = ?", id)
	q = entity.Active(q)

	if err := q.Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperr.ErrNotFound
		}
		return nil, fmt.Errorf("get pegawai %d: %w", id, err)
	}
	return pegawai, nil
}

// NIPExists reports whether an active pegawai already uses nip. A positive
// excludeID is ignored, which lets updates keep their own NIP.
func (r *Repository) NIPExists(ctx context.Context, nip string, excludeID int64) (bool, error) {
	q := r.db.NewSelect().
		Model((*Entity)(nil)).
		Where("p.nip = ?", nip)
	q = entity.Active(q)
	if excludeID > 0 {
		q = q.Where("p.id <> ?", excludeID)
	}

	exists, err := q.Exists(ctx)
	if err != nil {
		return false, fmt.Errorf("check pegawai nip %q: %w", nip, err)
	}
	return exists, nil
}

// Create inserts a new pegawai and sets its generated id.
func (r *Repository) Create(ctx context.Context, pegawai *Entity) error {
	_, err := r.db.NewInsert().
		Model(pegawai).
		Returning("id").
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("insert pegawai: %w", err)
	}
	return nil
}

// Update replaces the editable fields of an active pegawai.
func (r *Repository) Update(ctx context.Context, pegawai *Entity) error {
	res, err := r.db.NewUpdate().
		Model(pegawai).
		Column("bidang_id", "nama", "nip", "tanggal_lahir", "alamat", "is_male", "updated_at", "updated_by").
		WherePK().
		Where("deleted_at IS NULL").
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("update pegawai %d: %w", pegawai.ID, err)
	}
	return database.AffectedOrNotFound(res, fmt.Sprintf("update pegawai %d", pegawai.ID))
}

// Delete soft deletes an active pegawai.
func (r *Repository) Delete(ctx context.Context, id, actorID int64) error {
	pegawai := &Entity{Base: entity.Base{ID: id}}
	pegawai.MarkDeleted(actorID)

	res, err := r.db.NewUpdate().
		Model(pegawai).
		Column("deleted_at", "deleted_by").
		WherePK().
		Where("deleted_at IS NULL").
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete pegawai %d: %w", id, err)
	}
	return database.AffectedOrNotFound(res, fmt.Sprintf("delete pegawai %d", id))
}
