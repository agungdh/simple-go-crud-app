package service

import (
	"context"
	"errors"

	"simple-go-crud-app/internal/apperr"
	"simple-go-crud-app/internal/dto"
	"simple-go-crud-app/internal/model"
	"simple-go-crud-app/internal/repository"
)

const (
	duplicateNIPMessage  = "NIP sudah digunakan."
	missingBidangMessage = "Bidang tidak ditemukan."
)

// PegawaiService implements the pegawai use cases on top of the repository.
type PegawaiService struct {
	repo   *repository.PegawaiRepository
	bidang *repository.BidangRepository
}

func NewPegawai(repo *repository.PegawaiRepository, bidang *repository.BidangRepository) *PegawaiService {
	return &PegawaiService{repo: repo, bidang: bidang}
}

// referenceErrors checks the values that must exist or stay unique: NIP among
// active pegawai and bidang_id among active bidang.
func (s *PegawaiService) referenceErrors(ctx context.Context, fields pegawaiFields, excludeID int64) (apperr.FieldErrors, error) {
	errs := apperr.FieldErrors{}

	nipExists, err := s.repo.NIPExists(ctx, fields.NIP, excludeID)
	if err != nil {
		return nil, err
	}
	if nipExists {
		errs.Add("nip", duplicateNIPMessage)
	}

	if _, err := s.bidang.GetByID(ctx, fields.BidangID); err != nil {
		if errors.Is(err, apperr.ErrNotFound) {
			errs.Add("bidang_id", missingBidangMessage)
		} else {
			return nil, err
		}
	}

	if errs.Any() {
		return errs, nil
	}
	return nil, nil
}

// List returns active pegawai, optionally filtered by a search keyword.
func (s *PegawaiService) List(ctx context.Context, q string) ([]dto.Pegawai, error) {
	query, err := normalizeQuery(q)
	if err != nil {
		return nil, err
	}

	items, err := s.repo.List(ctx, query)
	if err != nil {
		return nil, err
	}

	result := make([]dto.Pegawai, 0, len(items))
	for _, item := range items {
		result = append(result, toPegawaiDTO(item))
	}
	return result, nil
}

// Get returns one active pegawai.
func (s *PegawaiService) Get(ctx context.Context, id int64) (dto.Pegawai, error) {
	item, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return dto.Pegawai{}, err
	}
	return toPegawaiDTO(*item), nil
}

// Create validates and inserts a new pegawai.
func (s *PegawaiService) Create(ctx context.Context, input dto.PegawaiInput) error {
	fields, err := validatePegawaiInput(input)
	if err != nil {
		return err
	}

	referenceErrs, err := s.referenceErrors(ctx, fields, 0)
	if err != nil {
		return err
	}
	if referenceErrs != nil {
		return referenceErrs
	}

	item := &model.Pegawai{
		BidangID:     fields.BidangID,
		Nama:         fields.Nama,
		NIP:          fields.NIP,
		TanggalLahir: fields.TanggalLahir,
		Alamat:       fields.Alamat,
		IsMale:       fields.IsMale,
	}
	item.MarkCreated(0)
	if err := s.repo.Create(ctx, item); err != nil {
		// Re-check on write failure so a concurrent insert still returns a
		// validation error instead of leaking a database error as 500.
		if referenceErrs, checkErr := s.referenceErrors(ctx, fields, 0); checkErr == nil && referenceErrs != nil {
			return referenceErrs
		}
		return err
	}
	return nil
}

// Update validates and replaces the editable fields of an active pegawai.
func (s *PegawaiService) Update(ctx context.Context, id int64, input dto.PegawaiInput) error {
	fields, err := validatePegawaiInput(input)
	if err != nil {
		return err
	}

	referenceErrs, err := s.referenceErrors(ctx, fields, id)
	if err != nil {
		return err
	}
	if referenceErrs != nil {
		return referenceErrs
	}

	item := &model.Pegawai{
		Base:         model.Base{ID: id},
		BidangID:     fields.BidangID,
		Nama:         fields.Nama,
		NIP:          fields.NIP,
		TanggalLahir: fields.TanggalLahir,
		Alamat:       fields.Alamat,
		IsMale:       fields.IsMale,
	}
	item.MarkUpdated(0)
	if err := s.repo.Update(ctx, item); err != nil {
		if referenceErrs, checkErr := s.referenceErrors(ctx, fields, id); checkErr == nil && referenceErrs != nil {
			return referenceErrs
		}
		return err
	}
	return nil
}

// Delete soft deletes an active pegawai.
func (s *PegawaiService) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id, 0)
}

// toPegawaiDTO maps stored values to the client representation: business dates
// become DD-MM-YYYY and the boolean becomes an Indonesian label.
func toPegawaiDTO(item model.Pegawai) dto.Pegawai {
	jenisKelamin := dto.Wanita
	if item.IsMale {
		jenisKelamin = dto.Pria
	}
	return dto.Pegawai{
		ID:           item.ID,
		BidangID:     item.BidangID,
		Nama:         item.Nama,
		NIP:          item.NIP,
		TanggalLahir: formatTanggalLahir(item.TanggalLahir),
		Alamat:       item.Alamat,
		JenisKelamin: jenisKelamin,
	}
}
