package handler

import (
	"net/http"

	"ticket-service/internal/repository"
	"ticket-service/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ServiceCenterHandler struct {
	svc *service.ServiceCenterService
}

func NewServiceCenterHandler(svc *service.ServiceCenterService) *ServiceCenterHandler {
	return &ServiceCenterHandler{svc: svc}
}

// GET /api/v1/service-centers
func (h *ServiceCenterHandler) List(c *gin.Context) {
	scs, err := h.svc.List(c.Request.Context())
	if err != nil {
		handleErr(c, err)
		return
	}
	respondOK(c, scs)
}

// POST /api/v1/service-centers
func (h *ServiceCenterHandler) Create(c *gin.Context) {
	var body struct {
		Codice string `json:"codice" binding:"required,max=20"`
		Nome   string `json:"nome"   binding:"required,max=200"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	sc, err := h.svc.Create(c.Request.Context(), repository.CreateSCInput{
		Codice: body.Codice,
		Nome:   body.Nome,
	})
	if err != nil {
		handleErr(c, err)
		return
	}
	respondCreated(c, sc)
}

// GET /api/v1/service-centers/:id
func (h *ServiceCenterHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id non valido"})
		return
	}
	sc, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		handleErr(c, err)
		return
	}
	respondOK(c, sc)
}
