package job

import "time"

type Status string

const (
	StatusPending    Status = "pending"
	StatusProcessing Status = "processing"
	StatusCompleted  Status = "completed"
	StatusFailed     Status = "failed"
)

type Info struct {
	JobID        string                 `json:"job_id"`
	JobType      string                 `json:"job_type"`
	ResourceType string                 `json:"resource_type"`
	ResourceName string                 `json:"resource_name"`
	Status       Status                 `json:"status"`
	Progress     int                    `json:"progress"`
	Message      string                 `json:"message"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	LogPath      string                 `json:"log_path"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	StartedAt    time.Time              `json:"started_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	FinishedAt   time.Time              `json:"finished_at"`
}
