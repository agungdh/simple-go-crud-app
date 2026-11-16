package bidang

import (
	"context"

	"simple-go-crud-app/internal/shared/apperr"
	"simple-go-crud-app/internal/shared/entity"
	"simple-go-crud-app/internal/shared/search"
)

const duplicateBidangMessage = "Bidang sudah digunakan."

// Service implements the bidang use cases on top of the repository.
type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// duplicateError reports whether another active bidang already uses name.
func (s *Service) duplicateError(ctx context.Context, name string, excludeID int64) (apperr.FieldErrors, error) {
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
		result = append(result, Response{ID: item.ID, Bidang: item.Bidang})
	}
	return result, nil
}

// Get returns one active bidang.
func (s *Service) Get(ctx context.Context, id int64) (Response, error) {
	item, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Response{}, err
	}
	return Response{ID: item.ID, Bidang: item.Bidang}, nil
}

// Create validates and inserts a new bidang. actorID is stored in the audit
// columns as created_by.
func (s *Service) Create(ctx context.Context, input Input, actorID int64) error {
	value, err := validateInput(input)
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

	item := &Entity{Bidang: value}
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
func (s *Service) Update(ctx context.Context, id int64, input Input, actorID int64) error {
	value, err := validateInput(input)
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

	item := &Entity{Base: entity.Base{ID: id}, Bidang: value}
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
func (s *Service) Delete(ctx context.Context, id, actorID int64) error {
	return s.repo.Delete(ctx, id, actorID)
}
