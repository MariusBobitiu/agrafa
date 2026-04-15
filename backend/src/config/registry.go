package config

import (
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
)

type SettingKey string

type SettingType string

const (
	SettingTypeString SettingType = "string"
	SettingTypeBool   SettingType = "bool"
	SettingTypeInt    SettingType = "int"
	SettingTypeEnum   SettingType = "enum"
)

const (
	SettingKeyPostgresURI                    SettingKey = "postgres.uri"
	SettingKeyAppEnv                         SettingKey = "app.env"
	SettingKeyPort                           SettingKey = "app.port"
	SettingKeyAppBaseURL                     SettingKey = "app.base_url"
	SettingKeyAppAllowedOrigins              SettingKey = "app.allowed_origins"
	SettingKeyAppSecret                      SettingKey = "app.secret"
	SettingKeyNodeHeartbeatTTLSeconds        SettingKey = "node.heartbeat_ttl_seconds"
	SettingKeyNodeExpiryCheckIntervalSeconds SettingKey = "node.expiry_check_interval_seconds"
	SettingKeyManagedCheckIntervalSeconds    SettingKey = "managed_service.check_interval_seconds"
	SettingKeyManagedCheckTimeoutSeconds     SettingKey = "managed_service.check_timeout_seconds"
	SettingKeySessionTTLDays                 SettingKey = "session.ttl_days"
	SettingKeySessionRememberTTLDays         SettingKey = "session.remember_ttl_days"
	SettingKeyEmailEnabled                   SettingKey = "email.enabled"
	SettingKeyEmailProvider                  SettingKey = "email.provider"
	SettingKeyEmailResendAPIKey              SettingKey = "email.resend_api_key"
	SettingKeyEmailResendDomain              SettingKey = "email.resend_domain"
)

type SettingDefinition struct {
	Key          SettingKey
	Group        string
	Label        string
	Description  string
	Type         SettingType
	DefaultValue *string
	IsRequired   bool
	IsSensitive  bool
	IsEncrypted  bool
	IsEnvOnly    bool
	EnvVars      []string
	Options      []string
	MinInt       *int
}

type ResolvedSettingValue struct {
	Definition SettingDefinition
	Value      string
	HasValue   bool
	Source     string
}

const (
	ValueSourceEnv     = "env"
	ValueSourceDefault = "default"
	ValueSourceDB      = "db"
)

type EnvLookupFunc func(string) (string, bool)

