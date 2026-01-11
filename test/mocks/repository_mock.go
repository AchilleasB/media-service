// Package mocks provides mock implementations for testing.
// Following hexagonal architecture, we mock the PORTS (interfaces)
// to test components in isolation.
package mocks

import (
	"context"
	"errors"
	"sync"

	"github.com/AchilleasB/baby-kliniek/media-service/internal/core/domain"
)

// MockVideoRepository implements ports.VideoRepository for testing.
// It allows seeding test data and injecting errors.
type MockVideoRepository struct {
	mu sync.RWMutex

	// Storage for test data
	videos map[string]*domain.Video

	// Error injection
	GetVideosError    error
	GetVideoByIDError error
	CreateVideoError  error
	DeleteVideoError  error

	// Call tracking
	GetVideosCalled    int
	GetVideoByIDCalled int
	CreateVideoCalled  int
	DeleteVideoCalled  int
}

// NewMockVideoRepository creates a new mock repository with empty state.
func NewMockVideoRepository() *MockVideoRepository {
	return &MockVideoRepository{
		videos: make(map[string]*domain.Video),
	}
}

// SeedVideo adds a video to the mock repository for testing.
func (m *MockVideoRepository) SeedVideo(video *domain.Video) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.videos[video.ID] = video
}

// Reset clears all state from the mock repository.
func (m *MockVideoRepository) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.videos = make(map[string]*domain.Video)

	m.GetVideosError = nil
	m.GetVideoByIDError = nil
	m.CreateVideoError = nil
	m.DeleteVideoError = nil

	m.GetVideosCalled = 0
	m.GetVideoByIDCalled = 0
	m.CreateVideoCalled = 0
	m.DeleteVideoCalled = 0
}

// GetVideos implements ports.VideoRepository.
func (m *MockVideoRepository) GetVideos(ctx context.Context) ([]domain.Video, error) {
	m.mu.Lock()
	m.GetVideosCalled++
	m.mu.Unlock()

	if m.GetVideosError != nil {
		return nil, m.GetVideosError
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	videos := make([]domain.Video, 0, len(m.videos))
	for _, v := range m.videos {
		videos = append(videos, *v)
	}
	return videos, nil
}

// GetVideoByID implements ports.VideoRepository.
func (m *MockVideoRepository) GetVideoByID(ctx context.Context, id string) (*domain.Video, error) {
	m.mu.Lock()
	m.GetVideoByIDCalled++
	m.mu.Unlock()

	if m.GetVideoByIDError != nil {
		return nil, m.GetVideoByIDError
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	video, exists := m.videos[id]
	if !exists {
		return nil, errors.New("video not found")
	}
	return video, nil
}

// CreateVideo implements ports.VideoRepository.
func (m *MockVideoRepository) CreateVideo(ctx context.Context, video domain.Video) (*domain.Video, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.CreateVideoCalled++

	if m.CreateVideoError != nil {
		return nil, m.CreateVideoError
	}

	m.videos[video.ID] = &video
	return &video, nil
}

// DeleteVideo implements ports.VideoRepository.
func (m *MockVideoRepository) DeleteVideo(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.DeleteVideoCalled++

	if m.DeleteVideoError != nil {
		return m.DeleteVideoError
	}

	if _, exists := m.videos[id]; !exists {
		return errors.New("video not found")
	}

	delete(m.videos, id)
	return nil
}
