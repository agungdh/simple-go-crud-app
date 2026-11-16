package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/uptrace/bun"

	"simple-go-crud-app/internal/apperr"
	"simple-go-crud-app/internal/model"
)

// PegawaiRepository persists pegawai records.
type PegawaiRepository struct {
	db *bun.DB
}

func NewPegawaiRepository(db *bun.DB) *PegawaiRepository {
	return &PegawaiRepository{db: db}
}

// List returns active pegawai ordered by id. When query is not empty, only
// rows whose search_text matches are returned. Callers must enforce the
// three-character minimum required by the trigram tokenizer.
func (r *PegawaiRepository) List(ctx context.Context, query string) ([]model.Pegawai, error) {
	items := make([]model.Pegawai, 0)

	q := r.db.NewSelect().
		Model(&items).
		Column(model.PublicColumns(r.db, (*model.Pegawai)(nil))...).
		Order("p.id ASC")
	q = model.Active(q)

	if query != "" {
		q = q.Join("JOIN pegawai_fts AS f ON f.rowid = p.id").
			Where("f.search_text MATCH ?", ftsPhrase(query))
	}

	if err := q.Scan(ctx); err != nil {
		return nil, fmt.Errorf("list pegawai: %w", err)
	}
	return items, nil
}

// ftsPhrase wraps user input in a quoted FTS5 phrase so that operators such as
// AND, OR, NOT, and * are matched as literal text.
func ftsPhrase(query string) string {
	return `"` + strings.ReplaceAll(query, `"`, `""`) + `"`
}

// GetByID returns one active pegawai or apperr.ErrNotFound.
func (r *PegawaiRepository) GetByID(ctx context.Context, id int64) (*model.Pegawai, error) {
	pegawai := new(model.Pegawai)

	q := r.db.NewSelect().
		Model(pegawai).
		Column(model.PublicColumns(r.db, (*model.Pegawai)(nil))...).
		Where("p.id = ?", id)
	q = model.Active(q)

	if err := q.Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperr.ErrNotFound
		}
		return nil, fmt.Errorf("get pegawai %d: %w", id, err)
	}
	return pegawai, nil
}

// Create inserts a new pegawai and sets its generated id.
func (r *PegawaiRepository) Create(ctx context.Context, pegawai *model.Pegawai) error {
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
func (r *PegawaiRepository) Update(ctx context.Context, pegawai *model.Pegawai) error {
	res, err := r.db.NewUpdate().
		Model(pegawai).
		Column("bidang_id", "nama", "nip", "tanggal_lahir", "alamat", "is_male", "updated_at", "updated_by").
		WherePK().
		Where("deleted_at IS NULL").
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("update pegawai %d: %w", pegawai.ID, err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update pegawai %d rows affected: %w", pegawai.ID, err)
	}
	if affected == 0 {
		return apperr.ErrNotFound
	}
	return nil
}

// Delete soft deletes an active pegawai.
func (r *PegawaiRepository) Delete(ctx context.Context, id, actorID int64) error {
	pegawai := &model.Pegawai{Base: model.Base{ID: id}}
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

	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete pegawai %d rows affected: %w", id, err)
	}
	if affected == 0 {
		return apperr.ErrNotFound
	}
	return nil
}