var registry = map[SettingKey]SettingDefinition{
	SettingKeyPostgresURI: {
		Key:         SettingKeyPostgresURI,
		Group:       "postgres",
		Label:       "Postgres URI",
		Description: "Primary PostgreSQL connection string for the backend.",
		Type:        SettingTypeString,
		IsRequired:  true,
		IsEnvOnly:   true,
		EnvVars:     []string{"POSTGRES_URI"},
	},
	SettingKeyAppEnv: {
		Key:          SettingKeyAppEnv,
		Group:        "app",
		Label:        "Environment",
		Description:  "Application environment name.",
		Type:         SettingTypeString,
		DefaultValue: stringSetting("development"),
		IsEnvOnly:    true,
		EnvVars:      []string{"APP_ENV"},
	},
	SettingKeyPort: {
		Key:          SettingKeyPort,
		Group:        "app",
		Label:        "Port",
		Description:  "HTTP listen port.",
		Type:         SettingTypeString,
		DefaultValue: stringSetting("8080"),
		IsEnvOnly:    true,
		EnvVars:      []string{"PORT"},
	},
	SettingKeyAppBaseURL: {
		Key:          SettingKeyAppBaseURL,
		Group:        "app",
		Label:        "Base URL",
		Description:  "Public frontend base URL used in generated links.",
		Type:         SettingTypeString,
		DefaultValue: stringSetting("http://localhost:3000"),
		IsEnvOnly:    true,
		EnvVars:      []string{"APP_BASE_URL"},
	},
	SettingKeyAppAllowedOrigins: {
		Key:         SettingKeyAppAllowedOrigins,
		Group:       "app",
		Label:       "Allowed Origins",
		Description: "Comma-separated browser origins allowed to call the API via CORS.",
		Type:        SettingTypeString,
		IsEnvOnly:   true,
		EnvVars:     []string{"APP_ALLOWED_ORIGINS"},
	},
	SettingKeyAppSecret: {
		Key:         SettingKeyAppSecret,
		Group:       "app",
		Label:       "App Secret",
		Description: "Application secret used for cryptographic derivation.",
		Type:        SettingTypeString,
		IsRequired:  true,
		IsSensitive: true,
		IsEnvOnly:   true,
		EnvVars:     []string{"APP_SECRET"},
	},
	SettingKeyNodeHeartbeatTTLSeconds: {
		Key:          SettingKeyNodeHeartbeatTTLSeconds,
		Group:        "node",
		Label:        "Heartbeat TTL",
		Description:  "TTL in seconds before a node is treated as offline.",
		Type:         SettingTypeInt,
		DefaultValue: stringSetting("60"),
		IsEnvOnly:    true,
		EnvVars:      []string{"NODE_HEARTBEAT_TTL_SECONDS"},
		MinInt:       intSetting(1),
	},
	SettingKeyNodeExpiryCheckIntervalSeconds: {
		Key:          SettingKeyNodeExpiryCheckIntervalSeconds,
		Group:        "node",
		Label:        "Expiry Check Interval",
		Description:  "Interval in seconds between node expiry checks.",
		Type:         SettingTypeInt,
		DefaultValue: stringSetting("15"),
		IsEnvOnly:    true,
		EnvVars:      []string{"NODE_EXPIRY_CHECK_INTERVAL_SECONDS"},
		MinInt:       intSetting(1),
	},
	SettingKeyManagedCheckIntervalSeconds: {
		Key:          SettingKeyManagedCheckIntervalSeconds,
		Group:        "managed_service",
		Label:        "Check Interval",
		Description:  "Interval in seconds between managed service checks.",
		Type:         SettingTypeInt,
		DefaultValue: stringSetting("15"),
		IsEnvOnly:    true,
		EnvVars:      []string{"MANAGED_SERVICE_CHECK_INTERVAL_SECONDS"},
		MinInt:       intSetting(1),
	},
	SettingKeyManagedCheckTimeoutSeconds: {
		Key:          SettingKeyManagedCheckTimeoutSeconds,
		Group:        "managed_service",
		Label:        "Check Timeout",
		Description:  "Timeout in seconds for managed service checks.",
		Type:         SettingTypeInt,
		DefaultValue: stringSetting("10"),
		IsEnvOnly:    true,
		EnvVars:      []string{"MANAGED_SERVICE_CHECK_TIMEOUT_SECONDS"},
		MinInt:       intSetting(1),
	},
	SettingKeySessionTTLDays: {
		Key:          SettingKeySessionTTLDays,
		Group:        "session",
		Label:        "Session TTL",
		Description:  "Session lifetime in days.",
		Type:         SettingTypeInt,
		DefaultValue: stringSetting("7"),
		IsEnvOnly:    true,
		EnvVars:      []string{"SESSION_TTL_DAYS"},
		MinInt:       intSetting(1),
	},
	SettingKeySessionRememberTTLDays: {
		Key:          SettingKeySessionRememberTTLDays,
		Group:        "session",
		Label:        "Remember Session TTL",
		Description:  "Remember-me session lifetime in days.",
		Type:         SettingTypeInt,
		DefaultValue: stringSetting("30"),
		IsEnvOnly:    true,
		EnvVars:      []string{"SESSION_REMEMBER_TTL_DAYS"},
		MinInt:       intSetting(1),
	},
	SettingKeyEmailEnabled: {
		Key:          SettingKeyEmailEnabled,
		Group:        "email",
		Label:        "Email Enabled",
		Description:  "Enables outbound email delivery when provider config is complete.",
		Type:         SettingTypeBool,
		DefaultValue: stringSetting("false"),
		EnvVars:      []string{"EMAIL_ENABLED"},
	},
	SettingKeyEmailProvider: {
		Key:          SettingKeyEmailProvider,
		Group:        "email",
		Label:        "Email Provider",
		Description:  "Outbound email provider.",
		Type:         SettingTypeEnum,
		DefaultValue: stringSetting("resend"),
		EnvVars:      []string{"EMAIL_PROVIDER"},
		Options:      []string{"resend"},
	},
	SettingKeyEmailResendAPIKey: {
		Key:         SettingKeyEmailResendAPIKey,
		Group:       "email",
		Label:       "Resend API Key",
		Description: "Resend API key used for sending emails.",
		Type:        SettingTypeString,
		IsSensitive: true,
		IsEncrypted: true,
		EnvVars:     []string{"EMAIL_RESEND_API_KEY"},
	},
	SettingKeyEmailResendDomain: {
		Key:         SettingKeyEmailResendDomain,
		Group:       "email",
		Label:       "Resend Domain",
		Description: "Resend sending domain.",
		Type:        SettingTypeString,
		EnvVars:     []string{"EMAIL_RESEND_DOMAIN"},
	},
}

