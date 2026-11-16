package pegawai

import (
	"context"

	"simple-go-crud-app/internal/bidang"
	"simple-go-crud-app/internal/shared/apperr"
	"simple-go-crud-app/internal/shared/entity"
	"simple-go-crud-app/internal/shared/search"
)

const (
	duplicateNIPMessage  = "NIP sudah digunakan."
	missingBidangMessage = "Bidang tidak ditemukan."
)

// Service implements the pegawai use cases on top of the repository.
type Service struct {
	repo   *Repository
	bidang *bidang.Repository
}

func NewService(repo *Repository, bidangRepo *bidang.Repository) *Service {
	return &Service{repo: repo, bidang: bidangRepo}
}

// referenceErrors checks the values that must exist or stay unique: NIP among
// active pegawai and bidang_id among active bidang.
func (s *Service) referenceErrors(ctx context.Context, values pegawaiFields, excludeID int64) (apperr.FieldErrors, error) {
	errs := apperr.FieldErrors{}

	nipExists, err := s.repo.NIPExists(ctx, values.NIP, excludeID)
	if err != nil {
		return nil, err
	}
	if nipExists {
		errs.Add("nip", duplicateNIPMessage)
	}

	exists, err := s.bidang.ExistsActive(ctx, values.BidangID)
	if err != nil {
		return nil, err
	}
	if !exists {
		errs.Add("bidang_id", missingBidangMessage)
	}

	if errs.Any() {
		return errs, nil
	}
	return nil, nil
}

// List returns active pegawai, optionally filtered by a search keyword.
func (s *Service) List(ctx context.Context, q string) ([]Response, error) {
	query, err := search.Normalize(q)
	if err != nil {
		return nil, err
	}

	items, err := s.repo.List(ctx, query)
	if err != nil {
		return nil, err
	}

	result := make([]Response, 0, len(items))
	for _, item := range items {
		result = append(result, toResponse(item))
	}
	return result, nil
}

// Get returns one active pegawai.
func (s *Service) Get(ctx context.Context, id int64) (Response, error) {
	item, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Response{}, err
	}
	return toResponse(*item), nil
}

// Create validates and inserts a new pegawai. actorID is stored in the audit
// columns as created_by.
func (s *Service) Create(ctx context.Context, input Input, actorID int64) error {
	values, err := validateInput(input)
	if err != nil {
		return err
	}

	referenceErrs, err := s.referenceErrors(ctx, values, 0)
	if err != nil {
		return err
	}
	if referenceErrs != nil {
		return referenceErrs
	}

	item := &Entity{
		BidangID:     values.BidangID,
		Nama:         values.Nama,
		NIP:          values.NIP,
		TanggalLahir: values.TanggalLahir,
		Alamat:       values.Alamat,
		IsMale:       values.IsMale,
	}
	item.MarkCreated(actorID)
	if err := s.repo.Create(ctx, item); err != nil {
		// Re-check on write failure so a concurrent insert still returns a
		// validation error instead of leaking a database error as 500.
		if referenceErrs, checkErr := s.referenceErrors(ctx, values, 0); checkErr == nil && referenceErrs != nil {
			return referenceErrs
		}
		return err
	}
	return nil
}

// Update validates and replaces the editable fields of an active pegawai.
// actorID is stored in the audit columns as updated_by.
func (s *Service) Update(ctx context.Context, id int64, input Input, actorID int64) error {
	values, err := validateInput(input)
	if err != nil {
		return err
	}

	referenceErrs, err := s.referenceErrors(ctx, values, id)
	if err != nil {
		return err
	}
	if referenceErrs != nil {
		return referenceErrs
	}

	item := &Entity{
		Base:         entity.Base{ID: id},
		BidangID:     values.BidangID,
		Nama:         values.Nama,
		NIP:          values.NIP,
		TanggalLahir: values.TanggalLahir,
		Alamat:       values.Alamat,
		IsMale:       values.IsMale,
	}
	item.MarkUpdated(actorID)
	if err := s.repo.Update(ctx, item); err != nil {
		if referenceErrs, checkErr := s.referenceErrors(ctx, values, id); checkErr == nil && referenceErrs != nil {
			return referenceErrs
		}
		return err
	}
	return nil
}

// Delete soft deletes an active pegawai and records deleted_by.
func (s *Service) Delete(ctx context.Context, id, actorID int64) error {
	return s.repo.Delete(ctx, id, actorID)
}

// toResponse maps stored values to the client representation: business dates
// become DD-MM-YYYY and the boolean becomes an Indonesian label.
func toResponse(item Entity) Response {
	jenisKelamin := Wanita
	if item.IsMale {
		jenisKelamin = Pria
	}
	return Response{
		ID:           item.ID,
		BidangID:     item.BidangID,
		Nama:         item.Nama,
		NIP:          item.NIP,
		TanggalLahir: formatTanggalLahir(item.TanggalLahir),
		Alamat:       item.Alamat,
		JenisKelamin: jenisKelamin,
	}
}
