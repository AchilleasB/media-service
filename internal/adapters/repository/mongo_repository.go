package repository

import (
	"context"
	"errors"

	"github.com/AchilleasB/baby-kliniek/media-service/internal/config"
	"github.com/AchilleasB/baby-kliniek/media-service/internal/core/domain"
	"github.com/AchilleasB/baby-kliniek/media-service/internal/core/ports"
	"github.com/sony/gobreaker"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type MongoRepository struct {
	mongoVideoCollection *mongo.Collection
	cb                   *gobreaker.CircuitBreaker
}

var _ ports.VideoRepository = (*MongoRepository)(nil)

func NewMongoRepository(mongodb *mongo.Database) *MongoRepository {
	vidCollection := mongodb.Collection("videos")

	// Configure circuit breaker for MongoDB operations
	cb := config.NewCircuitBreaker("MongoDB")

	return &MongoRepository{
		mongoVideoCollection: vidCollection,
		cb:                   cb,
	}
}

func (r *MongoRepository) GetVideos(ctx context.Context) ([]domain.Video, error) {
	result, err := r.cb.Execute(func() (interface{}, error) {
		filter := bson.M{}

		// Cursor is a MongoDB stream that we can iterate over
		cursor, err := r.mongoVideoCollection.Find(ctx, filter)
		if err != nil {
			return nil, err
		}
		defer cursor.Close(ctx)

		videos := make([]domain.Video, 0)

		if err := cursor.All(ctx, &videos); err != nil {
			return nil, err
		}

		return videos, nil
	})
	if err != nil {
		return nil, err
	}
	return result.([]domain.Video), nil
}

func (r *MongoRepository) GetVideoByID(ctx context.Context, id string) (*domain.Video, error) {
	result, err := r.cb.Execute(func() (interface{}, error) {
		var video domain.Video

		filter := bson.M{"_id": id}

		err := r.mongoVideoCollection.FindOne(ctx, filter).Decode(&video)
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				return nil, errors.New("video not found")
			}
			return nil, err
		}

		return &video, nil
	})
	if err != nil {
		return nil, err
	}
	return result.(*domain.Video), nil
}

func (r *MongoRepository) CreateVideo(ctx context.Context, video domain.Video) (*domain.Video, error) {
	_, err := r.cb.Execute(func() (interface{}, error) {
		_, err := r.mongoVideoCollection.InsertOne(ctx, video)
		if err != nil {
			return nil, err
		}

		return &video, nil
	})
	if err != nil {
		return nil, err
	}
	return &video, nil
}

func (r *MongoRepository) DeleteVideo(ctx context.Context, id string) error {
	_, err := r.cb.Execute(func() (interface{}, error) {
		filter := bson.M{"_id": id}

		result, err := r.mongoVideoCollection.DeleteOne(ctx, filter)
		if err != nil {
			return nil, err
		}

		if result.DeletedCount == 0 {
			return nil, errors.New("video not found")
		}

		return nil, nil
	})
	return err
}
