package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/MariusBobitiu/agrafa-backend/src/config"
	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	emailpkg "github.com/MariusBobitiu/agrafa-backend/src/email"
	"github.com/MariusBobitiu/agrafa-backend/src/repositories"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
)

type instanceSettingRepository interface {
	GetByKey(ctx context.Context, key string) (generated.InstanceSetting, error)
	List(ctx context.Context) ([]generated.InstanceSetting, error)
	Upsert(ctx context.Context, params generated.UpsertInstanceSettingParams) (generated.InstanceSetting, error)
	DeleteByKey(ctx context.Context, key string) (int64, error)
}

type InstanceSettingView struct {
	Key         string
	Description string
	Value       *string
	IsSensitive bool
	IsEncrypted bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type EmailRuntimeConfig struct {
	Enabled             bool
	Provider            string
	ResendAPIKey        string
	ResendDomain        string
	AlertsFrom          string
	NotificationsFrom   string
	SecurityFrom        string
	IsAvailable         bool
	UnavailableReason   string
	EnabledValueSource  string
	ProviderValueSource string
}

type InstanceSettingService struct {
	repo      instanceSettingRepository
	encryptor *config.Encryptor
	envLookup config.EnvLookupFunc
}

func NewInstanceSettingService(
	repo *repositories.InstanceSettingRepository,
	encryptor *config.Encryptor,
	envLookup config.EnvLookupFunc,
) *InstanceSettingService {
	return &InstanceSettingService{
		repo:      repo,
		encryptor: encryptor,
		envLookup: envLookup,
	}
}

func (s *InstanceSettingService) Upsert(ctx context.Context, key config.SettingKey, value *string) (InstanceSettingView, error) {
	definition, ok := config.LookupDefinition(key)
	if !ok {
		return InstanceSettingView{}, types.ErrInvalidInstanceSettingKey
	}
	if definition.IsEnvOnly {
		return InstanceSettingView{}, fmt.Errorf("%w: %s is env-only", types.ErrInvalidInstanceSettingKey, key)
	}
	if err := config.ValidateSettingValue(definition, value); err != nil {
		return InstanceSettingView{}, fmt.Errorf("%w: %v", types.ErrInvalidInstanceSettingValue, err)
	}

	var storedValue sql.NullString
	if value != nil {
		nextValue := *value
		if definition.IsEncrypted {
			if s.encryptor == nil {
				return InstanceSettingView{}, fmt.Errorf("instance setting encryption is not configured")
			}

			encryptedValue, err := s.encryptor.Encrypt(nextValue)
			if err != nil {
				return InstanceSettingView{}, fmt.Errorf("encrypt instance setting %s: %w", key, err)
			}
			nextValue = encryptedValue
		}

		storedValue = sql.NullString{String: nextValue, Valid: true}
	}

	setting, err := s.repo.Upsert(ctx, generated.UpsertInstanceSettingParams{
		Key:         string(definition.Key),
		Value:       storedValue,
		Description: sql.NullString{String: definition.Description, Valid: definition.Description != ""},
		IsSensitive: definition.IsSensitive,
		IsEncrypted: definition.IsEncrypted,
	})
	if err != nil {
		return InstanceSettingView{}, fmt.Errorf("upsert instance setting %s: %w", key, err)
	}

	return mapInstanceSettingView(setting, definition, false), nil
}

func (s *InstanceSettingService) Get(ctx context.Context, key config.SettingKey) (InstanceSettingView, error) {
	definition, ok := config.LookupDefinition(key)
	if !ok {
		return InstanceSettingView{}, types.ErrInvalidInstanceSettingKey
	}
	if definition.IsEnvOnly {
		return InstanceSettingView{}, fmt.Errorf("%w: %s is env-only", types.ErrInvalidInstanceSettingKey, key)
	}

	setting, err := s.repo.GetByKey(ctx, string(key))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return InstanceSettingView{
				Key:         string(definition.Key),
				Description: definition.Description,
				IsSensitive: definition.IsSensitive,
				IsEncrypted: definition.IsEncrypted,
			}, nil
		}

		return InstanceSettingView{}, fmt.Errorf("get instance setting %s: %w", key, err)
	}

	return mapInstanceSettingView(setting, definition, false), nil
}

