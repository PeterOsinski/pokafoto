package model

import "time"

type JobStatus string

const (
	JobStatusQueued     JobStatus = "queued"
	JobStatusProcessing JobStatus = "processing"
	JobStatusCompleted  JobStatus = "completed"
	JobStatusSkipped    JobStatus = "skipped"
	JobStatusFailed     JobStatus = "failed"
)

type JobStage string

const (
	JobStageHashing    JobStage = "hashing"
	JobStageDedup      JobStage = "dedup"
	JobStageStoring    JobStage = "storing"
	JobStageExif       JobStage = "exif"
	JobStageThumbnails JobStage = "thumbnails"
)

type UploadJob struct {
	ID               string    `json:"id" db:"id"`
	BatchID          string    `json:"batch_id" db:"batch_id"`
	UserID           string    `json:"user_id" db:"user_id"`
	Filename         string    `json:"filename" db:"filename"`
	SizeBytes        int64     `json:"size_bytes" db:"size_bytes"`
	TempPath         string    `json:"temp_path" db:"temp_path"`
	FolderID         *string   `json:"folder_id,omitempty" db:"folder_id"`
	SkipNameSizeDedup bool     `json:"skip_name_size_dedup" db:"skip_name_size_dedup"`
	Status           JobStatus `json:"status" db:"status"`
	Stage            *JobStage `json:"stage,omitempty" db:"stage"`
	Progress         float64   `json:"progress" db:"progress"`
	Error            *string   `json:"error,omitempty" db:"error"`
	Reason           *string   `json:"reason,omitempty" db:"reason"`
	FileID           *string   `json:"file_id,omitempty" db:"file_id"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}
