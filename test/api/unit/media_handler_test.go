// Package unit contains unit tests for the media handler.
// Handler unit tests verify HTTP routing, request parsing, and response formatting.
//
// HANDLER UNIT TEST PATTERN:
// ═══════════════════════════════════════════════════════════════════════════════
//
//	┌─────────────────────────────────────────────────────────────────────────┐
//	│                      HANDLER UNIT TEST ENVIRONMENT                       │
//	│                                                                          │
//	│    ┌──────────────┐        ┌──────────────┐        ┌──────────────┐     │
//	│    │ HTTP Request │───────▶│ MediaHandler │───────▶│    Mock      │     │
//	│    │  (httptest)  │        │   (real)     │        │   Service    │     │
//	│    └──────────────┘        └──────────────┘        └──────────────┘     │
//	│           │                                                │             │
//	│           ▼                                                ▼             │
//	│    ┌──────────────┐                                ┌──────────────┐     │
//	│    │HTTP Response │                                │ Verify Calls │     │
//	│    │  Recorder    │                                │              │     │
//	│    └──────────────┘                                └──────────────┘     │
//	└─────────────────────────────────────────────────────────────────────────┘
package unit

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/AchilleasB/baby-kliniek/media-service/internal/adapters/handler"
	"github.com/AchilleasB/baby-kliniek/media-service/internal/core/domain"
	"github.com/AchilleasB/baby-kliniek/media-service/internal/core/services"
	"github.com/AchilleasB/baby-kliniek/media-service/test/mocks"
)

// Helper to create a JSON request body reader
func jsonReader(jsonStr string) *strings.Reader {
	return strings.NewReader(jsonStr)
}

// TestMediaHandler_GetVideos_InvalidMethod tests method validation.
func TestMediaHandler_GetVideos_InvalidMethod(t *testing.T) {
	mockRepo := mocks.NewMockVideoRepository()
	service := services.NewVideoService(mockRepo)
	h := handler.NewMediaHandler(service)

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/media/videos", nil)
			rec := httptest.NewRecorder()

			h.GetVideos(rec, req)

			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
			}
		})
	}
}

