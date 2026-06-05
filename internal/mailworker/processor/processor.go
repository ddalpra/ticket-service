// Package processor contiene la logica di business del mail worker:
// decide se creare un nuovo ticket o aggiungere un commento a uno esistente.
package processor

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"ticket-service/ent"
	"ticket-service/ent/comment"
	"ticket-service/ent/company"
	"ticket-service/ent/servicecenter"
	"ticket-service/ent/ticket"
	"ticket-service/ent/user"
	"ticket-service/pkg/gmail"

	"github.com/google/uuid"
)

// Action descrive cosa ha fatto il processor con l'email.
type Action string

const (
	ActionCreatedTicket Action = "ticket_creato"
	ActionAddedComment  Action = "commento_aggiunto"
)

// Result è il risultato dell'elaborazione di un'email.
type Result struct {
	Action   Action
	TicketID string
}

// Processor elabora un singolo messaggio Gmail.
type Processor struct {
	db                     *ent.Client
	defaultServiceCenterID string
}

// New crea un Processor.
func New(db *ent.Client, defaultServiceCenterID string) *Processor {
	return &Processor{db: db, defaultServiceCenterID: defaultServiceCenterID}
}

// Process elabora un messaggio Gmail:
//   - cerca un ticket esistente con stesso oggetto e stesso dominio mittente
//   - se trovato: aggiunge l'email come commento
//   - se non trovato: crea un nuovo ticket (con o senza cliente)
func (p *Processor) Process(ctx context.Context, msg gmail.Message) (*Result, error) {
	// ── 1. Cerca ticket esistente ─────────────────────────────────────────────
	existing, err := p.findExistingTicket(ctx, msg.Subject, msg.FromDomain)
	if err != nil {
		return nil, fmt.Errorf("ricerca ticket esistente: %w", err)
	}

	if existing != nil {
		// ── 2a. Ticket trovato → aggiungi come commento ───────────────────────
		if err := p.addComment(ctx, existing, msg); err != nil {
			return nil, fmt.Errorf("aggiunta commento: %w", err)
		}
		return &Result{Action: ActionAddedComment, TicketID: existing.ID.String()}, nil
	}

	// ── 2b. Nessun ticket → crea nuovo ───────────────────────────────────────
	t, err := p.createTicket(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("creazione ticket: %w", err)
	}
	return &Result{Action: ActionCreatedTicket, TicketID: t.ID.String()}, nil
}

// ── Ricerca ticket esistente ──────────────────────────────────────────────────

// findExistingTicket cerca un ticket APERTO con:
//   - titolo uguale all'oggetto dell'email (case-insensitive)
//   - dominio del mittente uguale al dominio dell'azienda cliente del ticket
func (p *Processor) findExistingTicket(ctx context.Context, subject, fromDomain string) (*ent.Ticket, error) {
	if subject == "" || fromDomain == "" {
		return nil, nil
	}

	// Normalizza il soggetto (rimuove Re:, Fwd: ecc.)
	normalizedSubject := normalizeSubject(subject)

	// Cerca tra i ticket non chiusi
	tickets, err := p.db.Ticket.Query().
		Where(
			ticket.StateJobNEQ(ticket.StateJobCHIUSA),
		).
		WithCompany().
		All(ctx)
	if err != nil {
		return nil, err
	}

	for _, t := range tickets {
		// Confronta il titolo normalizzato
		if !strings.EqualFold(normalizeSubject(t.Title), normalizedSubject) {
			continue
		}

		// Verifica che il dominio del mittente corrisponda all'azienda
		// (controlla nel dominio email dell'azienda o nel codice fiscale — usiamo il dominio nell'email dei clienti)
		if t.Edges.Company != nil {
			// Cerca utenti dell'azienda con quel dominio email
			domainMatch, err := p.db.User.Query().
				Where(
					user.HasCompanyWith(company.ID(t.Edges.Company.ID)),
					user.EmailContains("@"+fromDomain),
				).
				Exist(ctx)
			if err != nil {
				log.Printf("[processor] errore verifica dominio: %v", err)
				continue
			}
			if domainMatch {
				return t, nil
			}
		}

		// Ticket senza cliente: confronta solo il titolo
		if t.Edges.Company == nil {
			return t, nil
		}
	}

	return nil, nil
}

// ── Crea nuovo ticket ─────────────────────────────────────────────────────────

