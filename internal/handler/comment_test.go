package handler_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/stpnv0/CommentTree/internal/domain"
	"github.com/stpnv0/CommentTree/internal/handler"
	"github.com/stpnv0/CommentTree/internal/handler/dto"
	"github.com/stpnv0/CommentTree/internal/handler/mocks"
	wbflogger "github.com/wb-go/wbf/logger"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func testLogger() wbflogger.Logger {
	log, _ := wbflogger.InitLogger(wbflogger.SlogEngine, "test", "test")
	return log
}

func setupRouter(h *handler.CommentHandler) *gin.Engine {
	r := gin.New()
	r.POST("/api/comments", h.Create)
	r.GET("/api/comments", h.GetTree)
	r.DELETE("/api/comments/:id", h.Delete)
	r.GET("/api/comments/search", h.Search)
	return r
}

func TestCommentHandler_Create_Success(t *testing.T) {
	svc := mocks.NewMockCommentService(t)
	h := handler.NewCommentHandler(svc, testLogger())
	r := setupRouter(h)

	now := time.Now().UTC().Truncate(time.Second)
	expected := domain.Comment{
		ID:        1,
		Author:    "Alice",
		Body:      "Hello world",
		Path:      "1",
		CreatedAt: now,
		Depth:     1,
	}

	svc.On("Create", mock.Anything, domain.CreateCommentInput{
		Author: "Alice",
		Body:   "Hello world",
	}).Return(expected, nil)

	body, _ := json.Marshal(map[string]string{"author": "Alice", "body": "Hello world"})
	req := httptest.NewRequest(http.MethodPost, "/api/comments", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var got dto.CommentResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, int64(1), got.ID)
	assert.Equal(t, "Alice", got.Author)
	assert.Equal(t, "Hello world", got.Body)
}

func TestCommentHandler_Create_ValidationError(t *testing.T) {
	svc := mocks.NewMockCommentService(t)
	h := handler.NewCommentHandler(svc, testLogger())
	r := setupRouter(h)

	svc.On("Create", mock.Anything, domain.CreateCommentInput{
		Author: "",
		Body:   "",
	}).Return(domain.Comment{}, domain.ErrInvalidInput)

	body, _ := json.Marshal(map[string]string{"author": "", "body": ""})
	req := httptest.NewRequest(http.MethodPost, "/api/comments", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCommentHandler_Create_ParentNotFound(t *testing.T) {
	svc := mocks.NewMockCommentService(t)
	h := handler.NewCommentHandler(svc, testLogger())
	r := setupRouter(h)

	parentID := int64(999)
	svc.On("Create", mock.Anything, domain.CreateCommentInput{
		ParentID: &parentID,
		Author:   "Bob",
		Body:     "reply",
	}).Return(domain.Comment{}, domain.ErrParentNotFound)

	body, _ := json.Marshal(map[string]any{"parent_id": 999, "author": "Bob", "body": "reply"})
	req := httptest.NewRequest(http.MethodPost, "/api/comments", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "parent comment not found")
}

func TestCommentHandler_GetTree_Success(t *testing.T) {
	svc := mocks.NewMockCommentService(t)
	h := handler.NewCommentHandler(svc, testLogger())
	r := setupRouter(h)

	resp := domain.CommentTreeResult{
		Comments: []domain.Comment{
			{ID: 1, Author: "Alice", Body: "Hello", Path: "1", CreatedAt: time.Now(), Depth: 1},
		},
		Total: 1,
	}
	svc.On("GetTree", mock.Anything, (*int64)(nil), mock.AnythingOfType("domain.PaginationInput")).
		Return(resp, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/comments?limit=20&offset=0", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var got dto.CommentTreeResponseDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, 1, got.Total)
	assert.Len(t, got.Comments, 1)
}

func TestCommentHandler_GetTree_WithParentID(t *testing.T) {
	svc := mocks.NewMockCommentService(t)
	h := handler.NewCommentHandler(svc, testLogger())
	r := setupRouter(h)

	parentID := int64(5)
	resp := domain.CommentTreeResult{
		Comments: []domain.Comment{},
		Total:    0,
	}
	svc.On("GetTree", mock.Anything, &parentID, mock.AnythingOfType("domain.PaginationInput")).
		Return(resp, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/comments?parent_id=5", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCommentHandler_GetTree_InvalidParentID(t *testing.T) {
	svc := mocks.NewMockCommentService(t)
	h := handler.NewCommentHandler(svc, testLogger())
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/comments?parent_id=abc", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid parent_id")
}

func TestCommentHandler_Delete_Success(t *testing.T) {
	svc := mocks.NewMockCommentService(t)
	h := handler.NewCommentHandler(svc, testLogger())
	r := setupRouter(h)

	svc.On("Delete", mock.Anything, int64(1)).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/comments/1", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestCommentHandler_Delete_NotFound(t *testing.T) {
	svc := mocks.NewMockCommentService(t)
	h := handler.NewCommentHandler(svc, testLogger())
	r := setupRouter(h)

	svc.On("Delete", mock.Anything, int64(999)).Return(domain.ErrCommentNotFound)

	req := httptest.NewRequest(http.MethodDelete, "/api/comments/999", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCommentHandler_Delete_InvalidID(t *testing.T) {
	svc := mocks.NewMockCommentService(t)
	h := handler.NewCommentHandler(svc, testLogger())
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodDelete, "/api/comments/abc", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCommentHandler_Search_Success(t *testing.T) {
	svc := mocks.NewMockCommentService(t)
	h := handler.NewCommentHandler(svc, testLogger())
	r := setupRouter(h)

	resp := domain.SearchResult{
		Comments: []domain.Comment{
			{ID: 1, Author: "Alice", Body: "search match", Path: "1", CreatedAt: time.Now(), Depth: 1},
		},
		Total: 1,
	}
	svc.On("Search", mock.Anything, "match", mock.AnythingOfType("domain.PaginationInput")).
		Return(resp, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/comments/search?q=match", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var got dto.SearchResponseDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Len(t, got.Comments, 1)
}

func TestCommentHandler_Search_MissingQuery(t *testing.T) {
	svc := mocks.NewMockCommentService(t)
	h := handler.NewCommentHandler(svc, testLogger())
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/comments/search", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "'q' is required")
}

func TestCommentHandler_Create_InternalError(t *testing.T) {
	svc := mocks.NewMockCommentService(t)
	h := handler.NewCommentHandler(svc, testLogger())
	r := setupRouter(h)

	svc.On("Create", mock.Anything, domain.CreateCommentInput{
		Author: "Alice",
		Body:   "Hello",
	}).Return(domain.Comment{}, errors.New("unexpected"))

	body, _ := json.Marshal(map[string]string{"author": "Alice", "body": "Hello"})
	req := httptest.NewRequest(http.MethodPost, "/api/comments", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
