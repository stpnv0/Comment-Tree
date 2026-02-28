package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/stpnv0/CommentTree/internal/domain"
	"github.com/stpnv0/CommentTree/internal/service"
	"github.com/stpnv0/CommentTree/internal/service/mocks"
	wbflogger "github.com/wb-go/wbf/logger"
)

func testLogger() wbflogger.Logger {
	log, _ := wbflogger.InitLogger(wbflogger.SlogEngine, "test", "test")
	return log
}

func TestCommentService_Create_RootComment(t *testing.T) {
	repo := mocks.NewMockCommentRepository(t)
	svc := service.NewCommentService(repo, testLogger())

	now := time.Now().UTC().Truncate(time.Second)
	expected := &domain.Comment{
		ID:        1,
		Author:    "Alice",
		Body:      "Hello",
		Path:      "1",
		CreatedAt: now,
		Depth:     1,
	}

	repo.On("Create", mock.Anything, (*int64)(nil), "Alice", "Hello").
		Return(expected, nil)

	input := domain.CreateCommentInput{Author: "Alice", Body: "Hello"}
	got, err := svc.Create(context.Background(), input)

	require.NoError(t, err)
	assert.Equal(t, int64(1), got.ID)
	assert.Equal(t, "Alice", got.Author)
	assert.Equal(t, "1", got.Path)
}

func TestCommentService_Create_WithParent(t *testing.T) {
	repo := mocks.NewMockCommentRepository(t)
	svc := service.NewCommentService(repo, testLogger())

	parentID := int64(1)
	now := time.Now().UTC().Truncate(time.Second)

	parent := &domain.Comment{
		ID:        1,
		Author:    "Alice",
		Body:      "Hello",
		Path:      "1",
		CreatedAt: now,
		Depth:     1,
	}
	child := &domain.Comment{
		ID:        2,
		ParentID:  &parentID,
		Author:    "Bob",
		Body:      "Reply",
		Path:      "1.2",
		CreatedAt: now,
		Depth:     2,
	}

	repo.On("GetByID", mock.Anything, parentID).Return(parent, nil)
	repo.On("Create", mock.Anything, &parentID, "Bob", "Reply").
		Return(child, nil)

	input := domain.CreateCommentInput{ParentID: &parentID, Author: "Bob", Body: "Reply"}
	got, err := svc.Create(context.Background(), input)

	require.NoError(t, err)
	assert.Equal(t, int64(2), got.ID)
	assert.Equal(t, "1.2", got.Path)
}

func TestCommentService_Create_ParentNotFound(t *testing.T) {
	repo := mocks.NewMockCommentRepository(t)
	svc := service.NewCommentService(repo, testLogger())

	parentID := int64(999)
	repo.On("GetByID", mock.Anything, parentID).Return((*domain.Comment)(nil), domain.ErrCommentNotFound)

	input := domain.CreateCommentInput{ParentID: &parentID, Author: "Bob", Body: "Reply"}
	_, err := svc.Create(context.Background(), input)

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrParentNotFound)
}

func TestCommentService_Create_MaxDepth(t *testing.T) {
	repo := mocks.NewMockCommentRepository(t)
	svc := service.NewCommentService(repo, testLogger())

	parentID := int64(1)
	parent := &domain.Comment{
		ID:    1,
		Path:  "1",
		Depth: 50,
	}

	repo.On("GetByID", mock.Anything, parentID).Return(parent, nil)

	input := domain.CreateCommentInput{ParentID: &parentID, Author: "Bob", Body: "Reply"}
	_, err := svc.Create(context.Background(), input)

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrInvalidInput)
	assert.Contains(t, err.Error(), "maximum nesting depth")
}

func TestCommentService_GetTree_Defaults(t *testing.T) {
	repo := mocks.NewMockCommentRepository(t)
	svc := service.NewCommentService(repo, testLogger())

	now := time.Now().UTC().Truncate(time.Second)
	comments := []domain.Comment{
		{ID: 1, Author: "Alice", Body: "Hello", Path: "1", CreatedAt: now, Depth: 1},
	}

	// Service forces sort to "path" ASC and default limit 20
	repo.On("GetTree", mock.Anything, (*int64)(nil), 20, 0, "path", "ASC").
		Return(comments, 1, nil)

	params := domain.PaginationInput{} // zeroed; defaults will be applied
	got, err := svc.GetTree(context.Background(), nil, params)

	require.NoError(t, err)
	assert.Equal(t, 1, got.Total)
	assert.Len(t, got.Comments, 1)
}

func TestCommentService_Delete_Success(t *testing.T) {
	repo := mocks.NewMockCommentRepository(t)
	svc := service.NewCommentService(repo, testLogger())

	repo.On("Delete", mock.Anything, int64(1)).Return(nil)

	err := svc.Delete(context.Background(), 1)
	require.NoError(t, err)
}

func TestCommentService_Delete_NotFound(t *testing.T) {
	repo := mocks.NewMockCommentRepository(t)
	svc := service.NewCommentService(repo, testLogger())

	repo.On("Delete", mock.Anything, int64(999)).Return(domain.ErrCommentNotFound)

	err := svc.Delete(context.Background(), 999)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrCommentNotFound)
}

func TestCommentService_Search_Success(t *testing.T) {
	repo := mocks.NewMockCommentRepository(t)
	svc := service.NewCommentService(repo, testLogger())

	now := time.Now().UTC().Truncate(time.Second)
	comments := []domain.Comment{
		{ID: 1, Author: "Alice", Body: "matching text", Path: "1", CreatedAt: now, Depth: 1},
	}

	repo.On("Search", mock.Anything, "matching", 20, 0).
		Return(comments, 1, nil)

	params := domain.PaginationInput{}
	got, err := svc.Search(context.Background(), "matching", params)

	require.NoError(t, err)
	assert.Equal(t, 1, got.Total)
	assert.Len(t, got.Comments, 1)
}

func TestCommentService_Search_Empty(t *testing.T) {
	repo := mocks.NewMockCommentRepository(t)
	svc := service.NewCommentService(repo, testLogger())

	repo.On("Search", mock.Anything, "nonexistent", 20, 0).
		Return([]domain.Comment(nil), 0, nil)

	params := domain.PaginationInput{}
	got, err := svc.Search(context.Background(), "nonexistent", params)

	require.NoError(t, err)
	assert.Equal(t, 0, got.Total)
	assert.Empty(t, got.Comments)
}
