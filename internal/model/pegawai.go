package model

import "github.com/uptrace/bun"

// Pegawai is an employee record.
//
// TanggalLahir is stored as ISO-8601 text (YYYY-MM-DD). Only audit columns use
// epoch milliseconds; business dates stay TEXT.
type Pegawai struct {
	bun.BaseModel `bun:"table:pegawai,alias:p"`

	Base

	Nama         string `bun:"nama,notnull"`
	NIP          string `bun:"nip,notnull"`
	TanggalLahir string `bun:"tanggal_lahir,notnull"`
	Alamat       string `bun:"alamat,notnull"`
	IsMale       bool   `bun:"is_male,notnull"`
}
