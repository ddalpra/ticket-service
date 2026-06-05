# Ticket Service

Servizio REST in Go per la gestione di ticket di supporto.

**Stack:** Go 1.22 · Gin · Ent ORM · PostgreSQL · Keycloak

---

## Struttura del progetto

```
ticket-service/
├── cmd/server/main.go          # entrypoint: wiring di tutti i layer
├── config/config.go            # lettura variabili d'ambiente
│
├── ent/                        # ORM generato
│   ├── generate.go             # go:generate
│   └── schema/                 # definizione entità
│       ├── company.go
│       ├── service_center.go
│       ├── user.go
│       ├── ticket.go
│       ├── comment.go
│       └── attachment.go
│
├── internal/
│   ├── auth/
│   │   └── jwt.go              # verifica JWT con JWKS Keycloak
│   ├── middleware/
│   │   └── middleware.go       # JWT · ResolveUser · RequireRole
│   ├── repository/             # query DB (Ent)
│   │   ├── user.go
│   │   ├── ticket.go
│   │   └── company_sc.go
│   ├── service/                # business logic
│   │   ├── ticket.go
│   │   ├── user.go
│   │   └── company_sc.go
│   └── handler/                # Gin handlers (HTTP layer)
│       ├── helpers.go
│       ├── ticket.go
│       ├── user.go
│       ├── company.go
│       └── service_center.go
│
└── pkg/
    ├── keycloak/client.go      # Admin API Keycloak
    └── apperrors/errors.go     # errori applicativi con codice HTTP
```

---

## Setup rapido

```bash
# 1. Copia e personalizza le variabili d'ambiente
cp .env.example .env

# 2. Avvia l'infrastruttura (Podman, dalla cartella ticket-env)
cd ../ticket-env && make infra-up && make wait-ready && cd ../ticket-service

# 3. Genera il codice Ent
make ent-gen

# 4. Scarica le dipendenze
go mod tidy

# 5. Avvia il server
make run

# Oppure con hot-reload:
make tools   # installa air
make run-air
```

---

## API Reference

### Autenticazione

Tutte le chiamate richiedono `Authorization: Bearer <JWT>`.

Ottenere un token:
```bash
curl -X POST http://localhost:8080/realms/ticket/protocol/openid-connect/token \
  -d "grant_type=password&client_id=ticket-service&client_secret=ticket-service-secret" \
  -d "username=daniele1&password=Password1!"
```

---

### Ticket

| Metodo | Path | Ruoli | Descrizione |
|--------|------|-------|-------------|
| `GET`  | `/api/v1/tickets` | tutti | Lista ticket (filtrata per ruolo) |
| `POST` | `/api/v1/tickets` | customer | Crea ticket |
| `GET`  | `/api/v1/tickets/:id` | tutti | Dettaglio ticket |
| `GET`  | `/api/v1/tickets/mine` | L1, L2 | Ticket assegnati a me |
| `GET`  | `/api/v1/tickets/unassigned` | L1, L2, supervisor | Non assegnati del mio centro |
| `GET`  | `/api/v1/tickets/center` | supervisor | Tutti i ticket del centro |
| `PUT`  | `/api/v1/tickets/:id/take` | L1, L2 | Prendi in carico |
| `PUT`  | `/api/v1/tickets/:id/priority` | L1, L2, supervisor | Cambia priorità |
| `PUT`  | `/api/v1/tickets/:id/state` | L1, L2, supervisor | Cambia stato |
| `PUT`  | `/api/v1/tickets/:id/escalate` | L1 | Scala a L2 |
| `PUT`  | `/api/v1/tickets/:id/assign` | supervisor | Riassegna a utente |
| `POST` | `/api/v1/tickets/:id/comments` | tutti | Aggiungi commento |
| `POST` | `/api/v1/tickets/:id/attachments` | tutti | Carica allegato |

### Companies

| Metodo | Path | Ruoli |
|--------|------|-------|
| `GET`  | `/api/v1/companies` | supervisor |
| `POST` | `/api/v1/companies` | supervisor |
| `GET`  | `/api/v1/companies/:id` | supervisor |
| `PUT`  | `/api/v1/companies/:id` | supervisor |

### Service Centers

| Metodo | Path | Ruoli |
|--------|------|-------|
| `GET`  | `/api/v1/service-centers` | supervisor |
| `POST` | `/api/v1/service-centers` | supervisor |
| `GET`  | `/api/v1/service-centers/:id` | supervisor |

### Utenti (Admin)

| Metodo | Path | Descrizione |
|--------|------|-------------|
| `POST` | `/api/v1/admin/users/customer` | Registra customer |
| `POST` | `/api/v1/admin/users/support` | Registra supporto/supervisor |
| `GET`  | `/api/v1/admin/users` | Utenti del centro |
| `PUT`  | `/api/v1/admin/users/:id/active` | Abilita/disabilita utente |

---

## Esempi di payload

**Crea ticket:**
```json
POST /api/v1/tickets
{
  "title": "Stampante non funziona",
  "question": "La stampante dell'ufficio 3 non stampa da stamattina.",
  "service_center_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

**Registra customer:**
```json
POST /api/v1/admin/users/customer
{
  "username": "marco1",
  "email": "marco@acme.com",
  "first_name": "Marco",
  "last_name": "Verdi",
  "password": "Password1!",
  "company_id": "550e8400-e29b-41d4-a716-446655440001"
}
```

**Escalation a L2:**
```json
PUT /api/v1/tickets/:id/escalate
{
  "user_id": "550e8400-e29b-41d4-a716-446655440002"
}
```

---

## State machine ticket

```
APERTA ──► PRESA_IN_CARICO ──► IN_ATTESA_CLIENTE
                │                      │
                ▼                      ▼
   IN_ATTESA_CENTRO_SERVIZI ◄──── (cliente risponde)
                │
                ▼
             CHIUSA
```

---

## Test

```bash
make test          # tutti i test con output verbose
make test-short    # esclude test di integrazione
make coverage      # genera coverage.html
make coverage-pct  # mostra percentuale totale
```

---

## Ruoli e visibilità

| Ruolo | Vede | Può fare |
|-------|------|----------|
| `customer` | Ticket della sua azienda | Crea ticket, commenta, allega |
| `support_l1` | Ticket assegnati a lui + non assegnati del centro | Prende in carico, risponde, scala a L2 |
| `support_l2` | Come L1 | Come L1, riceve escalation |
| `supervisor` | Tutti i ticket del centro | Tutto + riassegna, registra utenti |
