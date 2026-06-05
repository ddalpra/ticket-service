// Package mailworker implementa la goroutine di polling Gmail.
// Ad ogni ciclo:
//  1. Scarica le email non lette
//  2. Per ogni email cerca se esiste già un ticket con stesso oggetto + dominio mittente
//     → sì: aggiunge l'email come commento
//     → no: crea un nuovo ticket (intestato al cliente se il dominio è noto, altrimenti senza cliente)
//  3. Sposta l'email nella label "Processata"
package mailworker

import (
	"context"
	"log"
	"time"

	"ticket-service/ent"
	"ticket-service/internal/mailworker/processor"
	"ticket-service/pkg/gmail"
)

// Config contiene la configurazione del worker.
type Config struct {
	// Intervallo tra un ciclo di polling e il successivo
	PollInterval time.Duration
	// Numero massimo di email elaborate per ciclo
	MaxPerCycle int64
	// Centro servizi di default a cui assegnare i ticket da email
	DefaultServiceCenterID string
}

// Worker è la goroutine di polling Gmail.
type Worker struct {
	cfg       Config
	gmail     *gmail.Client
	processor *processor.Processor
}

// New crea un nuovo Worker.
func New(cfg Config, gmailClient *gmail.Client, db *ent.Client) *Worker {
	return &Worker{
		cfg:       cfg,
		gmail:     gmailClient,
		processor: processor.New(db, cfg.DefaultServiceCenterID),
	}
}

// Start avvia il worker in background. Termina quando ctx viene cancellato.
func (w *Worker) Start(ctx context.Context) {
	log.Printf("[mailworker] avviato — polling ogni %s, max %d email/ciclo",
		w.cfg.PollInterval, w.cfg.MaxPerCycle)

	ticker := time.NewTicker(w.cfg.PollInterval)
	defer ticker.Stop()

	// Primo ciclo immediato all'avvio
	w.runCycle(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Println("[mailworker] fermato.")
			return
		case <-ticker.C:
			w.runCycle(ctx)
		}
	}
}

// runCycle esegue un singolo ciclo di polling.
func (w *Worker) runCycle(ctx context.Context) {
	log.Println("[mailworker] inizio ciclo polling...")

	messages, err := w.gmail.FetchUnread(ctx, w.cfg.MaxPerCycle)
	if err != nil {
		log.Printf("[mailworker] errore fetch email: %v", err)
		return
	}

	if len(messages) == 0 {
		log.Println("[mailworker] nessuna email da processare.")
		return
	}

	log.Printf("[mailworker] trovate %d email da processare", len(messages))

	for _, msg := range messages {
		result, err := w.processor.Process(ctx, msg)
		if err != nil {
			log.Printf("[mailworker] errore processing email %s (%s): %v",
				msg.ID, msg.Subject, err)
			// Non sposta l'email: verrà ritentata al prossimo ciclo
			continue
		}

		log.Printf("[mailworker] email %s → %s (ticket: %s)",
			msg.ID, result.Action, result.TicketID)

		// Sposta nella label "Processata"
		if err := w.gmail.MarkProcessed(ctx, msg.ID); err != nil {
			log.Printf("[mailworker] avviso: impossibile spostare email %s: %v", msg.ID, err)
		}
	}

	log.Printf("[mailworker] ciclo completato — %d email elaborate", len(messages))
}
