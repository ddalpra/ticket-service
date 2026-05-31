package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"ticket-service/internal/repository"
	"ticket-service/internal/service"
)

type CompanyHandler struct {
	svc *service.CompanyService
}

func NewCompanyHandler(svc *service.CompanyService) *CompanyHandler {
	return &CompanyHandler{svc: svc}
}

// GET /api/v1/companies
func (h *CompanyHandler) List(c *gin.Context) {
	companies, err := h.svc.List(c.Request.Context())
	if err != nil {
		handleErr(c, err)
		return
	}
	respondOK(c, companies)
}

// POST /api/v1/companies
func (h *CompanyHandler) Create(c *gin.Context) {
	var body struct {
		Codice         string `json:"codice"          binding:"required,max=15"`
		RagioneSociale string `json:"ragione_sociale" binding:"required,max=200"`
		Piva           string `json:"piva"            binding:"required,max=16"`
		Cf             string `json:"cf"              binding:"required,max=16"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	company, err := h.svc.Create(c.Request.Context(), repository.CreateCompanyInput{
		Codice:         body.Codice,
		RagioneSociale: body.RagioneSociale,
		Piva:           body.Piva,
		Cf:             body.Cf,
	})
	if err != nil {
		handleErr(c, err)
		return
	}
	respondCreated(c, company)
}

// GET /api/v1/companies/:id
func (h *CompanyHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id non valido"})
		return
	}
	company, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		handleErr(c, err)
		return
	}
	respondOK(c, company)
}

// PUT /api/v1/companies/:id
func (h *CompanyHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id non valido"})
		return
	}
	var body struct {
		RagioneSociale *string `json:"ragione_sociale"`
		Active         *bool   `json:"active"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	company, err := h.svc.Update(c.Request.Context(), id, repository.UpdateCompanyInput{
		RagioneSociale: body.RagioneSociale,
		Active:         body.Active,
	})
	if err != nil {
		handleErr(c, err)
		return
	}
	respondOK(c, company)
}
