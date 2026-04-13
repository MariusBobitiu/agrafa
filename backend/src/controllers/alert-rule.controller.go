package controllers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/MariusBobitiu/agrafa-backend/src/types"
	"github.com/MariusBobitiu/agrafa-backend/src/utils"
	"github.com/go-chi/chi/v5"
)

type alertRuleService interface {
	Create(ctx context.Context, input types.CreateAlertRuleInput) (types.AlertRuleReadData, error)
	GetByID(ctx context.Context, alertRuleID int64) (types.AlertRuleReadData, error)
	Update(ctx context.Context, input types.UpdateAlertRuleInput) (types.AlertRuleReadData, error)
	Delete(ctx context.Context, alertRuleID int64) error
}

type AlertRuleController struct {
	alertRuleService alertRuleService
}

func NewAlertRuleController(alertRuleService alertRuleService) *AlertRuleController {
	return &AlertRuleController{alertRuleService: alertRuleService}
}

// Create creates an alert rule.
//
// @Summary      Create alert rule
// @Description  Creates an enabled alert rule for a project, node, or service target.
// @Tags         alerts
// @Accept       json
// @Produce      json
// @Param        request  body      types.AlertRuleCreateRequest  true  "Alert rule payload"
// @Success      201      {object}  types.AlertRuleResponse
// @Failure      400      {object}  types.ErrorResponse
// @Failure      404      {object}  types.ErrorResponse
// @Failure      409      {object}  types.ErrorResponse
// @Failure      500      {object}  types.ErrorResponse
// @Router       /alert-rules [post]
func (c *AlertRuleController) Create(w http.ResponseWriter, r *http.Request) {
	var request types.AlertRuleCreateRequest

	if err := utils.DecodeJSON(r, &request); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid alert rule payload")
		return
	}

	rule, err := c.alertRuleService.Create(r.Context(), types.CreateAlertRuleInput{
		ProjectID:      request.ProjectID,
		NodeID:         request.NodeID,
		ServiceID:      request.ServiceID,
		RuleType:       request.RuleType,
		Severity:       request.Severity,
		ThresholdValue: request.ThresholdValue,
	})
	if err != nil {
		if utils.WriteDomainError(w, err) {
			return
		}

		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusCreated, map[string]any{"alert_rule": rule})
}

// Get returns an alert rule by id.
//
// @Summary      Get alert rule
// @Description  Returns alert rule details for a rule the current user can read.
// @Tags         alerts
// @Produce      json
// @Param        id   path      int  true  "Alert rule ID"
// @Success      200  {object}  types.AlertRuleResponse
// @Failure      400  {object}  types.ErrorResponse
// @Failure      401  {object}  types.ErrorResponse
// @Failure      403  {object}  types.ErrorResponse
// @Failure      404  {object}  types.ErrorResponse
// @Failure      500  {object}  types.ErrorResponse
// @Router       /alert-rules/{id} [get]
func (c *AlertRuleController) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		utils.WriteError(w, http.StatusBadRequest, "id must be a positive integer")
		return
	}

	rule, err := c.alertRuleService.GetByID(r.Context(), id)
	if err != nil {
		if utils.WriteDomainError(w, err) {
			return
		}

		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]any{"alert_rule": rule})
}

// Update updates an alert rule.
//
// @Summary      Update alert rule
// @Description  Updates the target, threshold, severity, and enabled state of an alert rule.
// @Tags         alerts
// @Accept       json
// @Produce      json
// @Param        id       path      int                          true  "Alert rule ID"
// @Param        request  body      types.AlertRuleUpdateRequest true  "Alert rule update payload"
// @Success      200      {object}  types.AlertRuleResponse
// @Failure      400      {object}  types.ErrorResponse
// @Failure      404      {object}  types.ErrorResponse
// @Failure      500      {object}  types.ErrorResponse
// @Router       /alert-rules/{id} [patch]
func (c *AlertRuleController) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		utils.WriteError(w, http.StatusBadRequest, "id must be a positive integer")
		return
	}

	var request types.AlertRuleUpdateRequest
	if err := utils.DecodeJSON(r, &request); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid alert rule update payload")
		return
	}

	if request.NodeID == nil && request.ServiceID == nil && request.Severity == nil &&
		request.ThresholdValue == nil && request.IsEnabled == nil {
		utils.WriteError(w, http.StatusBadRequest, "at least one field must be provided")
		return
	}

	rule, err := c.alertRuleService.Update(r.Context(), types.UpdateAlertRuleInput{
		ID:             id,
		NodeID:         request.NodeID,
		ServiceID:      request.ServiceID,
		Severity:       request.Severity,
		ThresholdValue: request.ThresholdValue,
		IsEnabled:      request.IsEnabled,
	})
	if err != nil {
		if utils.WriteDomainError(w, err) {
			return
		}

		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]any{"alert_rule": rule})
}

// Delete deletes an alert rule.
//
// @Summary      Delete alert rule
// @Description  Deletes an alert rule.
// @Tags         alerts
// @Produce      json
// @Param        id  path      int  true  "Alert rule ID"
// @Success      204
// @Failure      400  {object}  types.ErrorResponse
// @Failure      401  {object}  types.ErrorResponse
// @Failure      403  {object}  types.ErrorResponse
// @Failure      404  {object}  types.ErrorResponse
// @Failure      500  {object}  types.ErrorResponse
// @Router       /alert-rules/{id} [delete]
func (c *AlertRuleController) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		utils.WriteError(w, http.StatusBadRequest, "id must be a positive integer")
		return
	}

	if err := c.alertRuleService.Delete(r.Context(), id); err != nil {
		if utils.WriteDomainError(w, err) {
			return
		}

		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
