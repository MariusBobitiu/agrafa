package controllers

import (
	"context"
	"net/http"

	"github.com/MariusBobitiu/agrafa-backend/src/types"
	"github.com/MariusBobitiu/agrafa-backend/src/utils"
)

type instanceSettingService interface {
	ListForUI(ctx context.Context) ([]types.InstanceSettingReadData, error)
	UpdateBatchForUI(ctx context.Context, updates []types.InstanceSettingsUpdateItemRequest) ([]types.InstanceSettingReadData, error)
}

type InstanceSettingController struct {
	instanceSettingService instanceSettingService
}

func NewInstanceSettingController(instanceSettingService instanceSettingService) *InstanceSettingController {
	return &InstanceSettingController{instanceSettingService: instanceSettingService}
}

// List returns DB-capable instance settings for the Instance Settings UI.
//
// @Summary      List instance settings
// @Description  Returns email-related DB-capable instance settings with UI metadata, masked sensitive values, and env override indicators.
// @Tags         settings
// @Produce      json
// @Success      200  {object}  types.InstanceSettingsResponse
// @Failure      400  {object}  types.ErrorResponse
// @Failure      500  {object}  types.ErrorResponse
// @Router       /instance-settings [get]
func (c *InstanceSettingController) List(w http.ResponseWriter, r *http.Request) {
	settings, err := c.instanceSettingService.ListForUI(r.Context())
	if err != nil {
		if utils.WriteDomainError(w, err) {
			return
		}

		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]any{"settings": settings})
}

// Update batch-upserts DB-capable instance settings for the Instance Settings UI.
//
// @Summary      Update instance settings
// @Description  Validates and upserts email-related DB-capable instance settings. Sensitive values are encrypted before storage.
// @Tags         settings
// @Accept       json
// @Produce      json
// @Param        request  body      types.InstanceSettingsUpdateRequest  true  "Instance settings update payload"
// @Success      200      {object}  types.InstanceSettingsResponse
// @Failure      400      {object}  types.ErrorResponse
// @Failure      500      {object}  types.ErrorResponse
// @Router       /instance-settings [patch]
func (c *InstanceSettingController) Update(w http.ResponseWriter, r *http.Request) {
	var request types.InstanceSettingsUpdateRequest
	if err := utils.DecodeJSON(r, &request); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid instance settings payload")
		return
	}

	settings, err := c.instanceSettingService.UpdateBatchForUI(r.Context(), request.Settings)
	if err != nil {
		if utils.WriteDomainError(w, err) {
			return
		}

		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]any{"settings": settings})
}
