package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ticket-service/config"
	"ticket-service/internal/auth"
	"ticket-service/internal/handler"
	"ticket-service/internal/middleware"
	"ticket-service/internal/repository"
	"ticket-service/internal/service"
	"ticket-service/pkg/keycloak"

	"github.com/joho/godotenv"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"

	"ticket-service/ent"
)

func main() {
	// ── Config ────────────────────────────────────────────────────────────────
	_ = godotenv.Load() // carica .env se esiste, silenzioso in produzione
	cfg := config.Load()

	// ── Database ──────────────────────────────────────────────────────────────
	// Usa entsql per aprire la connessione (l'import 'entsql' va benissimo)
	drv, err := entsql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("errore connessione database: %v", err)
	}

	// Chiedi direttamente al driver di Ent l'istanza di *sql.DB per configurare il pool
	db := drv.DB()
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	entClient := ent.NewClient(ent.Driver(drv))
	defer entClient.Close()

	// Esegui auto-migration
	ctx := context.Background()
	if err := entClient.Schema.Create(ctx); err != nil {
		log.Fatalf("errore migration schema: %v", err)
	}
	log.Println("Schema database aggiornato.")

	// ── Keycloak client ───────────────────────────────────────────────────────
	kcClient := keycloak.NewClient(cfg.KeycloakURL, cfg.KeycloakRealm,
		cfg.KeycloakClientID, cfg.KeycloakClientSecret)

	// ── JWT verifier ──────────────────────────────────────────────────────────
	jwtVerifier, err := auth.NewJWTVerifier(cfg.KeycloakURL, cfg.KeycloakRealm)
	if err != nil {
		log.Fatalf("errore inizializzazione JWT verifier: %v", err)
	}

	// ── Repository layer ─────────────────────────────────────────────────────
	userRepo := repository.NewUserRepository(entClient)
	companyRepo := repository.NewCompanyRepository(entClient)
	scRepo := repository.NewServiceCenterRepository(entClient)
	ticketRepo := repository.NewTicketRepository(entClient)

	// ── Service layer ─────────────────────────────────────────────────────────
	userSvc := service.NewUserService(userRepo, companyRepo, scRepo, kcClient)
	companySvc := service.NewCompanyService(companyRepo)
	scSvc := service.NewServiceCenterService(scRepo)
	ticketSvc := service.NewTicketService(ticketRepo, userRepo)

	// ── Handler layer ─────────────────────────────────────────────────────────
	userH := handler.NewUserHandler(userSvc)
	companyH := handler.NewCompanyHandler(companySvc)
	scH := handler.NewServiceCenterHandler(scSvc)
	ticketH := handler.NewTicketHandler(ticketSvc)

	// ── Router ────────────────────────────────────────────────────────────────
	if cfg.GinMode != "" {
		gin.SetMode(cfg.GinMode)
	}
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	// Health check (pubblico)
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "ticket-service"})
	})

	authMW := middleware.JWT(jwtVerifier)
	resolveUser := middleware.ResolveUser(userRepo)

	v1 := r.Group("/api/v1", authMW, resolveUser)

	// ── Routes ────────────────────────────────────────────────────────────────
	// Ticket — customer & support
	v1.GET("/tickets", ticketH.List)
	v1.POST("/tickets", middleware.RequireRole("customer"), ticketH.Create)
	v1.GET("/tickets/:id", ticketH.Get)
	v1.POST("/tickets/:id/comments", ticketH.AddComment)
	v1.POST("/tickets/:id/attachments", ticketH.AddAttachment)

	// Ticket — support
	v1.GET("/tickets/unassigned", middleware.RequireRole("support_l1", "support_l2", "supervisor"), ticketH.ListUnassigned)
	v1.GET("/tickets/mine", middleware.RequireRole("support_l1", "support_l2"), ticketH.ListMine)
	v1.PUT("/tickets/:id/take", middleware.RequireRole("support_l1", "support_l2"), ticketH.Take)
	v1.PUT("/tickets/:id/priority", middleware.RequireRole("support_l1", "support_l2", "supervisor"), ticketH.SetPriority)
	v1.PUT("/tickets/:id/escalate", middleware.RequireRole("support_l1"), ticketH.Escalate)
	v1.PUT("/tickets/:id/state", middleware.RequireRole("support_l1", "support_l2", "supervisor"), ticketH.SetState)

	// Ticket — supervisor
	v1.GET("/tickets/center", middleware.RequireRole("supervisor"), ticketH.ListCenter)
	v1.PUT("/tickets/:id/assign", middleware.RequireRole("supervisor"), ticketH.Assign)

	// Companies
	v1.GET("/companies", middleware.RequireRole("supervisor"), companyH.List)
	v1.POST("/companies", middleware.RequireRole("supervisor"), companyH.Create)
	v1.GET("/companies/:id", middleware.RequireRole("supervisor"), companyH.Get)
	v1.PUT("/companies/:id", middleware.RequireRole("supervisor"), companyH.Update)

	// Service centers
	v1.GET("/service-centers", middleware.RequireRole("supervisor"), scH.List)
	v1.POST("/service-centers", middleware.RequireRole("supervisor"), scH.Create)
	v1.GET("/service-centers/:id", middleware.RequireRole("supervisor"), scH.Get)

	// Users — admin
	adm := v1.Group("/admin", middleware.RequireRole("supervisor"))
	adm.POST("/users/customer", userH.RegisterCustomer)
	adm.POST("/users/support", userH.RegisterSupport)
	adm.GET("/users", userH.ListCenterUsers)
	adm.PUT("/users/:id/active", userH.SetActive)

	// ── HTTP server con graceful shutdown ─────────────────────────────────────
	srv := &http.Server{
		Addr:         ":" + cfg.ServerPort,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Ticket service in ascolto su :%s (mode: %s)", cfg.ServerPort, cfg.GinMode)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutdown in corso...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Shutdown forzato: %v", err)
	}
	log.Println("Server fermato.")
}
