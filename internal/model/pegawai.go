// Package model contains the database entities used by the application.
package model

import (
	"time"

	"github.com/uptrace/bun"
)

// Pegawai represents one employee record.
type Pegawai struct {
	bun.BaseModel `bun:"table:pegawai,alias:p" json:"-"`

	ID         int64     `bun:"id,pk,autoincrement" json:"id"`
	NIP        string    `bun:"nip,notnull,unique" json:"nip"`
	Nama       string    `bun:"nama,notnull" json:"nama"`
	Jabatan    string    `bun:"jabatan,notnull" json:"jabatan"`
	BidangID   int64     `bun:"bidang_id,notnull" json:"bidang_id"`
	BidangNama string    `bun:"bidang_nama,scanonly" json:"bidang_nama"`
	Email      string    `bun:"email,notnull" json:"email"`
	Telepon    string    `bun:"telepon,notnull" json:"telepon"`
	Alamat     string    `bun:"alamat,notnull" json:"alamat"`
	CreatedAt  time.Time `bun:"created_at,notnull" json:"created_at"`
	UpdatedAt  time.Time `bun:"updated_at,notnull" json:"updated_at"`
}
