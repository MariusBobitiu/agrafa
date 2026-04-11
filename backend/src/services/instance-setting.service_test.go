package services

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/MariusBobitiu/agrafa-backend/src/config"
	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
)

type fakeInstanceSettingRepository struct {
	items map[string]generated.InstanceSetting
	now   func() time.Time
}

func (r *fakeInstanceSettingRepository) GetByKey(_ context.Context, key string) (generated.InstanceSetting, error) {
	if setting, ok := r.items[key]; ok {
		return setting, nil
	}

	return generated.InstanceSetting{}, sql.ErrNoRows
}

func (r *fakeInstanceSettingRepository) List(_ context.Context) ([]generated.InstanceSetting, error) {
	rows := make([]generated.InstanceSetting, 0, len(r.items))
	for _, item := range r.items {
		rows = append(rows, item)
	}

	return rows, nil
}

func (r *fakeInstanceSettingRepository) Upsert(_ context.Context, params generated.UpsertInstanceSettingParams) (generated.InstanceSetting, error) {
	now := time.Now().UTC()
	if r.now != nil {
		now = r.now()
	}

	setting, ok := r.items[params.Key]
	if !ok {
		setting = generated.InstanceSetting{
			ID:        int64(len(r.items) + 1),
			Key:       params.Key,
			CreatedAt: now,
		}
	}

	setting.Value = params.Value
	setting.Description = params.Description
	setting.IsSensitive = params.IsSensitive
	setting.IsEncrypted = params.IsEncrypted
	setting.UpdatedAt = now
	r.items[params.Key] = setting

	return setting, nil
}

func (r *fakeInstanceSettingRepository) DeleteByKey(_ context.Context, key string) (int64, error) {
	if _, ok := r.items[key]; !ok {
		return 0, nil
	}

	delete(r.items, key)
	return 1, nil
}

func TestInstanceSettingServiceUpsertEncryptsSensitiveValuesAndStoresRegistryMetadata(t *testing.T) {
	t.Parallel()

	encryptor, err := config.NewEncryptor("test-app-secret")
	if err != nil {
		t.Fatalf("NewEncryptor() error = %v", err)
	}

	repo := &fakeInstanceSettingRepository{items: map[string]generated.InstanceSetting{}}
	service := &InstanceSettingService{
		repo:      repo,
		encryptor: encryptor,
	}

	value := "re_test_secret"
	view, err := service.Upsert(context.Background(), config.SettingKeyEmailResendAPIKey, &value)
	if err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}

	stored := repo.items[string(config.SettingKeyEmailResendAPIKey)]
	if !stored.Value.Valid {
		t.Fatal("stored value is not valid")
	}
	if stored.Value.String == value {
		t.Fatal("sensitive value was stored in plaintext")
	}
	if stored.Description.String != config.MustDefinition(config.SettingKeyEmailResendAPIKey).Description {
		t.Fatalf("stored description = %q", stored.Description.String)
	}
	if !stored.IsSensitive || !stored.IsEncrypted {
		t.Fatalf("stored flags = sensitive:%t encrypted:%t", stored.IsSensitive, stored.IsEncrypted)
	}
	if view.Value == nil || *view.Value != "********" {
		t.Fatalf("view.Value = %#v, want masked", view.Value)
	}
}

func TestInstanceSettingServiceResolveHonorsPrecedence(t *testing.T) {
	t.Parallel()

	encryptor, err := config.NewEncryptor("test-app-secret")
	if err != nil {
		t.Fatalf("NewEncryptor() error = %v", err)
	}

	repo := &fakeInstanceSettingRepository{
		items: map[string]generated.InstanceSetting{
			string(config.SettingKeyEmailEnabled): {
				ID:          1,
				Key:         string(config.SettingKeyEmailEnabled),
				Value:       sql.NullString{String: "true", Valid: true},
				Description: sql.NullString{String: "db value", Valid: true},
				CreatedAt:   time.Now().UTC(),
				UpdatedAt:   time.Now().UTC(),
			},
		},
	}

	service := &InstanceSettingService{
		repo:      repo,
		encryptor: encryptor,
		envLookup: func(name string) (string, bool) {
			if name == "EMAIL_ENABLED" {
				return "false", true
			}
			return "", false
		},
	}

	value, resolved, err := service.ResolveBool(context.Background(), config.SettingKeyEmailEnabled)
	if err != nil {
		t.Fatalf("ResolveBool() error = %v", err)
	}
	if value {
		t.Fatal("ResolveBool() = true, want false from env override")
	}
	if resolved.Source != config.ValueSourceEnv {
		t.Fatalf("resolved.Source = %q, want env", resolved.Source)
	}

	service.envLookup = nil
	value, resolved, err = service.ResolveBool(context.Background(), config.SettingKeyEmailEnabled)
	if err != nil {
		t.Fatalf("ResolveBool() error = %v", err)
	}
	if !value {
		t.Fatal("ResolveBool() = false, want true from db")
	}
	if resolved.Source != config.ValueSourceDB {
		t.Fatalf("resolved.Source = %q, want db", resolved.Source)
	}

	delete(repo.items, string(config.SettingKeyEmailEnabled))
	value, resolved, err = service.ResolveBool(context.Background(), config.SettingKeyEmailEnabled)
	if err != nil {
		t.Fatalf("ResolveBool() error = %v", err)
	}
	if value {
		t.Fatal("ResolveBool() = true, want false from default")
	}
	if resolved.Source != config.ValueSourceDefault {
		t.Fatalf("resolved.Source = %q, want default", resolved.Source)
	}
}