func (p *Processor) createTicket(ctx context.Context, msg gmail.Message) (*ent.Ticket, error) {
	// Cerca il centro servizi di default
	scID, err := p.resolveServiceCenter(ctx)
	if err != nil {
		return nil, err
	}

	// Tronca il titolo a 150 caratteri
	title := truncate(msg.Subject, 150)
	if title == "" {
		title = fmt.Sprintf("Email da %s — %s", msg.FromAddress, msg.ReceivedAt.Format("02/01/2006 15:04"))
	}

	// Corpo del messaggio come testo del ticket
	body := buildBody(msg)

	q := p.db.Ticket.Create().
		SetTitle(title).
		SetQuestion(body).
		SetServiceCenterID(scID).
		SetState(ticket.StateOPEN).
		SetStateJob(ticket.StateJobAPERTA).
		SetPriority(ticket.PriorityLOW)

	// Prova ad associare il cliente se il dominio è noto
	companyID, err := p.findCompanyByDomain(ctx, msg.FromDomain)
	if err != nil {
		log.Printf("[processor] avviso: ricerca azienda per dominio '%s': %v", msg.FromDomain, err)
	}
	if companyID != nil {
		q = q.SetCompanyID(*companyID)
		log.Printf("[processor] ticket associato all'azienda con dominio '%s'", msg.FromDomain)
	} else {
		log.Printf("[processor] dominio '%s' non riconosciuto — ticket senza cliente", msg.FromDomain)
	}

	t, err := q.Save(ctx)
	if err != nil {
		return nil, err
	}

	log.Printf("[processor] creato ticket '%s' (id: %s)", title, t.ID)
	return t, nil
}

// ── Aggiunge commento ─────────────────────────────────────────────────────────

func (p *Processor) addComment(ctx context.Context, t *ent.Ticket, msg gmail.Message) error {
	// Conta i commenti esistenti per impostare order_comment
	count, err := p.db.Comment.Query().
		Where(comment.HasTicketWith(ticket.ID(t.ID))).
		Count(ctx)
	if err != nil {
		return err
	}

	body := fmt.Sprintf(
		"📧 Email ricevuta da: %s\nData: %s\n\n%s",
		msg.From,
		msg.ReceivedAt.Format("02/01/2006 15:04"),
		msg.Body,
	)

	_, err = p.db.Comment.Create().
		SetTicketID(t.ID).
		SetOrderComment(count + 1).
		SetCommentDetail(truncate(body, 10000)).
		Save(ctx)
	if err != nil {
		return err
	}

	// Aggiorna data_updating del ticket
	_, err = p.db.Ticket.UpdateOneID(t.ID).
		SetDataUpdating(time.Now()).
		Save(ctx)
	return err
}

// ── Helper: risolve il centro servizi di default ──────────────────────────────

func (p *Processor) resolveServiceCenter(ctx context.Context) (uuid.UUID, error) {
	if p.defaultServiceCenterID != "" {
		id, err := uuid.Parse(p.defaultServiceCenterID)
		if err == nil {
			return id, nil
		}
		log.Printf("[processor] MAIL_DEFAULT_SC_ID non valido: %v", err)
	}

	// Fallback: usa il primo centro servizi disponibile
	sc, err := p.db.ServiceCenter.Query().
		Where(servicecenter.Active(true)).
		First(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("nessun centro servizi attivo trovato: %w", err)
	}
	return sc.ID, nil
}

// ── Helper: cerca azienda per dominio email ───────────────────────────────────

// findCompanyByDomain cerca un'azienda i cui utenti hanno email con quel dominio.
func (p *Processor) findCompanyByDomain(ctx context.Context, domain string) (*uuid.UUID, error) {
	if domain == "" {
		return nil, nil
	}

	u, err := p.db.User.Query().
		Where(
			user.EmailContains("@"+domain),
			user.HasCompany(),
		).
		WithCompany().
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	if u.Edges.Company != nil {
		id := u.Edges.Company.ID
		return &id, nil
	}
	return nil, nil
}

// ── Utilities ─────────────────────────────────────────────────────────────────

// normalizeSubject rimuove prefissi comuni come Re:, Fwd:, R:, I:
func normalizeSubject(s string) string {
	s = strings.TrimSpace(s)
	prefixes := []string{"re:", "fwd:", "fw:", "r:", "i:", "aw:"}
	for {
		lower := strings.ToLower(s)
		found := false
		for _, p := range prefixes {
			if strings.HasPrefix(lower, p) {
				s = strings.TrimSpace(s[len(p):])
				found = true
				break
			}
		}
		if !found {
			break
		}
	}
	return s
}

// buildBody costruisce il testo del ticket a partire dal messaggio email.
func buildBody(msg gmail.Message) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Da: %s\n", msg.From))
	sb.WriteString(fmt.Sprintf("Data: %s\n", msg.ReceivedAt.Format("02/01/2006 15:04")))
	sb.WriteString("\n")
	sb.WriteString(msg.Body)
	return truncate(sb.String(), 10000)
}

func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-3]) + "..."
}
