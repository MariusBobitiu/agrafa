package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/MariusBobitiu/agrafa-agent/src/types"
)

const (
	defaultSource               = "agent"
	defaultHeartbeatSeconds     = 15
	defaultMetricsSeconds       = 15
	defaultHealthSeconds        = 30
	defaultConfigRefreshSeconds = 30
	defaultHealthHTTPTimeout    = 10
	defaultAPIRetryCount        = 1
	maxAPIRetryCount            = 2
	defaultDiskPath             = "/"
	healthChecksFileEnvKey      = "AGRAFA_HEALTH_CHECKS_FILE"
	healthChecksJSONEnvKey      = "AGRAFA_HEALTH_CHECKS_JSON"
	apiBaseURLEnvKey            = "AGRAFA_API_BASE_URL"
	apiTimeoutEnvKey            = "AGRAFA_API_TIMEOUT_SECONDS"
	apiRetryCountEnvKey         = "AGRAFA_API_RETRY_COUNT"
	agentTokenEnvKey            = "AGRAFA_AGENT_TOKEN"
	nodeIDEnvKey                = "AGRAFA_NODE_ID"
	sourceEnvKey                = "AGRAFA_SOURCE"
	heartbeatIntervalEnvKey     = "AGRAFA_HEARTBEAT_INTERVAL_SECONDS"
	metricsIntervalEnvKey       = "AGRAFA_METRICS_INTERVAL_SECONDS"
	healthIntervalEnvKey        = "AGRAFA_HEALTH_INTERVAL_SECONDS"
	configRefreshEnvKey         = "AGRAFA_CONFIG_REFRESH_SECONDS"
	httpTimeoutEnvKey           = "AGRAFA_HTTP_TIMEOUT_SECONDS"
	diskPathEnvKey              = "AGRAFA_DISK_PATH"
)

func Load() (types.Config, error) {
	env := loadEnvironment()

	apiBaseURL := strings.TrimRight(env[apiBaseURLEnvKey], "/")
	if apiBaseURL == "" {
		return types.Config{}, fmt.Errorf("%s is required", apiBaseURLEnvKey)
	}

	if _, err := url.ParseRequestURI(apiBaseURL); err != nil {
		return types.Config{}, fmt.Errorf("invalid %s: %w", apiBaseURLEnvKey, err)
	}

	agentToken := strings.TrimSpace(env[agentTokenEnvKey])
	if agentToken == "" {
		return types.Config{}, fmt.Errorf("%s is required", agentTokenEnvKey)
	}

	nodeID, err := parseRequiredInt64(env, nodeIDEnvKey)
	if err != nil {
		return types.Config{}, err
	}

	heartbeatSeconds, err := parsePositiveInt(env, heartbeatIntervalEnvKey, defaultHeartbeatSeconds)
	if err != nil {
		return types.Config{}, err
	}

	metricsSeconds, err := parsePositiveInt(env, metricsIntervalEnvKey, defaultMetricsSeconds)
	if err != nil {
		return types.Config{}, err
	}

	healthSeconds, err := parsePositiveInt(env, healthIntervalEnvKey, defaultHealthSeconds)
	if err != nil {
		return types.Config{}, err
	}

	configRefreshSeconds, err := parsePositiveInt(env, configRefreshEnvKey, defaultConfigRefreshSeconds)
	if err != nil {
		return types.Config{}, err
	}

	healthHTTPTimeoutSeconds, err := parsePositiveInt(env, httpTimeoutEnvKey, defaultHealthHTTPTimeout)
	if err != nil {
		return types.Config{}, err
	}

	apiTimeoutSeconds, err := parseAPITimeoutSeconds(env, healthHTTPTimeoutSeconds)
	if err != nil {
		return types.Config{}, err
	}

	apiRetryCount, err := parseAPIRetryCount(env)
	if err != nil {
		return types.Config{}, err
	}

	healthChecks, err := loadHealthChecks(env)
	if err != nil {
		return types.Config{}, err
	}

	config := types.Config{
		APIBaseURL:            apiBaseURL,
		AgentToken:            agentToken,
		NodeID:                nodeID,
		Source:                firstNonEmpty(env[sourceEnvKey], defaultSource),
		APITimeout:            time.Duration(apiTimeoutSeconds) * time.Second,
		APIRetryCount:         apiRetryCount,
		HeartbeatInterval:     time.Duration(heartbeatSeconds) * time.Second,
		MetricsInterval:       time.Duration(metricsSeconds) * time.Second,
		HealthInterval:        time.Duration(healthSeconds) * time.Second,
		ConfigRefreshInterval: time.Duration(configRefreshSeconds) * time.Second,
		HTTPTimeout:           time.Duration(healthHTTPTimeoutSeconds) * time.Second,
		DiskPath:              firstNonEmpty(env[diskPathEnvKey], defaultDiskPath),
		HealthChecks:          healthChecks,
	}

	if err := validateHealthChecks(config.HealthChecks); err != nil {
		return types.Config{}, err
	}

	return config, nil
}