func TestInstanceSettingServiceResolveEmailConfigFromDB(t *testing.T) {
	t.Parallel()

	encryptor, err := config.NewEncryptor("test-app-secret")
	if err != nil {
		t.Fatalf("NewEncryptor() error = %v", err)
	}

	encryptedAPIKey, err := encryptor.Encrypt("re_db_secret")
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	repo := &fakeInstanceSettingRepository{
		items: map[string]generated.InstanceSetting{
			string(config.SettingKeyEmailEnabled): {
				Key:   string(config.SettingKeyEmailEnabled),
				Value: sql.NullString{String: "true", Valid: true},
			},
			string(config.SettingKeyEmailResendAPIKey): {
				Key:         string(config.SettingKeyEmailResendAPIKey),
				Value:       sql.NullString{String: encryptedAPIKey, Valid: true},
				IsSensitive: true,
				IsEncrypted: true,
			},
			string(config.SettingKeyEmailResendDomain): {
				Key:   string(config.SettingKeyEmailResendDomain),
				Value: sql.NullString{String: "email.agrafa.co", Valid: true},
			},
		},
	}

	service := &InstanceSettingService{
		repo:      repo,
		encryptor: encryptor,
	}

	emailConfig, err := service.ResolveEmailConfig(context.Background())
	if err != nil {
		t.Fatalf("ResolveEmailConfig() error = %v", err)
	}
	if !emailConfig.IsAvailable {
		t.Fatalf("ResolveEmailConfig() unavailable: %s", emailConfig.UnavailableReason)
	}
	if emailConfig.Provider != "resend" {
		t.Fatalf("Provider = %q, want resend", emailConfig.Provider)
	}
	if emailConfig.ResendAPIKey != "re_db_secret" {
		t.Fatalf("ResendAPIKey = %q", emailConfig.ResendAPIKey)
	}
	if emailConfig.AlertsFrom != "Agrafa Alerts <alerts@email.agrafa.co>" {
		t.Fatalf("AlertsFrom = %q", emailConfig.AlertsFrom)
	}
	if emailConfig.NotificationsFrom != "Agrafa Notifications <notifications@email.agrafa.co>" {
		t.Fatalf("NotificationsFrom = %q", emailConfig.NotificationsFrom)
	}
	if emailConfig.SecurityFrom != "Agrafa Security <security@email.agrafa.co>" {
		t.Fatalf("SecurityFrom = %q", emailConfig.SecurityFrom)
	}
}

func TestInstanceSettingServiceResolveEmailConfigHandlesMissingOptionalConfigGracefully(t *testing.T) {
	t.Parallel()

	encryptor, err := config.NewEncryptor("test-app-secret")
	if err != nil {
		t.Fatalf("NewEncryptor() error = %v", err)
	}

	repo := &fakeInstanceSettingRepository{
		items: map[string]generated.InstanceSetting{
			string(config.SettingKeyEmailEnabled): {
				Key:   string(config.SettingKeyEmailEnabled),
				Value: sql.NullString{String: "true", Valid: true},
			},
		},
	}

	service := &InstanceSettingService{
		repo:      repo,
		encryptor: encryptor,
	}

	emailConfig, err := service.ResolveEmailConfig(context.Background())
	if err != nil {
		t.Fatalf("ResolveEmailConfig() error = %v", err)
	}
	if emailConfig.IsAvailable {
		t.Fatal("ResolveEmailConfig() unexpectedly available")
	}
	if emailConfig.UnavailableReason != "resend api key or resend domain is missing" {
		t.Fatalf("UnavailableReason = %q", emailConfig.UnavailableReason)
	}
}

