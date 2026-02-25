package dto

import (
	"strings"

	"github.com/stpnv0/CommentTree/internal/domain"
)

// CreateCommentRequest is the JSON body for POST /comments.
type CreateCommentRequest struct {
	ParentID *int64 `json:"parent_id,omitempty"`
	Author   string `json:"author"`
	Body     string `json:"body"`
}

// ToDomain converts a handler-level DTO to a domain input.
func (r *CreateCommentRequest) ToDomain() domain.CreateCommentInput {
	return domain.CreateCommentInput{
		ParentID: r.ParentID,
		Author:   strings.TrimSpace(r.Author),
		Body:     strings.TrimSpace(r.Body),
	}
}

// PaginationQuery binds query-string pagination params.
type PaginationQuery struct {
	Limit     int    `form:"limit"`
	Offset    int    `form:"offset"`
	SortBy    string `form:"sort_by"`
	SortOrder string `form:"sort_order"`
}

// ToDomain converts to domain-level pagination.
func (p *PaginationQuery) ToDomain() domain.PaginationInput {
	return domain.PaginationInput{
		Limit:     p.Limit,
		Offset:    p.Offset,
		SortBy:    p.SortBy,
		SortOrder: p.SortOrder,
	}
}

// CommentResponse is a single comment in JSON.
type CommentResponse struct {
	ID        int64  `json:"id"`
	ParentID  *int64 `json:"parent_id,omitempty"`
	Author    string `json:"author"`
	Body      string `json:"body"`
	Path      string `json:"path"`
	CreatedAt string `json:"created_at"`
	Depth     int    `json:"depth"`
}

// CommentTreeResponseDTO is the paginated tree response.
type CommentTreeResponseDTO struct {
	Comments []CommentResponse `json:"comments"`
	Total    int               `json:"total"`
	Limit    int               `json:"limit"`
	Offset   int               `json:"offset"`
}

// SearchResponseDTO is the paginated search response.
type SearchResponseDTO struct {
	Comments []CommentResponse `json:"comments"`
	Total    int               `json:"total"`
	Query    string            `json:"query"`
	Limit    int               `json:"limit"`
	Offset   int               `json:"offset"`
}

// commentToResponse converts a domain.Comment to a JSON-safe DTO.
func CommentToResponse(c domain.Comment) CommentResponse {
	return CommentResponse{
		ID:        c.ID,
		ParentID:  c.ParentID,
		Author:    c.Author,
		Body:      c.Body,
		Path:      c.Path,
		CreatedAt: c.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		Depth:     c.Depth,
	}
}

// commentsToResponse converts a slice of domain comments.
func CommentsToResponse(cs []domain.Comment) []CommentResponse {
	// FIX: always return non-nil slice so JSON is [] not null
	out := make([]CommentResponse, 0, len(cs))
	for _, c := range cs {
		out = append(out, CommentToResponse(c))
	}
	return out
}
