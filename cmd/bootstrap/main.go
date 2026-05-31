// bootstrap è un comando one-shot che popola il DB con i dati iniziali:
// - Centro servizi di default
// - Aziende demo (acme, pippo)
// - Utenti Keycloak già presenti nel realm → specializzati nel DB locale
//
// Uso:
//   go run ./cmd/bootstrap
//
// Le variabili d'ambiente necessarie sono le stesse del server (.env).

package main

import (
	"context"
	"log"
	"os"

	"entgo.io/ent/dialect/sql"
	_ "github.com/lib/pq"
	"github.com/joho/godotenv"

	"ticket-service/ent"
	"ticket-service/ent/user"
	"ticket-service/pkg/keycloak"
)

func main() {
	// Carica .env se presente
	_ = godotenv.Load()

	dbURL := requireEnv("DATABASE_URL")
	kcURL := requireEnv("KEYCLOAK_URL")
	kcRealm := requireEnv("KEYCLOAK_REALM")
	kcClientID := requireEnv("KEYCLOAK_CLIENT_ID")
	kcClientSecret := requireEnv("KEYCLOAK_CLIENT_SECRET")

	ctx := context.Background()

	// ── DB ────────────────────────────────────────────────────────────────────
	drv, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("connessione DB: %v", err)
	}
	client := ent.NewClient(ent.Driver(drv))
	defer client.Close()

	if err := client.Schema.Create(ctx); err != nil {
		log.Fatalf("migration: %v", err)
	}
	log.Println("✓ Schema aggiornato")

	// ── Keycloak client ───────────────────────────────────────────────────────
	kc := keycloak.NewClient(kcURL, kcRealm, kcClientID, kcClientSecret)

	// ── 1. Centro servizi ─────────────────────────────────────────────────────
	sc, err := client.ServiceCenter.Create().
		SetCodice("CS-01").
		SetNome("Centro Servizi Principale").
		Save(ctx)
	if err != nil {
		log.Printf("centro servizi già presente o errore: %v", err)
		sc, err = client.ServiceCenter.Query().First(ctx)
		if err != nil {
			log.Fatalf("impossibile recuperare centro servizi: %v", err)
		}
	} else {
		log.Printf("✓ Centro servizi creato: %s (%s)", sc.Nome, sc.ID)
	}

	// ── 2. Aziende ────────────────────────────────────────────────────────────
	acme, err := client.Company.Create().
		SetCodice("ACME-01").
		SetRagioneSociale("Acme S.r.l.").
		SetPiva("12345678901").
		SetCf("12345678901").
		Save(ctx)
	if err != nil {
		log.Printf("azienda acme già presente: %v", err)
		acme, _ = client.Company.Query().Where(
			func(s *sql.Selector) { s.Where(sql.EQ("codice", "ACME-01")) },
		).First(ctx)
	} else {
		log.Printf("✓ Azienda creata: %s (%s)", acme.RagioneSociale, acme.ID)
	}

	pippo, err := client.Company.Create().
		SetCodice("PIPPO-01").
		SetRagioneSociale("Pippo S.p.A.").
		SetPiva("98765432109").
		SetCf("98765432109").
		Save(ctx)
	if err != nil {
		log.Printf("azienda pippo già presente: %v", err)
		pippo, _ = client.Company.Query().Where(
			func(s *sql.Selector) { s.Where(sql.EQ("codice", "PIPPO-01")) },
		).First(ctx)
	} else {
		log.Printf("✓ Azienda creata: %s (%s)", pippo.RagioneSociale, pippo.ID)
	}

	// ── 3. Recupera keycloak_id degli utenti già presenti nel realm ───────────
	kcUsers, err := kc.ListUsers(ctx)
	if err != nil {
		log.Fatalf("impossibile listare utenti Keycloak: %v", err)
	}
	kcMap := make(map[string]string) // username → keycloak_id
	for _, u := range kcUsers {
		kcMap[u.Username] = u.ID
		log.Printf("  KC user: %s → %s", u.Username, u.ID)
	}

	// ── 4. Specializza gli utenti nel DB locale ────────────────────────────────
	type seedUser struct {
		Username  string
		Email     string
		FirstName string
		LastName  string
		Role      user.Role
		CompanyID *ent.Company
		ScID      *ent.ServiceCenter
	}

	seeds := []seedUser{
		{
			Username: "supervisor1", Email: "supervisor1@example.com",
			FirstName: "Mario", LastName: "Rossi",
			Role: user.RoleSupervisor, ScID: sc,
		},
		{
			Username: "supportl1", Email: "support.l1@centro.com",
			FirstName: "Luigi", LastName: "Support",
			Role: user.RoleSupportL1, ScID: sc,
		},
		{
			Username: "supportl2", Email: "support.l2@centro.com",
			FirstName: "Sara", LastName: "Support",
			Role: user.RoleSupportL2, ScID: sc,
		},
		{
			Username: "daniele1", Email: "daniele1@acme.com",
			FirstName: "Daniele", LastName: "Acme",
			Role: user.RoleCustomer, CompanyID: acme,
		},
		{
			Username: "davide1", Email: "davide1@acme.com",
			FirstName: "Davide", LastName: "Acme",
			Role: user.RoleCustomer, CompanyID: acme,
		},
		{
			Username: "filippo1", Email: "filippo1@pippo.com",
			FirstName: "Filippo", LastName: "Pippo",
			Role: user.RoleCustomer, CompanyID: pippo,
		},
	}

	for _, s := range seeds {
		kcID, ok := kcMap[s.Username]
		if !ok {
			log.Printf("⚠ utente %s non trovato in Keycloak, salto", s.Username)
			continue
		}

		// Verifica se già presente nel DB
		exists, _ := client.User.Query().
			Where(user.KeycloakID(kcID)).
			Exist(ctx)
		if exists {
			log.Printf("  già presente: %s", s.Username)
			continue
		}

		q := client.User.Create().
			SetKeycloakID(kcID).
			SetUsername(s.Username).
			SetEmail(s.Email).
			SetFirstName(s.FirstName).
			SetLastName(s.LastName).
			SetRole(s.Role)

		if s.CompanyID != nil {
			q = q.SetCompany(s.CompanyID)
		}
		if s.ScID != nil {
			q = q.SetServiceCenter(s.ScID)
		}

		u, err := q.Save(ctx)
		if err != nil {
			log.Printf("✗ errore creazione %s: %v", s.Username, err)
			continue
		}
		log.Printf("✓ utente creato: %s (%s) → role: %s", u.Username, u.ID, u.Role)
	}

	log.Println("\nBootstrap completato.")
	log.Printf("  Centro servizi ID : %s\n", sc.ID)
	if acme != nil {
		log.Printf("  Acme ID           : %s\n", acme.ID)
	}
	if pippo != nil {
		log.Printf("  Pippo ID          : %s\n", pippo.ID)
	}
}

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("variabile d'ambiente obbligatoria mancante: %s", key)
	}
	return v
}
