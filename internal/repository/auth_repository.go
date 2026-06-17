package repository

import (
	"dsr-automation/internal/models"
	"errors"

	"gorm.io/gorm"
)

type AuthRepository interface {
	Create(user *models.User) error
	GetByEmail(email string) (*models.User, error)
}

type authRepository struct {
	db *gorm.DB
}

func NewAuthRepository(db *gorm.DB) AuthRepository {
	return &authRepository{db: db}
}

func (r *authRepository) Create(user *models.User) error {
	return r.db.Create(user).Error
}

func (r *authRepository) GetByEmail(email string) (*models.User, error) {
	var user models.User
	err := r.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}
