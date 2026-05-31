package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"ticket-service/ent"
	
	// IMPORT FONDAMENTALI: I sotto-pacchetti generati da Ent
	"ticket-service/ent/comment"
	"ticket-service/ent/company"
	"ticket-service/ent/servicecenter"
	"ticket-service/ent/ticket"
	"ticket-service/ent/user"
)

type TicketRepository struct {
	client *ent.Client
}

func NewTicketRepository(client *ent.Client) *TicketRepository {
	return &TicketRepository{client: client}
}

// withEdges carica le edge comuni su ogni query ticket.
func withEdges(q *ent.TicketQuery) *ent.TicketQuery {
	return q.
		WithCompany().
		WithServiceCenter().
		WithAssignedTo().
		WithCreatedBy().
		WithComments(func(cq *ent.CommentQuery) {
			// comment.FieldOrderComment ora viene riconosciuto grazie all'import
			cq.WithAttachments().WithAuthor().Order(ent.Asc(comment.FieldOrderComment))
		}).
		WithAttachments()
}

func (r *TicketRepository) FindByID(ctx context.Context, id uuid.UUID) (*ent.Ticket, error) {
	t, err := withEdges(r.client.Ticket.Query().Where(ticket.ID(id))).Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("ticket FindByID: %w", err)
	}
	return t, nil
}

func (r *TicketRepository) FindByCompany(ctx context.Context, companyID uuid.UUID) ([]*ent.Ticket, error) {
	return withEdges(r.client.Ticket.Query().
		Where(ticket.HasCompanyWith(company.ID(companyID)))).
		All(ctx)
}

func (r *TicketRepository) FindByAssignedUser(ctx context.Context, userID uuid.UUID) ([]*ent.Ticket, error) {
	return withEdges(r.client.Ticket.Query().
		Where(ticket.HasAssignedToWith(user.ID(userID)))).
		All(ctx)
}

func (r *TicketRepository) FindUnassignedByServiceCenter(ctx context.Context, scID uuid.UUID) ([]*ent.Ticket, error) {
	return withEdges(r.client.Ticket.Query().
		Where(
			ticket.HasServiceCenterWith(servicecenter.ID(scID)),
			ticket.Not(ticket.HasAssignedTo()),
			ticket.StateJobNEQ(ticket.StateJobCHIUSA), // ticket.StateJobCHIUSA è corretto
		)).
		All(ctx)
}

func (r *TicketRepository) FindByServiceCenter(ctx context.Context, scID uuid.UUID) ([]*ent.Ticket, error) {
	return withEdges(r.client.Ticket.Query().
		Where(ticket.HasServiceCenterWith(servicecenter.ID(scID)))).
		All(ctx)
}

func (r *TicketRepository) Create(ctx context.Context, input CreateTicketInput) (*ent.Ticket, error) {
	t, err := r.client.Ticket.Create().
		SetTitle(input.Title).
		SetQuestion(input.Question).
		SetCompanyID(input.CompanyID).
		SetServiceCenterID(input.ServiceCenterID).
		SetCreatedByID(input.CreatedByID).
		SetUpdatedByID(input.CreatedByID).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("ticket Create: %w", err)
	}
	return t, nil
}

func (r *TicketRepository) TakeTicket(ctx context.Context, id, userID uuid.UUID) (*ent.Ticket, error) {
	now := time.Now()
	return r.client.Ticket.UpdateOneID(id).
		SetAssignedToID(userID).
		SetUpdatedByID(userID).
		SetDataAssigned(now).
		SetStateJob(ticket.StateJobPRESA_IN_CARICO).
		Save(ctx)
}

func (r *TicketRepository) Escalate(ctx context.Context, id, newUserID, requestedByID uuid.UUID) (*ent.Ticket, error) {
	return r.client.Ticket.UpdateOneID(id).
		SetAssignedToID(newUserID).
		SetUpdatedByID(requestedByID).
		SetStateJob(ticket.StateJobIN_ATTESA_CENTRO_SERVIZI).
		Save(ctx)
}

func (r *TicketRepository) Assign(ctx context.Context, id, toUserID, byUserID uuid.UUID) (*ent.Ticket, error) {
	now := time.Now()
	return r.client.Ticket.UpdateOneID(id).
		SetAssignedToID(toUserID).
		SetUpdatedByID(byUserID).
		SetDataAssigned(now).
		SetStateJob(ticket.StateJobPRESA_IN_CARICO).
		Save(ctx)
}

func (r *TicketRepository) SetPriority(ctx context.Context, id uuid.UUID, p ticket.Priority, byUserID uuid.UUID) (*ent.Ticket, error) {
	return r.client.Ticket.UpdateOneID(id).
		SetPriority(p).
		SetUpdatedByID(byUserID).
		Save(ctx)
}

func (r *TicketRepository) SetState(ctx context.Context, id uuid.UUID, sj ticket.StateJob, byUserID uuid.UUID) (*ent.Ticket, error) {
	return r.client.Ticket.UpdateOneID(id).
		SetStateJob(sj).
		SetUpdatedByID(byUserID).
		Save(ctx)
}

func (r *TicketRepository) AddComment(ctx context.Context, input AddCommentInput) (*ent.Comment, error) {
	// Calcola il prossimo order_comment
	count, err := r.client.Comment.Query().
		Where(comment.HasTicketWith(ticket.ID(input.TicketID))).
		Count(ctx)
	if err != nil {
		return nil, err
	}
	return r.client.Comment.Create().
		SetTicketID(input.TicketID).
		SetAuthorID(input.AuthorID).
		SetOrderComment(count + 1).
		SetCommentDetail(input.Detail).
		Save(ctx)
}

type CreateTicketInput struct {
	Title           string
	Question        string
	CompanyID       uuid.UUID
	ServiceCenterID uuid.UUID
	CreatedByID     uuid.UUID
}

type AddCommentInput struct {
	TicketID uuid.UUID
	AuthorID uuid.UUID
	Detail   string
}
