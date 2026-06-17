package models

import "time"

type GitIntegration struct {
	ID          string `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID      string `gorm:"type:uuid;index;not null;uniqueIndex:idx_user_provider"`
	Provider    string `gorm:"not null;uniqueIndex:idx_user_provider"`
	BaseURL     string `gorm:"not null"`
	AccessToken string `gorm:"not null"`
	GitUsername string `gorm:"not null"`
	GitUserID   string `gorm:"not null"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`

	User     User         `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Projects []GitProject `gorm:"foreignKey:GitIntegrationID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

func (GitIntegration) TableName() string {
	return "git_integrations"
}
