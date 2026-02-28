// Package service contains application business logic.
package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/stpnv0/CommentTree/internal/domain"
	"github.com/wb-go/wbf/logger"
)

const (
	defaultLimit = 20
	maxLimit     = 100
	maxDepth     = 50
)

type CommentRepository interface {
	Create(ctx context.Context, parentID *int64, author, body string) (*domain.Comment, error)
	GetByID(ctx context.Context, id int64) (*domain.Comment, error)
	GetTree(ctx context.Context, parentID *int64, limit, offset int, sortBy, sortOrder string) ([]domain.Comment, int, error)
	Delete(ctx context.Context, id int64) error
	Search(ctx context.Context, query string, limit, offset int) ([]domain.Comment, int, error)
}

type CommentService struct {
	repo CommentRepository
	log  logger.Logger
}

func NewCommentService(repo CommentRepository, log logger.Logger) *CommentService {
	return &CommentService{
		repo: repo,
		log:  log.With("component", "CommentService"),
	}
}

// Create creates a new comment
func (s *CommentService) Create(ctx context.Context, input domain.CreateCommentInput) (domain.Comment, error) {
	const op = "CommentService.Create"

	if err := validateCreateInput(input); err != nil {
		return domain.Comment{}, err
	}

	if input.ParentID != nil {
		parent, err := s.repo.GetByID(ctx, *input.ParentID)
		if err != nil {
			s.log.LogAttrs(ctx, logger.WarnLevel, "parent lookup failed",
				logger.String("error", err.Error()),
			)
			if errors.Is(err, domain.ErrCommentNotFound) {
				return domain.Comment{}, fmt.Errorf("%s: %w", op, domain.ErrParentNotFound)
			}
			return domain.Comment{}, fmt.Errorf("%s: %w", op, err)
		}
		if parent.Depth >= maxDepth {
			return domain.Comment{}, fmt.Errorf("%w: maximum nesting depth (%d) reached", domain.ErrInvalidInput, maxDepth)
		}
	}

	comment, err := s.repo.Create(ctx, input.ParentID, input.Author, input.Body)
	if err != nil {
		s.log.LogAttrs(ctx, logger.ErrorLevel, "repo create failed",
			logger.String("error", err.Error()),
		)
		return domain.Comment{}, fmt.Errorf("%s: %w", op, err)
	}

	s.log.LogAttrs(ctx, logger.InfoLevel, "comment created",
		logger.Int64("comment_id", comment.ID),
	)

	return *comment, nil
}

// GetTree returns a paginated comment tree
func (s *CommentService) GetTree(
	ctx context.Context,
	parentID *int64,
	params domain.PaginationInput,
) (domain.CommentTreeResult, error) {
	const op = "CommentService.GetTree"

	params = normalizePagination(params)
	params.SortBy = "path"
	params.SortOrder = "ASC"

	comments, total, err := s.repo.GetTree(
		ctx, parentID,
		params.Limit, params.Offset,
		params.SortBy, params.SortOrder,
	)
	if err != nil {
		s.log.LogAttrs(ctx, logger.ErrorLevel, "repo get tree failed",
			logger.String("error", err.Error()),
		)
		return domain.CommentTreeResult{}, fmt.Errorf("%s: %w", op, err)
	}

	return domain.CommentTreeResult{
		Comments: comments,
		Total:    total,
	}, nil
}

// Delete removes a comment and its entire subtree
func (s *CommentService) Delete(ctx context.Context, id int64) error {
	const op = "CommentService.Delete"

	if id <= 0 {
		return fmt.Errorf("%w: invalid comment id", domain.ErrInvalidInput)
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		s.log.LogAttrs(ctx, logger.ErrorLevel, "repo delete failed",
			logger.Int64("comment_id", id),
			logger.String("error", err.Error()),
		)
		return fmt.Errorf("%s: %w", op, err)
	}

	s.log.LogAttrs(ctx, logger.InfoLevel, "comment deleted",
		logger.Int64("comment_id", id),
	)

	return nil
}

// Search performs a full-text search over comment bodies
func (s *CommentService) Search(
	ctx context.Context,
	query string,
	params domain.PaginationInput,
) (domain.SearchResult, error) {
	const op = "CommentService.Search"

	query = strings.TrimSpace(query)
	if query == "" {
		return domain.SearchResult{}, domain.ErrInvalidInput
	}
	params = normalizePagination(params)

	comments, total, err := s.repo.Search(ctx, query, params.Limit, params.Offset)
	if err != nil {
		s.log.LogAttrs(ctx, logger.ErrorLevel, "repo search failed",
			logger.String("query", query),
			logger.String("error", err.Error()),
		)
		return domain.SearchResult{}, fmt.Errorf("%s: %w", op, err)
	}

	return domain.SearchResult{
		Comments: comments,
		Total:    total,
	}, nil
}

func validateCreateInput(input domain.CreateCommentInput) error {
	if input.Author == "" {
		return fmt.Errorf("%w: author must not be empty", domain.ErrInvalidInput)
	}
	if input.Body == "" {
		return fmt.Errorf("%w: body must not be empty", domain.ErrInvalidInput)
	}
	if utf8.RuneCountInString(input.Author) > 255 {
		return fmt.Errorf("%w: author too long (max 255)", domain.ErrInvalidInput)
	}
	if utf8.RuneCountInString(input.Body) > 10000 {
		return fmt.Errorf("%w: body too long (max 10000)", domain.ErrInvalidInput)
	}
	if input.ParentID != nil && *input.ParentID <= 0 {
		return fmt.Errorf("%w: invalid parent_id", domain.ErrInvalidInput)
	}
	return nil
}

// normalizePagination clamps pagination parameters to valid ranges
func normalizePagination(p domain.PaginationInput) domain.PaginationInput {
	if p.Limit <= 0 {
		p.Limit = defaultLimit
	}
	if p.Limit > maxLimit {
		p.Limit = maxLimit
	}
	if p.Offset < 0 {
		p.Offset = 0
	}
	return p
}
