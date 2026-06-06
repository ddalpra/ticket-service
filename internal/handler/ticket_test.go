package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ticket-service/ent"
	entticket "ticket-service/ent/ticket"
	"ticket-service/internal/handler"
	"ticket-service/internal/middleware"
	"ticket-service/internal/service"
	"ticket-service/pkg/apperrors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ── Mock TicketService ────────────────────────────────────────────────────────

type mockTicketSvc struct{ mock.Mock }

func (m *mockTicketSvc) List(ctx context.Context, caller *ent.User) ([]*ent.Ticket, error) {
	args := m.Called(ctx, caller)
	return args.Get(0).([]*ent.Ticket), args.Error(1)
}
func (m *mockTicketSvc) Get(ctx context.Context, id uuid.UUID, caller *ent.User) (*ent.Ticket, error) {
	args := m.Called(ctx, id, caller)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ent.Ticket), args.Error(1)
}
func (m *mockTicketSvc) Create(ctx context.Context, input service.CreateTicketInput, caller *ent.User) (*ent.Ticket, error) {
	args := m.Called(ctx, input, caller)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ent.Ticket), args.Error(1)
}
func (m *mockTicketSvc) ListUnassigned(ctx context.Context, caller *ent.User) ([]*ent.Ticket, error) {
	args := m.Called(ctx, caller)
	return args.Get(0).([]*ent.Ticket), args.Error(1)
}
func (m *mockTicketSvc) ListMine(ctx context.Context, caller *ent.User) ([]*ent.Ticket, error) {
	args := m.Called(ctx, caller)
	return args.Get(0).([]*ent.Ticket), args.Error(1)
}
func (m *mockTicketSvc) ListCenter(ctx context.Context, caller *ent.User) ([]*ent.Ticket, error) {
	args := m.Called(ctx, caller)
	return args.Get(0).([]*ent.Ticket), args.Error(1)
}
func (m *mockTicketSvc) Take(ctx context.Context, id uuid.UUID, caller *ent.User) (*ent.Ticket, error) {
	args := m.Called(ctx, id, caller)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ent.Ticket), args.Error(1)
}
func (m *mockTicketSvc) AddComment(ctx context.Context, id uuid.UUID, detail string, caller *ent.User) (*ent.Comment, error) {
	args := m.Called(ctx, id, detail, caller)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ent.Comment), args.Error(1)
}

// ── Setup router di test ──────────────────────────────────────────────────────

func setupRouter(caller *ent.User, svc *mockTicketSvc) *gin.Engine {
	r := gin.New()
	// Inietta l'utente corrente senza passare per JWT/DB
	injectUser := func(c *gin.Context) {
		c.Set(middleware.KeyUser, caller)
		c.Next()
	}
	h := handler.NewTicketHandler(svc)
	v1 := r.Group("/api/v1", injectUser)
	v1.GET("/tickets", h.List)
	v1.GET("/tickets/mine", h.ListMine)
	v1.GET("/tickets/unassigned", h.ListUnassigned)
	v1.GET("/tickets/:id", h.Get)
	v1.POST("/tickets", h.Create)
	v1.PUT("/tickets/:id/take", h.Take)
	v1.POST("/tickets/:id/comments", h.AddComment)
	return r
}

func customerUser() *ent.User {
	u := &ent.User{}
	u.ID = uuid.New()
	u.Role = "customer"
	u.Edges.Company = &ent.Company{ID: uuid.New()}
	return u
}

func supportUser() *ent.User {
	u := &ent.User{}
	u.ID = uuid.New()
	u.Role = "support_l1"
	u.Edges.ServiceCenter = &ent.ServiceCenter{ID: uuid.New()}
	return u
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestListTickets_200(t *testing.T) {
	caller := customerUser()
	svc := new(mockTicketSvc)
	tickets := []*ent.Ticket{{ID: uuid.New(), Title: "Problema rete"}}
	svc.On("List", mock.Anything, caller).Return(tickets, nil)

	r := setupRouter(caller, svc)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/tickets", nil))

	assert.Equal(t, http.StatusOK, w.Code)
	var result []map[string]any
	json.Unmarshal(w.Body.Bytes(), &result)
	assert.Len(t, result, 1)
	svc.AssertExpectations(t)
}

func TestCreateTicket_201(t *testing.T) {
	caller := customerUser()
	scID := uuid.New()
	svc := new(mockTicketSvc)

	newTicket := &ent.Ticket{
		ID:       uuid.New(),
		Title:    "Stampante rotta",
		Question: "La stampante non funziona",
		StateJob: entticket.StateJobAPERTA,
	}
	svc.On("Create", mock.Anything, service.CreateTicketInput{
		Title:           "Stampante rotta",
		Question:        "La stampante non funziona",
		ServiceCenterID: scID,
	}, caller).Return(newTicket, nil)

	body, _ := json.Marshal(map[string]string{
		"title":             "Stampante rotta",
		"question":          "La stampante non funziona",
		"service_center_id": scID.String(),
	})
	r := setupRouter(caller, svc)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/tickets",
		bytes.NewReader(body)))

	assert.Equal(t, http.StatusCreated, w.Code)
	svc.AssertExpectations(t)
}

func TestCreateTicket_400_MissingFields(t *testing.T) {
	caller := customerUser()
	svc := new(mockTicketSvc)

	body, _ := json.Marshal(map[string]string{"title": "solo titolo"})
	r := setupRouter(caller, svc)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/tickets",
		bytes.NewReader(body)))

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetTicket_404(t *testing.T) {
	caller := customerUser()
	svc := new(mockTicketSvc)
	id := uuid.New()
	svc.On("Get", mock.Anything, id, caller).Return(nil, apperrors.NotFound("ticket non trovato"))

	r := setupRouter(caller, svc)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/tickets/"+id.String(), nil))

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestTakeTicket_200(t *testing.T) {
	caller := supportUser()
	svc := new(mockTicketSvc)
	id := uuid.New()
	taken := &ent.Ticket{ID: id, StateJob: entticket.StateJobPRESA_IN_CARICO}
	svc.On("Take", mock.Anything, id, caller).Return(taken, nil)

	r := setupRouter(caller, svc)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/v1/tickets/"+id.String()+"/take", nil))

	assert.Equal(t, http.StatusOK, w.Code)
	svc.AssertExpectations(t)
}

func TestAddComment_201(t *testing.T) {
	caller := customerUser()
	svc := new(mockTicketSvc)
	ticketID := uuid.New()
	comment := &ent.Comment{ID: uuid.New(), OrderComment: 1}
	svc.On("AddComment", mock.Anything, ticketID, "Il problema persiste", caller).
		Return(comment, nil)

	body, _ := json.Marshal(map[string]string{"detail": "Il problema persiste"})
	r := setupRouter(caller, svc)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost,
		"/api/v1/tickets/"+ticketID.String()+"/comments",
		bytes.NewReader(body)))

	assert.Equal(t, http.StatusCreated, w.Code)
	svc.AssertExpectations(t)
}
