package service

import (
	"context"

	"github.com/google/uuid"
	"ticket-service/ent"
	"ticket-service/internal/repository"
	"ticket-service/pkg/apperrors"
)

// ── CompanyService ────────────────────────────────────────────────────────────

type CompanyService struct {
	repo *repository.CompanyRepository
}

func NewCompanyService(repo *repository.CompanyRepository) *CompanyService {
	return &CompanyService{repo: repo}
}

func (s *CompanyService) List(ctx context.Context) ([]*ent.Company, error) {
	return s.repo.FindAll(ctx)
}

func (s *CompanyService) Get(ctx context.Context, id uuid.UUID) (*ent.Company, error) {
	c, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, apperrors.NotFound("azienda non trovata")
	}
	return c, nil
}

func (s *CompanyService) Create(ctx context.Context, input repository.CreateCompanyInput) (*ent.Company, error) {
	if input.Codice == "" || input.RagioneSociale == "" {
		return nil, apperrors.BadRequest("codice e ragione sociale sono obbligatori")
	}
	c, err := s.repo.Create(ctx, input)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	return c, nil
}

func (s *CompanyService) Update(ctx context.Context, id uuid.UUID, input repository.UpdateCompanyInput) (*ent.Company, error) {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return nil, apperrors.NotFound("azienda non trovata")
	}
	return s.repo.Update(ctx, id, input)
}

// ── ServiceCenterService ──────────────────────────────────────────────────────

type ServiceCenterService struct {
	repo *repository.ServiceCenterRepository
}

func NewServiceCenterService(repo *repository.ServiceCenterRepository) *ServiceCenterService {
	return &ServiceCenterService{repo: repo}
}

func (s *ServiceCenterService) List(ctx context.Context) ([]*ent.ServiceCenter, error) {
	return s.repo.FindAll(ctx)
}

func (s *ServiceCenterService) Get(ctx context.Context, id uuid.UUID) (*ent.ServiceCenter, error) {
	sc, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, apperrors.NotFound("centro servizi non trovato")
	}
	return sc, nil
}

func (s *ServiceCenterService) Create(ctx context.Context, input repository.CreateSCInput) (*ent.ServiceCenter, error) {
	if input.Codice == "" || input.Nome == "" {
		return nil, apperrors.BadRequest("codice e nome sono obbligatori")
	}
	sc, err := s.repo.Create(ctx, input)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	return sc, nil
}
