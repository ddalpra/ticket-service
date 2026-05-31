package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"ticket-service/ent"
	"ticket-service/internal/middleware"
	"ticket-service/pkg/apperrors"
)

// currentUser recupera l'utente corrente dal contesto Gin.
func currentUser(c *gin.Context) *ent.User {
	v, _ := c.Get(middleware.KeyUser)
	u, _ := v.(*ent.User)
	return u
}

// respond invia una risposta JSON con il codice HTTP specificato.
func respond(c *gin.Context, code int, data any) {
	c.JSON(code, data)
}

// respondOK invia 200 con i dati.
func respondOK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, data)
}

// respondCreated invia 201 con i dati.
func respondCreated(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, data)
}

// handleErr mappa un errore applicativo alla risposta HTTP appropriata.
func handleErr(c *gin.Context, err error) {
	if ae, ok := apperrors.As(err); ok {
		c.JSON(ae.Code, gin.H{"error": ae.Message})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": "errore interno del server"})
}
