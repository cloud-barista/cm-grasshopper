package job

import "time"

type ExecutionResponse struct {
	JobID             string    `json:"job_id"`
	JobType           string    `json:"job_type"`
	ResourceType      string    `json:"resource_type"`
	ResourceName      string    `json:"resource_name"`
	SourceClusterName string    `json:"source_cluster_name,omitempty"`
	TargetClusterName string    `json:"target_cluster_name,omitempty"`
	SourceNamespace   string    `json:"source_namespace,omitempty"`
	TargetNamespace   string    `json:"target_namespace,omitempty"`
	Status            string    `json:"status"`
	Progress          int       `json:"progress"`
	Message           string    `json:"message"`
	CurrentStage      string    `json:"current_stage,omitempty"`
	Metadata          string    `json:"metadata"`
	LogPath           string    `json:"log_path"`
	ErrorMessage      string    `json:"error_message"`
	StartedAt         time.Time `json:"started_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	FinishedAt        time.Time `json:"finished_at"`
}

type StartResponse struct {
	JobID        string                 `json:"job_id"`
	JobType      string                 `json:"job_type"`
	ResourceType string                 `json:"resource_type"`
	ResourceName string                 `json:"resource_name"`
	Status       string                 `json:"status"`
	Progress     int                    `json:"progress"`
	Message      string                 `json:"message"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	LogPath      string                 `json:"log_path"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	StartedAt    time.Time              `json:"started_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	FinishedAt   time.Time              `json:"finished_at"`
}

type LogResponse struct {
	JobID   string `json:"job_id"`
	Status  string `json:"status"`
	Message string `json:"message"`
	Log     string `json:"log"`
}
