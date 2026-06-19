package models

import "time"

const (
	PlanningActivityAssigned     = "assigned"
	PlanningActivityStatusChange = "status_change"
	PlanningActivityCommentAdded = "comment_added"
	PlanningActivityCompleted    = "completed"
)

type PlanningActivity struct {
	ID                    string    `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID                string    `gorm:"type:uuid;index;not null"`
	PlanningIntegrationID string    `gorm:"type:uuid;index;not null"`
	Provider              string    `gorm:"not null;index"`
	ActivityType          string    `gorm:"not null;index"`
	ItemKey               string    `gorm:"not null;index"`
	ItemTitle             string    `gorm:"type:text"`
	Status                string
	PreviousStatus        string
	Comment               string    `gorm:"type:text"`
	ExternalID            string    `gorm:"not null;uniqueIndex:idx_planning_activity_external"`
	OccurredAt            time.Time `gorm:"index;not null"`
	CreatedAt             time.Time `gorm:"autoCreateTime"`

	PlanningIntegration PlanningIntegration `gorm:"foreignKey:PlanningIntegrationID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	User                User                `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

func (PlanningActivity) TableName() string {
	return "planning_activities"
}
