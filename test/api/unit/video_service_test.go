// Package unit contains unit tests for the video service.
// Unit tests verify that the VideoService correctly delegates to the repository
// and handles errors appropriately.
//
// UNIT TEST PATTERN IN HEXAGONAL ARCHITECTURE:
// ═══════════════════════════════════════════════════════════════════════════════
//
//	┌─────────────────────────────────────────────────────────────────────────┐
//	│                         UNIT TEST ENVIRONMENT                           │
//	│                                                                         │
//	│    ┌──────────────┐        ┌──────────────┐        ┌──────────────┐     │
//	│    │  Test Case   │──────▶│VideoService   │──────▶│    Mock      │     │
//	│    │              │        │   (real)     │        │  Repository  │     │
//	│    └──────────────┘        └──────────────┘        └──────────────┘     │
//	│                                                                         │
//	└─────────────────────────────────────────────────────────────────────────┘
//
// The service under test is REAL, but dependencies are MOCKED.
package unit

import (
	"context"
	"errors"
	"testing"

	"github.com/AchilleasB/baby-kliniek/media-service/internal/core/domain"
	"github.com/AchilleasB/baby-kliniek/media-service/internal/core/services"
	"github.com/AchilleasB/baby-kliniek/media-service/test/mocks"
)

func TestVideoService_GetVideos(t *testing.T) {
	tests := []struct {
		name          string
		setupMock     func(*mocks.MockVideoRepository)
		expectedCount int
		expectError   bool
	}{
		{
			name: "returns_all_videos",
			setupMock: func(m *mocks.MockVideoRepository) {
				m.SeedVideo(mocks.CreateTestVideo("1", "http://example.com/v1.mp4", domain.Temperature, "Temperature video"))
				m.SeedVideo(mocks.CreateTestVideo("2", "http://example.com/v2.mp4", domain.Weighting, "Weighting video"))
			},
			expectedCount: 2,
			expectError:   false,
		},
		{
			name:          "returns_empty_list_when_no_videos",
			setupMock:     func(m *mocks.MockVideoRepository) {},
			expectedCount: 0,
			expectError:   false,
		},
		{
			name: "returns_error_on_repository_failure",
			setupMock: func(m *mocks.MockVideoRepository) {
				m.GetVideosError = errors.New("database connection failed")
			},
			expectedCount: 0,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := mocks.NewMockVideoRepository()
			tt.setupMock(mockRepo)

			service := services.NewVideoService(mockRepo)
			videos, err := service.GetVideos(context.Background())

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(videos) != tt.expectedCount {
				t.Errorf("expected %d videos, got %d", tt.expectedCount, len(videos))
			}

			if mockRepo.GetVideosCalled != 1 {
				t.Errorf("expected GetVideos to be called once, got %d", mockRepo.GetVideosCalled)
			}
		})
	}
}

func TestVideoService_GetVideoByID(t *testing.T) {
	tests := []struct {
		name        string
		videoID     string
		setupMock   func(*mocks.MockVideoRepository)
		expectError bool
		expectedURL string
	}{
		{
			name:    "returns_video_when_found",
			videoID: "video-123",
			setupMock: func(m *mocks.MockVideoRepository) {
				m.SeedVideo(mocks.CreateTestVideo("video-123", "http://example.com/found.mp4", domain.Sleeping, "Found video"))
			},
			expectError: false,
			expectedURL: "http://example.com/found.mp4",
		},
		{
			name:    "returns_error_when_not_found",
			videoID: "nonexistent",
			setupMock: func(m *mocks.MockVideoRepository) {
				// No video seeded
			},
			expectError: true,
		},
		{
			name:    "returns_error_on_repository_failure",
			videoID: "video-123",
			setupMock: func(m *mocks.MockVideoRepository) {
				m.GetVideoByIDError = errors.New("database timeout")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := mocks.NewMockVideoRepository()
			tt.setupMock(mockRepo)

			service := services.NewVideoService(mockRepo)
			video, err := service.GetVideoByID(context.Background(), tt.videoID)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if video.URL != tt.expectedURL {
				t.Errorf("expected URL %q, got %q", tt.expectedURL, video.URL)
			}

			if mockRepo.GetVideoByIDCalled != 1 {
				t.Errorf("expected GetVideoByID to be called once, got %d", mockRepo.GetVideoByIDCalled)
			}
		})
	}
}

func TestVideoService_CreateVideo(t *testing.T) {
	tests := []struct {
		name        string
		video       domain.Video
		setupMock   func(*mocks.MockVideoRepository)
		expectError bool
	}{
		{
			name: "creates_video_successfully",
			video: domain.Video{
				ID:          "new-video-1",
				URL:         "http://example.com/new.mp4",
				ContentType: domain.BreastFeeding,
				Description: "New video",
			},
			setupMock:   func(m *mocks.MockVideoRepository) {},
			expectError: false,
		},
		{
			name: "returns_error_on_repository_failure",
			video: domain.Video{
				ID:          "new-video-2",
				URL:         "http://example.com/fail.mp4",
				ContentType: domain.DiaperChange,
				Description: "Fail video",
			},
			setupMock: func(m *mocks.MockVideoRepository) {
				m.CreateVideoError = errors.New("duplicate key error")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := mocks.NewMockVideoRepository()
			tt.setupMock(mockRepo)

			service := services.NewVideoService(mockRepo)
			created, err := service.CreateVideo(context.Background(), tt.video)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if created.ID != tt.video.ID {
				t.Errorf("expected ID %q, got %q", tt.video.ID, created.ID)
			}

			if mockRepo.CreateVideoCalled != 1 {
				t.Errorf("expected CreateVideo to be called once, got %d", mockRepo.CreateVideoCalled)
			}
		})
	}
}

func TestVideoService_DeleteVideo(t *testing.T) {
	tests := []struct {
		name        string
		videoID     string
		setupMock   func(*mocks.MockVideoRepository)
		expectError bool
	}{
		{
			name:    "deletes_video_successfully",
			videoID: "delete-me",
			setupMock: func(m *mocks.MockVideoRepository) {
				m.SeedVideo(mocks.CreateTestVideo("delete-me", "http://example.com/delete.mp4", domain.BottleFeeding, "Delete me"))
			},
			expectError: false,
		},
		{
			name:    "returns_error_when_not_found",
			videoID: "nonexistent",
			setupMock: func(m *mocks.MockVideoRepository) {
				// No video seeded
			},
			expectError: true,
		},
		{
			name:    "returns_error_on_repository_failure",
			videoID: "fail-delete",
			setupMock: func(m *mocks.MockVideoRepository) {
				m.SeedVideo(mocks.CreateTestVideo("fail-delete", "http://example.com/fail.mp4", domain.Sleeping, "Fail delete"))
				m.DeleteVideoError = errors.New("connection reset")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := mocks.NewMockVideoRepository()
			tt.setupMock(mockRepo)

			service := services.NewVideoService(mockRepo)
			err := service.DeleteVideo(context.Background(), tt.videoID)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if mockRepo.DeleteVideoCalled != 1 {
				t.Errorf("expected DeleteVideo to be called once, got %d", mockRepo.DeleteVideoCalled)
			}
		})
	}
}

func TestVideoService_ContextCancellation(t *testing.T) {
	mockRepo := mocks.NewMockVideoRepository()
	service := services.NewVideoService(mockRepo)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Note: The mock doesn't check context, but real implementations should.
	// This test documents the expected behavior.
	_, err := service.GetVideos(ctx)
	if err != nil {
		// If the mock respects context, this would fail with context.Canceled
		t.Logf("Context cancellation handling: %v", err)
	}
}
