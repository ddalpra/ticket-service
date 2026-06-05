package service

import (
	"context"

	"ticket-service/ent"
	"ticket-service/ent/ticket"
	"ticket-service/internal/repository"
	"ticket-service/pkg/apperrors"

	"github.com/google/uuid"
)

type TicketService struct {
	repo     *repository.TicketRepository
	userRepo *repository.UserRepository
}

func NewTicketService(repo *repository.TicketRepository, userRepo *repository.UserRepository) *TicketService {
	return &TicketService{repo: repo, userRepo: userRepo}
}

// List restituisce i ticket visibili al caller in base al ruolo.
func (s *TicketService) List(ctx context.Context, caller *ent.User) ([]*ent.Ticket, error) {
	switch caller.Role.String() {
	case "customer":
		comp, err := caller.Edges.Company, error(nil)
		if comp == nil {
			comp, err = caller.QueryCompany().Only(ctx)
			if err != nil {
				return nil, apperrors.Internal(err)
			}
		}
		return s.repo.FindByCompany(ctx, comp.ID)

	case "support_l1", "support_l2":
		return s.repo.FindByAssignedUser(ctx, caller.ID)

	case "supervisor":
		sc := caller.Edges.ServiceCenter
		if sc == nil {
			var err error
			sc, err = caller.QueryServiceCenter().Only(ctx)
			if err != nil {
				return nil, apperrors.Internal(err)
			}
		}
		return s.repo.FindByServiceCenter(ctx, sc.ID)
	}
	return nil, apperrors.Forbidden("ruolo non riconosciuto")
}

// Get restituisce un ticket se il caller ha visibilità su di esso.
func (s *TicketService) Get(ctx context.Context, id uuid.UUID, caller *ent.User) (*ent.Ticket, error) {
	t, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, apperrors.NotFound("ticket non trovato")
	}
	if err := s.checkVisibility(ctx, t, caller); err != nil {
		return nil, err
	}
	return t, nil
}

// Create crea un nuovo ticket. Solo i customer possono creare ticket.
func (s *TicketService) Create(ctx context.Context, input CreateTicketInput, caller *ent.User) (*ent.Ticket, error) {
	comp := caller.Edges.Company
	if comp == nil {
		var err error
		comp, err = caller.QueryCompany().Only(ctx)
		if err != nil {
			return nil, apperrors.BadRequest("utente non associato ad un'azienda")
		}
	}
	return s.repo.Create(ctx, repository.CreateTicketInput{
		Title:           input.Title,
		Question:        input.Question,
		CompanyID:       comp.ID,
		ServiceCenterID: input.ServiceCenterID,
		CreatedByID:     caller.ID,
	})
}

// ListUnassigned restituisce i ticket non assegnati del centro servizi del caller.
func (s *TicketService) ListUnassigned(ctx context.Context, caller *ent.User) ([]*ent.Ticket, error) {
	sc, err := s.callerServiceCenter(ctx, caller)
	if err != nil {
		return nil, err
	}
	return s.repo.FindUnassignedByServiceCenter(ctx, sc.ID)
}

// ListMine restituisce i ticket assegnati al caller.
func (s *TicketService) ListMine(ctx context.Context, caller *ent.User) ([]*ent.Ticket, error) {
	return s.repo.FindByAssignedUser(ctx, caller.ID)
}

// ListCenter restituisce tutti i ticket del centro servizi (solo supervisor).
func (s *TicketService) ListCenter(ctx context.Context, caller *ent.User) ([]*ent.Ticket, error) {
	sc, err := s.callerServiceCenter(ctx, caller)
	if err != nil {
		return nil, err
	}
	return s.repo.FindByServiceCenter(ctx, sc.ID)
}

// Take prende in carico un ticket non assegnato.
func (s *TicketService) Take(ctx context.Context, id uuid.UUID, caller *ent.User) (*ent.Ticket, error) {
	t, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, apperrors.NotFound("ticket non trovato")
	}
	if t.Edges.AssignedTo != nil {
		return nil, apperrors.Conflict("ticket già assegnato")
	}
	if t.StateJob == ticket.StateJobCHIUSA {
		return nil, apperrors.BadRequest("il ticket è chiuso")
	}
	// Verifica che il ticket appartenga al centro servizi del caller
	sc, err := s.callerServiceCenter(ctx, caller)
	if err != nil {
		return nil, err
	}
	if t.Edges.ServiceCenter == nil || t.Edges.ServiceCenter.ID != sc.ID {
		return nil, apperrors.Forbidden("il ticket non appartiene al tuo centro servizi")
	}
	return s.repo.TakeTicket(ctx, id, caller.ID)
}

