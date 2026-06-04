package models

import "time"

type DSRReport struct {
	ID string `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`

	UserID string `gorm:"type:uuid;index;not null"`
	User   User   `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`

	ReportDate time.Time `gorm:"type:date;index"`

	YesterdayWork string `gorm:"type:text"`
	TodayPlan     string `gorm:"type:text"`
	Blockers      string `gorm:"type:text"`
	Summary       string `gorm:"type:text"`

	AIModel       string
	PromptVersion string

	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (DSRReport) TableName() string {
	return "dsr_reports"
}
