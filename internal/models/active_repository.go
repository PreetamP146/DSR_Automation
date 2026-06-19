package models

import "time"

type ActiveRepository struct {
	ID             string    `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID         string    `gorm:"type:uuid;uniqueIndex;not null"`
	RepositoryName string    `gorm:"not null"`
	RepositoryPath string    `gorm:"not null"`
	Source         string    `gorm:"not null;default:local"`
	UpdatedAt      time.Time `gorm:"autoUpdateTime"`
	CreatedAt      time.Time `gorm:"autoCreateTime"`

	User User `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

func (ActiveRepository) TableName() string {
	return "active_repositories"
}
