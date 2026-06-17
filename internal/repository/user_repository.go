package repository

import (
	"dsr-automation/internal/models"
)

type UserRepository interface {
	Create(user *models.User) error
	GetByEmail(email string) (*models.User, error)
}
