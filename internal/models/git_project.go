package models

import "time"

type GitProject struct {
	ID                string `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	GitIntegrationID  string `gorm:"type:uuid;index;not null;uniqueIndex:idx_integration_git_project"`
	UserID            string `gorm:"type:uuid;index;not null"`
	GitProjectID      string `gorm:"not null;uniqueIndex:idx_integration_git_project"`
	Name              string `gorm:"not null"`
	Path              string `gorm:"not null"`
	PathWithNamespace string `gorm:"not null"`
	WebURL            string `gorm:"not null"`
	Description       string
	DefaultBranch     string
	CreatedAt         time.Time `gorm:"autoCreateTime"`
	UpdatedAt         time.Time `gorm:"autoUpdateTime"`

	GitIntegration GitIntegration `gorm:"foreignKey:GitIntegrationID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	User           User           `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

func (GitProject) TableName() string {
	return "git_projects"
}
