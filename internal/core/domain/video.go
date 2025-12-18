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
	ID          string      `json:"id" bson:"_id"`
	URL         string      `json:"url" bson:"url"`
	ContentType ContentType `json:"content_type" bson:"content_type"`
	Description string      `json:"description" bson:"description"`
	CreatedAt   time.Time   `json:"created_at" bson:"created_at"`
}
