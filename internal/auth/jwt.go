package auth

import (
	"context"
	"fmt"
	"strings"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

// Claims rappresenta i claim del JWT emesso da Keycloak.
type Claims struct {
	jwt.RegisteredClaims
	RealmAccess struct {
		Roles []string `json:"roles"`
	} `json:"realm_access"`
	PreferredUsername string `json:"preferred_username"`
	Email             string `json:"email"`
	GivenName         string `json:"given_name"`
	FamilyName        string `json:"family_name"`
}

// HasRole restituisce true se il claim contiene il ruolo specificato.
func (c *Claims) HasRole(role string) bool {
	for _, r := range c.RealmAccess.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// HasAnyRole restituisce true se il claim contiene almeno uno dei ruoli.
func (c *Claims) HasAnyRole(roles ...string) bool {
	for _, r := range roles {
		if c.HasRole(r) {
			return true
		}
	}
	return false
}

// JWTVerifier verifica i JWT usando la JWKS pubblica di Keycloak.
type JWTVerifier struct {
	jwks     keyfunc.Keyfunc
	issuer   string
}

// NewJWTVerifier crea un verifier che scarica automaticamente la JWKS da Keycloak.
func NewJWTVerifier(keycloakURL, realm string) (*JWTVerifier, error) {
	jwksURL := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/certs",
		strings.TrimRight(keycloakURL, "/"), realm)

	jwks, err := keyfunc.NewDefaultCtx(context.Background(), []string{jwksURL})
	if err != nil {
		return nil, fmt.Errorf("inizializzazione JWKS: %w", err)
	}

	issuer := fmt.Sprintf("%s/realms/%s",
		strings.TrimRight(keycloakURL, "/"), realm)

	return &JWTVerifier{jwks: jwks, issuer: issuer}, nil
}

// Verify analizza e valida il token JWT. Restituisce i Claims se valido.
func (v *JWTVerifier) Verify(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, v.jwks.Keyfunc,
		jwt.WithIssuer(v.issuer),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, fmt.Errorf("token non valido: %w", err)
	}
	if !token.Valid {
		return nil, fmt.Errorf("token non valido")
	}
	return claims, nil
}
