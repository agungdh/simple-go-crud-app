package model

import (
	"time"

	"github.com/uptrace/bun"
)

// Bidang represents one organizational unit.
type Bidang struct {
	bun.BaseModel `bun:"table:bidang,alias:b" json:"-"`

	ID        int64     `bun:"id,pk,autoincrement" json:"id"`
	Kode      string    `bun:"kode,notnull,unique" json:"kode"`
	Nama      string    `bun:"nama,notnull" json:"nama"`
	CreatedAt time.Time `bun:"created_at,notnull" json:"created_at"`
	UpdatedAt time.Time `bun:"updated_at,notnull" json:"updated_at"`
}
