package repository

import (
	"context"
	"fmt"

	"ticket-service/ent"
	"ticket-service/ent/company"
	"ticket-service/ent/servicecenter"

	"github.com/google/uuid"
)

// ── Company ───────────────────────────────────────────────────────────────────

type CompanyRepository struct {
	client *ent.Client
}

func NewCompanyRepository(client *ent.Client) *CompanyRepository {
	return &CompanyRepository{client: client}
}

func (r *CompanyRepository) FindAll(ctx context.Context) ([]*ent.Company, error) {
	return r.client.Company.Query().Order(ent.Asc(company.FieldRagioneSociale)).All(ctx)
}

func (r *CompanyRepository) FindByID(ctx context.Context, id uuid.UUID) (*ent.Company, error) {
	c, err := r.client.Company.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("company FindByID: %w", err)
	}
	return c, nil
}

func (r *CompanyRepository) Create(ctx context.Context, input CreateCompanyInput) (*ent.Company, error) {
	return r.client.Company.Create().
		SetCodice(input.Codice).
		SetRagioneSociale(input.RagioneSociale).
		SetPiva(input.Piva).
		SetCf(input.Cf).
		Save(ctx)
}

func (r *CompanyRepository) Update(ctx context.Context, id uuid.UUID, input UpdateCompanyInput) (*ent.Company, error) {
	q := r.client.Company.UpdateOneID(id)
	if input.RagioneSociale != nil {
		q = q.SetRagioneSociale(*input.RagioneSociale)
	}
	if input.Active != nil {
		q = q.SetActive(*input.Active)
	}
	return q.Save(ctx)
}

type CreateCompanyInput struct {
	Codice         string
	RagioneSociale string
	Piva           string
	Cf             string
}

type UpdateCompanyInput struct {
	RagioneSociale *string
	Active         *bool
}

// ── ServiceCenter ─────────────────────────────────────────────────────────────

type ServiceCenterRepository struct {
	client *ent.Client
}

func NewServiceCenterRepository(client *ent.Client) *ServiceCenterRepository {
	return &ServiceCenterRepository{client: client}
}

func (r *ServiceCenterRepository) FindAll(ctx context.Context) ([]*ent.ServiceCenter, error) {
	return r.client.ServiceCenter.Query().Order(ent.Asc(servicecenter.FieldNome)).All(ctx)
}

func (r *ServiceCenterRepository) FindByID(ctx context.Context, id uuid.UUID) (*ent.ServiceCenter, error) {
	sc, err := r.client.ServiceCenter.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("service_center FindByID: %w", err)
	}
	return sc, nil
}

func (r *ServiceCenterRepository) Create(ctx context.Context, input CreateSCInput) (*ent.ServiceCenter, error) {
	return r.client.ServiceCenter.Create().
		SetCodice(input.Codice).
		SetNome(input.Nome).
		Save(ctx)
}

type CreateSCInput struct {
	Codice string
	Nome   string
}
