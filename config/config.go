package config

import (
	"log"
	"os"
)

// Config contiene tutte le variabili d'ambiente dell'applicazione.
type Config struct {
	DatabaseURL          string
	KeycloakURL          string
	KeycloakRealm        string
	KeycloakClientID     string
	KeycloakClientSecret string
	ServerPort           string
	GinMode              string
}

// Load legge le variabili d'ambiente e restituisce una Config.
// Termina l'applicazione se mancano variabili obbligatorie.
func Load() *Config {
	cfg := &Config{
		DatabaseURL:          requireEnv("DATABASE_URL"),
		KeycloakURL:          requireEnv("KEYCLOAK_URL"),
		KeycloakRealm:        requireEnv("KEYCLOAK_REALM"),
		KeycloakClientID:     requireEnv("KEYCLOAK_CLIENT_ID"),
		KeycloakClientSecret: requireEnv("KEYCLOAK_CLIENT_SECRET"),
		ServerPort:           getEnv("SERVER_PORT", "3000"),
		GinMode:              getEnv("GIN_MODE", "debug"),
	}
	return cfg
}

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
