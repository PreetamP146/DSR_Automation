package repository

import (
	"dsr-automation/internal/models"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GitRepository interface {
	GetByUserAndProvider(userID, provider string) (*models.GitIntegration, error)
	ListIntegrationsByUserID(userID string) ([]models.GitIntegration, error)
	Save(integration *models.GitIntegration) error
	SyncProjects(integrationID, userID string, projects []models.GitProject) error
	ListProjectsByUserID(userID, provider string, trackedOnly bool) ([]models.GitProject, error)
	ListAllTrackedProjects() ([]models.GitProject, error)
	GetProjectsByIDsAndUserID(userID string, projectIDs []string) ([]models.GitProject, error)
	CountProjectsByIntegrationID(integrationID string) (total int64, tracked int64, err error)
	SetTrackedProjects(userID string, projectIDs []string) error
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

func (r *gitRepository) ListIntegrationsByUserID(userID string) ([]models.GitIntegration, error) {
	var integrations []models.GitIntegration
	if err := r.db.Where("user_id = ?", userID).Order("provider ASC").Find(&integrations).Error; err != nil {
		return nil, err
	}
	return integrations, nil
}

func (r *gitRepository) Save(integration *models.GitIntegration) error {
	return r.db.Save(integration).Error
}

func (r *gitRepository) SyncProjects(integrationID, userID string, projects []models.GitProject) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var existing []models.GitProject
		if err := tx.Where("git_integration_id = ?", integrationID).Find(&existing).Error; err != nil {
			return err
		}

		trackedByGitID := make(map[string]bool, len(existing))
		existingByGitID := make(map[string]string, len(existing))
		for _, project := range existing {
			trackedByGitID[project.RemoteProjectID] = project.IsTracked
			existingByGitID[project.RemoteProjectID] = project.ID
		}

		incomingGitIDs := make(map[string]struct{}, len(projects))
		for i := range projects {
			projects[i].GitIntegrationID = integrationID
			projects[i].UserID = userID
			if tracked, ok := trackedByGitID[projects[i].RemoteProjectID]; ok {
				projects[i].IsTracked = tracked
				projects[i].ID = existingByGitID[projects[i].RemoteProjectID]
			}
			incomingGitIDs[projects[i].RemoteProjectID] = struct{}{}
		}

		var removedIDs []string
		for _, project := range existing {
			if _, ok := incomingGitIDs[project.RemoteProjectID]; !ok {
				removedIDs = append(removedIDs, project.ID)
			}
		}
		if len(removedIDs) > 0 {
			if err := tx.Where("id IN ?", removedIDs).Delete(&models.GitProject{}).Error; err != nil {
				return err
			}
		}

		if len(projects) == 0 {
			return nil
		}

		return tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "git_integration_id"},
				{Name: "remote_project_id"},
			},
			DoUpdates: clause.AssignmentColumns([]string{
				"name",
				"path",
				"path_with_namespace",
				"web_url",
				"description",
				"default_branch",
				"updated_at",
			}),
		}).Create(&projects).Error
	})
}

func (r *gitRepository) ListProjectsByUserID(userID, provider string, trackedOnly bool) ([]models.GitProject, error) {
	query := r.db.
		Preload("GitIntegration").
		Where("user_id = ?", userID)

	if provider != "" {
		query = query.Joins(
			"JOIN git_integrations ON git_integrations.id = git_projects.git_integration_id AND git_integrations.provider = ?",
			provider,
		)
	}

	if trackedOnly {
		query = query.Where("git_projects.is_tracked = ?", true)
	}

	var projects []models.GitProject
	if err := query.Order("git_projects.name ASC").Find(&projects).Error; err != nil {
		return nil, err
	}
	return projects, nil
}

func (r *gitRepository) ListAllTrackedProjects() ([]models.GitProject, error) {
	var projects []models.GitProject
	if err := r.db.
		Preload("GitIntegration").
		Where("is_tracked = ?", true).
		Order("user_id ASC, name ASC").
		Find(&projects).Error; err != nil {
		return nil, err
	}
	return projects, nil
}

func (r *gitRepository) GetProjectsByIDsAndUserID(userID string, projectIDs []string) ([]models.GitProject, error) {
	if len(projectIDs) == 0 {
		return nil, nil
	}

	var projects []models.GitProject
	if err := r.db.
		Preload("GitIntegration").
		Where("user_id = ? AND id IN ?", userID, projectIDs).
		Order("name ASC").
		Find(&projects).Error; err != nil {
		return nil, err
	}
	return projects, nil
}

func (r *gitRepository) CountProjectsByIntegrationID(integrationID string) (int64, int64, error) {
	var total int64
	if err := r.db.Model(&models.GitProject{}).Where("git_integration_id = ?", integrationID).Count(&total).Error; err != nil {
		return 0, 0, err
	}

	var tracked int64
	if err := r.db.Model(&models.GitProject{}).
		Where("git_integration_id = ? AND is_tracked = ?", integrationID, true).
		Count(&tracked).Error; err != nil {
		return 0, 0, err
	}

	return total, tracked, nil
}

func (r *gitRepository) SetTrackedProjects(userID string, projectIDs []string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if len(projectIDs) > 0 {
			var count int64
			if err := tx.Model(&models.GitProject{}).
				Where("user_id = ? AND id IN ?", userID, projectIDs).
				Count(&count).Error; err != nil {
				return err
			}
			if count != int64(len(projectIDs)) {
				return ErrGitProjectNotFound
			}
		}

		if err := tx.Model(&models.GitProject{}).
			Where("user_id = ?", userID).
			Update("is_tracked", false).Error; err != nil {
			return err
		}

		if len(projectIDs) == 0 {
			return nil
		}

		return tx.Model(&models.GitProject{}).
			Where("user_id = ? AND id IN ?", userID, projectIDs).
			Update("is_tracked", true).Error
	})
}

var ErrGitProjectNotFound = errors.New("one or more git projects not found")
