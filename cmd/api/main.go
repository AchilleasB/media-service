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
	"github.com/AchilleasB/baby-kliniek/media-service/internal/core/ports"
	"github.com/AchilleasB/baby-kliniek/media-service/internal/core/services"
)

func main() {

	cfg := config.Load()

	ctx := context.Background()
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer mongoClient.Disconnect(ctx)

	var mediaRepo ports.MediaRepository = repository.NewMongoRepository(mongoClient)

	authMiddleware := middleware.NewAuthMiddleware(cfg.JWTPublicKey)
	mediaService := services.NewVideoService(mediaRepo)

	mediaHandler := handler.NewMediaHandler(mediaService)
	// healthHandler := handler.NewHealthHandler(mongoClient)

	mux := http.NewServeMux()

	// Health endpoints (OpenShift compatible)
	// mux.HandleFunc("/health", healthHandler.Health)
	// mux.HandleFunc("/health/ready", healthHandler.Ready)
	// mux.HandleFunc("/health/live", healthHandler.Live)

	// API endpoints
	mux.HandleFunc("GET /media/videos", mediaHandler.GetVideos)
	mux.HandleFunc("GET /media/videos/{id}", mediaHandler.GetOneVideo)

	mux.Handle("POST /media/videos",
		authMiddleware.RequireRole("ADMIN", http.HandlerFunc(mediaHandler.CreateVideo)),
	)
	mux.Handle("DELETE /media/videos/{id}",
		authMiddleware.RequireRole("ADMIN", http.HandlerFunc(mediaHandler.DeleteVideo)),
	)

	log.Printf("Starting server on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, mux); err != nil {
		log.Fatalf("Could not start server: %s\n", err)
	}
}