func (s *InstanceSettingService) List(ctx context.Context) ([]InstanceSettingView, error) {
	rows, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list instance settings: %w", err)
	}

	views := make([]InstanceSettingView, 0, len(rows))
	for _, row := range rows {
		definition, ok := config.LookupDefinition(config.SettingKey(row.Key))
		if !ok || definition.IsEnvOnly {
			continue
		}

		views = append(views, mapInstanceSettingView(row, definition, false))
	}

	return views, nil
}

func (s *InstanceSettingService) ListForUI(ctx context.Context) ([]types.InstanceSettingReadData, error) {
	definitions := instanceSettingsUIDefinitions()
	items := make([]types.InstanceSettingReadData, 0, len(definitions))
	for _, definition := range definitions {
		resolved, err := s.resolve(ctx, definition.Key)
		if err != nil {
			return nil, fmt.Errorf("resolve instance setting %s: %w", definition.Key, err)
		}

		item, err := mapInstanceSettingReadData(definition, resolved)
		if err != nil {
			return nil, fmt.Errorf("map instance setting %s: %w", definition.Key, err)
		}

		items = append(items, item)
	}

	return items, nil
}

func (s *InstanceSettingService) UpdateBatchForUI(ctx context.Context, updates []types.InstanceSettingsUpdateItemRequest) ([]types.InstanceSettingReadData, error) {
	type normalizedUpdate struct {
		key   config.SettingKey
		value *string
	}

	normalized := make([]normalizedUpdate, 0, len(updates))
	for _, update := range updates {
		key := config.SettingKey(update.Key)
		definition, ok := instanceSettingsUIDefinition(key)
		if !ok {
			return nil, fmt.Errorf("%w: %s is not supported", types.ErrInvalidInstanceSettingKey, update.Key)
		}

		value, err := normalizeInstanceSettingInputValue(definition, update.Value)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", types.ErrInvalidInstanceSettingValue, err)
		}
		if err := config.ValidateSettingValue(definition, value); err != nil {
			return nil, fmt.Errorf("%w: %v", types.ErrInvalidInstanceSettingValue, err)
		}

		normalized = append(normalized, normalizedUpdate{
			key:   key,
			value: value,
		})
	}

	for _, update := range normalized {
		if _, err := s.Upsert(ctx, update.key, update.value); err != nil {
			return nil, err
		}
	}

	return s.ListForUI(ctx)
}

func (s *InstanceSettingService) ResolveString(ctx context.Context, key config.SettingKey) (config.ResolvedSettingValue, error) {
	return s.resolve(ctx, key)
}

func (s *InstanceSettingService) ResolveBool(ctx context.Context, key config.SettingKey) (bool, config.ResolvedSettingValue, error) {
	resolved, err := s.resolve(ctx, key)
	if err != nil {
		return false, config.ResolvedSettingValue{}, err
	}
	if !resolved.HasValue {
		return false, resolved, nil
	}

	value, err := strconv.ParseBool(strings.TrimSpace(resolved.Value))
	if err != nil {
		return false, config.ResolvedSettingValue{}, fmt.Errorf("parse boolean setting %s: %w", key, err)
	}

	return value, resolved, nil
}

