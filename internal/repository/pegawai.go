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
	// ErrNotFound is returned when a pegawai row does not exist.
	ErrNotFound = errors.New("pegawai tidak ditemukan")
	// ErrDuplicateNIP is returned when the NIP is already used by another row.
	ErrDuplicateNIP = errors.New("NIP sudah terdaftar")
)

// PegawaiRepository persists pegawai rows in SQLite.
type PegawaiRepository struct {
	db *bun.DB
}

// NewPegawai returns a repository that uses db.
func NewPegawai(db *bun.DB) *PegawaiRepository {
	return &PegawaiRepository{db: db}
}

// joinBidang adds the bidang join so aliased bidang columns are available.
func joinBidang(q *bun.SelectQuery) *bun.SelectQuery {
	return q.Join("LEFT JOIN bidang AS b ON b.id = p.bidang_id")
}

// withBidang joins the bidang table and exposes its name as bidang_nama.
func withBidang(q *bun.SelectQuery) *bun.SelectQuery {
	return joinBidang(q.Column("p.*").ColumnExpr("b.nama AS bidang_nama"))
}

// PegawaiQuery describes a server-side DataTables request.
type PegawaiQuery struct {
	Search   string
	OrderBy  string
	OrderDir string
	Limit    int
	Offset   int
}

// Query returns a filtered, sorted page of pegawai along with the total and
// filtered row counts expected by DataTables server-side processing.
func (r *PegawaiRepository) Query(ctx context.Context, q PegawaiQuery) ([]model.Pegawai, int64, int64, error) {
	total, err := r.db.NewSelect().Model((*model.Pegawai)(nil)).Count(ctx)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("count pegawai: %w", err)
	}

	filtered, err := joinBidang(applySearch(r.db.NewSelect().Model((*model.Pegawai)(nil)), q.Search)).Count(ctx)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("count filtered pegawai: %w", err)
	}

	items := make([]model.Pegawai, 0)
	list := withBidang(applySearch(r.db.NewSelect().Model(&items), q.Search)).
		OrderExpr(q.OrderBy + " " + q.OrderDir).
		Limit(int64(q.Limit)).
		Offset(int64(q.Offset))
	if err := list.Scan(ctx); err != nil {
		return nil, 0, 0, fmt.Errorf("query pegawai: %w", err)
	}

	return items, total, filtered, nil
}

func applySearch(q *bun.SelectQuery, search string) *bun.SelectQuery {
	search = strings.TrimSpace(search)
	if search == "" {
		return q
	}

	pattern := "%" + strings.ToLower(escapeLike(search)) + "%"
	return q.WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
		return q.
			Where("LOWER(p.nip) LIKE ? ESCAPE '\\'", pattern).
			WhereOr("LOWER(p.nama) LIKE ? ESCAPE '\\'", pattern).
			WhereOr("LOWER(p.jabatan) LIKE ? ESCAPE '\\'", pattern).
			WhereOr("LOWER(b.kode) LIKE ? ESCAPE '\\'", pattern).
			WhereOr("LOWER(b.nama) LIKE ? ESCAPE '\\'", pattern).
			WhereOr("LOWER(p.email) LIKE ? ESCAPE '\\'", pattern).
			WhereOr("LOWER(p.telepon) LIKE ? ESCAPE '\\'", pattern).
			WhereOr("LOWER(p.alamat) LIKE ? ESCAPE '\\'", pattern)
	})
}

func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

// Get returns a single pegawai by ID.
func (r *PegawaiRepository) Get(ctx context.Context, id int64) (*model.Pegawai, error) {
	p := new(model.Pegawai)
	err := withBidang(r.db.NewSelect().Model(p)).Where("p.id = ?", id).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get pegawai: %w", err)
	}
	return p, nil
}

// Create inserts a new pegawai and fills in its generated ID and timestamps.
func (r *PegawaiRepository) Create(ctx context.Context, p *model.Pegawai) error {
	now := time.Now().UTC()
	p.CreatedAt = now
	p.UpdatedAt = now

	_, err := r.db.NewInsert().Model(p).Returning("id").Exec(ctx)
	if err != nil {
		return mapWriteError(err, "create pegawai")
	}
	return nil
}

// Update replaces the mutable fields of an existing pegawai.
func (r *PegawaiRepository) Update(ctx context.Context, p *model.Pegawai) error {
	p.UpdatedAt = time.Now().UTC()

	res, err := r.db.NewUpdate().
		Model(p).
		Column("nip", "nama", "jabatan", "bidang_id", "email", "telepon", "alamat", "updated_at").
		Where("p.id = ?", p.ID).
		Exec(ctx)
	if err != nil {
		return mapWriteError(err, "update pegawai")
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update pegawai rows: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

// Delete removes a pegawai by ID.
func (r *PegawaiRepository) Delete(ctx context.Context, id int64) error {
	res, err := r.db.NewDelete().Model((*model.Pegawai)(nil)).Where("p.id = ?", id).Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete pegawai: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete pegawai rows: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

// NIPExists reports whether nip is already stored, optionally ignoring the row
// with the given ID (useful while updating).
func (r *PegawaiRepository) NIPExists(ctx context.Context, nip string, excludeID int64) (bool, error) {
	q := r.db.NewSelect().Model((*model.Pegawai)(nil)).Where("p.nip = ?", nip)
	if excludeID != 0 {
		q = q.Where("p.id <> ?", excludeID)
	}
	exists, err := q.Exists(ctx)
	if err != nil {
		return false, fmt.Errorf("check nip: %w", err)
	}
	return exists, nil
}

func mapWriteError(err error, op string) error {
	if isUniqueViolation(err) {
		return ErrDuplicateNIP
	}
	return fmt.Errorf("%s: %w", op, err)
}

func isUniqueViolation(err error) bool {
	// modernc.org/sqlite reports constraint failures as text; checking the
	// message keeps the repository independent from driver-specific types.
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique constraint") || strings.Contains(msg, "constraint failed: unique")
}