// Escalate assegna il ticket a un operatore L2 dello stesso centro servizi.
func (s *TicketService) Escalate(ctx context.Context, id, targetUserID uuid.UUID, caller *ent.User) (*ent.Ticket, error) {
	t, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, apperrors.NotFound("ticket non trovato")
	}
	// Solo il proprietario del ticket può fare escalation
	if t.Edges.AssignedTo == nil || t.Edges.AssignedTo.ID != caller.ID {
		return nil, apperrors.Forbidden("puoi fare escalation solo dei tuoi ticket")
	}
	// Verifica che il target sia L2
	target, err := s.userRepo.FindByID(ctx, targetUserID)
	if err != nil {
		return nil, apperrors.NotFound("utente target non trovato")
	}
	if target.Role.String() != "support_l2" {
		return nil, apperrors.BadRequest("l'escalation richiede un operatore di livello 2")
	}
	return s.repo.Escalate(ctx, id, targetUserID, caller.ID)
}

// Assign (supervisor) riassegna il ticket a qualsiasi utente del centro.
func (s *TicketService) Assign(ctx context.Context, id, toUserID uuid.UUID, caller *ent.User) (*ent.Ticket, error) {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return nil, apperrors.NotFound("ticket non trovato")
	}
	return s.repo.Assign(ctx, id, toUserID, caller.ID)
}

// SetPriority aggiorna la priorità del ticket.
func (s *TicketService) SetPriority(ctx context.Context, id uuid.UUID, p string, caller *ent.User) (*ent.Ticket, error) {
	pEnum, err := ticket.PriorityValidator(ticket.Priority(p))
	if err != nil {
		return nil, apperrors.BadRequest("priorità non valida: usa LOW, MEDIUM, HIGH")
	}
	return s.repo.SetPriority(ctx, id, pEnum.(ticket.Priority), caller.ID)
}

// SetState aggiorna lo state_job del ticket.
func (s *TicketService) SetState(ctx context.Context, id uuid.UUID, sj string, caller *ent.User) (*ent.Ticket, error) {
	sjEnum, err := ticket.StateJobValidator(ticket.StateJob(sj))
	if err != nil {
		return nil, apperrors.BadRequest("stato non valido")
	}
	return s.repo.SetState(ctx, id, sjEnum.(ticket.StateJob), caller.ID)
}

// AddComment aggiunge un commento al ticket.
func (s *TicketService) AddComment(ctx context.Context, ticketID uuid.UUID, detail string, caller *ent.User) (*ent.Comment, error) {
	t, err := s.repo.FindByID(ctx, ticketID)
	if err != nil {
		return nil, apperrors.NotFound("ticket non trovato")
	}
	if err := s.checkVisibility(ctx, t, caller); err != nil {
		return nil, err
	}
	return s.repo.AddComment(ctx, repository.AddCommentInput{
		TicketID: ticketID,
		AuthorID: caller.ID,
		Detail:   detail,
	})
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func (s *TicketService) callerServiceCenter(ctx context.Context, caller *ent.User) (*ent.ServiceCenter, error) {
	sc := caller.Edges.ServiceCenter
	if sc != nil {
		return sc, nil
	}
	sc, err := caller.QueryServiceCenter().Only(ctx)
	if err != nil {
		return nil, apperrors.BadRequest("utente non associato ad un centro servizi")
	}
	return sc, nil
}

func (s *TicketService) checkVisibility(ctx context.Context, t *ent.Ticket, caller *ent.User) error {
	switch caller.Role.String() {
	case "customer":
		comp := caller.Edges.Company
		if comp == nil {
			return apperrors.Forbidden("accesso negato")
		}
		if t.Edges.Company == nil || t.Edges.Company.ID != comp.ID {
			return apperrors.Forbidden("accesso negato")
		}
	case "support_l1", "support_l2", "supervisor":
		sc, err := s.callerServiceCenter(ctx, caller)
		if err != nil {
			return err
		}
		if t.Edges.ServiceCenter == nil || t.Edges.ServiceCenter.ID != sc.ID {
			return apperrors.Forbidden("accesso negato")
		}
	}
	return nil
}

// ── Input DTOs ────────────────────────────────────────────────────────────────

type CreateTicketInput struct {
	Title           string
	Question        string
	ServiceCenterID uuid.UUID
}
