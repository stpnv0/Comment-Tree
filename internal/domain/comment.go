// Package domain contains core business entities.
package domain

import "time"

// Comment represents a single comment in a tree structure.
type Comment struct {
	ID        int64     `json:"id"`
	ParentID  *int64    `json:"parent_id,omitempty"`
	Author    string    `json:"author"`
	Body      string    `json:"body"`
	Path      string    `json:"path"`
	CreatedAt time.Time `json:"created_at"`
	Depth     int       `json:"depth"` // nlevel(path): 1 = корень
}

// CreateCommentInput — clean domain input for creating a comment.
type CreateCommentInput struct {
	ParentID *int64
	Author   string
	Body     string
}

// PaginationInput — domain-level pagination parameters.
type PaginationInput struct {
	Limit     int
	Offset    int
	SortBy    string
	SortOrder string
}

// CommentTreeResult — domain-level result of a tree query.
type CommentTreeResult struct {
	Comments []Comment
	Total    int
}

// SearchResult — domain-level result of a search query.
type SearchResult struct {
	Comments []Comment
	Total    int
}
