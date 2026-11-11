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
	dateLayout = "2006-01-02"

	maxNamaLength   = 150
	maxNIPLength    = 50
	maxAlamatLength = 500
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

func (s *PegawaiService) List(ctx context.Context) ([]model.Pegawai, error) {
	return s.repo.List(ctx)
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

	jenisKelamin, err := normalizeJenisKelamin(in.JenisKelamin)
	if err != nil {
		return nil, err
	}

	return &model.Pegawai{
		Nama:         nama,
		NIP:          nip,
		TanggalLahir: tanggalLahir,
		Alamat:       alamat,
		JenisKelamin: jenisKelamin,
	}, nil
}

func parseTanggalLahir(value string) (int64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, apperr.Validation("tanggal_lahir wajib diisi")
	}

	t, err := time.Parse(dateLayout, value)
	if err != nil {
		return 0, apperr.Validation("tanggal_lahir harus berformat YYYY-MM-DD")
	}

	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	if t.After(today) {
		return 0, apperr.Validation("tanggal_lahir tidak boleh di masa depan")
	}

	return t.UnixMilli(), nil
}

func normalizeJenisKelamin(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "l", "laki-laki", "laki laki":
		return "L", nil
	case "p", "perempuan":
		return "P", nil
	default:
		return "", apperr.Validation("jenis_kelamin harus L atau P")
	}
}

// FormatTanggalLahir converts stored epoch milliseconds to YYYY-MM-DD.
func FormatTanggalLahir(ms int64) string {
	return time.UnixMilli(ms).UTC().Format(dateLayout)
}