// TestMediaHandler_GetVideos_Success tests successful video retrieval.
func TestMediaHandler_GetVideos_Success(t *testing.T) {
	mockRepo := mocks.NewMockVideoRepository()
	mockRepo.SeedVideo(mocks.CreateTestVideo("v1", "http://example.com/v1.mp4", domain.Temperature, "Temp"))
	mockRepo.SeedVideo(mocks.CreateTestVideo("v2", "http://example.com/v2.mp4", domain.Weighting, "Weight"))

	service := services.NewVideoService(mockRepo)
	h := handler.NewMediaHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/media/videos", nil)
	rec := httptest.NewRecorder()

	h.GetVideos(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", contentType)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	videos, ok := response["videos"].([]interface{})
	if !ok {
		t.Fatal("response should contain 'videos' array")
	}

	if len(videos) != 2 {
		t.Errorf("expected 2 videos, got %d", len(videos))
	}
}

// TestMediaHandler_GetVideos_DatabaseError tests error handling.
func TestMediaHandler_GetVideos_DatabaseError(t *testing.T) {
	mockRepo := mocks.NewMockVideoRepository()
	mockRepo.GetVideosError = errors.New("database connection failed")

	service := services.NewVideoService(mockRepo)
	h := handler.NewMediaHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/media/videos", nil)
	rec := httptest.NewRecorder()

	h.GetVideos(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

// TestMediaHandler_GetOneVideo_Success tests successful single video retrieval.
func TestMediaHandler_GetOneVideo_Success(t *testing.T) {
	mockRepo := mocks.NewMockVideoRepository()
	mockRepo.SeedVideo(mocks.CreateTestVideo("test-id-123", "http://example.com/test.mp4", domain.Sleeping, "Test video"))

	service := services.NewVideoService(mockRepo)
	h := handler.NewMediaHandler(service)

	// Use Go 1.22+ path value pattern
	mux := http.NewServeMux()
	mux.HandleFunc("GET /media/videos/{id}", h.GetOneVideo)

	req := httptest.NewRequest(http.MethodGet, "/media/videos/test-id-123", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response["id"] != "test-id-123" {
		t.Errorf("expected id 'test-id-123', got %v", response["id"])
	}
}

// TestMediaHandler_GetOneVideo_NotFound tests 404 response.
func TestMediaHandler_GetOneVideo_NotFound(t *testing.T) {
	mockRepo := mocks.NewMockVideoRepository()
	service := services.NewVideoService(mockRepo)
	h := handler.NewMediaHandler(service)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /media/videos/{id}", h.GetOneVideo)

	req := httptest.NewRequest(http.MethodGet, "/media/videos/nonexistent", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

// TestMediaHandler_GetOneVideo_InvalidMethod tests method validation.
func TestMediaHandler_GetOneVideo_InvalidMethod(t *testing.T) {
	mockRepo := mocks.NewMockVideoRepository()
	service := services.NewVideoService(mockRepo)
	h := handler.NewMediaHandler(service)

	mux := http.NewServeMux()
	mux.HandleFunc("/media/videos/{id}", h.GetOneVideo)

	req := httptest.NewRequest(http.MethodPost, "/media/videos/test-id", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
}

// TestMediaHandler_CreateVideo_Success tests successful video creation.
func TestMediaHandler_CreateVideo_Success(t *testing.T) {
	mockRepo := mocks.NewMockVideoRepository()
	service := services.NewVideoService(mockRepo)
	h := handler.NewMediaHandler(service)

	body := `{"url":"http://example.com/new.mp4","content_type":"TEMPERATURE","description":"New video"}`
	req := httptest.NewRequest(http.MethodPost, "/media/videos", jsonReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.CreateVideo(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response["url"] != "http://example.com/new.mp4" {
		t.Errorf("expected url 'http://example.com/new.mp4', got %v", response["url"])
	}

	if response["id"] == "" {
		t.Error("response should contain generated 'id'")
	}

	if mockRepo.CreateVideoCalled != 1 {
		t.Errorf("expected CreateVideo to be called once, got %d", mockRepo.CreateVideoCalled)
	}
}

// TestMediaHandler_CreateVideo_InvalidMethod tests method validation.
func TestMediaHandler_CreateVideo_InvalidMethod(t *testing.T) {
	mockRepo := mocks.NewMockVideoRepository()
	service := services.NewVideoService(mockRepo)
	h := handler.NewMediaHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/media/videos", nil)
	rec := httptest.NewRecorder()

	h.CreateVideo(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
}

// TestMediaHandler_CreateVideo_InvalidJSON tests bad request handling.
func TestMediaHandler_CreateVideo_InvalidJSON(t *testing.T) {
	mockRepo := mocks.NewMockVideoRepository()
	service := services.NewVideoService(mockRepo)
	h := handler.NewMediaHandler(service)

	body := `{"invalid json`
	req := httptest.NewRequest(http.MethodPost, "/media/videos", jsonReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.CreateVideo(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

// TestMediaHandler_CreateVideo_DatabaseError tests error handling.
func TestMediaHandler_CreateVideo_DatabaseError(t *testing.T) {
	mockRepo := mocks.NewMockVideoRepository()
	mockRepo.CreateVideoError = errors.New("database error")

	service := services.NewVideoService(mockRepo)
	h := handler.NewMediaHandler(service)

	body := `{"url":"http://example.com/new.mp4","content_type":"TEMPERATURE","description":"New video"}`
	req := httptest.NewRequest(http.MethodPost, "/media/videos", jsonReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.CreateVideo(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

// TestMediaHandler_DeleteVideo_Success tests successful video deletion.
func TestMediaHandler_DeleteVideo_Success(t *testing.T) {
	mockRepo := mocks.NewMockVideoRepository()
	mockRepo.SeedVideo(mocks.CreateTestVideo("delete-me", "http://example.com/delete.mp4", domain.DiaperChange, "Delete"))

	service := services.NewVideoService(mockRepo)
	h := handler.NewMediaHandler(service)

	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /media/videos/{id}", h.DeleteVideo)

	req := httptest.NewRequest(http.MethodDelete, "/media/videos/delete-me", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if mockRepo.DeleteVideoCalled != 1 {
		t.Errorf("expected DeleteVideo to be called once, got %d", mockRepo.DeleteVideoCalled)
	}
}

// TestMediaHandler_DeleteVideo_InvalidMethod tests method validation.
func TestMediaHandler_DeleteVideo_InvalidMethod(t *testing.T) {
	mockRepo := mocks.NewMockVideoRepository()
	service := services.NewVideoService(mockRepo)
	h := handler.NewMediaHandler(service)

	mux := http.NewServeMux()
	mux.HandleFunc("/media/videos/{id}", h.DeleteVideo)

	req := httptest.NewRequest(http.MethodGet, "/media/videos/some-id", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
}

// TestMediaHandler_DeleteVideo_NotFound tests error handling.
func TestMediaHandler_DeleteVideo_NotFound(t *testing.T) {
	mockRepo := mocks.NewMockVideoRepository()
	// Don't seed any video

	service := services.NewVideoService(mockRepo)
	h := handler.NewMediaHandler(service)

	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /media/videos/{id}", h.DeleteVideo)

	req := httptest.NewRequest(http.MethodDelete, "/media/videos/nonexistent", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	// Handler returns 500 for delete errors (could be improved to 404)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

// TestMediaHandler_ResponseContentType tests that all responses have correct Content-Type.
func TestMediaHandler_ResponseContentType(t *testing.T) {
	mockRepo := mocks.NewMockVideoRepository()
	mockRepo.SeedVideo(mocks.CreateTestVideo("v1", "http://example.com/v1.mp4", domain.Temperature, "Test"))

	service := services.NewVideoService(mockRepo)
	h := handler.NewMediaHandler(service)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /media/videos", h.GetVideos)
	mux.HandleFunc("GET /media/videos/{id}", h.GetOneVideo)
	mux.HandleFunc("POST /media/videos", h.CreateVideo)

	tests := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodGet, "/media/videos", ""},
		{http.MethodGet, "/media/videos/v1", ""},
		{http.MethodPost, "/media/videos", `{"url":"http://test.com","content_type":"SLEEPING","description":"test"}`},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			var req *http.Request
			if tt.body != "" {
				req = httptest.NewRequest(tt.method, tt.path, jsonReader(tt.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, tt.path, nil)
			}
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, req)

			contentType := rec.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("expected Content-Type application/json, got %s", contentType)
			}
		})
	}
}
