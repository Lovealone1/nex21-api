package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config represents the application's global configuration.
type Config struct {
	App           AppConfig
	HTTP          HTTPConfig
	Database      DatabaseConfig
	Redis         RedisConfig
	Auth          AuthConfig
	Observability ObservabilityConfig
}

// AppConfig contains application-level settings.
type AppConfig struct {
	Name     string
	Env      string // dev, stage, prod
	LogLevel string
}

// HTTPConfig contains web server settings.
type HTTPConfig struct {
	Host           string
	Port           string
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	IdleTimeout    time.Duration
	AllowedOrigins []string
}

// DatabaseConfig contains DB connection settings.
type DatabaseConfig struct {
	URL             string // Senstive: Redacted in String() (DATABASE_URL)
	DirectURL       string // Sensitive: Redacted in String() (DIRECT_URL)
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	QueryTimeout    time.Duration
}

// RedisConfig contains cache connection settings.
type RedisConfig struct {
	Addr         string
	Password     string // Sensitive: Redacted in String()
	DB           int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// AuthConfig contains authentication provider settings (Supabase).
type AuthConfig struct {
	SupabaseURL            string
	SupabaseAnonKey        string // Sensitive: Redacted in String()
	SupabaseServiceRoleKey string // Sensitive: Redacted in String()
	SupabaseJWTSecret      string // Sensitive: Redacted in String()
}

// ObservabilityConfig contains OpenTelemetry/tracing settings.
type ObservabilityConfig struct {
	Enabled        bool
	Endpoint       string
	ServiceName    string
	ServiceVersion string
}

// Load reads configuration from env vars (and .env) and validates it.
// It will panic if critical fields are missing.
func Load() *Config {
	_ = godotenv.Load()

	cfg := &Config{
		App: AppConfig{
			Name:     getEnv("APP_NAME", "nex21-api"),
			Env:      getEnv("APP_ENV", "dev"),
			LogLevel: getEnv("LOG_LEVEL", "debug"),
		},
		HTTP: HTTPConfig{
			Host:           getEnv("HTTP_HOST", "0.0.0.0"),
			Port:           getEnv("HTTP_PORT", "8080"),
			ReadTimeout:    getEnvDuration("HTTP_READ_TIMEOUT", 10*time.Second),
			WriteTimeout:   getEnvDuration("HTTP_WRITE_TIMEOUT", 10*time.Second),
			IdleTimeout:    getEnvDuration("HTTP_IDLE_TIMEOUT", 60*time.Second),
			AllowedOrigins: strings.Split(getEnv("CORS_ALLOWED_ORIGINS", "*"), ","),
		},
		Database: DatabaseConfig{
			URL:             getEnvOrPanic("DATABASE_URL"),
			DirectURL:       getEnvOrPanic("DIRECT_URL"),
			MaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 25),
			ConnMaxLifetime: getEnvDuration("DB_CONN_MAX_LIFETIME", 15*time.Minute),
			QueryTimeout:    getEnvDuration("DB_QUERY_TIMEOUT", 5*time.Second),
		},
		Redis: RedisConfig{
			Addr:         getEnvOrPanic("REDIS_ADDR"),
			Password:     getEnv("REDIS_PASSWORD", ""),
			DB:           getEnvInt("REDIS_DB", 0),
			DialTimeout:  getEnvDuration("REDIS_DIAL_TIMEOUT", 5*time.Second),
			ReadTimeout:  getEnvDuration("REDIS_READ_TIMEOUT", 3*time.Second),
			WriteTimeout: getEnvDuration("REDIS_WRITE_TIMEOUT", 3*time.Second),
		},
		Auth: AuthConfig{
			SupabaseURL:            getEnvOrPanic("SUPABASE_URL"),
			SupabaseAnonKey:        getEnvOrPanic("SUPABASE_ANON_KEY"),
			SupabaseServiceRoleKey: getEnvOrPanic("SUPABASE_SERVICE_ROLE_KEY"),
			SupabaseJWTSecret:      getEnv("SUPABASE_JWT_SECRET", ""), // Assuming it might be absent in some setups since we use API keys
		},
		Observability: ObservabilityConfig{
			Enabled:        getEnvBool("OTEL_ENABLED", false),
			Endpoint:       getEnv("OTEL_ENDPOINT", ""),
			ServiceName:    getEnv("OTEL_SERVICE_NAME", "nex21-api"),
			ServiceVersion: getEnv("OTEL_SERVICE_VERSION", "1.0.0"),
		},
	}

	log.Println("Config loaded successfully")
	return cfg
}

// String implements the fmt.Stringer interface to securely mask sensitive data
// when printing the configuration to logs.
func (c *Config) String() string {
	return fmt.Sprintf(
		"Config(App: %+v, HTTP: %+v, DB: [URL: REDACTED, DirectURL: REDACTED, MaxOpen: %d, MaxIdle: %d, MaxLifetime: %v], "+
			"Redis: [Addr: %s, Password: REDACTED, DB: %d], "+
			"Auth: [SupabaseURL: %s, SupabaseAnonKey: REDACTED, SupabaseServiceRoleKey: REDACTED, JWTSecret: REDACTED], "+
			"Observability: %+v)",
		c.App, c.HTTP, c.Database.MaxOpenConns, c.Database.MaxIdleConns, c.Database.ConnMaxLifetime,
		c.Redis.Addr, c.Redis.DB,
		c.Auth.SupabaseURL,
		c.Observability,
	)
}

// --- Helper Functions ---

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getEnvOrPanic(key string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}
	panic(fmt.Sprintf("CRITICAL CONFIG ERROR: Required environment variable %s is not set", key))
}

func getEnvInt(key string, fallback int) int {
	str := getEnv(key, "")
	if str == "" {
		return fallback
	}
	val, err := strconv.Atoi(str)
	if err != nil {
		log.Printf("Warning: invalid integer for %s, using fallback %d. Error: %v", key, fallback, err)
		return fallback
	}
	return val
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	str := getEnv(key, "")
	if str == "" {
		return fallback
	}
	val, err := time.ParseDuration(str)
	if err != nil {
		log.Printf("Warning: invalid duration for %s, using fallback %v. Error: %v", key, fallback, err)
		return fallback
	}
	return val
}

func getEnvBool(key string, fallback bool) bool {
	str := getEnv(key, "")
	if str == "" {
		return fallback
	}
	val, err := strconv.ParseBool(str)
	if err != nil {
		log.Printf("Warning: invalid boolean for %s, using fallback %v. Error: %v", key, fallback, err)
		return fallback
	}
	return val
}
