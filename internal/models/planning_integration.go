package models

import "time"

type PlanningIntegration struct {
	ID          string `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID      string `gorm:"type:uuid;index;not null;uniqueIndex:idx_user_planning_provider"`
	Provider    string `gorm:"not null;uniqueIndex:idx_user_planning_provider"`
	BaseURL     string
	Email       string
	APIToken    string `gorm:"not null"`
	APIKey      string
	AccountID   string `gorm:"not null"`
	DisplayName string `gorm:"not null"`
	LastSyncAt  *time.Time
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`

	User User `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

func (PlanningIntegration) TableName() string {
	return "planning_integrations"
}
