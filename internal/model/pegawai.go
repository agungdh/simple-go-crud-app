package model

import "github.com/uptrace/bun"

// Pegawai is an employee record.
//
// TanggalLahir is stored as epoch milliseconds (UTC), following the project
// convention for time values. The HTTP layer converts it to/from YYYY-MM-DD.
type Pegawai struct {
	bun.BaseModel `bun:"table:pegawai,alias:p"`

	Base

	Nama         string `bun:"nama,notnull"`
	NIP          string `bun:"nip,notnull"`
	TanggalLahir int64  `bun:"tanggal_lahir,notnull"`
	Alamat       string `bun:"alamat,notnull"`
	JenisKelamin string `bun:"jenis_kelamin,notnull"`
}
