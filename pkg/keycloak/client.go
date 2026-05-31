package keycloak

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client gestisce le chiamate all'API admin di Keycloak.
type Client struct {
	baseURL      string
	realm        string
	clientID     string
	clientSecret string
	httpClient   *http.Client
}

func NewClient(baseURL, realm, clientID, clientSecret string) *Client {
	return &Client{
		baseURL:      strings.TrimRight(baseURL, "/"),
		realm:        realm,
		clientID:     clientID,
		clientSecret: clientSecret,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
	}
}

// ── Token ─────────────────────────────────────────────────────────────────────

func (c *Client) adminToken(ctx context.Context) (string, error) {
	data := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {c.clientID},
		"client_secret": {c.clientSecret},
	}
	resp, err := c.httpClient.PostForm(
		fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token", c.baseURL, c.realm),
		data,
	)
	if err != nil {
		return "", fmt.Errorf("keycloak token request: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("keycloak token decode: %w", err)
	}
	if result.Error != "" {
		return "", fmt.Errorf("keycloak token error: %s", result.Error)
	}
	return result.AccessToken, nil
}

// ── User creation ─────────────────────────────────────────────────────────────

// CreateUserRequest contiene i dati per creare un utente in Keycloak.
type CreateUserRequest struct {
	Username  string
	Email     string
	FirstName string
	LastName  string
	Password  string
	Roles     []string
}

// CreateUser crea l'utente nel realm e assegna i ruoli. Restituisce il keycloak_id (UUID sub).
func (c *Client) CreateUser(ctx context.Context, req CreateUserRequest) (string, error) {
	token, err := c.adminToken(ctx)
	if err != nil {
		return "", err
	}

	body := map[string]any{
		"username":      req.Username,
		"email":         req.Email,
		"firstName":     req.FirstName,
		"lastName":      req.LastName,
		"enabled":       true,
		"emailVerified": true,
		"credentials": []map[string]any{
			{"type": "password", "value": req.Password, "temporary": false},
		},
	}

	userID, err := c.postAdmin(ctx, token,
		fmt.Sprintf("/admin/realms/%s/users", c.realm), body)
	if err != nil {
		return "", fmt.Errorf("crea utente keycloak: %w", err)
	}

	// Assegna i ruoli realm
	if len(req.Roles) > 0 {
		if err := c.assignRealmRoles(ctx, token, userID, req.Roles); err != nil {
			return "", fmt.Errorf("assegna ruoli keycloak: %w", err)
		}
	}
	return userID, nil
}

// assignRealmRoles assegna i ruoli realm all'utente.
func (c *Client) assignRealmRoles(ctx context.Context, token, userID string, roles []string) error {
	// Recupera i role ID dal realm
	type roleRep struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	var allRoles []roleRep
	if err := c.getAdmin(ctx, token,
		fmt.Sprintf("/admin/realms/%s/roles", c.realm), &allRoles); err != nil {
		return err
	}

	roleMap := make(map[string]roleRep, len(allRoles))
	for _, r := range allRoles {
		roleMap[r.Name] = r
	}

	var toAssign []map[string]any
	for _, name := range roles {
		r, ok := roleMap[name]
		if !ok {
			return fmt.Errorf("ruolo '%s' non trovato nel realm", name)
		}
		toAssign = append(toAssign, map[string]any{"id": r.ID, "name": r.Name})
	}

	_, err := c.postAdmin(ctx, token,
		fmt.Sprintf("/admin/realms/%s/users/%s/role-mappings/realm", c.realm, userID),
		toAssign)
	return err
}

// UserRepresentation è la rappresentazione minimale di un utente Keycloak.
type UserRepresentation struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Enabled   bool   `json:"enabled"`
}

// ListUsers restituisce tutti gli utenti del realm (usato dal bootstrap).
func (c *Client) ListUsers(ctx context.Context) ([]UserRepresentation, error) {
	token, err := c.adminToken(ctx)
	if err != nil {
		return nil, err
	}
	var users []UserRepresentation
	if err := c.getAdmin(ctx, token,
		fmt.Sprintf("/admin/realms/%s/users?max=200", c.realm), &users); err != nil {
		return nil, err
	}
	return users, nil
}

// DisableUser disabilita un utente in Keycloak.
func (c *Client) DisableUser(ctx context.Context, keycloakID string) error {
	token, err := c.adminToken(ctx)
	if err != nil {
		return err
	}
	return c.putAdmin(ctx, token,
		fmt.Sprintf("/admin/realms/%s/users/%s", c.realm, keycloakID),
		map[string]any{"enabled": false})
}

// ── HTTP helpers ──────────────────────────────────────────────────────────────

// postAdmin invia una POST all'admin API e restituisce l'ID dalla Location header.
func (c *Client) postAdmin(ctx context.Context, token, path string, body any) (string, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+path, bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 && resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("keycloak POST %s → %d: %s", path, resp.StatusCode, string(body))
	}

	// L'ID è nell'header Location: .../users/{id}
	loc := resp.Header.Get("Location")
	if loc != "" {
		parts := strings.Split(loc, "/")
		return parts[len(parts)-1], nil
	}
	return "", nil
}

func (c *Client) putAdmin(ctx context.Context, token, path string, body any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut,
		c.baseURL+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("keycloak PUT %s → %d: %s", path, resp.StatusCode, string(body))
	}
	return nil
}

func (c *Client) getAdmin(ctx context.Context, token, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("keycloak GET %s → %d: %s", path, resp.StatusCode, string(body))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
