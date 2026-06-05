# Configurazione Gmail OAuth2

Guida passo-passo per abilitare il mail worker.

---

## 1. Crea un progetto Google Cloud

1. Vai su https://console.cloud.google.com
2. Crea un nuovo progetto (es. `ticket-system`)

## 2. Abilita la Gmail API

1. Menu → **API e servizi** → **Libreria**
2. Cerca `Gmail API` → **Abilita**

## 3. Configura la schermata di consenso OAuth

1. Menu → **API e servizi** → **Schermata consenso OAuth**
2. Tipo utente: **Esterno** → Crea
3. Compila i campi obbligatori (nome app, email)
4. Nella sezione **Ambiti** aggiungi:
   - `https://www.googleapis.com/auth/gmail.readonly`
   - `https://www.googleapis.com/auth/gmail.modify`
   - `https://www.googleapis.com/auth/gmail.labels`
5. Nella sezione **Utenti di test** aggiungi la tua email Gmail
6. Salva

## 4. Crea le credenziali OAuth2

1. Menu → **API e servizi** → **Credenziali**
2. **Crea credenziali** → **ID client OAuth**
3. Tipo applicazione: **App desktop**
4. Nome: `ticket-mailworker`
5. **Crea** → Scarica il JSON
6. Rinomina il file in `credentials.json` e copialo nella cartella `ticket-service/`

```
ticket-service/
├── credentials.json    ← file scaricato da Google Cloud
├── gmail_token.json    ← creato automaticamente al primo avvio
└── ...
```

## 5. Prima autenticazione (una tantum)

```bash
cd ticket-service
make mailworker-auth
```

Il programma stampa un URL. Aprilo nel browser, accetta le autorizzazioni,
copia il codice e incollalo nel terminale. Il token viene salvato in
`gmail_token.json` e riutilizzato automaticamente.

> **Nota:** Il token viene rinnovato automaticamente finché il refresh token
> è valido. Se dopo un lungo periodo smette di funzionare, elimina
> `gmail_token.json` e ripeti l'autenticazione.

## 6. Avvio normale

```bash
# Avvia solo il mail worker
make mailworker

# Oppure insieme al server applicativo (due terminali)
make run          # terminale 1
make mailworker   # terminale 2
```

## 7. Variabili d'ambiente

```bash
GMAIL_CREDENTIALS_PATH=credentials.json  # percorso credentials.json
MAIL_POLL_INTERVAL=5m                    # frequenza polling (1m, 5m, 15m...)
MAIL_MAX_PER_CYCLE=20                    # max email per ciclo
MAIL_DEFAULT_SC_ID=<uuid>               # centro servizi di default
```

## 8. Comportamento

| Situazione | Azione |
|-----------|--------|
| Email da dominio **noto** (azienda registrata) | Ticket creato e intestato all'azienda |
| Email da dominio **sconosciuto** | Ticket creato senza cliente — l'operatore lo assegna |
| Email con stesso oggetto + stesso dominio di un ticket aperto | Aggiunta come commento al ticket esistente |
| Email processata con successo | Spostata nella label Gmail **"Processata"** |
| Errore durante il processing | Email lasciata in inbox — ritentata al prossimo ciclo |

## 9. Label Gmail

Il worker crea automaticamente la label **"Processata"** in Gmail al primo avvio.
Le email elaborate correttamente vengono spostate lì e rimosse dalla inbox.
