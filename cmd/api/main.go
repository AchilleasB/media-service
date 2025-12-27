package main

import (
	"context"
	"log"
	"net/http"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/AchilleasB/baby-kliniek/media-service/internal/adapters/handler"
	"github.com/AchilleasB/baby-kliniek/media-service/internal/adapters/middleware"
	"github.com/AchilleasB/baby-kliniek/media-service/internal/adapters/repository"
	"github.com/AchilleasB/baby-kliniek/media-service/internal/config"
	"github.com/AchilleasB/baby-kliniek/media-service/internal/core/services"
	"github.com/redis/go-redis/v9"
)

func main() {

	cfg := config.Load()
	ctx := context.Background()

	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer mongoClient.Disconnect(ctx)

	mongoRepo := repository.NewMongoRepository(mongoClient)

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddress,
		Password: cfg.RedisPassword,
		DB:       0,
	})

	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("failed to connect to redis: %v", err)
	}
	log.Println("Authenticated with Redis successfully")

	authMiddleware := middleware.NewAuthMiddleware(cfg.JWTPublicKey, redisClient)

	mediaService := services.NewVideoService(mongoRepo)

	mediaHandler := handler.NewMediaHandler(mediaService)

	mux := http.NewServeMux()

	// API endpoints
	mux.Handle("GET /media/videos",
		authMiddleware.RequireRole([]string{"ADMIN", "PARENT"}, http.HandlerFunc(mediaHandler.GetVideos)),
	)

	mux.Handle("GET /media/videos/{id}",
		authMiddleware.RequireRole([]string{"ADMIN", "PARENT"}, http.HandlerFunc(mediaHandler.GetOneVideo)),
	)

	mux.Handle("POST /media/videos",
		authMiddleware.RequireRole([]string{"ADMIN"}, http.HandlerFunc(mediaHandler.CreateVideo)),
	)
	mux.Handle("DELETE /media/videos/{id}",
		authMiddleware.RequireRole([]string{"ADMIN"}, http.HandlerFunc(mediaHandler.DeleteVideo)),
	)

	log.Printf("Starting server on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, mux); err != nil {
		log.Fatalf("Could not start server: %s\n", err)
	}
}
