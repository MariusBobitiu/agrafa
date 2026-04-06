package controllers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	authmiddleware "github.com/MariusBobitiu/agrafa-backend/src/middleware"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
	"github.com/MariusBobitiu/agrafa-backend/src/utils"
	"github.com/go-chi/chi/v5"
)

type projectService interface {
	Create(ctx context.Context, userID string, name string) (generated.Project, error)
	ListForUser(ctx context.Context, userID string) ([]types.ProjectSummaryData, error)
	Get(ctx context.Context, userID string, projectID int64) (types.ProjectDetailData, error)
	Update(ctx context.Context, userID string, projectID int64, input types.UpdateProjectInput) (types.ProjectDetailData, error)
	Delete(ctx context.Context, projectID int64) error
}

type ProjectController struct {
	projectService projectService
}

func NewProjectController(projectService projectService) *ProjectController {
	return &ProjectController{projectService: projectService}
}

// Create creates a project.
//
// @Summary      Create project
// @Description  Creates a project inventory record.
// @Tags         inventory
// @Accept       json
// @Produce      json
// @Param        request  body      types.ProjectCreateRequest  true  "Project payload"
// @Success      201      {object}  types.ProjectResponse
// @Failure      400      {object}  types.ErrorResponse
// @Failure      409      {object}  types.ErrorResponse
// @Failure      500      {object}  types.ErrorResponse
// @Router       /projects [post]
func (c *ProjectController) Create(w http.ResponseWriter, r *http.Request) {
	var request types.ProjectCreateRequest

	if err := utils.DecodeJSON(r, &request); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid project payload")
		return
	}

	user, ok := authmiddleware.AuthenticatedUser(r.Context())
	if !ok {
		utils.WriteError(w, http.StatusInternalServerError, "authenticated user missing from context")
		return
	}

	project, err := c.projectService.Create(r.Context(), user.ID, request.Name)
	if err != nil {
		if utils.WriteDomainError(w, err) {
			return
		}

		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusCreated, map[string]any{"project": project})
}

// List lists projects for the authenticated user.
//
// @Summary      List projects
// @Description  Returns only projects the current authenticated user belongs to.
// @Tags         inventory
// @Produce      json
// @Success      200  {object}  types.ProjectsResponse
// @Failure      401  {object}  types.ErrorResponse
// @Failure      500  {object}  types.ErrorResponse
// @Router       /projects [get]
func (c *ProjectController) List(w http.ResponseWriter, r *http.Request) {
	user, ok := authmiddleware.AuthenticatedUser(r.Context())
	if !ok {
		utils.WriteError(w, http.StatusInternalServerError, "authenticated user missing from context")
		return
	}

	projects, err := c.projectService.ListForUser(r.Context(), user.ID)
	if err != nil {
		if utils.WriteDomainError(w, err) {
			return
		}

		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]any{"projects": projects})
}

// Get returns a project by id.
//
// @Summary      Get project
// @Description  Returns project details for a project the current user can read.
// @Tags         inventory
// @Produce      json
// @Param        id   path      int  true  "Project ID"
// @Success      200  {object}  types.ProjectDetailResponse
// @Failure      400  {object}  types.ErrorResponse
// @Failure      401  {object}  types.ErrorResponse
// @Failure      403  {object}  types.ErrorResponse
// @Failure      404  {object}  types.ErrorResponse
// @Failure      500  {object}  types.ErrorResponse
// @Router       /projects/{id} [get]
func (c *ProjectController) Get(w http.ResponseWriter, r *http.Request) {
	projectID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || projectID <= 0 {
		utils.WriteError(w, http.StatusBadRequest, "id must be a positive integer")
		return
	}

	user, ok := authmiddleware.AuthenticatedUser(r.Context())
	if !ok {
		utils.WriteError(w, http.StatusInternalServerError, "authenticated user missing from context")
		return
	}

	project, err := c.projectService.Get(r.Context(), user.ID, projectID)
	if err != nil {
		if utils.WriteDomainError(w, err) {
			return
		}

		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]any{"project": project})
}

// Update updates a project.
//
// @Summary      Update project
// @Description  Updates the project name. Ownership and membership are not changed here.
// @Tags         inventory
// @Accept       json
// @Produce      json
// @Param        id       path      int                         true  "Project ID"
// @Param        request  body      types.ProjectUpdateRequest  true  "Project update payload"
// @Success      200      {object}  types.ProjectDetailResponse
// @Failure      400      {object}  types.ErrorResponse
// @Failure      401      {object}  types.ErrorResponse
// @Failure      403      {object}  types.ErrorResponse
// @Failure      404      {object}  types.ErrorResponse
// @Failure      500      {object}  types.ErrorResponse
// @Router       /projects/{id} [patch]
func (c *ProjectController) Update(w http.ResponseWriter, r *http.Request) {
	projectID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || projectID <= 0 {
		utils.WriteError(w, http.StatusBadRequest, "id must be a positive integer")
		return
	}

	var request types.ProjectUpdateRequest
	if err := utils.DecodeJSON(r, &request); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid project update payload")
		return
	}

	user, ok := authmiddleware.AuthenticatedUser(r.Context())
	if !ok {
		utils.WriteError(w, http.StatusInternalServerError, "authenticated user missing from context")
		return
	}

	project, err := c.projectService.Update(r.Context(), user.ID, projectID, types.UpdateProjectInput{
		Name: request.Name,
	})
	if err != nil {
		if utils.WriteDomainError(w, err) {
			return
		}

		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]any{"project": project})
}

// Delete deletes a project.
//
// @Summary      Delete project
// @Description  Deletes a project and dependent application resources.
// @Tags         inventory
// @Produce      json
// @Param        id  path      int  true  "Project ID"
// @Success      204
// @Failure      400  {object}  types.ErrorResponse
// @Failure      401  {object}  types.ErrorResponse
// @Failure      403  {object}  types.ErrorResponse
// @Failure      404  {object}  types.ErrorResponse
// @Failure      500  {object}  types.ErrorResponse
// @Router       /projects/{id} [delete]
func (c *ProjectController) Delete(w http.ResponseWriter, r *http.Request) {
	projectID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || projectID <= 0 {
		utils.WriteError(w, http.StatusBadRequest, "id must be a positive integer")
		return
	}

	if err := c.projectService.Delete(r.Context(), projectID); err != nil {
		if utils.WriteDomainError(w, err) {
			return
		}

		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
