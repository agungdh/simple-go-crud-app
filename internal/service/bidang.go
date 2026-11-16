package service

import (
	"context"

	"simple-go-crud-app/internal/dto"
	"simple-go-crud-app/internal/model"
	"simple-go-crud-app/internal/repository"
)

// BidangService implements the bidang use cases on top of the repository.
type BidangService struct {
	repo *repository.BidangRepository
}

func NewBidang(repo *repository.BidangRepository) *BidangService {
	return &BidangService{repo: repo}
}

// List returns active bidang, optionally filtered by a search keyword.
func (s *BidangService) List(ctx context.Context, q string) ([]dto.Bidang, error) {
	query, err := normalizeQuery(q)
	if err != nil {
		return nil, err
	}

	items, err := s.repo.List(ctx, query)
	if err != nil {
		return nil, err
	}

	result := make([]dto.Bidang, 0, len(items))
	for _, item := range items {
		result = append(result, dto.Bidang{ID: item.ID, Bidang: item.Bidang})
	}
	return result, nil
}

// Get returns one active bidang.
func (s *BidangService) Get(ctx context.Context, id int64) (dto.Bidang, error) {
	item, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return dto.Bidang{}, err
	}
	return dto.Bidang{ID: item.ID, Bidang: item.Bidang}, nil
}

// Create validates and inserts a new bidang.
func (s *BidangService) Create(ctx context.Context, input dto.BidangInput) error {
	value, err := validateBidangInput(input)
	if err != nil {
		return err
	}

	item := &model.Bidang{Bidang: value}
	item.MarkCreated(0)
	return s.repo.Create(ctx, item)
}

// Update validates and replaces the editable fields of an active bidang.
func (s *BidangService) Update(ctx context.Context, id int64, input dto.BidangInput) error {
	value, err := validateBidangInput(input)
	if err != nil {
		return err
	}

	item := &model.Bidang{Base: model.Base{ID: id}, Bidang: value}
	item.MarkUpdated(0)
	return s.repo.Update(ctx, item)
}

// Delete soft deletes an active bidang.
func (s *BidangService) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id, 0)
}
