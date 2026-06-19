package models

import "time"

type LocalGitCommit struct {
	ID                   string    `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	LocalGitRepositoryID string    `gorm:"type:uuid;index;not null;uniqueIndex:idx_local_repo_sha"`
	UserID               string    `gorm:"type:uuid;index;not null"`
	SHA                  string    `gorm:"not null;uniqueIndex:idx_local_repo_sha"`
	Message              string    `gorm:"type:text;not null"`
	Branch               string    `gorm:"not null"`
	RepositoryName       string    `gorm:"not null"`
	Author               string    `gorm:"not null"`
	CommittedAt          time.Time `gorm:"index;not null"`
	CreatedAt            time.Time `gorm:"autoCreateTime"`

	LocalGitRepository LocalGitRepository `gorm:"foreignKey:LocalGitRepositoryID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	User               User               `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

func (LocalGitCommit) TableName() string {
	return "local_git_commits"
}
