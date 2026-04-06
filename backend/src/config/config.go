package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port                    string
	PostgresURI             string
	NodeHeartbeatTTL        time.Duration
	NodeExpiryCheckInterval time.Duration
	ManagedCheckInterval    time.Duration
	ManagedCheckTimeout     time.Duration
	Environment             string
	SessionTTL              time.Duration
	SessionRememberTTL      time.Duration
	SessionCookieSecure     bool
	AppBaseURL              string
	ResendAPIKey            string
	ResendEmailDomain       string
	AlertsFromEmail         string
}

func Load() (Config, error) {
	_ = godotenv.Load(".env.local", ".env")

	postgresURI, err := requiredEnv("POSTGRES_URI")
	if err != nil {
		return Config{}, err
	}

	heartbeatTTL, err := envDurationSeconds("NODE_HEARTBEAT_TTL_SECONDS", 60)
	if err != nil {
		return Config{}, err
	}

	expiryInterval, err := envDurationSeconds("NODE_EXPIRY_CHECK_INTERVAL_SECONDS", 15)
	if err != nil {
		return Config{}, err
	}

	managedCheckInterval, err := envDurationSeconds("MANAGED_SERVICE_CHECK_INTERVAL_SECONDS", 15)
	if err != nil {
		return Config{}, err
	}

	managedCheckTimeout, err := envDurationSeconds("MANAGED_SERVICE_CHECK_TIMEOUT_SECONDS", 10)
	if err != nil {
		return Config{}, err
	}

	sessionTTL, err := envDurationDays("SESSION_TTL_DAYS", 7)
	if err != nil {
		return Config{}, err
	}

	sessionRememberTTL, err := envDurationDays("SESSION_REMEMBER_TTL_DAYS", 30)
	if err != nil {
		return Config{}, err
	}

	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = "8080"
	}

	environment := strings.TrimSpace(os.Getenv("APP_ENV"))
	if environment == "" {
		environment = "development"
	}

	return Config{
		Port:                    port,
		PostgresURI:             postgresURI,
		NodeHeartbeatTTL:        heartbeatTTL,
		NodeExpiryCheckInterval: expiryInterval,
		ManagedCheckInterval:    managedCheckInterval,
		ManagedCheckTimeout:     managedCheckTimeout,
		Environment:             environment,
		SessionTTL:              sessionTTL,
		SessionRememberTTL:      sessionRememberTTL,
		SessionCookieSecure:     strings.EqualFold(environment, "production"),
		AppBaseURL:              envStringDefault("APP_BASE_URL", "http://localhost:3000"),
		ResendAPIKey:            strings.TrimSpace(os.Getenv("RESEND_API_KEY")),
		ResendEmailDomain:       strings.TrimSpace(os.Getenv("RESEND_EMAIL_DOMAIN")),
		AlertsFromEmail:         strings.TrimSpace(os.Getenv("ALERTS_FROM_EMAIL")),
	}, nil
}

func requiredEnv(name string) (string, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return "", fmt.Errorf("%s is required", name)
	}

	return value, nil
}

func envDurationSeconds(name string, defaultSeconds int) (time.Duration, error) {
	rawValue := strings.TrimSpace(os.Getenv(name))
	if rawValue == "" {
		return time.Duration(defaultSeconds) * time.Second, nil
	}

	seconds, err := strconv.Atoi(rawValue)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer number of seconds: %w", name, err)
	}

	if seconds <= 0 {
		return 0, fmt.Errorf("%s must be greater than zero", name)
	}

	return time.Duration(seconds) * time.Second, nil
}

func envDurationDays(name string, defaultDays int) (time.Duration, error) {
	rawValue := strings.TrimSpace(os.Getenv(name))
	if rawValue == "" {
		return time.Duration(defaultDays) * 24 * time.Hour, nil
	}

	days, err := strconv.Atoi(rawValue)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer number of days: %w", name, err)
	}

	if days <= 0 {
		return 0, fmt.Errorf("%s must be greater than zero", name)
	}

	return time.Duration(days) * 24 * time.Hour, nil
}

func envStringDefault(name, defaultValue string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		value = defaultValue
	}

	return strings.TrimRight(value, "/")
}
