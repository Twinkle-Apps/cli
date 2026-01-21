package api

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type APITime struct {
	time.Time
}

func (t *APITime) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	raw := strings.Trim(string(data), "\"")
	if raw == "" {
		return nil
	}
	parsed, err := parseAPITime(raw)
	if err != nil {
		return err
	}
	t.Time = parsed
	return nil
}

func parseAPITime(value string) (time.Time, error) {
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.999999999",
		"2006-01-02T15:04:05",
	}
	for _, layout := range layouts {
		if layout == "2006-01-02T15:04:05.999999999" || layout == "2006-01-02T15:04:05" {
			parsed, err := time.ParseInLocation(layout, value, time.UTC)
			if err == nil {
				return parsed, nil
			}
			continue
		}
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("parse time %q", value)
}

type BuildID struct {
	value int
}

func (b BuildID) Int() int {
	return b.value
}

func (b *BuildID) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	if len(data) > 0 && data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		value, err := strconv.Atoi(s)
		if err != nil {
			return fmt.Errorf("parse build id %q: %w", s, err)
		}
		b.value = value
		return nil
	}

	var num json.Number
	if err := json.Unmarshal(data, &num); err != nil {
		return err
	}
	value, err := num.Int64()
	if err != nil {
		return fmt.Errorf("parse build id %q: %w", num.String(), err)
	}
	b.value = int(value)
	return nil
}

func (b BuildID) MarshalJSON() ([]byte, error) {
	return json.Marshal(b.value)
}

type Appcast struct {
	FeedURL     string   `json:"feed_url"`
	Message     string   `json:"message"`
	PublishedAt *APITime `json:"published_at"`
	Status      string   `json:"status"`
	URL         *string  `json:"url"`
}

type Build struct {
	BuildNumber string         `json:"build_number"`
	ID          int            `json:"id"`
	InsertedAt  APITime        `json:"inserted_at"`
	Metadata    *BuildMetadata `json:"metadata"`
	Status      string         `json:"status"`
	UpdatedAt   APITime        `json:"updated_at"`
	Version     string         `json:"version"`
}

type BuildMetadata struct {
	BuildNumber          *string                `json:"build_number"`
	BuildSize            *int                   `json:"build_size"`
	BuildVersion         *string                `json:"build_version"`
	MinimumSystemVersion *string                `json:"minimum_system_version"`
	ProcessingErrors     map[string]interface{} `json:"processing_errors"`
	Signature            *string                `json:"signature"`
}

type BuildResponse struct {
	Appcast Appcast `json:"appcast"`
	Build   Build   `json:"build"`
}

type BuildUploadParams struct {
	BuildNumber string `json:"build_number,omitempty"`
	ContentType string `json:"content_type,omitempty"`
	Version     string `json:"version,omitempty"`
}

type BuildUploadRequest struct {
	Build BuildUploadParams `json:"build"`
}

type BuildUploadResponse struct {
	BuildID     BuildID `json:"build_id"`
	CompleteURL string  `json:"complete_url"`
	StatusURL   string  `json:"status_url"`
	UploadURL   string  `json:"upload_url"`
	WaitURL     string  `json:"wait_url"`
}

type BuildUploadCompleteResponse struct {
	BuildID   BuildID `json:"build_id"`
	StatusURL string  `json:"status_url"`
	WaitURL   string  `json:"wait_url"`
}

type ErrorResponse struct {
	Details map[string]interface{} `json:"details"`
	Error   string                 `json:"error"`
}
