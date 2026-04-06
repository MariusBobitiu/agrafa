package controllers

import (
	"context"
	"net/http"
	"strings"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	authmiddleware "github.com/MariusBobitiu/agrafa-backend/src/middleware"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
	"github.com/MariusBobitiu/agrafa-backend/src/utils"
	"github.com/go-chi/chi/v5"
)

type projectInvitationService interface {
	Create(ctx context.Context, input types.CreateProjectInvitationInput) (types.ProjectInvitationReadData, error)
	CreateMany(ctx context.Context, inputs []types.CreateProjectInvitationInput) (types.ProjectInvitationCreateBatchData, error)
	List(ctx context.Context, projectID int64) ([]types.ProjectInvitationReadData, error)
	GetByToken(ctx context.Context, rawToken string) (types.ProjectInvitationLookupData, error)
	Accept(ctx context.Context, rawToken string, user generated.User) (bool, error)
	Delete(ctx context.Context, id string) error
}

type ProjectInvitationController struct {
	projectInvitationService projectInvitationService
}

func NewProjectInvitationController(projectInvitationService projectInvitationService) *ProjectInvitationController {
	return &ProjectInvitationController{projectInvitationService: projectInvitationService}
}

// Create creates a project invitation.
//
// @Summary      Create project invitation
// @Description  Creates a project invitation, stores a hashed invite token, and sends the invite email.
// @Tags         project-invitations
// @Accept       json
// @Produce      json
// @Param        request  body      types.ProjectInvitationCreateRequest  true  "Project invitation payload"
// @Success      200      {object}  types.ProjectInvitationCreateResponse
// @Failure      400      {object}  types.ErrorResponse
// @Failure      401      {object}  types.ErrorResponse
// @Failure      403      {object}  types.ErrorResponse
// @Failure      404      {object}  types.ErrorResponse
// @Failure      500      {object}  types.ErrorResponse
// @Router       /project-invitations [post]
func (c *ProjectInvitationController) Create(w http.ResponseWriter, r *http.Request) {
	var request types.ProjectInvitationCreateRequest
	if err := utils.DecodeJSON(r, &request); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid project invitation payload")
		return
	}

	user, ok := authmiddleware.AuthenticatedUser(r.Context())
	if !ok {
		utils.WriteError(w, http.StatusUnauthorized, types.ErrUnauthenticated.Error())
		return
	}

	inputs := make([]types.CreateProjectInvitationInput, 0, max(1, len(request.Invitations)))
	if request.Invitations != nil {
		if len(request.Invitations) == 0 {
			utils.WriteError(w, http.StatusBadRequest, types.ErrEmptyProjectInvitations.Error())
			return
		}

		for _, invitation := range request.Invitations {
			inputs = append(inputs, types.CreateProjectInvitationInput{
				ProjectID:       request.ProjectID,
				Email:           invitation.Email,
				Role:            invitation.Role,
				InvitedByUserID: user.ID,
				InvitedByName:   user.Name,
			})
		}
	} else {
		if strings.TrimSpace(request.Email) == "" || strings.TrimSpace(request.Role) == "" {
			utils.WriteError(w, http.StatusBadRequest, "single invite payload requires email and role")
			return
		}

		inputs = append(inputs, types.CreateProjectInvitationInput{
			ProjectID:       request.ProjectID,
			Email:           request.Email,
			Role:            request.Role,
			InvitedByUserID: user.ID,
			InvitedByName:   user.Name,
		})
	}

	invitations, err := c.projectInvitationService.CreateMany(r.Context(), inputs)
	if err != nil {
		if utils.WriteDomainError(w, err) {
			return
		}

		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, mapProjectInvitationCreateResponse(invitations))
}

// List lists project invitations for a project.
//
// @Summary      List project invitations
// @Description  Returns project invitations for the requested project.
// @Tags         project-invitations
// @Produce      json
// @Param        project_id  query     int  true  "Project ID"
// @Success      200         {object}  types.ProjectInvitationsResponse
// @Failure      400         {object}  types.ErrorResponse
// @Failure      401         {object}  types.ErrorResponse
// @Failure      403         {object}  types.ErrorResponse
// @Failure      500         {object}  types.ErrorResponse
// @Router       /project-invitations [get]
func (c *ProjectInvitationController) List(w http.ResponseWriter, r *http.Request) {
	projectID, err := utils.ParseOptionalPositiveInt64Query(r, "project_id")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if projectID == nil {
		utils.WriteError(w, http.StatusBadRequest, "project_id is required")
		return
	}

	invitations, err := c.projectInvitationService.List(r.Context(), *projectID)
	if err != nil {
		if utils.WriteDomainError(w, err) {
			return
		}

		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, types.ProjectInvitationsResponse{
		ProjectInvitations: mapProjectInvitationDocuments(invitations),
	})
}

