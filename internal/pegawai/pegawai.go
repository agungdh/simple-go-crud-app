// Package pegawai implements the pegawai (employee) domain end to end: model,
// persistence, business rules, DTOs, and HTTP handlers.
package pegawai

import (
	"github.com/uptrace/bun"

	"simple-go-crud-app/internal/shared/entity"
)

// Entity is an employee record.
//
// TanggalLahir is stored as ISO-8601 text (YYYY-MM-DD). Only audit columns use
// epoch milliseconds; business dates stay TEXT.
type Entity struct {
	bun.BaseModel `bun:"table:pegawai,alias:p"`

	entity.Base

	BidangID     int64  `bun:"bidang_id,notnull"`
	Nama         string `bun:"nama,notnull"`
	NIP          string `bun:"nip,notnull"`
	TanggalLahir string `bun:"tanggal_lahir,notnull"`
	Alamat       string `bun:"alamat,notnull"`
	IsMale       bool   `bun:"is_male,notnull"`
}
