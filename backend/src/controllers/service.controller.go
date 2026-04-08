package controllers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
	"github.com/MariusBobitiu/agrafa-backend/src/utils"
	"github.com/go-chi/chi/v5"
)

type serviceWriter interface {
	Create(ctx context.Context, input types.CreateServiceInput) (generated.Service, error)
	Update(ctx context.Context, serviceID int64, input types.UpdateServiceInput) (generated.Service, error)
	Delete(ctx context.Context, serviceID int64) error
}

type serviceReader interface {
	GetByID(ctx context.Context, serviceID int64) (types.ServiceDetailData, error)
}

type ServiceController struct {
	serviceService     serviceWriter
	serviceReadService serviceReader
}

func NewServiceController(serviceService serviceWriter, serviceReadService serviceReader) *ServiceController {
	return &ServiceController{
		serviceService:     serviceService,
		serviceReadService: serviceReadService,
	}
}

// Create creates a service.
//
// @Summary      Create service
// @Description  Creates a service under a project and resolves the execution node from execution_mode.
// @Tags         inventory
// @Accept       json
// @Produce      json
// @Param        request  body      types.ServiceCreateRequest  true  "Service payload"
// @Success      201      {object}  types.ServiceDetailResponse
// @Failure      400      {object}  types.ErrorResponse
// @Failure      404      {object}  types.ErrorResponse
// @Failure      409      {object}  types.ErrorResponse
// @Failure      500      {object}  types.ErrorResponse
// @Router       /services [post]
func (c *ServiceController) Create(w http.ResponseWriter, r *http.Request) {
	var request types.ServiceCreateRequest

	if err := utils.DecodeJSON(r, &request); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid service payload")
		return
	}

	service, err := c.serviceService.Create(r.Context(), types.CreateServiceInput{
		ProjectID:     request.ProjectID,
		NodeID:        request.NodeID,
		ExecutionMode: request.ExecutionMode,
		Name:          request.Name,
		CheckType:     request.CheckType,
		CheckTarget:   request.CheckTarget,
	})
	if err != nil {
		if utils.WriteDomainError(w, err) {
			return
		}

		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	item, err := c.serviceReadService.GetByID(r.Context(), service.ID)
	if err != nil {
		if utils.WriteDomainError(w, err) {
			return
		}

		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusCreated, map[string]any{"service": item})
}

// Get returns a service by id.
//
// @Summary      Get service
// @Description  Returns service details for a service the current user can read.
// @Tags         inventory
// @Produce      json
// @Param        id   path      int  true  "Service ID"
// @Success      200  {object}  types.ServiceDetailResponse
// @Failure      400  {object}  types.ErrorResponse
// @Failure      401  {object}  types.ErrorResponse
// @Failure      403  {object}  types.ErrorResponse
// @Failure      404  {object}  types.ErrorResponse
// @Failure      500  {object}  types.ErrorResponse
// @Router       /services/{id} [get]
func (c *ServiceController) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		utils.WriteError(w, http.StatusBadRequest, "id must be a positive integer")
		return
	}

	service, err := c.serviceReadService.GetByID(r.Context(), id)
	if err != nil {
		if utils.WriteDomainError(w, err) {
			return
		}

		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]any{"service": service})
}

// Update updates a service.
//
// @Summary      Update service
// @Description  Updates the editable service definition fields only. Execution mode and attached node are unchanged.
// @Tags         inventory
// @Accept       json
// @Produce      json
// @Param        id       path      int                         true  "Service ID"
// @Param        request  body      types.ServiceUpdateRequest  true  "Service update payload"
// @Success      200      {object}  types.ServiceDetailResponse
// @Failure      400      {object}  types.ErrorResponse
// @Failure      401      {object}  types.ErrorResponse
// @Failure      403      {object}  types.ErrorResponse
// @Failure      404      {object}  types.ErrorResponse
// @Failure      500      {object}  types.ErrorResponse
// @Router       /services/{id} [patch]
func (c *ServiceController) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		utils.WriteError(w, http.StatusBadRequest, "id must be a positive integer")
		return
	}

	var request types.ServiceUpdateRequest
	if err := utils.DecodeJSON(r, &request); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid service update payload")
		return
	}

	service, err := c.serviceService.Update(r.Context(), id, types.UpdateServiceInput{
		Name:        request.Name,
		CheckType:   request.CheckType,
		CheckTarget: request.CheckTarget,
	})
	if err != nil {
		if utils.WriteDomainError(w, err) {
			return
		}

		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	item, err := c.serviceReadService.GetByID(r.Context(), service.ID)
	if err != nil {
		if utils.WriteDomainError(w, err) {
			return
		}

		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]any{"service": item})
}

// Delete deletes a service.
//
// @Summary      Delete service
// @Description  Deletes a service and relies on existing schema cleanup for dependent operational records.
// @Tags         inventory
// @Produce      json
// @Param        id  path      int  true  "Service ID"
// @Success      204
// @Failure      400  {object}  types.ErrorResponse
// @Failure      401  {object}  types.ErrorResponse
// @Failure      403  {object}  types.ErrorResponse
// @Failure      404  {object}  types.ErrorResponse
// @Failure      500  {object}  types.ErrorResponse
// @Router       /services/{id} [delete]
func (c *ServiceController) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		utils.WriteError(w, http.StatusBadRequest, "id must be a positive integer")
		return
	}

	if err := c.serviceService.Delete(r.Context(), id); err != nil {
		if utils.WriteDomainError(w, err) {
			return
		}

		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