// GetByToken returns safe invitation details for an invitation token.
//
// @Summary      Get project invitation by token
// @Description  Returns safe invitation details when the invite token is valid and not expired or already accepted.
// @Tags         project-invitations
// @Produce      json
// @Param        token  query     string  true  "Raw invitation token"
// @Success      200    {object}  types.ProjectInvitationLookupResponse
// @Failure      400    {object}  types.ErrorResponse
// @Failure      500    {object}  types.ErrorResponse
// @Router       /project-invitations/by-token [get]
func (c *ProjectInvitationController) GetByToken(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimSpace(r.URL.Query().Get("token"))
	if token == "" {
		utils.WriteError(w, http.StatusBadRequest, "token is required")
		return
	}

	invitation, err := c.projectInvitationService.GetByToken(r.Context(), token)
	if err != nil {
		if utils.WriteDomainError(w, err) {
			return
		}

		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, types.ProjectInvitationLookupResponse{
		ProjectInvitation: mapProjectInvitationLookupDocument(invitation),
	})
}

// Accept accepts a project invitation for the authenticated user.
//
// @Summary      Accept project invitation
// @Description  Accepts a valid invitation token for the authenticated user when the invite email matches the user email.
// @Tags         project-invitations
// @Accept       json
// @Produce      json
// @Param        request  body      types.ProjectInvitationAcceptRequest  true  "Project invitation acceptance payload"
// @Success      200      {object}  types.ProjectInvitationAcceptResponse
// @Failure      400      {object}  types.ErrorResponse
// @Failure      401      {object}  types.ErrorResponse
// @Failure      403      {object}  types.ErrorResponse
// @Failure      500      {object}  types.ErrorResponse
// @Router       /project-invitations/accept [post]
func (c *ProjectInvitationController) Accept(w http.ResponseWriter, r *http.Request) {
	user, ok := authmiddleware.AuthenticatedUser(r.Context())
	if !ok {
		utils.WriteError(w, http.StatusUnauthorized, types.ErrUnauthenticated.Error())
		return
	}

	var request types.ProjectInvitationAcceptRequest
	if err := utils.DecodeJSON(r, &request); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid project invitation accept payload")
		return
	}

	alreadyMember, err := c.projectInvitationService.Accept(r.Context(), request.Token, user)
	if err != nil {
		if utils.WriteDomainError(w, err) {
			return
		}

		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, types.ProjectInvitationAcceptResponse{
		Status:        "ok",
		AlreadyMember: alreadyMember,
	})
}

// Delete deletes a project invitation.
//
// @Summary      Delete project invitation
// @Description  Deletes a project invitation by id.
// @Tags         project-invitations
// @Produce      json
// @Param        id  path      string  true  "Project invitation ID"
// @Success      204
// @Failure      400  {object}  types.ErrorResponse
// @Failure      401  {object}  types.ErrorResponse
// @Failure      403  {object}  types.ErrorResponse
// @Failure      404  {object}  types.ErrorResponse
// @Failure      500  {object}  types.ErrorResponse
// @Router       /project-invitations/{id} [delete]
func (c *ProjectInvitationController) Delete(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "id is required")
		return
	}

	if err := c.projectInvitationService.Delete(r.Context(), id); err != nil {
		if utils.WriteDomainError(w, err) {
			return
		}

		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func mapProjectInvitationDocument(data types.ProjectInvitationReadData) types.ProjectInvitationDocument {
	return types.ProjectInvitationDocument{
		ID:              data.ID,
		ProjectID:       data.ProjectID,
		Email:           data.Email,
		Role:            data.Role,
		InvitedByUserID: data.InvitedByUserID,
		ExpiresAt:       data.ExpiresAt,
		AcceptedAt:      data.AcceptedAt,
		CreatedAt:       data.CreatedAt,
	}
}

func mapProjectInvitationDocuments(items []types.ProjectInvitationReadData) []types.ProjectInvitationDocument {
	documents := make([]types.ProjectInvitationDocument, 0, len(items))
	for _, item := range items {
		documents = append(documents, mapProjectInvitationDocument(item))
	}

	return documents
}

func mapProjectInvitationLookupDocument(data types.ProjectInvitationLookupData) types.ProjectInvitationLookupDocument {
	return types.ProjectInvitationLookupDocument{
		ID:          data.ID,
		ProjectID:   data.ProjectID,
		ProjectName: data.ProjectName,
		Email:       data.Email,
		Role:        data.Role,
		ExpiresAt:   data.ExpiresAt,
	}
}

func mapProjectInvitationCreateResponse(data types.ProjectInvitationCreateBatchData) types.ProjectInvitationCreateResponse {
	results := make([]types.ProjectInvitationCreateResultDocument, 0, len(data.Results))
	for _, item := range data.Results {
		results = append(results, mapProjectInvitationCreateResultDocument(item))
	}

	return types.ProjectInvitationCreateResponse{
		ProjectID: data.ProjectID,
		Results:   results,
	}
}

func mapProjectInvitationCreateResultDocument(data types.ProjectInvitationCreateResultData) types.ProjectInvitationCreateResultDocument {
	var invitation *types.ProjectInvitationDocument
	if data.Invitation != nil {
		document := mapProjectInvitationDocument(*data.Invitation)
		invitation = &document
	}

	return types.ProjectInvitationCreateResultDocument{
		Email:        data.Email,
		Role:         data.Role,
		Status:       data.Status,
		Invitation:   invitation,
		ErrorCode:    data.ErrorCode,
		ErrorMessage: data.ErrorMessage,
	}
}
