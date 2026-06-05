package service_test

import (
	"context"
	"testing"

	"ticket-service/ent"
	entticket "ticket-service/ent/ticket"
	"ticket-service/internal/service"
	"ticket-service/pkg/apperrors"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ── Mock TicketRepository ─────────────────────────────────────────────────────

type mockTicketRepo struct{ mock.Mock }

func (m *mockTicketRepo) FindByID(ctx context.Context, id uuid.UUID) (*ent.Ticket, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ent.Ticket), args.Error(1)
}
func (m *mockTicketRepo) FindByCompany(ctx context.Context, id uuid.UUID) ([]*ent.Ticket, error) {
	args := m.Called(ctx, id)
	return args.Get(0).([]*ent.Ticket), args.Error(1)
}
func (m *mockTicketRepo) FindByAssignedUser(ctx context.Context, id uuid.UUID) ([]*ent.Ticket, error) {
	args := m.Called(ctx, id)
	return args.Get(0).([]*ent.Ticket), args.Error(1)
}
func (m *mockTicketRepo) FindUnassignedByServiceCenter(ctx context.Context, id uuid.UUID) ([]*ent.Ticket, error) {
	args := m.Called(ctx, id)
	return args.Get(0).([]*ent.Ticket), args.Error(1)
}
func (m *mockTicketRepo) FindByServiceCenter(ctx context.Context, id uuid.UUID) ([]*ent.Ticket, error) {
	args := m.Called(ctx, id)
	return args.Get(0).([]*ent.Ticket), args.Error(1)
}
func (m *mockTicketRepo) TakeTicket(ctx context.Context, id, userID uuid.UUID) (*ent.Ticket, error) {
	args := m.Called(ctx, id, userID)
	return args.Get(0).(*ent.Ticket), args.Error(1)
}

// ── Mock UserRepository ───────────────────────────────────────────────────────

type mockUserRepo struct{ mock.Mock }

func (m *mockUserRepo) FindByID(ctx context.Context, id uuid.UUID) (*ent.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ent.User), args.Error(1)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func customerUser(companyID uuid.UUID) *ent.User {
	u := &ent.User{}
	u.ID = uuid.New()
	u.Role = entticket.RoleCustomer // nota: use ent/user.RoleCustomer nella app reale
	u.Edges.Company = &ent.Company{ID: companyID}
	return u
}

func supportL1User(scID uuid.UUID) *ent.User {
	u := &ent.User{}
	u.ID = uuid.New()
	u.Role = "support_l1"
	u.Edges.ServiceCenter = &ent.ServiceCenter{ID: scID}
	return u
}

func openTicket(companyID, scID uuid.UUID) *ent.Ticket {
	t := &ent.Ticket{}
	t.ID = uuid.New()
	t.StateJob = entticket.StateJobAPERTA
	t.Edges.Company = &ent.Company{ID: companyID}
	t.Edges.ServiceCenter = &ent.ServiceCenter{ID: scID}
	return t
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestTicketService_List_Customer(t *testing.T) {
	companyID := uuid.New()
	caller := customerUser(companyID)

	repoMock := new(mockTicketRepo)
	expected := []*ent.Ticket{openTicket(companyID, uuid.New())}
	repoMock.On("FindByCompany", mock.Anything, companyID).Return(expected, nil)

	svc := service.NewTicketService(repoMock, nil)
	tickets, err := svc.List(context.Background(), caller)

	assert.NoError(t, err)
	assert.Len(t, tickets, 1)
	repoMock.AssertExpectations(t)
}

func TestTicketService_List_SupportL1_ReturnsMine(t *testing.T) {
	scID := uuid.New()
	caller := supportL1User(scID)

	repoMock := new(mockTicketRepo)
	repoMock.On("FindByAssignedUser", mock.Anything, caller.ID).
		Return([]*ent.Ticket{}, nil)

	svc := service.NewTicketService(repoMock, nil)
	tickets, err := svc.List(context.Background(), caller)

	assert.NoError(t, err)
	assert.Empty(t, tickets)
	repoMock.AssertExpectations(t)
}

func TestTicketService_Take_AlreadyAssigned(t *testing.T) {
	scID := uuid.New()
	caller := supportL1User(scID)
	ticketID := uuid.New()

	existingUser := &ent.User{ID: uuid.New()}
	t_ := openTicket(uuid.New(), scID)
	t_.Edges.AssignedTo = existingUser

	repoMock := new(mockTicketRepo)
	repoMock.On("FindByID", mock.Anything, ticketID).Return(t_, nil)

	svc := service.NewTicketService(repoMock, nil)
	_, err := svc.Take(context.Background(), ticketID, caller)

	assert.Error(t, err)
	ae, ok := apperrors.As(err)
	assert.True(t, ok)
	assert.Equal(t, 409, ae.Code)
}

func TestTicketService_Take_WrongServiceCenter(t *testing.T) {
	scID := uuid.New()
	otherSC := uuid.New()
	caller := supportL1User(scID)
	ticketID := uuid.New()

	t_ := openTicket(uuid.New(), otherSC) // ticket appartiene ad altro centro

	repoMock := new(mockTicketRepo)
	repoMock.On("FindByID", mock.Anything, ticketID).Return(t_, nil)

	svc := service.NewTicketService(repoMock, nil)
	_, err := svc.Take(context.Background(), ticketID, caller)

	assert.Error(t, err)
	ae, ok := apperrors.As(err)
	assert.True(t, ok)
	assert.Equal(t, 403, ae.Code)
}

func TestTicketService_Take_Success(t *testing.T) {
	scID := uuid.New()
	caller := supportL1User(scID)
	ticketID := uuid.New()

	t_ := openTicket(uuid.New(), scID)
	updated := openTicket(uuid.New(), scID)
	updated.Edges.AssignedTo = caller
	updated.StateJob = entticket.StateJobPRESA_IN_CARICO

	repoMock := new(mockTicketRepo)
	repoMock.On("FindByID", mock.Anything, ticketID).Return(t_, nil)
	repoMock.On("TakeTicket", mock.Anything, ticketID, caller.ID).Return(updated, nil)

	svc := service.NewTicketService(repoMock, nil)
	result, err := svc.Take(context.Background(), ticketID, caller)

	assert.NoError(t, err)
	assert.Equal(t, entticket.StateJobPRESA_IN_CARICO, result.StateJob)
	repoMock.AssertExpectations(t)
}

func TestTicketService_Get_CustomerCannotSeeOtherCompany(t *testing.T) {
	companyID := uuid.New()
	otherCompany := uuid.New()
	caller := customerUser(companyID)
	ticketID := uuid.New()

	t_ := openTicket(otherCompany, uuid.New())

	repoMock := new(mockTicketRepo)
	repoMock.On("FindByID", mock.Anything, ticketID).Return(t_, nil)

	svc := service.NewTicketService(repoMock, nil)
	_, err := svc.Get(context.Background(), ticketID, caller)

	assert.Error(t, err)
	ae, ok := apperrors.As(err)
	assert.True(t, ok)
	assert.Equal(t, 403, ae.Code)
}
