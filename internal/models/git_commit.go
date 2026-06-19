package models

import "time"

type GitCommit struct {
	ID           string    `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	GitProjectID string    `gorm:"type:uuid;index;not null;uniqueIndex:idx_project_sha"`
	UserID       string    `gorm:"type:uuid;index;not null"`
	SHA          string    `gorm:"not null;uniqueIndex:idx_project_sha"`
	Message      string    `gorm:"type:text;not null"`
	Author       string    `gorm:"not null"`
	CommittedAt  time.Time `gorm:"index;not null"`
	URL          string
	CreatedAt    time.Time `gorm:"autoCreateTime"`

	GitProject GitProject `gorm:"foreignKey:GitProjectID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	User       User       `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

func (GitCommit) TableName() string {
	return "git_commits"
}
