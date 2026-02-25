// Package repository provides data-access implementations.
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/wb-go/wbf/dbpg"
	"github.com/wb-go/wbf/retry"

	"github.com/stpnv0/CommentTree/internal/domain"
)

const tsConfig = "simple"

type CommentRepository struct {
	db       *dbpg.DB
	strategy retry.Strategy
}

// NewCommentPostgres creates a new CommentPostgres repository.
func NewCommentPostgres(db *dbpg.DB, strategy retry.Strategy) *CommentRepository {
	return &CommentRepository{
		db:       db,
		strategy: strategy,
	}
}

// Create inserts a new comment inside a transaction
func (r *CommentRepository) Create(ctx context.Context, parentID *int64, author, body string) (*domain.Comment, error) {
	const op = "CommentRepository.Create"

	var c domain.Comment
	err := r.db.WithTxWithRetry(ctx, r.strategy, func(tx *sql.Tx) error {
		// 1. если родитель указан, проверяем его существование
		var parentPath string
		if parentID != nil {
			queryPath := `SELECT path::text FROM comments WHERE id = $1 FOR UPDATE`
			if err := tx.QueryRowContext(ctx, queryPath, *parentID).Scan(&parentPath); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return domain.ErrParentNotFound
				}
				return fmt.Errorf("%s - fetch parent path: %w", op, err)
			}
		}

		// 2. вставляем строку с временным path
		insertQ := `INSERT INTO comments (parent_id, author, body, path)
		            VALUES ($1, $2, $3, 'tmp')
		            RETURNING id, created_at`
		if err := tx.QueryRowContext(ctx, insertQ, parentID, author, body).Scan(&c.ID, &c.CreatedAt); err != nil {
			return fmt.Errorf("%s - insert comment: %w", op, err)
		}

		// 3. строим реальный path
		var path string
		if parentID != nil {
			path = fmt.Sprintf("%s.%d", parentPath, c.ID)
		} else {
			path = fmt.Sprintf("%d", c.ID)
		}

		//4. обновляем путь
		updateQ := `UPDATE comments SET path = $1::ltree WHERE id = $2`
		if _, err := tx.ExecContext(ctx, updateQ, path, c.ID); err != nil {
			return fmt.Errorf("%s - update path: %w", op, err)
		}

		c.ParentID = parentID
		c.Author = author
		c.Body = body
		c.Path = path
		c.Depth = strings.Count(path, ".") + 1
		return nil
	})

	if err != nil {
		return nil, err
	}
	return &c, nil
}

// GetByID returns a single comment by ID
func (r *CommentRepository) GetByID(ctx context.Context, id int64) (*domain.Comment, error) {
	const op = "CommentRepository.GetByID"

	query := `SELECT id, parent_id, author, body, path::text, created_at, nlevel(path) AS depth
			  FROM comments
		 	  WHERE id = $1`
	row, err := r.db.QueryRowWithRetry(ctx, r.strategy, query, id)
	if err != nil {
		return nil, fmt.Errorf("%s - query comment %d: %w", op, id, err)
	}

	var c domain.Comment
	if err = row.Scan(&c.ID, &c.ParentID, &c.Author, &c.Body, &c.Path, &c.CreatedAt, &c.Depth); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrCommentNotFound
		}
		return nil, fmt.Errorf("%s - scan: %w", op, err)
	}
	return &c, nil
}

