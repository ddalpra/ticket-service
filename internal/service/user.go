package service

import (
	"context"

	"ticket-service/ent"
	"ticket-service/internal/repository"
	"ticket-service/pkg/apperrors"
	"ticket-service/pkg/keycloak"

	"github.com/google/uuid"
)

type UserService struct {
	repo        *repository.UserRepository
	companyRepo *repository.CompanyRepository
	scRepo      *repository.ServiceCenterRepository
	kc          *keycloak.Client
}

func NewUserService(
	repo *repository.UserRepository,
	companyRepo *repository.CompanyRepository,
	scRepo *repository.ServiceCenterRepository,
	kc *keycloak.Client,
) *UserService {
	return &UserService{repo: repo, companyRepo: companyRepo, scRepo: scRepo, kc: kc}
}

// RegisterCustomer crea un customer in Keycloak e nel DB locale.
func (s *UserService) RegisterCustomer(ctx context.Context, input RegisterCustomerInput) (*ent.User, error) {
	// Verifica che l'azienda esista
	if _, err := s.companyRepo.FindByID(ctx, input.CompanyID); err != nil {
		return nil, apperrors.NotFound("azienda non trovata")
	}

	// Crea in Keycloak
	kcID, err := s.kc.CreateUser(ctx, keycloak.CreateUserRequest{
		Username:  input.Username,
		Email:     input.Email,
		FirstName: input.FirstName,
		LastName:  input.LastName,
		Password:  input.Password,
		Roles:     []string{"customer"},
	})
	if err != nil {
		return nil, apperrors.Internal(err)
	}

	// Specializza nel DB locale
	return s.repo.Create(ctx, repository.CreateUserInput{
		KeycloakID: kcID,
		Username:   input.Username,
		Email:      input.Email,
		FirstName:  &input.FirstName,
		LastName:   &input.LastName,
		Role:       "customer",
		CompanyID:  &input.CompanyID,
	})
}

// RegisterSupport crea un operatore (L1/L2) o supervisor in Keycloak e nel DB.
func (s *UserService) RegisterSupport(ctx context.Context, input RegisterSupportInput) (*ent.User, error) {
	allowedRoles := map[string]bool{
		"support_l1": true,
		"support_l2": true,
		"supervisor": true,
	}
	if !allowedRoles[input.Role] {
		return nil, apperrors.BadRequest("ruolo non valido: usa support_l1, support_l2 o supervisor")
	}

	// Verifica che il centro servizi esista
	if _, err := s.scRepo.FindByID(ctx, input.ServiceCenterID); err != nil {
		return nil, apperrors.NotFound("centro servizi non trovato")
	}

	kcID, err := s.kc.CreateUser(ctx, keycloak.CreateUserRequest{
		Username:  input.Username,
		Email:     input.Email,
		FirstName: input.FirstName,
		LastName:  input.LastName,
		Password:  input.Password,
		Roles:     []string{input.Role},
	})
	if err != nil {
		return nil, apperrors.Internal(err)
	}

	return s.repo.Create(ctx, repository.CreateUserInput{
		KeycloakID:      kcID,
		Username:        input.Username,
		Email:           input.Email,
		FirstName:       &input.FirstName,
		LastName:        &input.LastName,
		Role:            input.Role,
		ServiceCenterID: &input.ServiceCenterID,
	})
}

// ListCenterUsers restituisce gli utenti del centro servizi del supervisor.
func (s *UserService) ListCenterUsers(ctx context.Context, caller *ent.User) ([]*ent.User, error) {
	sc := caller.Edges.ServiceCenter
	if sc == nil {
		return nil, apperrors.BadRequest("supervisor non associato ad un centro servizi")
	}
	return s.repo.FindByServiceCenter(ctx, sc.ID)
}

func (s *UserService) SetActive(ctx context.Context, id uuid.UUID, active bool, caller *ent.User) (*ent.User, error) {
	u, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, apperrors.NotFound("utente non trovato")
	}
	// Disabilita anche su Keycloak
	if !active {
		if err := s.kc.DisableUser(ctx, u.KeycloakID); err != nil {
			return nil, apperrors.Internal(err)
		}
	}
	return s.repo.SetActive(ctx, id, active)
}

// ── Input DTOs ────────────────────────────────────────────────────────────────

type RegisterCustomerInput struct {
	Username  string
	Email     string
	FirstName string
	LastName  string
	Password  string
	CompanyID uuid.UUID
}

type RegisterSupportInput struct {
	Username        string
	Email           string
	FirstName       string
	LastName        string
	Password        string
	Role            string
	ServiceCenterID uuid.UUID
}
