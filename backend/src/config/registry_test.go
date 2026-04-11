package config

import "testing"

func TestLookupDefinitionRejectsUnknownKey(t *testing.T) {
	t.Parallel()

	if _, ok := LookupDefinition(SettingKey("email.unknown")); ok {
		t.Fatal("LookupDefinition() unexpectedly accepted unknown key")
	}
}

func TestValidateSettingValueRejectsInvalidEnum(t *testing.T) {
	t.Parallel()

	definition := MustDefinition(SettingKeyEmailProvider)
	value := "ses"
	err := ValidateSettingValue(definition, &value)
	if err == nil || err.Error() != "email.provider must be one of resend" {
		t.Fatalf("ValidateSettingValue() error = %v", err)
	}
}

func TestValidateSettingValueRejectsInvalidType(t *testing.T) {
	t.Parallel()

	definition := MustDefinition(SettingKeyNodeHeartbeatTTLSeconds)
	value := "fast"
	err := ValidateSettingValue(definition, &value)
	if err == nil {
		t.Fatal("ValidateSettingValue() error = nil")
	}
}

func TestResolveEnvValueUsesDefault(t *testing.T) {
	t.Parallel()

	resolved, err := ResolveEnvValue(SettingKeyEmailEnabled, func(string) (string, bool) {
		return "", false
	})
	if err != nil {
		t.Fatalf("ResolveEnvValue() error = %v", err)
	}
	if !resolved.HasValue || resolved.Value != "false" || resolved.Source != ValueSourceDefault {
		t.Fatalf("unexpected resolved value: %#v", resolved)
	}
}

func TestResolveEnvValueRequiresRequiredValue(t *testing.T) {
	t.Parallel()

	_, err := ResolveEnvValue(SettingKeyAppSecret, func(string) (string, bool) {
		return "", false
	})
	if err == nil {
		t.Fatalf("ResolveEnvValue() error = %v", err)
	}
	if err.Error() != "APP_SECRET is required" {
		t.Fatalf("ResolveEnvValue() error = %v, want APP_SECRET is required", err)
	}
}
