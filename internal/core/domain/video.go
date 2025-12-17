package domain

import "time"

type ContentType string

const (
	Temperature   ContentType = "TEMPERATURE	"
	Weighting     ContentType = "WEIGHTING"
	BreastFeeding ContentType = "BREAST_FEEDING"
	BottleFeeding ContentType = "BOTTLE_FEEDING"
	DiaperChange  ContentType = "DIAPER_CHANGE"
	Sleeping      ContentType = "SLEEPING"
)

type Video struct {
	ID          string      `json:"id"`
	URL         string      `json:"url"`
	ContentType ContentType `json:"content_type"`
	Description string      `json:"description"`
	CreatedAt   time.Time   `json:"created_at"`
}
