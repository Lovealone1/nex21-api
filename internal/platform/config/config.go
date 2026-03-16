package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port               string
	DBUrl              string
	SupabaseURL        string
	SupabaseAnonKey    string
	SupabaseServiceKey string
}

func Load() *Config {
	_ = godotenv.Load()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbUrl := os.Getenv("DATABASE_URL")
	supaURL := os.Getenv("SUPABASE_URL")
	supaAnonKey := os.Getenv("SUPABASE_ANON_KEY")
	supaServiceKey := os.Getenv("SUPABASE_SERVICE_ROLE_KEY")

	log.Println("Config loaded")

	return &Config{
		Port:               port,
		DBUrl:              dbUrl,
		SupabaseURL:        supaURL,
		SupabaseAnonKey:    supaAnonKey,
		SupabaseServiceKey: supaServiceKey,
	}
}
