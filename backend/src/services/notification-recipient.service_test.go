package services

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
)

type fakeNotificationRecipientRepo struct {
	recipients      []generated.NotificationRecipient
	nextID          int64
	lastCreateInput generated.CreateNotificationRecipientParams
}

func (r *fakeNotificationRecipientRepo) Create(_ context.Context, params generated.CreateNotificationRecipientParams) (generated.NotificationRecipient, error) {
	r.nextID++
	r.lastCreateInput = params

	recipient := generated.NotificationRecipient{
		ID:          r.nextID,
		ProjectID:   params.ProjectID,
		ChannelType: params.ChannelType,
		Target:      params.Target,
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

func TestCreateNotificationRecipientAcceptsValidEmail(t *testing.T) {
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

	recipient, err := service.Create(context.Background(), types.CreateNotificationRecipientInput{
		ProjectID:   1,
		ChannelType: types.NotificationChannelTypeEmail,
		Target:      "Ops@Example.com",
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if recipient.Target != "ops@example.com" {
		t.Fatalf("expected lower-cased target, got %q", recipient.Target)
	}

	if repo.lastCreateInput.Target != "ops@example.com" {
		t.Fatalf("expected stored target to be normalized, got %q", repo.lastCreateInput.Target)
	}
}

func TestCreateNotificationRecipientRejectsInvalidEmail(t *testing.T) {
	t.Parallel()

	service := &NotificationRecipientService{
		notificationRecipientRepo: &fakeNotificationRecipientRepo{},
		projectRepo: &fakeNotificationProjectRepo{
			projects: map[int64]generated.Project{
				1: {ID: 1, Name: "Agrafa"},
			},
		},
	}

	_, err := service.Create(context.Background(), types.CreateNotificationRecipientInput{
		ProjectID:   1,
		ChannelType: types.NotificationChannelTypeEmail,
		Target:      "not-an-email",
	})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}

	if err != types.ErrInvalidNotificationTarget {
		t.Fatalf("expected ErrInvalidNotificationTarget, got %v", err)
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
}
