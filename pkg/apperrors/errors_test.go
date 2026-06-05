package apperrors_test

import (
	"errors"
	"net/http"
	"testing"

	"ticket-service/pkg/apperrors"

	"github.com/stretchr/testify/assert"
)

func TestNotFound(t *testing.T) {
	err := apperrors.NotFound("risorsa non trovata")
	assert.Equal(t, http.StatusNotFound, err.Code)
	assert.Equal(t, "risorsa non trovata", err.Message)
}

func TestInternal_WrapsError(t *testing.T) {
	inner := errors.New("db timeout")
	err := apperrors.Internal(inner)
	assert.Equal(t, http.StatusInternalServerError, err.Code)
	assert.True(t, errors.Is(err, inner))
}

func TestAs_ExtractsAppError(t *testing.T) {
	err := apperrors.Forbidden("accesso negato")
	ae, ok := apperrors.As(err)
	assert.True(t, ok)
	assert.Equal(t, http.StatusForbidden, ae.Code)
}

func TestAs_NonAppError(t *testing.T) {
	err := errors.New("errore generico")
	_, ok := apperrors.As(err)
	assert.False(t, ok)
}
