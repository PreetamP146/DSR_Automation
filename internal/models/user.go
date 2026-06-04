package models

import "time"

type User struct {
	ID           string    `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Name         string    `gorm:"not null"`
	Email        string    `gorm:"uniqueIndex;not null"`
	PasswordHash string    `gorm:"not null"`
	CreatedAt    time.Time `gorm:"autoCreateTime"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime"`

	DSRReports []DSRReport `gorm:"foreignKey:UserID;references:ID"`
}

func (User) TableName() string {
	return "users"
}
