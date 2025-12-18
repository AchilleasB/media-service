package repository

import (
	"context"
	"errors"

	"github.com/AchilleasB/baby-kliniek/media-service/internal/core/domain"
	"github.com/AchilleasB/baby-kliniek/media-service/internal/core/ports"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type MongoRepository struct {
	mongoVideoCollection *mongo.Collection
}

var _ ports.VideoRepository = (*MongoRepository)(nil)

func NewMongoRepository(mongodb *mongo.Client) *MongoRepository {
	vidCollection := mongodb.Database("media").Collection("videos")
	return &MongoRepository{
		mongoVideoCollection: vidCollection,
	}
}

func (r *MongoRepository) GetVideos(ctx context.Context) ([]domain.Video, error) {
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
}

func (r *MongoRepository) GetVideoByID(ctx context.Context, id string) (*domain.Video, error) {
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
}

func (r *MongoRepository) CreateVideo(ctx context.Context, video domain.Video) (*domain.Video, error) {
	_, err := r.mongoVideoCollection.InsertOne(ctx, video)
	if err != nil {
		return nil, err
	}

	return &video, nil
}

func (r *MongoRepository) DeleteVideo(ctx context.Context, id string) error {
	filter := bson.M{"_id": id}

	result, err := r.mongoVideoCollection.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return errors.New("video not found")
	}

	return nil
}
