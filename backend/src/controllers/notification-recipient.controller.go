package controllers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/MariusBobitiu/agrafa-backend/src/types"
	"github.com/MariusBobitiu/agrafa-backend/src/utils"
	"github.com/go-chi/chi/v5"
)

type notificationRecipientService interface {
	Create(ctx context.Context, input types.CreateNotificationRecipientsInput) ([]types.NotificationRecipientReadData, error)
	List(ctx context.Context, projectID *int64) ([]types.NotificationRecipientReadData, error)
	GetByID(ctx context.Context, notificationRecipientID int64) (types.NotificationRecipientReadData, error)
	SetEnabled(ctx context.Context, input types.UpdateNotificationRecipientInput) (types.NotificationRecipientReadData, error)
	Delete(ctx context.Context, notificationRecipientID int64) error
	SendTestEmail(ctx context.Context, projectID int64, email string) error
}

type NotificationRecipientController struct {
	notificationRecipientService notificationRecipientService
}

func NewNotificationRecipientController(notificationRecipientService notificationRecipientService) *NotificationRecipientController {
	return &NotificationRecipientController{notificationRecipientService: notificationRecipientService}
}

// Create creates a notification recipient.
//
// @Summary      Create notification recipient
// @Description  Creates a project-scoped notification recipient. Only email is supported for now.
// @Tags         notifications
// @Accept       json
// @Produce      json
// @Param        request  body      types.NotificationRecipientCreateRequest  true  "Notification recipient payload"
// @Success      201      {object}  types.NotificationRecipientsResponse
// @Failure      400      {object}  types.ErrorResponse
// @Failure      404      {object}  types.ErrorResponse
// @Failure      409      {object}  types.ErrorResponse
// @Failure      500      {object}  types.ErrorResponse
// @Router       /notification-recipients [post]
func (c *NotificationRecipientController) Create(w http.ResponseWriter, r *http.Request) {
	var request types.NotificationRecipientCreateRequest
	if err := utils.DecodeJSON(r, &request); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid notification recipient payload")
		return
	}

	recipientsInput := make([]types.CreateNotificationRecipientItemInput, 0, len(request.Recipients))
	for _, recipient := range request.Recipients {
		recipientsInput = append(recipientsInput, types.CreateNotificationRecipientItemInput{
			Target:      recipient.Target,
			MinSeverity: recipient.MinSeverity,
		})
	}

	recipients, err := c.notificationRecipientService.Create(r.Context(), types.CreateNotificationRecipientsInput{
		ProjectID:   request.ProjectID,
		ChannelType: request.ChannelType,
		Recipients:  recipientsInput,
	})
	if err != nil {
		if utils.WriteDomainError(w, err) {
			return
		}

		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusCreated, map[string]any{"notification_recipients": recipients})
}

// List lists notification recipients.
//
// @Summary      List notification recipients
// @Description  Returns project notification recipients ordered by newest first. Optionally filters by project_id.
// @Tags         notifications
// @Produce      json
// @Param        project_id  query     int  false  "Project ID"
// @Success      200         {object}  types.NotificationRecipientsResponse
// @Failure      400         {object}  types.ErrorResponse
// @Failure      500         {object}  types.ErrorResponse
// @Router       /notification-recipients [get]
func (c *NotificationRecipientController) List(w http.ResponseWriter, r *http.Request) {
	projectID, err := utils.ParseOptionalPositiveInt64Query(r, "project_id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	recipients, err := c.notificationRecipientService.List(r.Context(), projectID)
	if err != nil {
		if utils.WriteDomainError(w, err) {
			return
		}

		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]any{"notification_recipients": recipients})
}