func AllSettingDefinitions() []SettingDefinition {
	definitions := make([]SettingDefinition, 0, len(registry))
	for _, definition := range registry {
		definitions = append(definitions, definition)
	}
	slices.SortFunc(definitions, func(left SettingDefinition, right SettingDefinition) int {
		return strings.Compare(string(left.Key), string(right.Key))
	})

	return definitions
}

func DBSettingDefinitions() []SettingDefinition {
	definitions := make([]SettingDefinition, 0)
	for _, definition := range AllSettingDefinitions() {
		if definition.IsEnvOnly {
			continue
		}

		definitions = append(definitions, definition)
	}

	return definitions
}

func LookupDefinition(key SettingKey) (SettingDefinition, bool) {
	definition, ok := registry[key]
	return definition, ok
}

func MustDefinition(key SettingKey) SettingDefinition {
	definition, ok := LookupDefinition(key)
	if !ok {
		panic(fmt.Sprintf("unknown setting key %q", key))
	}

	return definition
}

func ResolveEnvValue(key SettingKey, lookup EnvLookupFunc) (ResolvedSettingValue, error) {
	definition, ok := LookupDefinition(key)
	if !ok {
		return ResolvedSettingValue{}, fmt.Errorf("unknown setting key %q", key)
	}

	rawValue, found := lookupValue(definition, lookup)
	if found {
		if err := ValidateSettingValue(definition, &rawValue); err != nil {
			return ResolvedSettingValue{}, err
		}

		return ResolvedSettingValue{
			Definition: definition,
			Value:      rawValue,
			HasValue:   true,
			Source:     ValueSourceEnv,
		}, nil
	}

	if definition.DefaultValue != nil {
		defaultValue := *definition.DefaultValue
		if err := ValidateSettingValue(definition, &defaultValue); err != nil {
			return ResolvedSettingValue{}, err
		}

		return ResolvedSettingValue{
			Definition: definition,
			Value:      defaultValue,
			HasValue:   true,
			Source:     ValueSourceDefault,
		}, nil
	}

	if definition.IsRequired {
		return ResolvedSettingValue{}, fmt.Errorf("%s is required", primaryEnvVar(definition))
	}

	return ResolvedSettingValue{Definition: definition}, nil
}

func ResolveEnvValueFromOS(key SettingKey) (ResolvedSettingValue, error) {
	return ResolveEnvValue(key, os.LookupEnv)
}

func ValidateSettingValue(definition SettingDefinition, value *string) error {
	if value == nil {
		if definition.IsRequired {
			return fmt.Errorf("%s is required", primaryEnvVar(definition))
		}

		return nil
	}

	rawValue := strings.TrimSpace(*value)
	if rawValue == "" {
		if definition.IsRequired {
			return fmt.Errorf("%s is required", primaryEnvVar(definition))
		}

		if definition.Type == SettingTypeBool || definition.Type == SettingTypeInt || definition.Type == SettingTypeEnum {
			return fmt.Errorf("%s cannot be blank", definition.Key)
		}

		return nil
	}

	switch definition.Type {
	case SettingTypeBool:
		if _, err := strconv.ParseBool(rawValue); err != nil {
			return fmt.Errorf("%s must be a boolean: %w", definition.Key, err)
		}
	case SettingTypeInt:
		parsedValue, err := strconv.Atoi(rawValue)
		if err != nil {
			return fmt.Errorf("%s must be an integer: %w", definition.Key, err)
		}
		if definition.MinInt != nil && parsedValue < *definition.MinInt {
			return fmt.Errorf("%s must be greater than or equal to %d", definition.Key, *definition.MinInt)
		}
	case SettingTypeEnum:
		if !slices.Contains(definition.Options, rawValue) {
			return fmt.Errorf("%s must be one of %s", definition.Key, strings.Join(definition.Options, ", "))
		}
	case SettingTypeString:
		return nil
	default:
		return fmt.Errorf("%s has unsupported type %q", definition.Key, definition.Type)
	}

	return nil
}

func lookupValue(definition SettingDefinition, lookup EnvLookupFunc) (string, bool) {
	if lookup == nil {
		return "", false
	}

	for _, envVar := range definition.EnvVars {
		value, ok := lookup(envVar)
		if !ok {
			continue
		}

		trimmedValue := strings.TrimSpace(value)
		if trimmedValue == "" {
			continue
		}

		return trimmedValue, true
	}

	return "", false
}

func primaryEnvVar(definition SettingDefinition) string {
	if len(definition.EnvVars) == 0 {
		return string(definition.Key)
	}

	return definition.EnvVars[0]
}

func stringSetting(value string) *string {
	return &value
}

func intSetting(value int) *int {
	return &value
}
