package middleware

import (
	"net/http"
	"strings"

	"ticket-service/internal/auth"
	"ticket-service/internal/repository"

	"github.com/gin-gonic/gin"
)

const (
	KeyClaims = "jwt_claims"
	KeyUser   = "current_user"
)

// JWT estrae e verifica il Bearer token da Authorization header.
func JWT(verifier *auth.JWTVerifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Authorization header mancante o non valido",
			})
			return
		}
		tokenStr := strings.TrimPrefix(header, "Bearer ")
		claims, err := verifier.Verify(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "token non valido: " + err.Error(),
			})
			return
		}
		c.Set(KeyClaims, claims)
		c.Next()
	}
}

// ResolveUser carica l'utente locale dal DB in base al keycloak_id nel JWT.
// Deve essere eseguito dopo il middleware JWT.
func ResolveUser(repo *repository.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := GetClaims(c)
		if claims == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "claims mancanti"})
			return
		}
		user, err := repo.FindByKeycloakID(c.Request.Context(), claims.Subject)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "utente non trovato nel sistema: " + claims.PreferredUsername,
			})
			return
		}
		c.Set(KeyUser, user)
		c.Next()
	}
}

// RequireRole verifica che l'utente abbia almeno uno dei ruoli specificati.
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := GetClaims(c)
		if claims == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "non autenticato"})
			return
		}
		if !claims.HasAnyRole(roles...) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "permesso negato",
			})
			return
		}
		c.Next()
	}
}

// GetClaims recupera i JWT claims dal contesto Gin.
func GetClaims(c *gin.Context) *auth.Claims {
	v, exists := c.Get(KeyClaims)
	if !exists {
		return nil
	}
	claims, _ := v.(*auth.Claims)
	return claims
}
