package models

import "time"

type LocalGitRepository struct {
	ID            string    `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID        string    `gorm:"type:uuid;index;not null;uniqueIndex:idx_user_local_repo_path"`
	Name          string    `gorm:"not null"`
	Path          string    `gorm:"not null;uniqueIndex:idx_user_local_repo_path"`
	IsTracked     bool      `gorm:"not null;default:true;index"`
	LastScannedAt *time.Time
	CreatedAt     time.Time `gorm:"autoCreateTime"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime"`

	User User `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

func (LocalGitRepository) TableName() string {
	return "local_git_repositories"
}
