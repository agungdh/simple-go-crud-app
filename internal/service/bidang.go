package service

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"simple-go-crud-app/internal/apperr"
	"simple-go-crud-app/internal/model"
	"simple-go-crud-app/internal/repository"
)

const maxBidangLength = 150

// BidangInput is the raw payload accepted from the HTTP layer.
type BidangInput struct {
	Bidang string
}

// BidangService contains bidang business rules.
type BidangService struct {
	repo *repository.BidangRepository
}

func NewBidangService(repo *repository.BidangRepository) *BidangService {
	return &BidangService{repo: repo}
}

func (s *BidangService) List(ctx context.Context, query string) ([]model.Bidang, error) {
	query = strings.TrimSpace(query)
	if query != "" && utf8.RuneCountInString(query) < minSearchLength {
		return nil, apperr.Validation(fmt.Sprintf("q minimal %d karakter", minSearchLength))
	}
	if utf8.RuneCountInString(query) > maxSearchLength {
		return nil, apperr.Validation(fmt.Sprintf("q maksimal %d karakter", maxSearchLength))
	}
	return s.repo.List(ctx, query)
}

func (s *BidangService) Get(ctx context.Context, id int64) (*model.Bidang, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *BidangService) Create(ctx context.Context, in BidangInput) (*model.Bidang, error) {
	bidang, err := buildBidang(in)
	if err != nil {
		return nil, err
	}
	bidang.MarkCreated(0)

	if err := s.repo.Create(ctx, bidang); err != nil {
		return nil, err
	}
	return bidang, nil
}

func (s *BidangService) Update(ctx context.Context, id int64, in BidangInput) (*model.Bidang, error) {
	bidang, err := buildBidang(in)
	if err != nil {
		return nil, err
	}
	bidang.ID = id
	bidang.MarkUpdated(0)

	if err := s.repo.Update(ctx, bidang); err != nil {
		return nil, err
	}
	return bidang, nil
}

func (s *BidangService) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id, 0)
}

// buildBidang validates and normalizes input into a model.
func buildBidang(in BidangInput) (*model.Bidang, error) {
	bidang := strings.TrimSpace(in.Bidang)

	switch {
	case bidang == "":
		return nil, apperr.Validation("bidang wajib diisi")
	case utf8.RuneCountInString(bidang) > maxBidangLength:
		return nil, apperr.Validation(fmt.Sprintf("bidang maksimal %d karakter", maxBidangLength))
	}

	return &model.Bidang{Bidang: bidang}, nil
}