func TestInstanceSettingServiceRejectsUnknownKeys(t *testing.T) {
	t.Parallel()

	service := &InstanceSettingService{}
	value := "true"
	_, err := service.Upsert(context.Background(), config.SettingKey("email.unknown"), &value)
	if err == nil {
		t.Fatal("Upsert() error = nil")
	}
	if !errors.Is(err, types.ErrInvalidInstanceSettingKey) {
		t.Fatalf("Upsert() error = %v", err)
	}
}

func TestInstanceSettingServiceListForUIMasksSensitiveValuesAndExposesOverrideMetadata(t *testing.T) {
	t.Parallel()

	encryptor, err := config.NewEncryptor("test-app-secret")
	if err != nil {
		t.Fatalf("NewEncryptor() error = %v", err)
	}

	encryptedAPIKey, err := encryptor.Encrypt("re_db_secret")
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	repo := &fakeInstanceSettingRepository{
		items: map[string]generated.InstanceSetting{
			string(config.SettingKeyEmailEnabled): {
				Key:   string(config.SettingKeyEmailEnabled),
				Value: sql.NullString{String: "false", Valid: true},
			},
			string(config.SettingKeyEmailResendAPIKey): {
				Key:         string(config.SettingKeyEmailResendAPIKey),
				Value:       sql.NullString{String: encryptedAPIKey, Valid: true},
				IsSensitive: true,
				IsEncrypted: true,
			},
			string(config.SettingKeyEmailResendDomain): {
				Key:   string(config.SettingKeyEmailResendDomain),
				Value: sql.NullString{String: "email.db.example.com", Valid: true},
			},
		},
	}

	service := &InstanceSettingService{
		repo:      repo,
		encryptor: encryptor,
		envLookup: func(name string) (string, bool) {
			switch name {
			case "EMAIL_ENABLED":
				return "true", true
			case "EMAIL_RESEND_API_KEY":
				return "re_env_secret", true
			default:
				return "", false
			}
		},
	}

	settings, err := service.ListForUI(context.Background())
	if err != nil {
		t.Fatalf("ListForUI() error = %v", err)
	}

	emailEnabled := findInstanceSettingByKey(t, settings, string(config.SettingKeyEmailEnabled))
	if emailEnabled.Value != true {
		t.Fatalf("email.enabled value = %#v, want true", emailEnabled.Value)
	}
	if !emailEnabled.IsEnvOverridden {
		t.Fatal("email.enabled should be marked as env overridden")
	}
	if !emailEnabled.IsEditable {
		t.Fatal("email.enabled should remain editable")
	}

	apiKey := findInstanceSettingByKey(t, settings, string(config.SettingKeyEmailResendAPIKey))
	if apiKey.Value != "********" {
		t.Fatalf("email.resend_api_key value = %#v, want masked", apiKey.Value)
	}
	if apiKey.IsConfigured == nil || !*apiKey.IsConfigured {
		t.Fatalf("email.resend_api_key configured = %#v, want true", apiKey.IsConfigured)
	}
	if !apiKey.IsEnvOverridden {
		t.Fatal("email.resend_api_key should be marked as env overridden")
	}

	domain := findInstanceSettingByKey(t, settings, string(config.SettingKeyEmailResendDomain))
	if domain.Value != "email.db.example.com" {
		t.Fatalf("email.resend_domain value = %#v", domain.Value)
	}

	provider := findInstanceSettingByKey(t, settings, string(config.SettingKeyEmailProvider))
	if provider.Value != "resend" {
		t.Fatalf("email.provider value = %#v, want default resend", provider.Value)
	}
}

