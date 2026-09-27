package toonify

import "time"

type JobStatus string

const (
	StatusPending    JobStatus = "pending"
	StatusProcessing JobStatus = "processing"
	StatusCompleted  JobStatus = "completed"
	StatusFailed     JobStatus = "failed"
)

type ToonifyJob struct {
	ID string `json:"id" bson:"_id"`

	UserID string `json:"user_id,omitempty" bson:"user_id,omitempty"`

	Status JobStatus `json:"status" bson:"status"`

	Style string `json:"style" bson:"style"`

	InputObject  string `json:"input_object" bson:"input_object"`
	OutputObject string `json:"output_object" bson:"output_object"`

	Error string `json:"error,omitempty" bson:"error,omitempty"`

	CreatedAt   time.Time  `json:"created_at" bson:"created_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty" bson:"completed_at,omitempty"`
}

type CreateUploadRequest struct {
	ContentType string `json:"content_type"`
	Style       string `json:"style"`
}

type CreateUploadResponse struct {
	JobID     string `json:"job_id"`
	UploadURL string `json:"upload_url"`
	Object    string `json:"object"`
}

type JobResponse struct {
	JobID  string    `json:"job_id"`
	Status JobStatus `json:"status"`
	Style  string    `json:"style"`

	DownloadURL string `json:"download_url,omitempty"`

	Error string `json:"error,omitempty"`
}
