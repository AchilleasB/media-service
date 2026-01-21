// Package integration contains integration tests for the Media Service API.
// Integration tests verify components work together with real dependencies.
//
// INTEGRATION TEST PATTERN IN HEXAGONAL ARCHITECTURE:
// ═══════════════════════════════════════════════════════════════════════════════
//
//	┌─────────────────────────────────────────────────────────────────────────┐
//	│                      INTEGRATION TEST ENVIRONMENT                        │
//	│                                                                          │
//	│    ┌──────────────┐        ┌──────────────┐        ┌──────────────┐     │
//	│    │  HTTP Client │───────▶│   API Server │◀───────│   MongoDB    │     │
//	│    │  (httptest)  │        │  (real mux)  │        │   (real)     │     │
//	│    └──────────────┘        └──────────────┘        └──────────────┘     │
//	│                                   │                                      │
//	│                                   ▼                                      │
//	│                            ┌──────────────┐                              │
//	│                            │   Services   │                              │
//	│                            │   (real)     │                              │
//	│                            └──────────────┘                              │
//	└─────────────────────────────────────────────────────────────────────────┘
//
// These tests use real dependencies (MongoDB) to catch issues that unit tests miss.
//
// RUNNING THESE TESTS:
// 1. Start MongoDB: docker-compose up -d mongo-media
// 2. Set environment: TEST_MONGO_URI=mongodb://localhost:27017
// 3. Run: go test -v ./test/api/integration/...
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/AchilleasB/baby-kliniek/media-service/internal/adapters/handler"
	"github.com/AchilleasB/baby-kliniek/media-service/internal/adapters/repository"
	"github.com/AchilleasB/baby-kliniek/media-service/internal/core/services"
)

var (
	testMongoClient *mongo.Client
	testDB          *mongo.Database
)

// TestMain sets up and tears down the test environment.
func TestMain(m *testing.M) {
	mongoURI := os.Getenv("TEST_MONGO_URI")
	if mongoURI == "" {
		fmt.Println("Skipping integration tests: TEST_MONGO_URI not set")
		fmt.Println("Run with: TEST_MONGO_URI='mongodb://localhost:27017' go test ./...")
		os.Exit(0)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var err error
	testMongoClient, err = mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		fmt.Printf("Failed to connect to MongoDB: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = testMongoClient.Disconnect(context.Background()) }()

	if err := testMongoClient.Ping(ctx, nil); err != nil {
		fmt.Printf("Failed to ping MongoDB: %v\n", err)
		os.Exit(1)
	}

	// Use a test database
	testDB = testMongoClient.Database("media_test")

	// Run tests
	code := m.Run()

	// Cleanup
	cleanupTestData()

	os.Exit(code)
}

func cleanupTestData() {
	ctx := context.Background()
	// Ensure a clean slate without destroying the collection namespace
	if _, err := testDB.Collection("videos").DeleteMany(ctx, bson.M{}); err != nil {
		fmt.Printf("Failed to cleanup test data: %v\n", err)
	}
}

func setupTestServer() *httptest.Server {
	repo := repository.NewMongoRepository(testDB)
	// Override to use test database
	testCollection := testDB.Collection("videos")
	_ = testCollection // For reference

	videoService := services.NewVideoService(repo)
	mediaHandler := handler.NewMediaHandler(videoService)
	healthHandler := handler.NewHealthHandler(testMongoClient)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler.Health)
	mux.HandleFunc("/health/ready", healthHandler.Ready)
	mux.HandleFunc("/health/live", healthHandler.Live)
	mux.HandleFunc("GET /media/videos", mediaHandler.GetVideos)
	mux.HandleFunc("GET /media/videos/{id}", mediaHandler.GetOneVideo)
	mux.HandleFunc("POST /media/videos", mediaHandler.CreateVideo)
	mux.HandleFunc("DELETE /media/videos/{id}", mediaHandler.DeleteVideo)

	return httptest.NewServer(mux)
}

// TestIntegration_HealthCheck tests health endpoints with real MongoDB.
func TestIntegration_HealthCheck(t *testing.T) {
	if testMongoClient == nil {
		t.Skip("Integration tests require MongoDB connection")
	}

	server := setupTestServer()
	defer server.Close()

	t.Run("liveness", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/health/live")
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
		}
	})

	t.Run("readiness_with_mongodb", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/health/ready")
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
		}
	})
}

