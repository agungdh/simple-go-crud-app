package dto

// JenisKelaminInput is the label accepted on requests. Comparison is
// case-insensitive on the server; the contract documents these lowercase
// values as canonical.
type JenisKelaminInput string

// JenisKelamin is the label used in responses.
type JenisKelamin string

const (
	InputPria   JenisKelaminInput = "pria"
	InputWanita JenisKelaminInput = "wanita"

	Pria   JenisKelamin = "Pria"
	Wanita JenisKelamin = "Wanita"
)

// Pegawai is the pegawai representation returned to clients. TanggalLahir is
// formatted as DD-MM-YYYY, while storage stays ISO-8601.
type Pegawai struct {
	ID           int64        `json:"id"`
	BidangID     int64        `json:"bidang_id"`
	Nama         string       `json:"nama"`
	NIP          string       `json:"nip"`
	TanggalLahir string       `json:"tanggal_lahir"`
	Alamat       string       `json:"alamat"`
	JenisKelamin JenisKelamin `json:"jenis_kelamin"`
}

// PegawaiInput is the request body for creating or updating a pegawai.
type PegawaiInput struct {
	BidangID     int64             `json:"bidang_id"`
	Nama         string            `json:"nama"`
	NIP          string            `json:"nip"`
	TanggalLahir string            `json:"tanggal_lahir"`
	Alamat       string            `json:"alamat"`
	JenisKelamin JenisKelaminInput `json:"jenis_kelamin"`
}

// PegawaiResponse is the single-object envelope.
type PegawaiResponse struct {
	Data Pegawai `json:"data"`
}

// PegawaiListResponse is the collection envelope. Data must marshal as [] when
// empty, never null.
type PegawaiListResponse struct {
	Data []Pegawai `json:"data"`
}
