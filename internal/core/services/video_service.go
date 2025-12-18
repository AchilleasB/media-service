package services

import (
	"context"

	"github.com/AchilleasB/baby-kliniek/media-service/internal/core/domain"
	"github.com/AchilleasB/baby-kliniek/media-service/internal/core/ports"
)

type VideoService struct {
	repo ports.VideoRepository
}

func NewVideoService(repo ports.VideoRepository) *VideoService {
	return &VideoService{
		repo: repo,
	}
}

func (s *VideoService) GetVideos(ctx context.Context) ([]domain.Video, error) {
	return s.repo.GetVideos(ctx)
}

func (s *VideoService) GetVideoByID(ctx context.Context, id string) (*domain.Video, error) {
	return s.repo.GetVideoByID(ctx, id)
}

func (s *VideoService) CreateVideo(ctx context.Context, video domain.Video) (*domain.Video, error) {
	return s.repo.CreateVideo(ctx, video)
}

func (s *VideoService) DeleteVideo(ctx context.Context, id string) error {
	return s.repo.DeleteVideo(ctx, id)
}
