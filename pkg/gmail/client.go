// Package gmail gestisce l'autenticazione OAuth2 e le operazioni
// sulla Gmail API (lettura messaggi, labeling).
package gmail

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"mime"
	"mime/multipart"
	"net/mail"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

const tokenFile = "gmail_token.json"

// Message è la rappresentazione semplificata di un messaggio Gmail.
type Message struct {
	ID          string
	ThreadID    string
	Subject     string
	From        string
	FromAddress string // solo la parte email
	FromDomain  string // dominio mittente (dopo la @)
	Body        string
	ReceivedAt  time.Time
}

// Client è il wrapper attorno alla Gmail API.
type Client struct {
	svc           *gmail.Service
	userID        string
	processedLabel string // nome della label "Processata"
}

// NewClient crea un Client Gmail usando OAuth2.
// credentialsPath è il percorso del file credentials.json scaricato da Google Cloud Console.
func NewClient(ctx context.Context, credentialsPath string) (*Client, error) {
	data, err := os.ReadFile(credentialsPath)
	if err != nil {
		return nil, fmt.Errorf("lettura credentials.json: %w", err)
	}

	cfg, err := google.ConfigFromJSON(data, gmail.GmailReadonlyScope, gmail.GmailLabelsScope, gmail.GmailModifyScope)
	if err != nil {
		return nil, fmt.Errorf("parsing credentials.json: %w", err)
	}

	token, err := loadOrCreateToken(ctx, cfg, credentialsPath)
	if err != nil {
		return nil, fmt.Errorf("token OAuth2: %w", err)
	}

	svc, err := gmail.NewService(ctx, option.WithTokenSource(cfg.TokenSource(ctx, token)))
	if err != nil {
		return nil, fmt.Errorf("creazione Gmail service: %w", err)
	}

	c := &Client{svc: svc, userID: "me", processedLabel: "Processata"}

	// Crea la label "Processata" se non esiste
	if err := c.ensureLabel(ctx); err != nil {
		log.Printf("[gmail] avviso: impossibile creare label '%s': %v", c.processedLabel, err)
	}

	return c, nil
}

// FetchUnread restituisce i messaggi non letti nella inbox.
// maxResults limita il numero di messaggi restituiti per ciclo.
func (c *Client) FetchUnread(ctx context.Context, maxResults int64) ([]Message, error) {
	listResp, err := c.svc.Users.Messages.List(c.userID).
		Q("is:unread in:inbox").
		MaxResults(maxResults).
		Context(ctx).
		Do()
	if err != nil {
		return nil, fmt.Errorf("listing messaggi Gmail: %w", err)
	}

	var messages []Message
	for _, m := range listResp.Messages {
		msg, err := c.fetchMessage(ctx, m.Id)
		if err != nil {
			log.Printf("[gmail] errore lettura messaggio %s: %v", m.Id, err)
			continue
		}
		messages = append(messages, *msg)
	}
	return messages, nil
}

// MarkProcessed sposta il messaggio nella label "Processata"
// e lo rimuove dalla inbox.
func (c *Client) MarkProcessed(ctx context.Context, messageID string) error {
	labelID, err := c.getLabelID(ctx, c.processedLabel)
	if err != nil {
		return err
	}

	_, err = c.svc.Users.Messages.Modify(c.userID, messageID, &gmail.ModifyMessageRequest{
		AddLabelIds:    []string{labelID},
		RemoveLabelIds: []string{"INBOX", "UNREAD"},
	}).Context(ctx).Do()
	return err
}

// ── Helpers interni ───────────────────────────────────────────────────────────

func (c *Client) fetchMessage(ctx context.Context, id string) (*Message, error) {
	raw, err := c.svc.Users.Messages.Get(c.userID, id).
		Format("full").
		Context(ctx).
		Do()
	if err != nil {
		return nil, err
	}

	msg := &Message{
		ID:       raw.Id,
		ThreadID: raw.ThreadId,
	}

	// Estrae header
	for _, h := range raw.Payload.Headers {
		switch h.Name {
		case "Subject":
			msg.Subject = h.Value
		case "From":
			msg.From = h.Value
			addr, err := mail.ParseAddress(h.Value)
			if err == nil {
				msg.FromAddress = addr.Address
				parts := strings.SplitN(addr.Address, "@", 2)
				if len(parts) == 2 {
					msg.FromDomain = strings.ToLower(parts[1])
				}
			} else {
				msg.FromAddress = h.Value
			}
		case "Date":
			if t, err := mail.ParseDate(h.Value); err == nil {
				msg.ReceivedAt = t
			}
		}
	}

	msg.Body = extractBody(raw.Payload)
	return msg, nil
}