func (s *InstanceSettingService) ResolveEmailConfig(ctx context.Context) (EmailRuntimeConfig, error) {
	enabled, enabledResolved, err := s.ResolveBool(ctx, config.SettingKeyEmailEnabled)
	if err != nil {
		return EmailRuntimeConfig{}, err
	}

	providerResolved, err := s.ResolveString(ctx, config.SettingKeyEmailProvider)
	if err != nil {
		return EmailRuntimeConfig{}, err
	}

	apiKeyResolved, err := s.ResolveString(ctx, config.SettingKeyEmailResendAPIKey)
	if err != nil {
		return EmailRuntimeConfig{}, err
	}

	domainResolved, err := s.ResolveString(ctx, config.SettingKeyEmailResendDomain)
	if err != nil {
		return EmailRuntimeConfig{}, err
	}

	emailConfig := EmailRuntimeConfig{
		Enabled:             enabled,
		Provider:            providerResolved.Value,
		ResendAPIKey:        apiKeyResolved.Value,
		ResendDomain:        domainResolved.Value,
		EnabledValueSource:  enabledResolved.Source,
		ProviderValueSource: providerResolved.Source,
	}

	if !emailConfig.Enabled {
		emailConfig.UnavailableReason = "email.enabled is false"
		return emailConfig, nil
	}

	if emailConfig.Provider == "" {
		emailConfig.UnavailableReason = "email provider is not configured"
		return emailConfig, nil
	}

	switch emailConfig.Provider {
	case "resend":
		if emailConfig.ResendAPIKey == "" || emailConfig.ResendDomain == "" {
			emailConfig.UnavailableReason = "resend api key or resend domain is missing"
			return emailConfig, nil
		}

		emailConfig.AlertsFrom = emailpkg.BuildAlertsFromAddress(emailConfig.ResendDomain, "")
		emailConfig.NotificationsFrom = emailpkg.BuildNotificationsFromAddress(emailConfig.ResendDomain)
		emailConfig.SecurityFrom = emailpkg.BuildSecurityFromAddress(emailConfig.ResendDomain)
	default:
		emailConfig.UnavailableReason = fmt.Sprintf("email provider %q is not supported", emailConfig.Provider)
		return emailConfig, nil
	}

	emailConfig.IsAvailable = true
	return emailConfig, nil
}

func (s *InstanceSettingService) resolve(ctx context.Context, key config.SettingKey) (config.ResolvedSettingValue, error) {
	definition, ok := config.LookupDefinition(key)
	if !ok {
		return config.ResolvedSettingValue{}, types.ErrInvalidInstanceSettingKey
	}

	envValue, found := lookupResolvedEnvValue(definition, s.envLookup)
	if found {
		if err := config.ValidateSettingValue(definition, &envValue); err != nil {
			return config.ResolvedSettingValue{}, fmt.Errorf("%w: %v", types.ErrInvalidInstanceSettingValue, err)
		}

		return config.ResolvedSettingValue{
			Definition: definition,
			Value:      envValue,
			HasValue:   true,
			Source:     config.ValueSourceEnv,
		}, nil
	}

	if !definition.IsEnvOnly {
		if s.repo == nil {
			return config.ResolvedSettingValue{}, fmt.Errorf("instance setting repository is not configured")
		}

		setting, err := s.repo.GetByKey(ctx, string(definition.Key))
		switch {
		case err == nil:
			rawValue := ""
			if setting.Value.Valid {
				rawValue = setting.Value.String
				if setting.IsEncrypted || definition.IsEncrypted {
					if s.encryptor == nil {
						return config.ResolvedSettingValue{}, fmt.Errorf("instance setting encryption is not configured")
					}

					rawValue, err = s.encryptor.Decrypt(rawValue)
					if err != nil {
						return config.ResolvedSettingValue{}, fmt.Errorf("decrypt instance setting %s: %w", key, err)
					}
				}

				if err := config.ValidateSettingValue(definition, &rawValue); err != nil {
					return config.ResolvedSettingValue{}, fmt.Errorf("%w: %v", types.ErrInvalidInstanceSettingValue, err)
				}

				return config.ResolvedSettingValue{
					Definition: definition,
					Value:      rawValue,
					HasValue:   true,
					Source:     config.ValueSourceDB,
				}, nil
			}
		case errors.Is(err, sql.ErrNoRows):
		default:
			return config.ResolvedSettingValue{}, fmt.Errorf("read instance setting %s: %w", key, err)
		}
	}

	if definition.DefaultValue != nil {
		defaultValue := *definition.DefaultValue
		if err := config.ValidateSettingValue(definition, &defaultValue); err != nil {
			return config.ResolvedSettingValue{}, fmt.Errorf("%w: %v", types.ErrInvalidInstanceSettingValue, err)
		}

		return config.ResolvedSettingValue{
			Definition: definition,
			Value:      defaultValue,
			HasValue:   true,
			Source:     config.ValueSourceDefault,
		}, nil
	}

	if definition.IsRequired {
		return config.ResolvedSettingValue{}, fmt.Errorf("%s is required", definition.EnvVars[0])
	}

	return config.ResolvedSettingValue{Definition: definition}, nil
}

