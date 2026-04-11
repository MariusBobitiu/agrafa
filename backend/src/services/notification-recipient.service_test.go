package services

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	emailpkg "github.com/MariusBobitiu/agrafa-backend/src/email"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
)

type fakeNotificationRecipientRepo struct {
	recipients   []generated.NotificationRecipient
	nextID       int64
	createCalls  int
	createInputs []generated.CreateNotificationRecipientParams
}

func (r *fakeNotificationRecipientRepo) Create(_ context.Context, params generated.CreateNotificationRecipientParams) (generated.NotificationRecipient, error) {
	r.createCalls++
	r.createInputs = append(r.createInputs, params)

	for index := range r.recipients {
		recipient := &r.recipients[index]
		if recipient.ProjectID == params.ProjectID && recipient.ChannelType == params.ChannelType && recipient.Target == params.Target {
			recipient.MinSeverity = params.MinSeverity
			recipient.IsEnabled = params.IsEnabled
			recipient.UpdatedAt = time.Date(2026, time.April, 5, 12, 5, 0, 0, time.UTC)
			return *recipient, nil
		}
	}

	r.nextID++
	recipient := generated.NotificationRecipient{
		ID:          r.nextID,
		ProjectID:   params.ProjectID,
		ChannelType: params.ChannelType,
		Target:      params.Target,
		MinSeverity: params.MinSeverity,
		IsEnabled:   params.IsEnabled,
		CreatedAt:   time.Date(2026, time.April, 5, 12, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2026, time.April, 5, 12, 0, 0, 0, time.UTC),
	}
	r.recipients = append(r.recipients, recipient)
	return recipient, nil
}

func (r *fakeNotificationRecipientRepo) List(_ context.Context, _ *int64) ([]generated.NotificationRecipient, error) {
	return r.recipients, nil
}

func (r *fakeNotificationRecipientRepo) ListByProjectAndChannel(_ context.Context, projectID int64, channelType string) ([]generated.NotificationRecipient, error) {
	items := make([]generated.NotificationRecipient, 0, len(r.recipients))
	for _, recipient := range r.recipients {
		if recipient.ProjectID == projectID && recipient.ChannelType == channelType {
			items = append(items, recipient)
		}
	}

	return items, nil
}

func (r *fakeNotificationRecipientRepo) GetByID(_ context.Context, id int64) (generated.NotificationRecipient, error) {
	for _, recipient := range r.recipients {
		if recipient.ID == id {
			return recipient, nil
		}
	}

	return generated.NotificationRecipient{}, sql.ErrNoRows
}

func (r *fakeNotificationRecipientRepo) UpdateEnabled(_ context.Context, id int64, isEnabled bool) (generated.NotificationRecipient, error) {
	for index := range r.recipients {
		if r.recipients[index].ID == id {
			r.recipients[index].IsEnabled = isEnabled
			return r.recipients[index], nil
		}
	}

	return generated.NotificationRecipient{}, sql.ErrNoRows
}

func (r *fakeNotificationRecipientRepo) Delete(_ context.Context, id int64) (int64, error) {
	for index := range r.recipients {
		if r.recipients[index].ID == id {
			r.recipients = append(r.recipients[:index], r.recipients[index+1:]...)
			return 1, nil
		}
	}

	return 0, nil
}

type fakeNotificationProjectRepo struct {
	projects map[int64]generated.Project
}

func (r *fakeNotificationProjectRepo) GetByID(_ context.Context, id int64) (generated.Project, error) {
	project, ok := r.projects[id]
	if !ok {
		return generated.Project{}, sql.ErrNoRows
	}

	return project, nil
}

type fakeNotificationRecipientEmailService struct {
	recipients []string
	lastData   emailpkg.NotificationRecipientTestTemplateData
}

func (s *fakeNotificationRecipientEmailService) SendNotificationRecipientTestEmail(_ context.Context, to string, data emailpkg.NotificationRecipientTestTemplateData) error {
	s.recipients = append(s.recipients, to)
	s.lastData = data
	return nil
}

