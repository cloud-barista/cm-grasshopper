package model

import "time"

type JobExecution struct {
	JobID             string    `json:"job_id" gorm:"primaryKey"`
	JobType           string    `json:"job_type"`
	ResourceType      string    `json:"resource_type"`
	ResourceName      string    `json:"resource_name"`
	SourceClusterName string    `json:"source_cluster_name"`
	TargetClusterName string    `json:"target_cluster_name"`
	SourceNamespace   string    `json:"source_namespace"`
	TargetNamespace   string    `json:"target_namespace"`
	Status            string    `json:"status"`
	Progress          int       `json:"progress"`
	Message           string    `json:"message"`
	Metadata          string    `json:"metadata"`
	LogPath           string    `json:"log_path"`
	ErrorMessage      string    `json:"error_message"`
	StartedAt         time.Time `json:"started_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	FinishedAt        time.Time `json:"finished_at"`
}

type JobLogRes struct {
	JobID   string `json:"job_id"`
	Status  string `json:"status"`
	Message string `json:"message"`
	Log     string `json:"log"`
}
