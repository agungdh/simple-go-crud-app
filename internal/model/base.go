package model

import (
	"time"

	"github.com/uptrace/bun"
)

// Base is embedded by every entity. It provides the primary key and audit
// columns. Audit columns are intentionally hidden from JSON: they are internal
// bookkeeping and must never be exposed to the frontend.
//
// Timestamps are stored as epoch milliseconds (UTC) and are nullable, with no
// database default. Values are set by the application.
type Base struct {
	ID int64 `bun:"id,pk,autoincrement" json:"id"`

	CreatedAt *int64 `bun:"created_at" json:"-"`
	CreatedBy *int64 `bun:"created_by" json:"-"`
	UpdatedAt *int64 `bun:"updated_at" json:"-"`
	UpdatedBy *int64 `bun:"updated_by" json:"-"`
	DeletedAt *int64 `bun:"deleted_at" json:"-"`
	DeletedBy *int64 `bun:"deleted_by" json:"-"`
}

// NowMillis returns the current UTC time as epoch milliseconds.
func NowMillis() int64 {
	return time.Now().UTC().UnixMilli()
}

// MarkCreated fills created_at and created_by. updated_* and deleted_* stay nil.
func (b *Base) MarkCreated(actorID int64) {
	now := NowMillis()
	b.CreatedAt = &now
	if actorID > 0 {
		b.CreatedBy = &actorID
	}
}

// MarkUpdated fills updated_at and updated_by.
func (b *Base) MarkUpdated(actorID int64) {
	now := NowMillis()
	b.UpdatedAt = &now
	if actorID > 0 {
		b.UpdatedBy = &actorID
	}
}

// MarkDeleted performs a soft delete by filling deleted_at and deleted_by.
func (b *Base) MarkDeleted(actorID int64) {
	now := NowMillis()
	b.DeletedAt = &now
	if actorID > 0 {
		b.DeletedBy = &actorID
	}
}

// IsDeleted reports whether the entity has been soft deleted.
func (b *Base) IsDeleted() bool {
	return b.DeletedAt != nil
}

// Active scopes a select query to non-deleted rows only.
//
// It must be applied to every read query, including list/datatable queries:
//
//	q := db.NewSelect().Model(&items)
//	q = model.Active(q)
//
// For joins, each joined table that has soft deletes must be filtered too.
func Active(q *bun.SelectQuery) *bun.SelectQuery {
	return q.Where("?TableAlias.deleted_at IS NULL")
}
