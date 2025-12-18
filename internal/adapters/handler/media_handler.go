package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/AchilleasB/baby-kliniek/media-service/internal/core/domain"
	"github.com/AchilleasB/baby-kliniek/media-service/internal/core/ports"
	"github.com/google/uuid"
)

type MediaHandler struct {
	videoService ports.VideoService
}

type CreateVideoRequest struct {
	URL         string `json:"url"`
	ContentType string `json:"content_type"`
	Description string `json:"description"`
}

type VideosResponse struct {
	Videos []VideoDTO `json:"videos"`
}

type VideoDTO struct {
	ID          string `json:"id"`
	URL         string `json:"url"`
	ContentType string `json:"content_type"`
	Description string `json:"description"`
}

func NewMediaHandler(video ports.VideoService) *MediaHandler {
	return &MediaHandler{
		videoService: video,
	}
}

func (h *MediaHandler) GetVideos(w http.ResponseWriter, r *http.Request) {
	videos, err := h.videoService.GetVideos(r.Context())
	if err != nil {
		http.Error(w, "Failed to get videos", http.StatusInternalServerError)
		return
	}

	response := VideosResponse{
		Videos: func() []VideoDTO {
			obj := make([]VideoDTO, len(videos))
			for i, v := range videos {
				obj[i] = VideoDTO{
					ID:          v.ID,
					URL:         v.URL,
					ContentType: string(v.ContentType),
					Description: v.Description,
				}
			}
			return obj
		}(),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}

	log.Printf("Retrieved %d videos", len(videos))
}
func (h *MediaHandler) GetOneVideo(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Missing video ID", http.StatusBadRequest)
		return
	}

	video, err := h.videoService.GetVideoByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Video not found", http.StatusNotFound)
		return
	}

	response := VideoDTO{
		ID:          video.ID,
		URL:         video.URL,
		ContentType: string(video.ContentType),
		Description: video.Description,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}
func (h *MediaHandler) CreateVideo(w http.ResponseWriter, r *http.Request) {
	var req CreateVideoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	newVideo := domain.Video{
		ID:          uuid.NewString(),
		URL:         req.URL,
		ContentType: domain.ContentType(req.ContentType),
		Description: req.Description,
	}

	createdVideo, err := h.videoService.CreateVideo(r.Context(), newVideo)
	if err != nil {
		log.Printf("Failed to add video: %v", err)
		http.Error(w, "Failed to create video", http.StatusInternalServerError)
		return
	}

	videoDTO := VideoDTO{
		ID:          createdVideo.ID,
		URL:         createdVideo.URL,
		ContentType: string(createdVideo.ContentType),
		Description: createdVideo.Description,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(videoDTO); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}
func (h *MediaHandler) DeleteVideo(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Missing video ID", http.StatusBadRequest)
		return
	}

	err := h.videoService.DeleteVideo(r.Context(), id)
	if err != nil {
		log.Printf("Failed to delete video %s: %v", id, err)
		http.Error(w, "Failed to delete video", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(map[string]string{
		"message": "Video deleted successfully",
	}); err != nil {
		log.Printf("Failed to write response: %v", err)
	}
}
