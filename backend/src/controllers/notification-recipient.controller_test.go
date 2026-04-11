package controllers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MariusBobitiu/agrafa-backend/src/types"
)

type fakeNotificationRecipientControllerService struct {
	testEmailProjectID int64
	testEmailRecipient string
	testEmailErr       error
}

func (s *fakeNotificationRecipientControllerService) Create(_ context.Context, _ types.CreateNotificationRecipientsInput) ([]types.NotificationRecipientReadData, error) {
	return nil, nil
}

func (s *fakeNotificationRecipientControllerService) List(_ context.Context, _ *int64) ([]types.NotificationRecipientReadData, error) {
	return nil, nil
}

func (s *fakeNotificationRecipientControllerService) GetByID(_ context.Context, _ int64) (types.NotificationRecipientReadData, error) {
	return types.NotificationRecipientReadData{}, nil
}

func (s *fakeNotificationRecipientControllerService) SetEnabled(_ context.Context, _ types.UpdateNotificationRecipientInput) (types.NotificationRecipientReadData, error) {
	return types.NotificationRecipientReadData{}, nil
}

func (s *fakeNotificationRecipientControllerService) Delete(_ context.Context, _ int64) error {
	return nil
}

func (s *fakeNotificationRecipientControllerService) SendTestEmail(_ context.Context, projectID int64, email string) error {
	s.testEmailProjectID = projectID
	s.testEmailRecipient = email
	return s.testEmailErr
}

func TestNotificationRecipientControllerSendTestEmailAcceptsPayload(t *testing.T) {
	t.Parallel()

	service := &fakeNotificationRecipientControllerService{}
	controller := NewNotificationRecipientController(service)
	request := httptest.NewRequest(http.MethodPost, "/v1/notification-recipients/test-email", strings.NewReader(`{"project_id":1,"email":"ops@example.com"}`))
	recorder := httptest.NewRecorder()

	controller.SendTestEmail(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if service.testEmailProjectID != 1 {
		t.Fatalf("projectID = %d, want 1", service.testEmailProjectID)
	}
	if service.testEmailRecipient != "ops@example.com" {
		t.Fatalf("email = %q", service.testEmailRecipient)
	}
	if !strings.Contains(recorder.Body.String(), `"status":"ok"`) {
		t.Fatalf("body = %s", recorder.Body.String())
	}
}

func TestNotificationRecipientControllerSendTestEmailRejectsMalformedPayload(t *testing.T) {
	t.Parallel()

	controller := NewNotificationRecipientController(&fakeNotificationRecipientControllerService{})
	request := httptest.NewRequest(http.MethodPost, "/v1/notification-recipients/test-email", strings.NewReader(`{"project_id":1`))
	recorder := httptest.NewRecorder()

	controller.SendTestEmail(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "invalid notification recipient test email payload") {
		t.Fatalf("body = %s", recorder.Body.String())
	}
}