// Get returns a notification recipient by id.
//
// @Summary      Get notification recipient
// @Description  Returns notification recipient details for a recipient the current user can read.
// @Tags         notifications
// @Produce      json
// @Param        id   path      int  true  "Notification recipient ID"
// @Success      200  {object}  types.NotificationRecipientResponse
// @Failure      400  {object}  types.ErrorResponse
// @Failure      401  {object}  types.ErrorResponse
// @Failure      403  {object}  types.ErrorResponse
// @Failure      404  {object}  types.ErrorResponse
// @Failure      500  {object}  types.ErrorResponse
// @Router       /notification-recipients/{id} [get]
func (c *NotificationRecipientController) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		utils.WriteError(w, http.StatusBadRequest, "id must be a positive integer")
		return
	}

	recipient, err := c.notificationRecipientService.GetByID(r.Context(), id)
	if err != nil {
		if utils.WriteDomainError(w, err) {
			return
		}

		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]any{"notification_recipient": recipient})
}

// Update updates a notification recipient.
//
// @Summary      Update notification recipient
// @Description  Updates the enabled state of a notification recipient.
// @Tags         notifications
// @Accept       json
// @Produce      json
// @Param        id       path      int                                     true  "Notification recipient ID"
// @Param        request  body      types.NotificationRecipientUpdateRequest true  "Notification recipient update payload"
// @Success      200      {object}  types.NotificationRecipientResponse
// @Failure      400      {object}  types.ErrorResponse
// @Failure      404      {object}  types.ErrorResponse
// @Failure      500      {object}  types.ErrorResponse
// @Router       /notification-recipients/{id} [patch]
func (c *NotificationRecipientController) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		utils.WriteError(w, http.StatusBadRequest, "id must be a positive integer")
		return
	}

	var request types.NotificationRecipientUpdateRequest
	if err := utils.DecodeJSON(r, &request); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid notification recipient update payload")
		return
	}

	if request.IsEnabled == nil {
		utils.WriteError(w, http.StatusBadRequest, "is_enabled is required")
		return
	}

	recipient, err := c.notificationRecipientService.SetEnabled(r.Context(), types.UpdateNotificationRecipientInput{
		ID:        id,
		IsEnabled: *request.IsEnabled,
	})
	if err != nil {
		if utils.WriteDomainError(w, err) {
			return
		}

		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]any{"notification_recipient": recipient})
}

// Delete deletes a notification recipient.
//
// @Summary      Delete notification recipient
// @Description  Deletes a notification recipient.
// @Tags         notifications
// @Produce      json
// @Param        id  path      int  true  "Notification recipient ID"
// @Success      204
// @Failure      400  {object}  types.ErrorResponse
// @Failure      401  {object}  types.ErrorResponse
// @Failure      403  {object}  types.ErrorResponse
// @Failure      404  {object}  types.ErrorResponse
// @Failure      500  {object}  types.ErrorResponse
// @Router       /notification-recipients/{id} [delete]
func (c *NotificationRecipientController) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		utils.WriteError(w, http.StatusBadRequest, "id must be a positive integer")
		return
	}

	if err := c.notificationRecipientService.Delete(r.Context(), id); err != nil {
		if utils.WriteDomainError(w, err) {
			return
		}

		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// SendTestEmail sends a test email to an existing notification recipient.
//
// @Summary      Send notification recipient test email
// @Description  Verifies that the provided email exists as a project notification recipient before sending a test email.
// @Tags         notifications
// @Accept       json
// @Produce      json
// @Param        request  body      types.NotificationRecipientTestEmailRequest  true  "Notification recipient test email payload"
// @Success      200      {object}  types.AuthLogoutResponse
// @Failure      400      {object}  types.ErrorResponse
// @Failure      404      {object}  types.ErrorResponse
// @Failure      409      {object}  types.ErrorResponse
// @Failure      500      {object}  types.ErrorResponse
// @Router       /notification-recipients/test-email [post]
func (c *NotificationRecipientController) SendTestEmail(w http.ResponseWriter, r *http.Request) {
	var request types.NotificationRecipientTestEmailRequest
	if err := utils.DecodeJSON(r, &request); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid notification recipient test email payload")
		return
	}

	if err := c.notificationRecipientService.SendTestEmail(r.Context(), request.ProjectID, request.Email); err != nil {
		if utils.WriteDomainError(w, err) {
			return
		}

		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, types.AuthLogoutResponse{Status: "ok"})
}
