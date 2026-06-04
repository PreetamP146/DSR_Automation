package config

import (
	"dsr-automation/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ConnectDatabase opens the DB and runs automigrations for known models.
func ConnectDatabase(databaseURL string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&models.User{}, &models.DSRReport{}); err != nil {
		return nil, err
	}

	return db, nil
}