func loadEnvironment() map[string]string {
	env := map[string]string{}

	mergeEnvironmentFile(env, ".env")
	mergeEnvironmentFile(env, ".env.local")
	mergeRuntimeEnvironment(env)

	return env
}

func mergeEnvironmentFile(target map[string]string, path string) {
	content, err := os.ReadFile(path)
	if err != nil {
		return
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		key, value, found := strings.Cut(trimmed, "=")
		if !found {
			continue
		}

		target[strings.TrimSpace(key)] = trimQuotes(strings.TrimSpace(value))
	}
}

func mergeRuntimeEnvironment(target map[string]string) {
	for _, entry := range os.Environ() {
		key, value, found := strings.Cut(entry, "=")
		if !found {
			continue
		}
		target[key] = value
	}
}

func trimQuotes(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
			return value[1 : len(value)-1]
		}
	}
	return value
}

func loadHealthChecks(env map[string]string) ([]types.HealthCheck, error) {
	if path := strings.TrimSpace(env[healthChecksFileEnvKey]); path != "" {
		content, err := os.ReadFile(filepath.Clean(path))
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", healthChecksFileEnvKey, err)
		}
		return decodeHealthChecks(content)
	}

	if rawJSON := strings.TrimSpace(env[healthChecksJSONEnvKey]); rawJSON != "" {
		return decodeHealthChecks([]byte(rawJSON))
	}

	return nil, nil
}

func decodeHealthChecks(content []byte) ([]types.HealthCheck, error) {
	var checksFile types.HealthChecksFile
	if err := json.Unmarshal(content, &checksFile); err == nil {
		return checksFile.HealthChecks, nil
	}

	var checks []types.HealthCheck
	if err := json.Unmarshal(content, &checks); err != nil {
		return nil, fmt.Errorf("decode health checks: %w", err)
	}

	return checks, nil
}

func validateHealthChecks(checks []types.HealthCheck) error {
	for _, check := range checks {
		if check.ServiceID <= 0 {
			return errors.New("health check service_id must be greater than 0")
		}
		if check.Name == "" {
			return errors.New("health check name is required")
		}
		if check.Type != "http" {
			return fmt.Errorf("unsupported health check type %q", check.Type)
		}
		if _, err := url.ParseRequestURI(check.Target); err != nil {
			return fmt.Errorf("invalid health check target for %q: %w", check.Name, err)
		}
		if check.TimeoutSeconds < 0 {
			return fmt.Errorf("health check timeout_seconds must be 0 or greater for %q", check.Name)
		}
		if check.IntervalSeconds < 0 {
			return fmt.Errorf("health check interval_seconds must be 0 or greater for %q", check.Name)
		}
	}

	return nil
}

func parseRequiredInt64(env map[string]string, key string) (int64, error) {
	rawValue := strings.TrimSpace(env[key])
	if rawValue == "" {
		return 0, fmt.Errorf("%s is required", key)
	}

	value, err := strconv.ParseInt(rawValue, 10, 64)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", key)
	}

	return value, nil
}

func parsePositiveInt(env map[string]string, key string, defaultValue int) (int, error) {
	rawValue := strings.TrimSpace(env[key])
	if rawValue == "" {
		return defaultValue, nil
	}

	value, err := strconv.Atoi(rawValue)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", key)
	}

	return value, nil
}

func parseAPITimeoutSeconds(env map[string]string, fallback int) (int, error) {
	rawValue := strings.TrimSpace(env[apiTimeoutEnvKey])
	if rawValue == "" {
		return fallback, nil
	}

	value, err := strconv.Atoi(rawValue)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", apiTimeoutEnvKey)
	}

	return value, nil
}

func parseAPIRetryCount(env map[string]string) (int, error) {
	rawValue := strings.TrimSpace(env[apiRetryCountEnvKey])
	if rawValue == "" {
		return defaultAPIRetryCount, nil
	}

	value, err := strconv.Atoi(rawValue)
	if err != nil || value < 0 {
		return 0, fmt.Errorf("%s must be 0, 1, or 2", apiRetryCountEnvKey)
	}

	if value > maxAPIRetryCount {
		return 0, fmt.Errorf("%s must be 0, 1, or 2", apiRetryCountEnvKey)
	}

	return value, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
