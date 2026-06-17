package repository

import (
	"dsr-automation/internal/models"
	"errors"

	"gorm.io/gorm"
)

type GitRepository interface {
	GetByUserAndProvider(userID, provider string) (*models.GitIntegration, error)
	Save(integration *models.GitIntegration) error
	ReplaceProjects(integrationID, userID string, projects []models.GitProject) error
}

type gitRepository struct {
	db *gorm.DB
}

func NewGitRepository(db *gorm.DB) GitRepository {
	return &gitRepository{db: db}
}

func (r *gitRepository) GetByUserAndProvider(userID, provider string) (*models.GitIntegration, error) {
	var integration models.GitIntegration
	err := r.db.Where("user_id = ? AND provider = ?", userID, provider).First(&integration).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &integration, nil
}

func (r *gitRepository) Save(integration *models.GitIntegration) error {
	return r.db.Save(integration).Error
}

func (r *gitRepository) ReplaceProjects(integrationID, userID string, projects []models.GitProject) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("git_integration_id = ?", integrationID).Delete(&models.GitProject{}).Error; err != nil {
			return err
		}
		if len(projects) == 0 {
			return nil
		}
		return tx.Create(&projects).Error
	})
}
