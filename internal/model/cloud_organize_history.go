package model

import "time"

// CloudOrganizeHistory 是一次云盘整理的记录。
type CloudOrganizeHistory struct {
	Base
	SourceAccountID string     `gorm:"size:36" json:"source_account_id"`
	SourceProvider  string     `gorm:"size:32" json:"source_provider"`
	SourcePath      string     `gorm:"size:1024" json:"source_path"`
	TargetAccountID string     `gorm:"size:36" json:"target_account_id"`
	TargetProvider  string     `gorm:"size:32" json:"target_provider"`
	TargetPath      string     `gorm:"size:1024" json:"target_path"`
	TotalFiles      int        `json:"total_files"`
	SuccessFiles    int        `json:"success_files"`
	FailedFiles     int        `json:"failed_files"`
	Status          string     `gorm:"size:16" json:"status"` // running/completed/failed/cancelled
	ErrorMessage    string     `gorm:"type:text" json:"error_message"`
	StartedAt       time.Time  `json:"started_at"`
	CompletedAt     *time.Time `json:"completed_at"`
}