func TestCreateNotificationRecipientsAcceptsValidEmailsAndPersistsMinSeverity(t *testing.T) {
	t.Parallel()

	repo := &fakeNotificationRecipientRepo{}
	service := &NotificationRecipientService{
		notificationRecipientRepo: repo,
		projectRepo: &fakeNotificationProjectRepo{
			projects: map[int64]generated.Project{
				1: {ID: 1, Name: "Agrafa"},
			},
		},
	}

	recipients, err := service.Create(context.Background(), types.CreateNotificationRecipientsInput{
		ProjectID:   1,
		ChannelType: types.NotificationChannelTypeEmail,
		Recipients: []types.CreateNotificationRecipientItemInput{
			{Target: "Ops@Example.com", MinSeverity: types.AlertSeverityCritical},
			{Target: "team@example.com", MinSeverity: types.AlertSeverityWarning},
		},
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if len(recipients) != 2 {
		t.Fatalf("len(recipients) = %d, want 2", len(recipients))
	}
	if recipients[0].Target != "ops@example.com" {
		t.Fatalf("expected lower-cased target, got %q", recipients[0].Target)
	}
	if recipients[0].MinSeverity != types.AlertSeverityCritical {
		t.Fatalf("expected min_severity %q, got %q", types.AlertSeverityCritical, recipients[0].MinSeverity)
	}
	if len(repo.createInputs) != 2 {
		t.Fatalf("len(createInputs) = %d, want 2", len(repo.createInputs))
	}
	if repo.createInputs[0].Target != "ops@example.com" {
		t.Fatalf("expected stored target to be normalized, got %q", repo.createInputs[0].Target)
	}
	if repo.createInputs[1].MinSeverity != types.AlertSeverityWarning {
		t.Fatalf("expected second stored min_severity %q, got %q", types.AlertSeverityWarning, repo.createInputs[1].MinSeverity)
	}
}

func TestCreateNotificationRecipientsRejectsInvalidEmail(t *testing.T) {
	t.Parallel()

	service := &NotificationRecipientService{
		notificationRecipientRepo: &fakeNotificationRecipientRepo{},
		projectRepo: &fakeNotificationProjectRepo{
			projects: map[int64]generated.Project{
				1: {ID: 1, Name: "Agrafa"},
			},
		},
	}

	_, err := service.Create(context.Background(), types.CreateNotificationRecipientsInput{
		ProjectID:   1,
		ChannelType: types.NotificationChannelTypeEmail,
		Recipients: []types.CreateNotificationRecipientItemInput{
			{Target: "not-an-email", MinSeverity: types.AlertSeverityInfo},
		},
	})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}

	if err != types.ErrInvalidNotificationTarget {
		t.Fatalf("expected ErrInvalidNotificationTarget, got %v", err)
	}
}

func TestCreateNotificationRecipientsRejectsInvalidMinSeverity(t *testing.T) {
	t.Parallel()

	service := &NotificationRecipientService{
		notificationRecipientRepo: &fakeNotificationRecipientRepo{},
		projectRepo: &fakeNotificationProjectRepo{
			projects: map[int64]generated.Project{
				1: {ID: 1, Name: "Agrafa"},
			},
		},
	}

	_, err := service.Create(context.Background(), types.CreateNotificationRecipientsInput{
		ProjectID:   1,
		ChannelType: types.NotificationChannelTypeEmail,
		Recipients: []types.CreateNotificationRecipientItemInput{
			{Target: "ops@example.com", MinSeverity: "urgent"},
		},
	})
	if !errors.Is(err, types.ErrInvalidNotificationMinSeverity) {
		t.Fatalf("Create() error = %v, want ErrInvalidNotificationMinSeverity", err)
	}
}

func TestCreateNotificationRecipientsRejectsEmptyRecipients(t *testing.T) {
	t.Parallel()

	service := &NotificationRecipientService{
		notificationRecipientRepo: &fakeNotificationRecipientRepo{},
		projectRepo: &fakeNotificationProjectRepo{
			projects: map[int64]generated.Project{
				1: {ID: 1, Name: "Agrafa"},
			},
		},
	}

	_, err := service.Create(context.Background(), types.CreateNotificationRecipientsInput{
		ProjectID:   1,
		ChannelType: types.NotificationChannelTypeEmail,
	})
	if !errors.Is(err, types.ErrEmptyNotificationRecipients) {
		t.Fatalf("Create() error = %v, want ErrEmptyNotificationRecipients", err)
	}
}

func TestCreateNotificationRecipientsDeduplicatesTargetsAndUpsertsExistingRecipient(t *testing.T) {
	t.Parallel()

	repo := &fakeNotificationRecipientRepo{
		recipients: []generated.NotificationRecipient{
			{
				ID:          7,
				ProjectID:   1,
				ChannelType: types.NotificationChannelTypeEmail,
				Target:      "ops@example.com",
				MinSeverity: types.AlertSeverityInfo,
				IsEnabled:   false,
			},
		},
		nextID: 7,
	}
	service := &NotificationRecipientService{
		notificationRecipientRepo: repo,
		projectRepo: &fakeNotificationProjectRepo{
			projects: map[int64]generated.Project{
				1: {ID: 1, Name: "Agrafa"},
			},
		},
	}

	recipients, err := service.Create(context.Background(), types.CreateNotificationRecipientsInput{
		ProjectID:   1,
		ChannelType: types.NotificationChannelTypeEmail,
		Recipients: []types.CreateNotificationRecipientItemInput{
			{Target: "ops@example.com", MinSeverity: types.AlertSeverityWarning},
			{Target: "OPS@example.com", MinSeverity: types.AlertSeverityCritical},
		},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if repo.createCalls != 1 {
		t.Fatalf("createCalls = %d, want 1", repo.createCalls)
	}
	if len(recipients) != 1 {
		t.Fatalf("len(recipients) = %d, want 1", len(recipients))
	}
	if recipients[0].MinSeverity != types.AlertSeverityCritical {
		t.Fatalf("min_severity = %q, want %q", recipients[0].MinSeverity, types.AlertSeverityCritical)
	}
	if !repo.recipients[0].IsEnabled {
		t.Fatal("expected existing recipient to be re-enabled on upsert")
	}
}

func TestDeleteNotificationRecipientRemovesExistingRecipient(t *testing.T) {
	t.Parallel()

	repo := &fakeNotificationRecipientRepo{
		recipients: []generated.NotificationRecipient{{ID: 7}},
	}
	service := &NotificationRecipientService{
		notificationRecipientRepo: repo,
		projectRepo:               &fakeNotificationProjectRepo{},
	}

	if err := service.Delete(context.Background(), 7); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if len(repo.recipients) != 0 {
		t.Fatalf("len(recipients) = %d, want 0", len(repo.recipients))
	}
}

func TestDeleteNotificationRecipientMissingReturnsNotFound(t *testing.T) {
	t.Parallel()

	service := &NotificationRecipientService{
		notificationRecipientRepo: &fakeNotificationRecipientRepo{},
		projectRepo:               &fakeNotificationProjectRepo{},
	}

	err := service.Delete(context.Background(), 99)
	if !errors.Is(err, types.ErrNotificationRecipientNotFound) {
		t.Fatalf("Delete() error = %v, want ErrNotificationRecipientNotFound", err)
	}
}

func TestGetNotificationRecipientByIDReturnsMappedRecipient(t *testing.T) {
	t.Parallel()

	service := &NotificationRecipientService{
		notificationRecipientRepo: &fakeNotificationRecipientRepo{
			recipients: []generated.NotificationRecipient{
				{
					ID:          7,
					ProjectID:   1,
					ChannelType: types.NotificationChannelTypeEmail,
					Target:      "ops@example.com",
					MinSeverity: types.AlertSeverityWarning,
					IsEnabled:   true,
				},
			},
		},
		projectRepo: &fakeNotificationProjectRepo{},
	}

	recipient, err := service.GetByID(context.Background(), 7)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if recipient.ID != 7 || recipient.Target != "ops@example.com" {
		t.Fatalf("unexpected recipient: %#v", recipient)
	}
	if recipient.MinSeverity != types.AlertSeverityWarning {
		t.Fatalf("recipient.MinSeverity = %q, want %q", recipient.MinSeverity, types.AlertSeverityWarning)
	}
}

func TestSendNotificationRecipientTestEmailSendsToExistingRecipient(t *testing.T) {
	t.Parallel()

	emailService := &fakeNotificationRecipientEmailService{}
	service := &NotificationRecipientService{
		notificationRecipientRepo: &fakeNotificationRecipientRepo{
			recipients: []generated.NotificationRecipient{
				{
					ID:          7,
					ProjectID:   1,
					ChannelType: types.NotificationChannelTypeEmail,
					Target:      "ops@example.com",
					IsEnabled:   false,
				},
			},
		},
		projectRepo: &fakeNotificationProjectRepo{
			projects: map[int64]generated.Project{
				1: {ID: 1, Name: "Agrafa"},
			},
		},
		emailService: emailService,
	}

	err := service.SendTestEmail(context.Background(), 1, "Ops@example.com")
	if err != nil {
		t.Fatalf("SendTestEmail() error = %v", err)
	}
	if len(emailService.recipients) != 1 || emailService.recipients[0] != "ops@example.com" {
		t.Fatalf("recipients = %#v", emailService.recipients)
	}
	if emailService.lastData.ProjectName != "Agrafa" {
		t.Fatalf("lastData.ProjectName = %q", emailService.lastData.ProjectName)
	}
	if emailService.lastData.Recipient != "ops@example.com" {
		t.Fatalf("lastData.Recipient = %q", emailService.lastData.Recipient)
	}
}

func TestSendNotificationRecipientTestEmailRejectsUnknownRecipient(t *testing.T) {
	t.Parallel()

	service := &NotificationRecipientService{
		notificationRecipientRepo: &fakeNotificationRecipientRepo{},
		projectRepo: &fakeNotificationProjectRepo{
			projects: map[int64]generated.Project{
				1: {ID: 1, Name: "Agrafa"},
			},
		},
		emailService: &fakeNotificationRecipientEmailService{},
	}

	err := service.SendTestEmail(context.Background(), 1, "ops@example.com")
	if !errors.Is(err, types.ErrNotificationRecipientNotFound) {
		t.Fatalf("SendTestEmail() error = %v, want ErrNotificationRecipientNotFound", err)
	}
}

func TestSendNotificationRecipientTestEmailRequiresConfiguredEmail(t *testing.T) {
	t.Parallel()

	service := &NotificationRecipientService{
		notificationRecipientRepo: &fakeNotificationRecipientRepo{},
		projectRepo: &fakeNotificationProjectRepo{
			projects: map[int64]generated.Project{
				1: {ID: 1, Name: "Agrafa"},
			},
		},
	}

	err := service.SendTestEmail(context.Background(), 1, "ops@example.com")
	if !errors.Is(err, types.ErrEmailNotConfigured) {
		t.Fatalf("SendTestEmail() error = %v, want ErrEmailNotConfigured", err)
	}
}
