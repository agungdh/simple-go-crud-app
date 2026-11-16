package pegawai

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

// Response is the pegawai representation returned to clients. TanggalLahir is
// formatted as DD-MM-YYYY, while storage stays ISO-8601.
type Response struct {
	ID           int64        `json:"id"`
	BidangID     int64        `json:"bidang_id"`
	Nama         string       `json:"nama"`
	NIP          string       `json:"nip"`
	TanggalLahir string       `json:"tanggal_lahir"`
	Alamat       string       `json:"alamat"`
	JenisKelamin JenisKelamin `json:"jenis_kelamin"`
}

// Input is the request body for creating or updating a pegawai.
type Input struct {
	BidangID     int64             `json:"bidang_id"`
	Nama         string            `json:"nama"`
	NIP          string            `json:"nip"`
	TanggalLahir string            `json:"tanggal_lahir"`
	Alamat       string            `json:"alamat"`
	JenisKelamin JenisKelaminInput `json:"jenis_kelamin"`
}