func TestInstanceSettingServiceUpdateBatchForUIUpdatesNonSensitiveValues(t *testing.T) {
	t.Parallel()

	repo := &fakeInstanceSettingRepository{items: map[string]generated.InstanceSetting{}}
	service := &InstanceSettingService{repo: repo}

	settings, err := service.UpdateBatchForUI(context.Background(), []types.InstanceSettingsUpdateItemRequest{
		{Key: string(config.SettingKeyEmailEnabled), Value: true},
		{Key: string(config.SettingKeyEmailResendDomain), Value: "email.example.com"},
	})
	if err != nil {
		t.Fatalf("UpdateBatchForUI() error = %v", err)
	}

	storedEnabled := repo.items[string(config.SettingKeyEmailEnabled)]
	if !storedEnabled.Value.Valid || storedEnabled.Value.String != "true" {
		t.Fatalf("stored email.enabled = %#v", storedEnabled.Value)
	}

	storedDomain := repo.items[string(config.SettingKeyEmailResendDomain)]
	if !storedDomain.Value.Valid || storedDomain.Value.String != "email.example.com" {
		t.Fatalf("stored email.resend_domain = %#v", storedDomain.Value)
	}

	emailEnabled := findInstanceSettingByKey(t, settings, string(config.SettingKeyEmailEnabled))
	if emailEnabled.Value != true {
		t.Fatalf("email.enabled response value = %#v, want true", emailEnabled.Value)
	}

	domain := findInstanceSettingByKey(t, settings, string(config.SettingKeyEmailResendDomain))
	if domain.Value != "email.example.com" {
		t.Fatalf("email.resend_domain response value = %#v", domain.Value)
	}
}

func TestInstanceSettingServiceUpdateBatchForUIEncryptsSensitiveValues(t *testing.T) {
	t.Parallel()

	encryptor, err := config.NewEncryptor("test-app-secret")
	if err != nil {
		t.Fatalf("NewEncryptor() error = %v", err)
	}

	repo := &fakeInstanceSettingRepository{items: map[string]generated.InstanceSetting{}}
	service := &InstanceSettingService{
		repo:      repo,
		encryptor: encryptor,
	}

	settings, err := service.UpdateBatchForUI(context.Background(), []types.InstanceSettingsUpdateItemRequest{
		{Key: string(config.SettingKeyEmailResendAPIKey), Value: "re_secret"},
	})
	if err != nil {
		t.Fatalf("UpdateBatchForUI() error = %v", err)
	}

	stored := repo.items[string(config.SettingKeyEmailResendAPIKey)]
	if !stored.Value.Valid {
		t.Fatal("stored sensitive value is not valid")
	}
	if stored.Value.String == "re_secret" {
		t.Fatal("sensitive value was stored in plaintext")
	}

	decrypted, err := encryptor.Decrypt(stored.Value.String)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}
	if decrypted != "re_secret" {
		t.Fatalf("decrypted sensitive value = %q", decrypted)
	}

	apiKey := findInstanceSettingByKey(t, settings, string(config.SettingKeyEmailResendAPIKey))
	if apiKey.Value != "********" {
		t.Fatalf("email.resend_api_key response value = %#v, want masked", apiKey.Value)
	}
}

func TestInstanceSettingServiceUpdateBatchForUIRejectsEnvOnlyKeys(t *testing.T) {
	t.Parallel()

	service := &InstanceSettingService{repo: &fakeInstanceSettingRepository{items: map[string]generated.InstanceSetting{}}}

	_, err := service.UpdateBatchForUI(context.Background(), []types.InstanceSettingsUpdateItemRequest{
		{Key: string(config.SettingKeyAppBaseURL), Value: "https://example.com"},
	})
	if err == nil {
		t.Fatal("UpdateBatchForUI() error = nil")
	}
	if !errors.Is(err, types.ErrInvalidInstanceSettingKey) {
		t.Fatalf("UpdateBatchForUI() error = %v", err)
	}
}

func TestInstanceSettingServiceUpdateBatchForUIRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	service := &InstanceSettingService{repo: &fakeInstanceSettingRepository{items: map[string]generated.InstanceSetting{}}}

	_, err := service.UpdateBatchForUI(context.Background(), []types.InstanceSettingsUpdateItemRequest{
		{Key: string(config.SettingKeyEmailProvider), Value: "ses"},
	})
	if err == nil {
		t.Fatal("UpdateBatchForUI() error = nil")
	}
	if !errors.Is(err, types.ErrInvalidInstanceSettingValue) {
		t.Fatalf("UpdateBatchForUI() error = %v", err)
	}
}

func findInstanceSettingByKey(t *testing.T, settings []types.InstanceSettingReadData, key string) types.InstanceSettingReadData {
	t.Helper()

	for _, setting := range settings {
		if setting.Key == key {
			return setting
		}
	}

	t.Fatalf("instance setting %q not found", key)
	return types.InstanceSettingReadData{}
}