// extractBody estrae il testo del messaggio (preferisce text/plain).
func extractBody(payload *gmail.MessagePart) string {
	if payload == nil {
		return ""
	}

	// Parte singola
	if payload.Body != nil && payload.Body.Size > 0 {
		data, err := base64.URLEncoding.DecodeString(payload.Body.Data)
		if err == nil {
			ct := payload.MimeType
			if ct == "text/plain" || ct == "text/html" {
				return strings.TrimSpace(string(data))
			}
		}
	}

	// Multipart: cerca text/plain prima, poi text/html
	var htmlFallback string
	for _, part := range payload.Parts {
		switch part.MimeType {
		case "text/plain":
			if part.Body != nil && part.Body.Size > 0 {
				data, err := base64.URLEncoding.DecodeString(part.Body.Data)
				if err == nil {
					return strings.TrimSpace(string(data))
				}
			}
		case "text/html":
			if part.Body != nil && part.Body.Size > 0 {
				data, err := base64.URLEncoding.DecodeString(part.Body.Data)
				if err == nil {
					htmlFallback = strings.TrimSpace(string(data))
				}
			}
		default:
			// Ricorsione per multipart annidati
			if nested := extractBody(part); nested != "" {
				return nested
			}
		}
	}

	if htmlFallback != "" {
		return htmlFallback
	}
	return ""
}

// ensureLabel crea la label se non esiste già.
func (c *Client) ensureLabel(ctx context.Context) error {
	_, err := c.getLabelID(ctx, c.processedLabel)
	if err == nil {
		return nil // già esiste
	}
	_, err = c.svc.Users.Labels.Create(c.userID, &gmail.Label{
		Name:                  c.processedLabel,
		MessageListVisibility: "show",
		LabelListVisibility:   "labelShow",
	}).Context(ctx).Do()
	return err
}

func (c *Client) getLabelID(ctx context.Context, name string) (string, error) {
	resp, err := c.svc.Users.Labels.List(c.userID).Context(ctx).Do()
	if err != nil {
		return "", err
	}
	for _, l := range resp.Labels {
		if strings.EqualFold(l.Name, name) {
			return l.Id, nil
		}
	}
	return "", fmt.Errorf("label '%s' non trovata", name)
}

// ── OAuth2 token management ───────────────────────────────────────────────────

func loadOrCreateToken(ctx context.Context, cfg *oauth2.Config, credPath string) (*oauth2.Token, error) {
	// Cerca il token nella stessa directory delle credentials
	dir := filepath.Dir(credPath)
	tokenPath := filepath.Join(dir, tokenFile)

	// Prova a caricare il token salvato
	if data, err := os.ReadFile(tokenPath); err == nil {
		var token oauth2.Token
		if err := json.Unmarshal(data, &token); err == nil {
			if token.Valid() {
				return &token, nil
			}
			// Token scaduto: prova a rinnovarlo con il refresh token
			if token.RefreshToken != "" {
				src := cfg.TokenSource(ctx, &token)
				newToken, err := src.Token()
				if err == nil {
					saveToken(tokenPath, newToken)
					return newToken, nil
				}
			}
		}
	}

	// Nessun token valido: flusso di autorizzazione interattivo
	authURL := cfg.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("\n[gmail] Nessun token OAuth2 trovato.\n")
	fmt.Printf("Apri questo URL nel browser e autorizza l'applicazione:\n\n%s\n\n", authURL)
	fmt.Print("Incolla il codice di autorizzazione: ")

	var code string
	fmt.Scan(&code)

	token, err := cfg.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("scambio codice OAuth2: %w", err)
	}
	saveToken(tokenPath, token)
	fmt.Printf("[gmail] Token salvato in %s\n", tokenPath)
	return token, nil
}

func saveToken(path string, token *oauth2.Token) {
	data, err := json.Marshal(token)
	if err != nil {
		log.Printf("[gmail] errore serializzazione token: %v", err)
		return
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		log.Printf("[gmail] errore salvataggio token in %s: %v", path, err)
	}
}

// Evita import inutilizzati in caso di rimozione di funzionalità
var _ = mime.FormatMediaType
var _ = multipart.NewReader