func mapInstanceSettingView(setting generated.InstanceSetting, definition config.SettingDefinition, includeSensitiveValue bool) InstanceSettingView {
	view := InstanceSettingView{
		Key:         setting.Key,
		Description: definition.Description,
		IsSensitive: definition.IsSensitive,
		IsEncrypted: definition.IsEncrypted,
		CreatedAt:   setting.CreatedAt,
		UpdatedAt:   setting.UpdatedAt,
	}

	if !setting.Value.Valid {
		return view
	}

	value := setting.Value.String
	if definition.IsSensitive && !includeSensitiveValue {
		value = maskSensitiveValue(value)
	}
	view.Value = &value

	return view
}

func maskSensitiveValue(value string) string {
	if value == "" {
		return ""
	}

	return "********"
}

func lookupResolvedEnvValue(definition config.SettingDefinition, lookup config.EnvLookupFunc) (string, bool) {
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

func instanceSettingsUIDefinitions() []config.SettingDefinition {
	definitions := make([]config.SettingDefinition, 0)
	for _, definition := range config.DBSettingDefinitions() {
		if definition.Group != "email" {
			continue
		}

		definitions = append(definitions, definition)
	}

	return definitions
}

func instanceSettingsUIDefinition(key config.SettingKey) (config.SettingDefinition, bool) {
	definition, ok := config.LookupDefinition(key)
	if !ok || definition.IsEnvOnly || definition.Group != "email" {
		return config.SettingDefinition{}, false
	}

	return definition, true
}

func mapInstanceSettingReadData(definition config.SettingDefinition, resolved config.ResolvedSettingValue) (types.InstanceSettingReadData, error) {
	item := types.InstanceSettingReadData{
		Key:             string(definition.Key),
		Group:           definition.Group,
		Label:           settingLabel(definition),
		Description:     definition.Description,
		Type:            string(definition.Type),
		IsSensitive:     definition.IsSensitive,
		IsEncrypted:     definition.IsEncrypted,
		IsEnvOverridden: resolved.Source == config.ValueSourceEnv,
		IsEditable:      true,
	}

	if definition.IsSensitive {
		configured := resolved.HasValue
		item.IsConfigured = &configured
		if configured {
			item.Value = maskSensitiveValue(resolved.Value)
		}

		return item, nil
	}

	if !resolved.HasValue {
		return item, nil
	}

	value, err := parseInstanceSettingValue(definition, resolved.Value)
	if err != nil {
		return types.InstanceSettingReadData{}, err
	}

	item.Value = value
	return item, nil
}

func settingLabel(definition config.SettingDefinition) string {
	if definition.Label != "" {
		return definition.Label
	}

	return string(definition.Key)
}

func parseInstanceSettingValue(definition config.SettingDefinition, value string) (any, error) {
	switch definition.Type {
	case config.SettingTypeBool:
		parsed, err := strconv.ParseBool(strings.TrimSpace(value))
		if err != nil {
			return nil, fmt.Errorf("%s must be a boolean: %w", definition.Key, err)
		}

		return parsed, nil
	case config.SettingTypeInt:
		parsed, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil {
			return nil, fmt.Errorf("%s must be an integer: %w", definition.Key, err)
		}

		return parsed, nil
	case config.SettingTypeEnum, config.SettingTypeString:
		return value, nil
	default:
		return nil, fmt.Errorf("%s has unsupported type %q", definition.Key, definition.Type)
	}
}

func normalizeInstanceSettingInputValue(definition config.SettingDefinition, value any) (*string, error) {
	if value == nil {
		return nil, nil
	}

	switch definition.Type {
	case config.SettingTypeBool:
		parsed, ok := value.(bool)
		if !ok {
			return nil, fmt.Errorf("%s expects a boolean value", definition.Key)
		}

		normalized := strconv.FormatBool(parsed)
		return &normalized, nil
	case config.SettingTypeInt:
		number, ok := value.(float64)
		if !ok {
			return nil, fmt.Errorf("%s expects an integer value", definition.Key)
		}
		if math.Trunc(number) != number {
			return nil, fmt.Errorf("%s expects an integer value", definition.Key)
		}

		normalized := strconv.Itoa(int(number))
		return &normalized, nil
	case config.SettingTypeEnum, config.SettingTypeString:
		parsed, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("%s expects a string value", definition.Key)
		}

		normalized := strings.TrimSpace(parsed)
		return &normalized, nil
	default:
		return nil, fmt.Errorf("%s has unsupported type %q", definition.Key, definition.Type)
	}
}
