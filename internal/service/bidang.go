package service

import (
	"context"

	"simple-go-crud-app/internal/apperr"
	"simple-go-crud-app/internal/dto"
	"simple-go-crud-app/internal/model"
	"simple-go-crud-app/internal/repository"
)

const duplicateBidangMessage = "Bidang sudah digunakan."

// BidangService implements the bidang use cases on top of the repository.
type BidangService struct {
	repo *repository.BidangRepository
}

func NewBidang(repo *repository.BidangRepository) *BidangService {
	return &BidangService{repo: repo}
}

// duplicateError reports whether another active bidang already uses name.
func (s *BidangService) duplicateError(ctx context.Context, name string, excludeID int64) (apperr.FieldErrors, error) {
	exists, err := s.repo.NameExists(ctx, name, excludeID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}
	return apperr.FieldErrors{"bidang": {duplicateBidangMessage}}, nil
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

// Create validates and inserts a new bidang. actorID is stored in the audit
// columns as created_by.
func (s *BidangService) Create(ctx context.Context, input dto.BidangInput, actorID int64) error {
	value, err := validateBidangInput(input)
	if err != nil {
		return err
	}

	duplicate, err := s.duplicateError(ctx, value, 0)
	if err != nil {
		return err
	}
	if duplicate != nil {
		return duplicate
	}

	item := &model.Bidang{Bidang: value}
	item.MarkCreated(actorID)
	if err := s.repo.Create(ctx, item); err != nil {
		// Re-check on write failure so a concurrent insert still returns a
		// validation error instead of leaking a database error as 500.
		if duplicate, checkErr := s.duplicateError(ctx, value, 0); checkErr == nil && duplicate != nil {
			return duplicate
		}
		return err
	}
	return nil
}

// Update validates and replaces the editable fields of an active bidang.
// actorID is stored in the audit columns as updated_by.
func (s *BidangService) Update(ctx context.Context, id int64, input dto.BidangInput, actorID int64) error {
	value, err := validateBidangInput(input)
	if err != nil {
		return err
	}

	duplicate, err := s.duplicateError(ctx, value, id)
	if err != nil {
		return err
	}
	if duplicate != nil {
		return duplicate
	}

	item := &model.Bidang{Base: model.Base{ID: id}, Bidang: value}
	item.MarkUpdated(actorID)
	if err := s.repo.Update(ctx, item); err != nil {
		if duplicate, checkErr := s.duplicateError(ctx, value, id); checkErr == nil && duplicate != nil {
			return duplicate
		}
		return err
	}
	return nil
}

// Delete soft deletes an active bidang and records deleted_by.
func (s *BidangService) Delete(ctx context.Context, id, actorID int64) error {
	return s.repo.Delete(ctx, id, actorID)
}
