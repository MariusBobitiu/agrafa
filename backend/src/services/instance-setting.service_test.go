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
