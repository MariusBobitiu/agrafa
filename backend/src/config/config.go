package config

import (
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	PostgresURI             string
	Port                    string
	Environment             string
	AppBaseURL              string
	AppSecret               string
	NodeHeartbeatTTL        time.Duration
	NodeExpiryCheckInterval time.Duration
	ManagedCheckInterval    time.Duration
	ManagedCheckTimeout     time.Duration
	SessionTTL              time.Duration
	SessionRememberTTL      time.Duration
	SessionCookieSecure     bool
}

func Load() (Config, error) {
	_ = godotenv.Load(".env.local", ".env")

	postgresURI, err := envString(SettingKeyPostgresURI)
	if err != nil {
		return Config{}, err
	}

	port, err := envString(SettingKeyPort)
	if err != nil {
		return Config{}, err
	}

	environment, err := envString(SettingKeyAppEnv)
	if err != nil {
		return Config{}, err
	}

	appBaseURL, err := envString(SettingKeyAppBaseURL)
	if err != nil {
		return Config{}, err
	}

	appSecret, err := envString(SettingKeyAppSecret)
	if err != nil {
		return Config{}, err
	}

	heartbeatTTLSeconds, err := envInt(SettingKeyNodeHeartbeatTTLSeconds)
	if err != nil {
		return Config{}, err
	}

	expiryIntervalSeconds, err := envInt(SettingKeyNodeExpiryCheckIntervalSeconds)
	if err != nil {
		return Config{}, err
	}

	managedCheckIntervalSeconds, err := envInt(SettingKeyManagedCheckIntervalSeconds)
	if err != nil {
		return Config{}, err
	}

	managedCheckTimeoutSeconds, err := envInt(SettingKeyManagedCheckTimeoutSeconds)
	if err != nil {
		return Config{}, err
	}

	sessionTTLDays, err := envInt(SettingKeySessionTTLDays)
	if err != nil {
		return Config{}, err
	}

	sessionRememberTTLDays, err := envInt(SettingKeySessionRememberTTLDays)
	if err != nil {
		return Config{}, err
	}

	return Config{
		PostgresURI:             postgresURI,
		Port:                    port,
		Environment:             environment,
		AppBaseURL:              trimTrailingSlash(appBaseURL),
		AppSecret:               appSecret,
		NodeHeartbeatTTL:        time.Duration(heartbeatTTLSeconds) * time.Second,
		NodeExpiryCheckInterval: time.Duration(expiryIntervalSeconds) * time.Second,
		ManagedCheckInterval:    time.Duration(managedCheckIntervalSeconds) * time.Second,
		ManagedCheckTimeout:     time.Duration(managedCheckTimeoutSeconds) * time.Second,
		SessionTTL:              time.Duration(sessionTTLDays) * 24 * time.Hour,
		SessionRememberTTL:      time.Duration(sessionRememberTTLDays) * 24 * time.Hour,
		SessionCookieSecure:     strings.EqualFold(environment, "production"),
	}, nil
}

func envString(key SettingKey) (string, error) {
	resolved, err := ResolveEnvValueFromOS(key)
	if err != nil {
		return "", err
	}

	return resolved.Value, nil
}

func envInt(key SettingKey) (int, error) {
	resolved, err := ResolveEnvValueFromOS(key)
	if err != nil {
		return 0, err
	}

	return strconv.Atoi(resolved.Value)
}

func trimTrailingSlash(value string) string {
	return strings.TrimRight(value, "/")
}
