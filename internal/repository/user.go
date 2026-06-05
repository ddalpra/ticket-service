package repository

import (
	"context"
	"fmt"

	"ticket-service/ent"
	"ticket-service/ent/user"

	"github.com/google/uuid"
)

type UserRepository struct {
	client *ent.Client
}

func NewUserRepository(client *ent.Client) *UserRepository {
	return &UserRepository{client: client}
}

func (r *UserRepository) FindByID(ctx context.Context, id uuid.UUID) (*ent.User, error) {
	u, err := r.client.User.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("user FindByID: %w", err)
	}
	return u, nil
}

func (r *UserRepository) FindByKeycloakID(ctx context.Context, keycloakID string) (*ent.User, error) {
	u, err := r.client.User.Query().
		Where(user.KeycloakID(keycloakID)).
		WithCompany().
		WithServiceCenter().
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("user FindByKeycloakID: %w", err)
	}
	return u, nil
}

func (r *UserRepository) FindByServiceCenter(ctx context.Context, scID uuid.UUID) ([]*ent.User, error) {
	return r.client.User.Query().
		Where(user.HasServiceCenterWith()).
		All(ctx)
}

func (r *UserRepository) Create(ctx context.Context, input CreateUserInput) (*ent.User, error) {
	q := r.client.User.Create().
		SetKeycloakID(input.KeycloakID).
		SetUsername(input.Username).
		SetEmail(input.Email).
		SetRole(user.Role(input.Role)).
		SetNillableFirstName(input.FirstName).
		SetNillableLastName(input.LastName)

	if input.CompanyID != nil {
		q = q.SetCompanyID(*input.CompanyID)
	}
	if input.ServiceCenterID != nil {
		q = q.SetServiceCenterID(*input.ServiceCenterID)
	}
	return q.Save(ctx)
}

func (r *UserRepository) SetActive(ctx context.Context, id uuid.UUID, active bool) (*ent.User, error) {
	return r.client.User.UpdateOneID(id).SetActive(active).Save(ctx)
}

type CreateUserInput struct {
	KeycloakID      string
	Username        string
	Email           string
	FirstName       *string
	LastName        *string
	Role            string
	CompanyID       *uuid.UUID
	ServiceCenterID *uuid.UUID
}
