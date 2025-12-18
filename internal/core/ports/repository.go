package ports

import (
	"context"

	"github.com/AchilleasB/baby-kliniek/media-service/internal/core/domain"
)

type VideoRepository interface {
	GetVideos(ctx context.Context) ([]domain.Video, error)
	GetVideoByID(ctx context.Context, id string) (*domain.Video, error)
	CreateVideo(ctx context.Context, video domain.Video) (*domain.Video, error)
	DeleteVideo(ctx context.Context, id string) error
}
