// Package handler contains HTTP request handlers.
package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/stpnv0/CommentTree/internal/handler/dto"
	"github.com/wb-go/wbf/ginext"
	"github.com/wb-go/wbf/logger"

	"github.com/stpnv0/CommentTree/internal/domain"
)

// CommentService defines the business-logic contract consumed by the handler.
type CommentService interface {
	Create(ctx context.Context, req domain.CreateCommentInput) (domain.Comment, error)
	GetTree(ctx context.Context, parentID *int64, params domain.PaginationInput) (domain.CommentTreeResult, error)
	Delete(ctx context.Context, id int64) error
	Search(ctx context.Context, query string, params domain.PaginationInput) (domain.SearchResult, error)
}

// CommentHandler handles HTTP requests for comments.
type CommentHandler struct {
	svc CommentService
	log logger.Logger
}

// NewCommentHandler creates a new CommentHandler.
func NewCommentHandler(svc CommentService, log logger.Logger) *CommentHandler {
	return &CommentHandler{
		svc: svc,
		log: log,
	}
}

// Create handles POST /comments.
func (h *CommentHandler) Create(c *ginext.Context) {
	var req dto.CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ginext.H{"error": "invalid request body"})
		return
	}

	input := req.ToDomain()

	comment, err := h.svc.Create(c.Request.Context(), input)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.CommentToResponse(comment))
}

// GetTree handles GET /comments.
func (h *CommentHandler) GetTree(c *ginext.Context) {
	var pq dto.PaginationQuery
	if err := c.ShouldBindQuery(&pq); err != nil {
		c.JSON(http.StatusBadRequest, ginext.H{"error": "invalid query parameters"})
		return
	}

	var parentID *int64
	if pidStr := c.Query("parent_id"); pidStr != "" {
		pid, err := strconv.ParseInt(pidStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, ginext.H{"error": "invalid parent_id"})
			return
		}
		parentID = &pid
	}

	params := pq.ToDomain()
	result, err := h.svc.GetTree(c.Request.Context(), parentID, params)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.CommentTreeResponseDTO{
		Comments: dto.CommentsToResponse(result.Comments),
		Total:    result.Total,
		Limit:    params.Limit,
		Offset:   params.Offset,
	})
}

// Delete handles DELETE /comments/:id.
func (h *CommentHandler) Delete(c *ginext.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ginext.H{"error": "invalid comment id"})
		return
	}

	if err = h.svc.Delete(c.Request.Context(), id); err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// Search handles GET /comments/search.
func (h *CommentHandler) Search(c *ginext.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, ginext.H{"error": "query parameter 'q' is required"})
		return
	}

	var pq dto.PaginationQuery
	if err := c.ShouldBindQuery(&pq); err != nil {
		c.JSON(http.StatusBadRequest, ginext.H{"error": "invalid query parameters"})
		return
	}

	params := pq.ToDomain()
	result, err := h.svc.Search(c.Request.Context(), query, params)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.SearchResponseDTO{
		Comments: dto.CommentsToResponse(result.Comments),
		Total:    result.Total,
		Query:    query,
		Limit:    params.Limit,
		Offset:   params.Offset,
	})
}

// handleServiceError maps domain errors to HTTP responses.
func (h *CommentHandler) handleServiceError(c *ginext.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrCommentNotFound):
		c.JSON(http.StatusNotFound, ginext.H{"error": "comment not found"})
	case errors.Is(err, domain.ErrParentNotFound):
		c.JSON(http.StatusBadRequest, ginext.H{"error": "parent comment not found"})
	case errors.Is(err, domain.ErrInvalidInput):
		c.JSON(http.StatusBadRequest, ginext.H{"error": err.Error()})
	default:
		h.log.LogAttrs(c.Request.Context(), logger.ErrorLevel, "unhandled service error",
			logger.String("error", err.Error()),
			logger.String("path", c.Request.URL.Path),
			logger.String("method", c.Request.Method),
		)
		c.JSON(http.StatusInternalServerError, ginext.H{"error": "internal server error"})
	}
}
