package repository

import (
	"context"

	"github.com/AchilleasB/baby-kliniek/media-service/internal/core/domain"
	"go.mongodb.org/mongo-driver/mongo"
)

type MongoRepo struct {
	mongoCollection *mongo.Collection
}

func NewMongoRepository(mongodb *mongo.Client) *MongoRepo {
	collection := mongodb.Database("media_service").Collection("media")
	return &MongoRepo{
		mongoCollection: collection,
	}
}

func (r *MongoRepo) GetVideos(ctx context.Context) ([]domain.Video, error) {
	// Implementation here
	return nil, nil
}

func (r *MongoRepo) GetVideoByID(ctx context.Context, id string) (*domain.Video, error) {
	// Implementation here
	return nil, nil
}

func (r *MongoRepo) CreateVideo(ctx context.Context, video domain.Video) (*domain.Video, error) {
	// Implementation here
	return nil, nil
}

func (r *MongoRepo) DeleteVideo(ctx context.Context, id string) error {
	// Implementation here
	return nil
}
