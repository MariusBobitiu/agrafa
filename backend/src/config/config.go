package config

import (
	"fmt"
	"net/url"
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
	AppAllowedOrigins       []string
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

	appAllowedOrigins, err := envOptionalString(SettingKeyAppAllowedOrigins)
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

	normalizedAppBaseURL := trimTrailingSlash(appBaseURL)
	allowedOrigins, err := normalizeAllowedOrigins(normalizedAppBaseURL, appAllowedOrigins)
	if err != nil {
		return Config{}, err
	}

	sessionCookieSecure, err := shouldUseSecureSessionCookies(normalizedAppBaseURL)
	if err != nil {
		return Config{}, err
	}

	return Config{
		PostgresURI:             postgresURI,
		Port:                    port,
		Environment:             environment,
		AppBaseURL:              normalizedAppBaseURL,
		AppAllowedOrigins:       allowedOrigins,
		AppSecret:               appSecret,
		NodeHeartbeatTTL:        time.Duration(heartbeatTTLSeconds) * time.Second,
		NodeExpiryCheckInterval: time.Duration(expiryIntervalSeconds) * time.Second,
		ManagedCheckInterval:    time.Duration(managedCheckIntervalSeconds) * time.Second,
		ManagedCheckTimeout:     time.Duration(managedCheckTimeoutSeconds) * time.Second,
		SessionTTL:              time.Duration(sessionTTLDays) * 24 * time.Hour,
		SessionRememberTTL:      time.Duration(sessionRememberTTLDays) * 24 * time.Hour,
		SessionCookieSecure:     sessionCookieSecure,
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

func envOptionalString(key SettingKey) (string, error) {
	resolved, err := ResolveEnvValueFromOS(key)
	if err != nil {
		return "", err
	}
	if !resolved.HasValue {
		return "", nil
	}

	return resolved.Value, nil
}

func trimTrailingSlash(value string) string {
	return strings.TrimRight(value, "/")
}

func normalizeAllowedOrigins(appBaseURL string, rawAllowedOrigins string) ([]string, error) {
	values := []string{appBaseURL}
	if rawAllowedOrigins != "" {
		values = append(values, strings.Split(rawAllowedOrigins, ",")...)
	}

	normalized := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		origin, err := normalizeOrigin(value)
		if err != nil {
			return nil, err
		}
		if origin == "" {
			continue
		}
		if _, ok := seen[origin]; ok {
			continue
		}
		seen[origin] = struct{}{}
		normalized = append(normalized, origin)
	}

	return normalized, nil
}

func normalizeOrigin(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}

	parsed, err := url.Parse(value)
	if err != nil {
		return "", fmt.Errorf("invalid origin %q: %w", value, err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("invalid origin %q", value)
	}

	return parsed.Scheme + "://" + parsed.Host, nil
}

func shouldUseSecureSessionCookies(appBaseURL string) (bool, error) {
	parsed, err := url.Parse(appBaseURL)
	if err != nil {
		return false, fmt.Errorf("invalid APP_BASE_URL %q: %w", appBaseURL, err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return false, fmt.Errorf("invalid APP_BASE_URL %q", appBaseURL)
	}

	return strings.EqualFold(parsed.Scheme, "https"), nil
}
