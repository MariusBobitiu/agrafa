package controllers

import (
	"net/http"
	"strconv"

	"github.com/MariusBobitiu/agrafa-backend/src/services"
	"github.com/MariusBobitiu/agrafa-backend/src/utils"
)

type NotificationDeliveryController struct {
	notificationDeliveryService *services.NotificationDeliveryService
}

func NewNotificationDeliveryController(notificationDeliveryService *services.NotificationDeliveryService) *NotificationDeliveryController {
	return &NotificationDeliveryController{notificationDeliveryService: notificationDeliveryService}
}

// List lists notification deliveries.
//
// @Summary      List notification deliveries
// @Description  Returns notification delivery attempts ordered by newest first, with optional project, status, and limit filters.
// @Tags         notifications
// @Produce      json
// @Param        project_id  query     int     false  "Project ID"
// @Param        status      query     string  false  "Delivery status"  Enums(sent, failed)
// @Param        limit       query     int     false  "Maximum number of deliveries"  minimum(1)  default(50)
// @Success      200         {object}  types.NotificationDeliveriesResponse
// @Failure      400         {object}  types.ErrorResponse
// @Failure      500         {object}  types.ErrorResponse
// @Router       /notification-deliveries [get]
func (c *NotificationDeliveryController) List(w http.ResponseWriter, r *http.Request) {
	projectID, err := utils.ParseOptionalPositiveInt64Query(r, "project_id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	var status *string
	if rawStatus := r.URL.Query().Get("status"); rawStatus != "" {
		status = &rawStatus
	}

	limit := int32(50)
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsed, err := strconv.ParseInt(rawLimit, 10, 32)
		if err != nil || parsed <= 0 {
			utils.WriteError(w, http.StatusBadRequest, "limit must be a positive integer")
			return
		}

		limit = int32(parsed)
	}

	deliveries, err := c.notificationDeliveryService.List(r.Context(), projectID, status, limit)
	if err != nil {
		if utils.WriteDomainError(w, err) {
			return
		}

		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]any{"notification_deliveries": deliveries})
}
