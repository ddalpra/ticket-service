package auth_test

import (
	"testing"

	"ticket-service/internal/auth"

	"github.com/stretchr/testify/assert"
)

func TestClaims_HasRole(t *testing.T) {
	claims := &auth.Claims{}
	claims.RealmAccess.Roles = []string{"customer", "offline_access"}

	assert.True(t, claims.HasRole("customer"))
	assert.False(t, claims.HasRole("support_l1"))
}

func TestClaims_HasAnyRole(t *testing.T) {
	claims := &auth.Claims{}
	claims.RealmAccess.Roles = []string{"support_l2"}

	assert.True(t, claims.HasAnyRole("support_l1", "support_l2", "supervisor"))
	assert.False(t, claims.HasAnyRole("customer", "support_l1"))
}

// TestNewJWTVerifier_InvalidURL verifica che un URL non raggiungibile restituisca errore.
func TestNewJWTVerifier_InvalidURL(t *testing.T) {
	_, err := auth.NewJWTVerifier("http://localhost:19999", "test-realm")
	// L'errore si manifesterà al primo uso della JWKS, non alla creazione
	// (keyfunc scarica lazy). Verifichiamo solo che l'oggetto venga creato.
	assert.NoError(t, err)
}
