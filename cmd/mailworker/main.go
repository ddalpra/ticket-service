// mailworker è un processo autonomo che esegue il polling Gmail
// e crea/aggiorna ticket nel sistema.
//
// Uso:
//
//	go run ./cmd/mailworker
//
// Variabili d'ambiente richieste (stesse del server + quelle specifiche):
//
//	DATABASE_URL                  — connessione PostgreSQL
//	KEYCLOAK_URL / REALM / ecc.  — non necessari per il worker
//	GMAIL_CREDENTIALS_PATH        — percorso del file credentials.json di Google
//	MAIL_POLL_INTERVAL            — es. "5m", "1m", "30s" (default: 5m)
//	MAIL_MAX_PER_CYCLE            — max email per ciclo (default: 20)
//	MAIL_DEFAULT_SC_ID            — UUID del centro servizi di default
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"ticket-service/ent"
	"ticket-service/internal/mailworker"
	"ticket-service/pkg/gmail"
)

func main() {
	// Carica .env se presente (sviluppo locale)
	_ = godotenv.Load()

	ctx, cancel := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// ── Config ────────────────────────────────────────────────────────────────
	dbURL := requireEnv("DATABASE_URL")
	credentialsPath := getEnv("GMAIL_CREDENTIALS_PATH", "credentials.json")
	pollInterval := parseDuration(getEnv("MAIL_POLL_INTERVAL", "5m"))
	maxPerCycle := parseInt64(getEnv("MAIL_MAX_PER_CYCLE", "20"))
	defaultSCID := getEnv("MAIL_DEFAULT_SC_ID", "")

	log.Printf("[mailworker] configurazione:")
	log.Printf("  credentials: %s", credentialsPath)
	log.Printf("  poll interval: %s", pollInterval)
	log.Printf("  max per ciclo: %d", maxPerCycle)
	if defaultSCID != "" {
		log.Printf("  centro servizi default: %s", defaultSCID)
	} else {
		log.Printf("  centro servizi default: (primo attivo)")
	}

	// ── Database ──────────────────────────────────────────────────────────────
	drv, err := entsql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("[mailworker] connessione DB: %v", err)
	}
	db := ent.NewClient(ent.Driver(drv))
	defer db.Close()

	// Verifica connessione
	if err := db.Schema.Create(ctx); err != nil {
		log.Fatalf("[mailworker] schema DB: %v", err)
	}
	log.Println("[mailworker] connessione DB OK")

	// ── Gmail client ──────────────────────────────────────────────────────────
	gmailClient, err := gmail.NewClient(ctx, credentialsPath)
	if err != nil {
		log.Fatalf("[mailworker] inizializzazione Gmail: %v", err)
	}
	log.Println("[mailworker] connessione Gmail OK")

	// ── Worker ────────────────────────────────────────────────────────────────
	cfg := mailworker.Config{
		PollInterval:           pollInterval,
		MaxPerCycle:            maxPerCycle,
		DefaultServiceCenterID: defaultSCID,
	}

	worker := mailworker.New(cfg, gmailClient, db)
	worker.Start(ctx) // bloccante fino a SIGINT/SIGTERM
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("variabile d'ambiente obbligatoria mancante: %s", key)
	}
	return v
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		log.Printf("[mailworker] MAIL_POLL_INTERVAL non valido ('%s'), uso 5m", s)
		return 5 * time.Minute
	}
	return d
}

func parseInt64(s string) int64 {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil || n <= 0 {
		return 20
	}
	return n
}