// TestIntegration_CreateAndGetVideo tests the full video lifecycle.
func TestIntegration_CreateAndGetVideo(t *testing.T) {
	if testMongoClient == nil {
		t.Skip("Integration tests require MongoDB connection")
	}

	cleanupTestData()
	server := setupTestServer()
	defer server.Close()

	// Create a video
	createBody := `{"url":"http://example.com/integration-test.mp4","content_type":"TEMPERATURE","description":"Integration test video"}`
	createResp, err := http.Post(
		server.URL+"/media/videos",
		"application/json",
		bytes.NewReader([]byte(createBody)),
	)
	if err != nil {
		t.Fatalf("create request failed: %v", err)
	}
	defer createResp.Body.Close()

	// Verify the server response
	if createResp.StatusCode != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, createResp.StatusCode)
	}

	var createResult map[string]interface{}
	if err := json.NewDecoder(createResp.Body).Decode(&createResult); err != nil {
		t.Fatalf("failed to decode create response: %v", err)
	}

	videoID, ok := createResult["id"].(string)
	if !ok || videoID == "" {
		t.Fatal("created video should have an id")
	}

	// Get the video by ID
	getResp, err := http.Get(server.URL + "/media/videos/" + videoID)
	if err != nil {
		t.Fatalf("get request failed: %v", err)
	}
	defer getResp.Body.Close()

	// Verify the get response
	if getResp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, getResp.StatusCode)
	}

	var getResult map[string]interface{}
	if err := json.NewDecoder(getResp.Body).Decode(&getResult); err != nil {
		t.Fatalf("failed to decode get response: %v", err)
	}

	// Validate the retrieved video
	if getResult["id"] != videoID {
		t.Errorf("expected id %q, got %q", videoID, getResult["id"])
	}

	if getResult["url"] != "http://example.com/integration-test.mp4" {
		t.Errorf("unexpected url: %v", getResult["url"])
	}
}

// TestIntegration_GetAllVideos tests listing all videos.
func TestIntegration_GetAllVideos(t *testing.T) {
	if testMongoClient == nil {
		t.Skip("Integration tests require MongoDB connection")
	}

	cleanupTestData()
	server := setupTestServer()
	defer server.Close()

	// Create multiple videos
	for i := 1; i <= 3; i++ {
		body := fmt.Sprintf(`{"url":"http://example.com/video%d.mp4","content_type":"SLEEPING","description":"Video %d"}`, i, i)
		resp, err := http.Post(server.URL+"/media/videos", "application/json", bytes.NewReader([]byte(body)))
		if err != nil {
			t.Fatalf("create request %d failed: %v", i, err)
		}
		_ = resp.Body.Close()
	}

	// Get all videos
	resp, err := http.Get(server.URL + "/media/videos")
	if err != nil {
		t.Fatalf("get all request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	videos, ok := result["videos"].([]interface{})
	if !ok {
		t.Fatal("response should contain 'videos' array")
	}

	if len(videos) != 3 {
		t.Errorf("expected 3 videos, got %d", len(videos))
	}
}

// TestIntegration_DeleteVideo tests video deletion.
func TestIntegration_DeleteVideo(t *testing.T) {
	if testMongoClient == nil {
		t.Skip("Integration tests require MongoDB connection")
	}

	cleanupTestData()
	server := setupTestServer()
	defer server.Close()

	// Create a video
	createBody := `{"url":"http://example.com/delete-me.mp4","content_type":"WEIGHTING","description":"Delete test"}`
	createResp, err := http.Post(server.URL+"/media/videos", "application/json", bytes.NewReader([]byte(createBody)))
	if err != nil {
		t.Fatalf("create request failed: %v", err)
	}

	var createResult map[string]interface{}
	if err := json.NewDecoder(createResp.Body).Decode(&createResult); err != nil {
		t.Fatalf("failed to decode create response: %v", err)
	}
	createResp.Body.Close()

	videoID := createResult["id"].(string)

	// Delete the video
	req, _ := http.NewRequest(http.MethodDelete, server.URL+"/media/videos/"+videoID, nil)
	deleteResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("delete request failed: %v", err)
	}
	defer deleteResp.Body.Close()

	if deleteResp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, deleteResp.StatusCode)
	}

	// Verify video is gone
	getResp, err := http.Get(server.URL + "/media/videos/" + videoID)
	if err != nil {
		t.Fatalf("get request failed: %v", err)
	}
	defer getResp.Body.Close()

	if getResp.StatusCode != http.StatusNotFound {
		t.Errorf("expected status %d after deletion, got %d", http.StatusNotFound, getResp.StatusCode)
	}
}

// TestIntegration_GetNonExistentVideo tests 404 handling.
func TestIntegration_GetNonExistentVideo(t *testing.T) {
	if testMongoClient == nil {
		t.Skip("Integration tests require MongoDB connection")
	}

	server := setupTestServer()
	defer server.Close()

	resp, err := http.Get(server.URL + "/media/videos/nonexistent-id-12345")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, resp.StatusCode)
	}
}

// TestIntegration_DatabasePersistence tests that data persists across requests.
func TestIntegration_DatabasePersistence(t *testing.T) {
	if testMongoClient == nil {
		t.Skip("Integration tests require MongoDB connection")
	}

	cleanupTestData()

	// Use a valid 24-char Hex ObjectId to ensure the App's repository can parse it correctly
	validObjectID := "111111111111111111111111"

	// Insert directly into MongoDB
	ctx := context.Background()
	collection := testDB.Collection("videos")
	_, err := collection.InsertOne(ctx, bson.M{
		"_id":          validObjectID,
		"url":          "http://example.com/direct.mp4",
		"content_type": "BREAST_FEEDING",
		"description":  "Direct insert",
		"created_at":   time.Now(),
	})
	if err != nil {
		t.Fatalf("direct insert failed: %v", err)
	}

	// Verify via API
	server := setupTestServer()
	defer server.Close()

	resp, err := http.Get(server.URL + "/media/videos/" + validObjectID)
	if err != nil {
		t.Fatalf("get request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var result map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&result)

	if result["url"] != "http://example.com/direct.mp4" {
		t.Errorf("expected url from direct insert, got: %v", result["url"])
	}
}
