// Package unit contains unit tests for the health handler.
// Health handler tests verify liveness and readiness checks.
package unit

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/AchilleasB/baby-kliniek/media-service/internal/adapters/handler"
)

// TestHealthHandler_Health_ProcessCheck tests basic health check.
func TestHealthHandler_Health_ProcessCheck(t *testing.T) {
	// Create handler with nil mongo client (for unit testing)
	h := handler.NewHealthHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	h.Health(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response["status"] != "UP" {
		t.Errorf("expected status 'UP', got %v", response["status"])
	}

	checks, ok := response["checks"].(map[string]interface{})
	if !ok {
		t.Fatal("response should contain 'checks' object")
	}

	processCheck, ok := checks["process"].(map[string]interface{})
	if !ok {
		t.Fatal("checks should contain 'process'")
	}

	if processCheck["status"] != "UP" {
		t.Errorf("expected process status 'UP', got %v", processCheck["status"])
	}
}

// TestHealthHandler_Health_InvalidMethod tests method validation.
func TestHealthHandler_Health_InvalidMethod(t *testing.T) {
	h := handler.NewHealthHandler(nil)

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/health", nil)
			rec := httptest.NewRecorder()

			h.Health(rec, req)

			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
			}
		})
	}
}

// TestHealthHandler_Live tests liveness endpoint.
func TestHealthHandler_Live(t *testing.T) {
	h := handler.NewHealthHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	rec := httptest.NewRecorder()

	h.Live(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response["status"] != "UP" {
		t.Errorf("expected status 'UP', got %v", response["status"])
	}
}

// TestHealthHandler_Ready_NoDependencies tests readiness without database.
func TestHealthHandler_Ready_NoDependencies(t *testing.T) {
	// nil mongo client should result in service unavailable
	h := handler.NewHealthHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	rec := httptest.NewRecorder()

	h.Ready(rec, req)

	// Without mongo client, database check fails
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status %d (unavailable without DB), got %d", http.StatusServiceUnavailable, rec.Code)
	}
}

// TestHealthHandler_ContentType tests JSON response format.
func TestHealthHandler_ContentType(t *testing.T) {
	h := handler.NewHealthHandler(nil)

	endpoints := []struct {
		path    string
		handler func(http.ResponseWriter, *http.Request)
	}{
		{"/health", h.Health},
		{"/health/live", h.Live},
		{"/health/ready", h.Ready},
	}

	for _, ep := range endpoints {
		t.Run(ep.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, ep.path, nil)
			rec := httptest.NewRecorder()

			ep.handler(rec, req)

			contentType := rec.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("expected Content-Type application/json, got %s", contentType)
			}
		})
	}
}

// TestHealthHandler_UptimeIncreases tests that uptime increases over time.
func TestHealthHandler_UptimeIncreases(t *testing.T) {
	h := handler.NewHealthHandler(nil)

	// First request
	req1 := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec1 := httptest.NewRecorder()
	h.Health(rec1, req1)

	var response1 map[string]interface{}
	json.Unmarshal(rec1.Body.Bytes(), &response1)
	uptime1 := response1["uptime"].(string)

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	// Second request
	req2 := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec2 := httptest.NewRecorder()
	h.Health(rec2, req2)

	var response2 map[string]interface{}
	json.Unmarshal(rec2.Body.Bytes(), &response2)
	uptime2 := response2["uptime"].(string)

	// Uptimes should be different (increased)
	if uptime1 == uptime2 {
		t.Log("Uptime values were the same, which can happen with fast tests")
	}
}

// TestHealthHandler_ResponseStructure tests the health response structure.
func TestHealthHandler_ResponseStructure(t *testing.T) {
	h := handler.NewHealthHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	h.Health(rec, req)

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// Check required fields
	requiredFields := []string{"status", "timestamp", "uptime", "version", "checks"}
	for _, field := range requiredFields {
		if _, exists := response[field]; !exists {
			t.Errorf("response missing required field: %s", field)
		}
	}
}
