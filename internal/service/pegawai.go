package service

import (
	"context"

	"simple-go-crud-app/internal/dto"
	"simple-go-crud-app/internal/model"
	"simple-go-crud-app/internal/repository"
)

// PegawaiService implements the pegawai use cases on top of the repository.
type PegawaiService struct {
	repo *repository.PegawaiRepository
}

func NewPegawai(repo *repository.PegawaiRepository) *PegawaiService {
	return &PegawaiService{repo: repo}
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

	item := &model.Pegawai{
		BidangID:     fields.BidangID,
		Nama:         fields.Nama,
		NIP:          fields.NIP,
		TanggalLahir: fields.TanggalLahir,
		Alamat:       fields.Alamat,
		IsMale:       fields.IsMale,
	}
	item.MarkCreated(0)
	return s.repo.Create(ctx, item)
}

// Update validates and replaces the editable fields of an active pegawai.
func (s *PegawaiService) Update(ctx context.Context, id int64, input dto.PegawaiInput) error {
	fields, err := validatePegawaiInput(input)
	if err != nil {
		return err
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
	return s.repo.Update(ctx, item)
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