// GetTree returns a paginated slice of comments (subtree or all)
func (r *CommentRepository) GetTree(
	ctx context.Context,
	parentID *int64,
	limit, offset int,
	sortBy, sortOrder string,
) ([]domain.Comment, int, error) {
	const op = "CommentRepository.GetTree"

	col := sanitizeSort(sortBy)
	dir := sanitizeOrder(sortOrder)

	var (
		whereClause string
		args        []any
	)

	if parentID != nil {
		parentPath, err := r.parentPath(ctx, *parentID)
		if err != nil {
			return nil, 0, fmt.Errorf("%s: %w", op, err)
		}
		whereClause = `WHERE path <@ $1::ltree`
		args = append(args, parentPath)
	}

	argIdx := len(args)
	limitPlaceholder := fmt.Sprintf("$%d", argIdx+1)
	offsetPlaceholder := fmt.Sprintf("$%d", argIdx+2)
	args = append(args, limit, offset)

	query := fmt.Sprintf(
		`SELECT id, parent_id, author, body,
		        path::text, created_at, nlevel(path) AS depth,
		        COUNT(*) OVER() AS total
		 FROM   comments
		 %s
		 ORDER  BY %s %s
		 LIMIT  %s OFFSET %s`,
		whereClause, col, dir, limitPlaceholder, offsetPlaceholder,
	)

	rows, err := r.db.QueryWithRetry(ctx, r.strategy, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("%s - query tree: %w", op, err)
	}
	defer rows.Close()

	var (
		comments []domain.Comment
		total    int
	)
	for rows.Next() {
		var c domain.Comment
		if err = rows.Scan(
			&c.ID, &c.ParentID, &c.Author, &c.Body,
			&c.Path, &c.CreatedAt, &c.Depth, &total,
		); err != nil {
			return nil, 0, fmt.Errorf("%s - scan: %w", op, err)
		}
		comments = append(comments, c)
	}
	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("%s - rows: %w", op, err)
	}

	if comments == nil {
		comments = []domain.Comment{}
	}

	return comments, total, nil
}

func (r *CommentRepository) parentPath(ctx context.Context, parentID int64) (string, error) {
	query := `SELECT path::text FROM comments WHERE id = $1`
	row, err := r.db.QueryRowWithRetry(ctx, r.strategy, query, parentID)
	if err != nil {
		return "", fmt.Errorf("query parent path: %w", err)
	}
	var path string
	if err = row.Scan(&path); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", domain.ErrParentNotFound
		}
		return "", fmt.Errorf("scan parent path: %w", err)
	}
	return path, nil
}

// Delete removes a comment and its entire subtree using ltree
func (r *CommentRepository) Delete(ctx context.Context, id int64) error {
	const op = "CommentRepository.Delete"

	query := `DELETE FROM comments
	          WHERE path <@ (SELECT path FROM comments WHERE id = $1)`
	res, err := r.db.ExecWithRetry(ctx, r.strategy, query, id)
	if err != nil {
		return fmt.Errorf("%s - delete subtree %d: %w", op, id, err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s - rows affected: %w", op, err)
	}
	if affected == 0 {
		return domain.ErrCommentNotFound
	}
	return nil
}

func (r *CommentRepository) Search(
	ctx context.Context,
	query string,
	limit, offset int,
) ([]domain.Comment, int, error) {
	const op = "CommentRepository.Search"

	dataQ := fmt.Sprintf(
		`SELECT id, parent_id, author, body,
		        path::text, created_at, nlevel(path) AS depth,
		        COUNT(*) OVER() AS total
		 FROM   comments
		 WHERE  to_tsvector('%s', body) @@ plainto_tsquery('%s', $1)
		 ORDER  BY ts_rank(
		           to_tsvector('%s', body),
		           plainto_tsquery('%s', $1)
		         ) DESC,
		         created_at DESC
		 LIMIT  $2 OFFSET $3`,
		tsConfig, tsConfig, tsConfig, tsConfig,
	)

	rows, err := r.db.QueryWithRetry(ctx, r.strategy, dataQ, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()
	var (
		comments []domain.Comment
		total    int
	)
	for rows.Next() {
		var c domain.Comment
		if err = rows.Scan(
			&c.ID, &c.ParentID, &c.Author, &c.Body,
			&c.Path, &c.CreatedAt, &c.Depth, &total,
		); err != nil {
			return nil, 0, fmt.Errorf("%s - scan: %w", op, err)
		}
		comments = append(comments, c)
	}
	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("%s - rows: %w", op, err)
	}

	if comments == nil {
		comments = []domain.Comment{}
	}

	return comments, total, nil
}

// sanitizeSort — whitelist допустимых колонок для ORDER BY
// защита от sql инъекций
func sanitizeSort(col string) string {
	switch col {
	case "created_at", "path", "author":
		return col
	default:
		return "created_at"
	}
}

// sanitizeOrder — whitelist направлений сортировки
func sanitizeOrder(dir string) string {
	if strings.EqualFold(dir, "desc") {
		return "DESC"
	}
	return "ASC"
}
