package service

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"simple-go-crud-app/internal/apperr"
	"simple-go-crud-app/internal/model"
	"simple-go-crud-app/internal/repository"
)

const (
	dateLayout     = "2006-01-02" // storage
	dateLayoutIndo = "02-01-2006" // DTO, e.g. 17-05-1985

	maxNamaLength   = 150
	maxNIPLength    = 50
	maxAlamatLength = 500
	minSearchLength = 3
	maxSearchLength = 100

	jenisKelaminPria   = "pria"
	jenisKelaminWanita = "wanita"
)

// PegawaiInput is the raw payload accepted from the HTTP layer. Values are
// validated and normalized by the service.
type PegawaiInput struct {
	Nama         string
	NIP          string
	TanggalLahir string
	Alamat       string
	JenisKelamin string
}

// PegawaiService contains pegawai business rules.
type PegawaiService struct {
	repo *repository.PegawaiRepository
}

func NewPegawaiService(repo *repository.PegawaiRepository) *PegawaiService {
	return &PegawaiService{repo: repo}
}

func (s *PegawaiService) List(ctx context.Context, query string) ([]model.Pegawai, error) {
	query = strings.TrimSpace(query)
	if query != "" && utf8.RuneCountInString(query) < minSearchLength {
		return nil, apperr.Validation(fmt.Sprintf("q minimal %d karakter", minSearchLength))
	}
	if utf8.RuneCountInString(query) > maxSearchLength {
		return nil, apperr.Validation(fmt.Sprintf("q maksimal %d karakter", maxSearchLength))
	}
	return s.repo.List(ctx, query)
}

func (s *PegawaiService) Get(ctx context.Context, id int64) (*model.Pegawai, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *PegawaiService) Create(ctx context.Context, in PegawaiInput) (*model.Pegawai, error) {
	pegawai, err := buildPegawai(in)
	if err != nil {
		return nil, err
	}
	pegawai.MarkCreated(0)

	if err := s.repo.Create(ctx, pegawai); err != nil {
		return nil, err
	}
	return pegawai, nil
}

func (s *PegawaiService) Update(ctx context.Context, id int64, in PegawaiInput) (*model.Pegawai, error) {
	pegawai, err := buildPegawai(in)
	if err != nil {
		return nil, err
	}
	pegawai.ID = id
	pegawai.MarkUpdated(0)

	if err := s.repo.Update(ctx, pegawai); err != nil {
		return nil, err
	}
	return pegawai, nil
}

func (s *PegawaiService) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id, 0)
}

// buildPegawai validates and normalizes input into a model.
func buildPegawai(in PegawaiInput) (*model.Pegawai, error) {
	nama := strings.TrimSpace(in.Nama)
	nip := strings.TrimSpace(in.NIP)
	alamat := strings.TrimSpace(in.Alamat)

	switch {
	case nama == "":
		return nil, apperr.Validation("nama wajib diisi")
	case utf8.RuneCountInString(nama) > maxNamaLength:
		return nil, apperr.Validation(fmt.Sprintf("nama maksimal %d karakter", maxNamaLength))
	case nip == "":
		return nil, apperr.Validation("nip wajib diisi")
	case utf8.RuneCountInString(nip) > maxNIPLength:
		return nil, apperr.Validation(fmt.Sprintf("nip maksimal %d karakter", maxNIPLength))
	case alamat == "":
		return nil, apperr.Validation("alamat wajib diisi")
	case utf8.RuneCountInString(alamat) > maxAlamatLength:
		return nil, apperr.Validation(fmt.Sprintf("alamat maksimal %d karakter", maxAlamatLength))
	}

	tanggalLahir, err := parseTanggalLahir(in.TanggalLahir)
	if err != nil {
		return nil, err
	}

	isMale, err := parseJenisKelamin(in.JenisKelamin)
	if err != nil {
		return nil, err
	}

	return &model.Pegawai{
		Nama:         nama,
		NIP:          nip,
		TanggalLahir: tanggalLahir,
		Alamat:       alamat,
		IsMale:       isMale,
	}, nil
}

func parseTanggalLahir(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", apperr.Validation("tanggal_lahir wajib diisi")
	}

	t, err := time.Parse(dateLayout, value)
	if err != nil {
		t, err = time.Parse(dateLayoutIndo, value)
	}
	if err != nil {
		return "", apperr.Validation("tanggal_lahir harus berformat DD-MM-YYYY atau YYYY-MM-DD")
	}

	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	if t.After(today) {
		return "", apperr.Validation("tanggal_lahir tidak boleh di masa depan")
	}

	return t.Format(dateLayout), nil
}

// FormatTanggalLahir returns the stored ISO date as DD-MM-YYYY for the DTO.
func FormatTanggalLahir(value string) string {
	t, err := time.Parse(dateLayout, value)
	if err != nil {
		return value
	}
	return t.Format(dateLayoutIndo)
}

// parseJenisKelamin maps the API label to the stored boolean.
func parseJenisKelamin(value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "":
		return false, apperr.Validation("jenis_kelamin wajib diisi")
	case jenisKelaminPria:
		return true, nil
	case jenisKelaminWanita:
		return false, nil
	default:
		return false, apperr.Validation("jenis_kelamin harus pria atau wanita")
	}
}

// FormatJenisKelamin maps the stored boolean to the DTO label.
func FormatJenisKelamin(isMale bool) string {
	if isMale {
		return "Pria"
	}
	return "Wanita"
}
