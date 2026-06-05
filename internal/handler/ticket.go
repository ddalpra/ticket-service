package handler

import (
	"net/http"

	"ticket-service/internal/service"
	"ticket-service/pkg/apperrors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type TicketHandler struct {
	svc *service.TicketService
}

func NewTicketHandler(svc *service.TicketService) *TicketHandler {
	return &TicketHandler{svc: svc}
}

// GET /api/v1/tickets
func (h *TicketHandler) List(c *gin.Context) {
	tickets, err := h.svc.List(c.Request.Context(), currentUser(c))
	if err != nil {
		handleErr(c, err)
		return
	}
	respondOK(c, tickets)
}

// POST /api/v1/tickets
func (h *TicketHandler) Create(c *gin.Context) {
	var body struct {
		Title           string `json:"title"            binding:"required,max=150"`
		Question        string `json:"question"         binding:"required"`
		ServiceCenterID string `json:"service_center_id" binding:"required,uuid"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	scID, _ := uuid.Parse(body.ServiceCenterID)
	t, err := h.svc.Create(c.Request.Context(), service.CreateTicketInput{
		Title:           body.Title,
		Question:        body.Question,
		ServiceCenterID: scID,
	}, currentUser(c))
	if err != nil {
		handleErr(c, err)
		return
	}
	respondCreated(c, t)
}

// GET /api/v1/tickets/:id
func (h *TicketHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id non valido"})
		return
	}
	t, err := h.svc.Get(c.Request.Context(), id, currentUser(c))
	if err != nil {
		handleErr(c, err)
		return
	}
	respondOK(c, t)
}

// GET /api/v1/tickets/unassigned
func (h *TicketHandler) ListUnassigned(c *gin.Context) {
	tickets, err := h.svc.ListUnassigned(c.Request.Context(), currentUser(c))
	if err != nil {
		handleErr(c, err)
		return
	}
	respondOK(c, tickets)
}

// GET /api/v1/tickets/mine
func (h *TicketHandler) ListMine(c *gin.Context) {
	tickets, err := h.svc.ListMine(c.Request.Context(), currentUser(c))
	if err != nil {
		handleErr(c, err)
		return
	}
	respondOK(c, tickets)
}

// GET /api/v1/tickets/center
func (h *TicketHandler) ListCenter(c *gin.Context) {
	tickets, err := h.svc.ListCenter(c.Request.Context(), currentUser(c))
	if err != nil {
		handleErr(c, err)
		return
	}
	respondOK(c, tickets)
}

// PUT /api/v1/tickets/:id/take
func (h *TicketHandler) Take(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id non valido"})
		return
	}
	t, err := h.svc.Take(c.Request.Context(), id, currentUser(c))
	if err != nil {
		handleErr(c, err)
		return
	}
	respondOK(c, t)
}

// PUT /api/v1/tickets/:id/escalate
func (h *TicketHandler) Escalate(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id non valido"})
		return
	}
	var body struct {
		UserID string `json:"user_id" binding:"required,uuid"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	targetID, _ := uuid.Parse(body.UserID)
	t, err := h.svc.Escalate(c.Request.Context(), id, targetID, currentUser(c))
	if err != nil {
		handleErr(c, err)
		return
	}
	respondOK(c, t)
}

// PUT /api/v1/tickets/:id/assign
func (h *TicketHandler) Assign(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id non valido"})
		return
	}
	var body struct {
		UserID string `json:"user_id" binding:"required,uuid"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	toID, _ := uuid.Parse(body.UserID)
	t, err := h.svc.Assign(c.Request.Context(), id, toID, currentUser(c))
	if err != nil {
		handleErr(c, err)
		return
	}
	respondOK(c, t)
}

// PUT /api/v1/tickets/:id/priority
func (h *TicketHandler) SetPriority(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id non valido"})
		return
	}
	var body struct {
		Priority string `json:"priority" binding:"required,oneof=LOW MEDIUM HIGH"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	t, err := h.svc.SetPriority(c.Request.Context(), id, body.Priority, currentUser(c))
	if err != nil {
		handleErr(c, err)
		return
	}
	respondOK(c, t)
}

// PUT /api/v1/tickets/:id/state
func (h *TicketHandler) SetState(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id non valido"})
		return
	}
	var body struct {
		StateJob string `json:"state_job" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	t, err := h.svc.SetState(c.Request.Context(), id, body.StateJob, currentUser(c))
	if err != nil {
		handleErr(c, err)
		return
	}
	respondOK(c, t)
}

// POST /api/v1/tickets/:id/comments
func (h *TicketHandler) AddComment(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id non valido"})
		return
	}
	var body struct {
		Detail string `json:"detail" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	comment, err := h.svc.AddComment(c.Request.Context(), id, body.Detail, currentUser(c))
	if err != nil {
		handleErr(c, err)
		return
	}
	respondCreated(c, comment)
}

// POST /api/v1/tickets/:id/attachments
// Gestisce l'upload di un file come multipart/form-data.
func (h *TicketHandler) AddAttachment(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id non valido"})
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file mancante nel form"})
		return
	}
	// Salva il file nel filesystem (da estendere con S3 o altro storage)
	savePath := "./uploads/" + id.String() + "_" + file.Filename
	if err := c.SaveUploadedFile(file, savePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "errore salvataggio file"})
		return
	}
	// TODO: chiama s.svc.AddAttachment per persistere i metadati
	respondCreated(c, gin.H{
		"ticket_id": id,
		"name":      file.Filename,
		"path":      savePath,
		"size":      file.Size,
	})
}

// parseID è un helper per evitare la ripetizione.
func parseID(c *gin.Context) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return uuid.Nil, apperrors.BadRequest("id non valido")
	}
	return id, nil
}
