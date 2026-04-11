package controllers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MariusBobitiu/agrafa-backend/src/types"
)

type fakeInstanceSettingControllerService struct {
	settings      []types.InstanceSettingReadData
	receivedPatch []types.InstanceSettingsUpdateItemRequest
	listErr       error
	updateErr     error
}

func (s *fakeInstanceSettingControllerService) ListForUI(_ context.Context) ([]types.InstanceSettingReadData, error) {
	return s.settings, s.listErr
}

func (s *fakeInstanceSettingControllerService) UpdateBatchForUI(_ context.Context, updates []types.InstanceSettingsUpdateItemRequest) ([]types.InstanceSettingReadData, error) {
	s.receivedPatch = updates
	return s.settings, s.updateErr
}

func TestInstanceSettingControllerListReturnsSettings(t *testing.T) {
	t.Parallel()

	controller := NewInstanceSettingController(&fakeInstanceSettingControllerService{
		settings: []types.InstanceSettingReadData{
			{
				Key:             "email.enabled",
				Group:           "email",
				Label:           "Email Enabled",
				Description:     "Enables outbound email delivery when provider config is complete.",
				Type:            "bool",
				Value:           true,
				IsSensitive:     false,
				IsEncrypted:     false,
				IsEnvOverridden: true,
				IsEditable:      true,
			},
		},
	})

	request := httptest.NewRequest(http.MethodGet, "/v1/instance-settings", nil)
	recorder := httptest.NewRecorder()

	controller.List(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), `"key":"email.enabled"`) {
		t.Fatalf("body = %s", recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"is_env_overridden":true`) {
		t.Fatalf("body = %s", recorder.Body.String())
	}
}

func TestInstanceSettingControllerUpdateDecodesBatchPayload(t *testing.T) {
	t.Parallel()

	service := &fakeInstanceSettingControllerService{
		settings: []types.InstanceSettingReadData{
			{
				Key:             "email.enabled",
				Group:           "email",
				Label:           "Email Enabled",
				Description:     "Enables outbound email delivery when provider config is complete.",
				Type:            "bool",
				Value:           true,
				IsSensitive:     false,
				IsEncrypted:     false,
				IsEnvOverridden: false,
				IsEditable:      true,
			},
		},
	}
	controller := NewInstanceSettingController(service)

	request := httptest.NewRequest(http.MethodPatch, "/v1/instance-settings", strings.NewReader(`{"settings":[{"key":"email.enabled","value":true},{"key":"email.resend_domain","value":"email.example.com"}]}`))
	recorder := httptest.NewRecorder()

	controller.Update(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if len(service.receivedPatch) != 2 {
		t.Fatalf("receivedPatch = %d, want 2", len(service.receivedPatch))
	}
	if value, ok := service.receivedPatch[0].Value.(bool); !ok || !value {
		t.Fatalf("first patch value = %#v, want boolean true", service.receivedPatch[0].Value)
	}
	if value, ok := service.receivedPatch[1].Value.(string); !ok || value != "email.example.com" {
		t.Fatalf("second patch value = %#v, want string", service.receivedPatch[1].Value)
	}
}

func TestInstanceSettingControllerUpdateRejectsInvalidPayload(t *testing.T) {
	t.Parallel()

	controller := NewInstanceSettingController(&fakeInstanceSettingControllerService{})

	request := httptest.NewRequest(http.MethodPatch, "/v1/instance-settings", strings.NewReader(`{"settings":[{"key":"email.enabled","value":true}]`))
	recorder := httptest.NewRecorder()

	controller.Update(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "invalid instance settings payload") {
		t.Fatalf("body = %s", recorder.Body.String())
	}
}
