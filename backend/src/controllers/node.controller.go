package controllers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	"github.com/MariusBobitiu/agrafa-backend/src/services"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
	"github.com/MariusBobitiu/agrafa-backend/src/utils"
	"github.com/go-chi/chi/v5"
)

type nodeService interface {
	Create(ctx context.Context, projectID int64, name string) (generated.Node, error)
	Update(ctx context.Context, nodeID int64, input types.UpdateNodeInput) (generated.Node, error)
	RegenerateAgentToken(ctx context.Context, nodeID int64) (string, error)
	Delete(ctx context.Context, nodeID int64) error
}

type nodeReader interface {
	GetByID(ctx context.Context, nodeID int64) (types.NodeReadData, error)
}

type NodeController struct {
	nodeService       nodeService
	nodeReader        nodeReader
	streamInterval    time.Duration
	streamMaxDuration time.Duration
}

func NewNodeController(nodeService nodeService, nodeReader nodeReader) *NodeController {
	return &NodeController{
		nodeService:       nodeService,
		nodeReader:        nodeReader,
		streamInterval:    5 * time.Second,
		streamMaxDuration: 25 * time.Second,
	}
}

// Create creates a node.
//
// @Summary      Create node
// @Description  Creates a node under a project with offline status.
// @Tags         inventory
// @Accept       json
// @Produce      json
// @Param        request  body      types.NodeCreateRequest  true  "Node payload"
// @Success      201      {object}  types.NodeResponse
// @Failure      400      {object}  types.ErrorResponse
// @Failure      404      {object}  types.ErrorResponse
// @Failure      409      {object}  types.ErrorResponse
// @Failure      500      {object}  types.ErrorResponse
// @Router       /nodes [post]
func (c *NodeController) Create(w http.ResponseWriter, r *http.Request) {
	var request types.NodeCreateRequest

	if err := utils.DecodeJSON(r, &request); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid node payload")
		return
	}

	node, err := c.nodeService.Create(r.Context(), request.ProjectID, request.Name)
	if err != nil {
		if utils.WriteDomainError(w, err) {
			return
		}

		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusCreated, map[string]any{"node": services.MapNodeResponse(node)})
}

// Get returns a node by id.
//
// @Summary      Get node
// @Description  Returns node details for a node the current user can read.
// @Tags         inventory
// @Produce      json
// @Param        id   path      int  true  "Node ID"
// @Success      200  {object}  types.NodeDetailResponse
// @Failure      400  {object}  types.ErrorResponse
// @Failure      401  {object}  types.ErrorResponse
// @Failure      403  {object}  types.ErrorResponse
// @Failure      404  {object}  types.ErrorResponse
// @Failure      500  {object}  types.ErrorResponse
// @Router       /nodes/{id} [get]
func (c *NodeController) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		utils.WriteError(w, http.StatusBadRequest, "id must be a positive integer")
		return
	}

	node, err := c.nodeReader.GetByID(r.Context(), id)
	if err != nil {
		if utils.WriteDomainError(w, err) {
			return
		}

		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]any{"node": node})
}

// Stream streams node detail snapshots over SSE.
//
// @Summary      Stream node detail
// @Description  Streams the current node detail payload over Server-Sent Events for a node the current user can read.
// @Tags         inventory
// @Produce      text/event-stream
// @Param        id   path      int  true  "Node ID"
// @Success      200
// @Failure      400  {object}  types.ErrorResponse
// @Failure      401  {object}  types.ErrorResponse
// @Failure      403  {object}  types.ErrorResponse
// @Failure      404  {object}  types.ErrorResponse
// @Failure      500  {object}  types.ErrorResponse
// @Router       /nodes/{id}/stream [get]
func (c *NodeController) Stream(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		utils.WriteError(w, http.StatusBadRequest, "id must be a positive integer")
		return
	}

	streamSSESnapshots(w, r, c.streamMaxDuration, c.streamInterval, func(ctx context.Context) (any, error) {
		node, err := c.nodeReader.GetByID(ctx, id)
		if err != nil {
			return nil, err
		}

		return map[string]any{"node": node}, nil
	})
}

// Update updates a node.
//
// @Summary      Update node
// @Description  Updates the editable node identity fields only.
// @Tags         inventory
// @Accept       json
// @Produce      json
// @Param        id       path      int                      true  "Node ID"
// @Param        request  body      types.NodeUpdateRequest  true  "Node update payload"
// @Success      200      {object}  types.NodeResponse
// @Failure      400      {object}  types.ErrorResponse
// @Failure      401      {object}  types.ErrorResponse
// @Failure      403      {object}  types.ErrorResponse
// @Failure      404      {object}  types.ErrorResponse
// @Failure      409      {object}  types.ErrorResponse
// @Failure      500      {object}  types.ErrorResponse
// @Router       /nodes/{id} [patch]
func (c *NodeController) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		utils.WriteError(w, http.StatusBadRequest, "id must be a positive integer")
		return
	}

	var request types.NodeUpdateRequest
	if err := utils.DecodeJSON(r, &request); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid node update payload")
		return
	}

	node, err := c.nodeService.Update(r.Context(), id, types.UpdateNodeInput{
		Name:       request.Name,
		Identifier: request.Identifier,
	})
	if err != nil {
		if utils.WriteDomainError(w, err) {
			return
		}

		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]any{"node": services.MapNodeResponse(node)})
}

// RegenerateAgentToken generates and stores a new per-node agent token.
//
// @Summary      Regenerate node agent token
// @Description  Generates a new random agent token for the node, stores only its hash, and returns the raw token once.
// @Tags         inventory
// @Accept       json
// @Produce      json
// @Param        id  path      int                           true  "Node ID"
// @Success      200 {object}  types.NodeAgentTokenResponse
// @Failure      400 {object}  types.ErrorResponse
// @Failure      404 {object}  types.ErrorResponse
// @Failure      500 {object}  types.ErrorResponse
// @Router       /nodes/{id}/regenerate-agent-token [post]
func (c *NodeController) RegenerateAgentToken(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		utils.WriteError(w, http.StatusBadRequest, "id must be a positive integer")
		return
	}

	token, err := c.nodeService.RegenerateAgentToken(r.Context(), id)
	if err != nil {
		if utils.WriteDomainError(w, err) {
			return
		}

		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, types.NodeAgentTokenResponse{
		NodeID:     id,
		AgentToken: token,
	})
}

// Delete deletes a node.
//
// @Summary      Delete node
// @Description  Deletes a node when it has no remaining services.
// @Tags         inventory
// @Produce      json
// @Param        id  path      int  true  "Node ID"
// @Success      204
// @Failure      400  {object}  types.ErrorResponse
// @Failure      401  {object}  types.ErrorResponse
// @Failure      403  {object}  types.ErrorResponse
// @Failure      404  {object}  types.ErrorResponse
// @Failure      409  {object}  types.ErrorResponse
// @Failure      500  {object}  types.ErrorResponse
// @Router       /nodes/{id} [delete]
func (c *NodeController) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		utils.WriteError(w, http.StatusBadRequest, "id must be a positive integer")
		return
	}

	if err := c.nodeService.Delete(r.Context(), id); err != nil {
		if utils.WriteDomainError(w, err) {
			return
		}

		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
