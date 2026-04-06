package controllers

import (
	"context"
	"net/http"
	"strings"

	"github.com/MariusBobitiu/agrafa-backend/src/types"
	"github.com/MariusBobitiu/agrafa-backend/src/utils"
	"github.com/go-chi/chi/v5"
)

type projectMemberService interface {
	List(ctx context.Context, projectID int64) ([]types.ProjectMemberReadData, error)
	GetByID(ctx context.Context, id string) (types.ProjectMemberReadData, error)
	Create(ctx context.Context, input types.CreateProjectMemberInput) (types.ProjectMemberReadData, error)
	UpdateRole(ctx context.Context, input types.UpdateProjectMemberInput) (types.ProjectMemberReadData, error)
	Delete(ctx context.Context, id string) error
}

type ProjectMemberController struct {
	projectMemberService projectMemberService
}

func NewProjectMemberController(projectMemberService projectMemberService) *ProjectMemberController {
	return &ProjectMemberController{projectMemberService: projectMemberService}
}

// List lists project members.
//
// @Summary      List project members
// @Description  Returns project members for a project, including joined user summary data.
// @Tags         project-members
// @Produce      json
// @Param        project_id  query     int  true  "Project ID"
// @Success      200         {object}  types.ProjectMembersResponse
// @Failure      400         {object}  types.ErrorResponse
// @Failure      401         {object}  types.ErrorResponse
// @Failure      403         {object}  types.ErrorResponse
// @Failure      500         {object}  types.ErrorResponse
// @Router       /project-members [get]
func (c *ProjectMemberController) List(w http.ResponseWriter, r *http.Request) {
	projectID, err := utils.ParseOptionalPositiveInt64Query(r, "project_id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if projectID == nil {
		utils.WriteError(w, http.StatusBadRequest, "project_id is required")
		return
	}

	projectMembers, err := c.projectMemberService.List(r.Context(), *projectID)
	if err != nil {
		if utils.WriteDomainError(w, err) {
			return
		}

		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]any{"project_members": projectMembers})
}

// Get returns a project member by id.
//
// @Summary      Get project member
// @Description  Returns a project membership with joined user summary data.
// @Tags         project-members
// @Produce      json
// @Param        id   path      string  true  "Project member ID"
// @Success      200  {object}  types.ProjectMemberResponse
// @Failure      400  {object}  types.ErrorResponse
// @Failure      401  {object}  types.ErrorResponse
// @Failure      403  {object}  types.ErrorResponse
// @Failure      404  {object}  types.ErrorResponse
// @Failure      500  {object}  types.ErrorResponse
// @Router       /project-members/{id} [get]
func (c *ProjectMemberController) Get(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "id is required")
		return
	}

	projectMember, err := c.projectMemberService.GetByID(r.Context(), id)
	if err != nil {
		if utils.WriteDomainError(w, err) {
			return
		}

		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]any{"project_member": projectMember})
}

// Create creates a project member.
//
// @Summary      Create project member
// @Description  Adds a user to a project with the requested role.
// @Tags         project-members
// @Accept       json
// @Produce      json
// @Param        request  body      types.ProjectMemberCreateRequest  true  "Project member payload"
// @Success      201      {object}  types.ProjectMemberResponse
// @Failure      400      {object}  types.ErrorResponse
// @Failure      401      {object}  types.ErrorResponse
// @Failure      403      {object}  types.ErrorResponse
// @Failure      404      {object}  types.ErrorResponse
// @Failure      409      {object}  types.ErrorResponse
// @Failure      500      {object}  types.ErrorResponse
// @Router       /project-members [post]
func (c *ProjectMemberController) Create(w http.ResponseWriter, r *http.Request) {
	var request types.ProjectMemberCreateRequest
	if err := utils.DecodeJSON(r, &request); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid project member payload")
		return
	}

	projectMember, err := c.projectMemberService.Create(r.Context(), types.CreateProjectMemberInput{
		ProjectID: request.ProjectID,
		UserID:    request.UserID,
		Role:      request.Role,
	})
	if err != nil {
		if utils.WriteDomainError(w, err) {
			return
		}

		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusCreated, map[string]any{"project_member": projectMember})
}

// Update updates a project member role.
//
// @Summary      Update project member
// @Description  Updates the role of an existing project membership.
// @Tags         project-members
// @Accept       json
// @Produce      json
// @Param        id       path      string                          true  "Project member ID"
// @Param        request  body      types.ProjectMemberUpdateRequest true  "Project member update payload"
// @Success      200      {object}  types.ProjectMemberResponse
// @Failure      400      {object}  types.ErrorResponse
// @Failure      401      {object}  types.ErrorResponse
// @Failure      403      {object}  types.ErrorResponse
// @Failure      404      {object}  types.ErrorResponse
// @Failure      500      {object}  types.ErrorResponse
// @Router       /project-members/{id} [patch]
func (c *ProjectMemberController) Update(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "id is required")
		return
	}

	var request types.ProjectMemberUpdateRequest
	if err := utils.DecodeJSON(r, &request); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid project member update payload")
		return
	}

	projectMember, err := c.projectMemberService.UpdateRole(r.Context(), types.UpdateProjectMemberInput{
		ID:   id,
		Role: request.Role,
	})
	if err != nil {
		if utils.WriteDomainError(w, err) {
			return
		}

		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]any{"project_member": projectMember})
}

// Delete removes a project member.
//
// @Summary      Delete project member
// @Description  Deletes an existing project membership when owner safety rules allow it.
// @Tags         project-members
// @Produce      json
// @Param        id  path      string  true  "Project member ID"
// @Success      204
// @Failure      400  {object}  types.ErrorResponse
// @Failure      401  {object}  types.ErrorResponse
// @Failure      403  {object}  types.ErrorResponse
// @Failure      404  {object}  types.ErrorResponse
// @Failure      500  {object}  types.ErrorResponse
// @Router       /project-members/{id} [delete]
func (c *ProjectMemberController) Delete(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "id is required")
		return
	}

	if err := c.projectMemberService.Delete(r.Context(), id); err != nil {
		if utils.WriteDomainError(w, err) {
			return
		}

		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
