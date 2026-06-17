package database

import (
	"dsr-automation/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Connect(databaseURL string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&models.User{}, &models.DSRReport{}, &models.GitIntegration{}, &models.GitProject{}); err != nil {
		return nil, err
	}

	return db, nil
}
