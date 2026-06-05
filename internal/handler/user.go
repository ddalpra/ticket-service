package handler

import (
	"net/http"

	"ticket-service/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type UserHandler struct {
	svc *service.UserService
}

func NewUserHandler(svc *service.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

// POST /api/v1/admin/users/customer
func (h *UserHandler) RegisterCustomer(c *gin.Context) {
	var body struct {
		Username  string `json:"username"   binding:"required"`
		Email     string `json:"email"      binding:"required,email"`
		FirstName string `json:"first_name" binding:"required"`
		LastName  string `json:"last_name"  binding:"required"`
		Password  string `json:"password"   binding:"required,min=8"`
		CompanyID string `json:"company_id" binding:"required,uuid"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	compID, _ := uuid.Parse(body.CompanyID)
	u, err := h.svc.RegisterCustomer(c.Request.Context(), service.RegisterCustomerInput{
		Username:  body.Username,
		Email:     body.Email,
		FirstName: body.FirstName,
		LastName:  body.LastName,
		Password:  body.Password,
		CompanyID: compID,
	})
	if err != nil {
		handleErr(c, err)
		return
	}
	respondCreated(c, u)
}

// POST /api/v1/admin/users/support
func (h *UserHandler) RegisterSupport(c *gin.Context) {
	var body struct {
		Username        string `json:"username"          binding:"required"`
		Email           string `json:"email"             binding:"required,email"`
		FirstName       string `json:"first_name"        binding:"required"`
		LastName        string `json:"last_name"         binding:"required"`
		Password        string `json:"password"          binding:"required,min=8"`
		Role            string `json:"role"              binding:"required,oneof=support_l1 support_l2 supervisor"`
		ServiceCenterID string `json:"service_center_id" binding:"required,uuid"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	scID, _ := uuid.Parse(body.ServiceCenterID)
	u, err := h.svc.RegisterSupport(c.Request.Context(), service.RegisterSupportInput{
		Username:        body.Username,
		Email:           body.Email,
		FirstName:       body.FirstName,
		LastName:        body.LastName,
		Password:        body.Password,
		Role:            body.Role,
		ServiceCenterID: scID,
	})
	if err != nil {
		handleErr(c, err)
		return
	}
	respondCreated(c, u)
}

// GET /api/v1/admin/users
func (h *UserHandler) ListCenterUsers(c *gin.Context) {
	users, err := h.svc.ListCenterUsers(c.Request.Context(), currentUser(c))
	if err != nil {
		handleErr(c, err)
		return
	}
	respondOK(c, users)
}

// PUT /api/v1/admin/users/:id/active
func (h *UserHandler) SetActive(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id non valido"})
		return
	}
	var body struct {
		Active bool `json:"active"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	u, err := h.svc.SetActive(c.Request.Context(), id, body.Active, currentUser(c))
	if err != nil {
		handleErr(c, err)
		return
	}
	respondOK(c, u)
}
