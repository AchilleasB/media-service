package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/AchilleasB/baby-kliniek/media-service/internal/adapters/handler"
	"github.com/AchilleasB/baby-kliniek/media-service/internal/adapters/middleware"
	"github.com/AchilleasB/baby-kliniek/media-service/internal/adapters/repository"
	"github.com/AchilleasB/baby-kliniek/media-service/internal/config"
	"github.com/AchilleasB/baby-kliniek/media-service/internal/core/services"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	redis "github.com/redis/go-redis/v9"
)

func main() {

	cfg := config.Load()
	ctx := context.Background()

	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer mongoClient.Disconnect(ctx)

	dbName := mongoClient.Database(cfg.MongoDb)
	mongoRepo := repository.NewMongoRepository(dbName)

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddress,
		Password: cfg.RedisPassword,
		DB:       0,
	})

	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Redis is not available yet: %v. App will continue and retry later.", err)
	} else {
		log.Println("Authenticated with Redis successfully")
	}

	authMiddleware := middleware.NewAuthMiddleware(cfg.JWTPublicKey, redisClient)

	mediaService := services.NewVideoService(mongoRepo)

	mediaHandler := handler.NewMediaHandler(mediaService)
	healthHandler := handler.NewHealthHandler(mongoClient)

	mux := http.NewServeMux()

	// Metrics endpoint
	mux.Handle("/metrics", promhttp.Handler())

	// Health endpoints (OpenShift compatible)
	mux.HandleFunc("/health", healthHandler.Health)
	mux.HandleFunc("/health/ready", healthHandler.Ready)
	mux.HandleFunc("/health/live", healthHandler.Live)

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

	// Apply middleware chain: CORS -> Metrics
	corsRouter := middleware.CORSMiddleware(cfg.CORSAllowedOrigins)(mux)
	loggedRouter := middleware.MetricsMiddleware(corsRouter)

	// Create HTTP server with timeouts
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      loggedRouter,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Starting server on :%s", cfg.Port)
		log.Printf("CORS allowed origins: %v", cfg.CORSAllowedOrigins)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Could not start server: %s\n", err)
		}
	}()

	// Setup signal handling for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Block until we receive a signal
	sig := <-quit
	log.Printf("Received signal %v, initiating graceful shutdown...", sig)

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	// Close MongoDB connection
	if err := mongoClient.Disconnect(shutdownCtx); err != nil {
		log.Printf("Error closing MongoDB: %v", err)
	}

	// Close Redis connection
	if err := redisClient.Close(); err != nil {
		log.Printf("Error closing Redis: %v", err)
	}

	log.Println("Server gracefully stopped")
}
