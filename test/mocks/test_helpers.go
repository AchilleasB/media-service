// Package mocks provides test helpers and utilities.
package mocks

import (
	"bytes"
	"encoding/json"
	"io"
	"time"

	"github.com/AchilleasB/baby-kliniek/media-service/internal/core/domain"
)

// JSONReader creates an io.Reader from a JSON string for use in HTTP requests.
func JSONReader(jsonStr string) io.Reader {
	return bytes.NewReader([]byte(jsonStr))
}

// CreateTestVideo creates a domain.Video with sensible defaults for testing.
func CreateTestVideo(id, url string, contentType domain.ContentType, description string) *domain.Video {
	return &domain.Video{
		ID:          id,
		URL:         url,
		ContentType: contentType,
		Description: description,
		CreatedAt:   time.Now(),
	}
}

// MustMarshal marshals a value to JSON, panicking on error.
// Use only in tests where errors indicate bugs.
func MustMarshal(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}

// ContentTypes provides easy access to all content types for testing.
var ContentTypes = struct {
	Temperature   domain.ContentType
	Weighting     domain.ContentType
	BreastFeeding domain.ContentType
	BottleFeeding domain.ContentType
	DiaperChange  domain.ContentType
	Sleeping      domain.ContentType
}{
	Temperature:   domain.Temperature,
	Weighting:     domain.Weighting,
	BreastFeeding: domain.BreastFeeding,
	BottleFeeding: domain.BottleFeeding,
	DiaperChange:  domain.DiaperChange,
	Sleeping:      domain.Sleeping,
}
