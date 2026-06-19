package repository

import (
	"dsr-automation/internal/models"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ActivityRepository interface {
	// Local git repositories
	SaveLocalRepository(repo *models.LocalGitRepository) error
	GetLocalRepositoryByIDAndUserID(id, userID string) (*models.LocalGitRepository, error)
	ListLocalRepositoriesByUserID(userID string, trackedOnly bool) ([]models.LocalGitRepository, error)
	ListAllTrackedLocalRepositories() ([]models.LocalGitRepository, error)
	DeleteLocalRepository(id, userID string) error
	SetTrackedLocalRepositories(userID string, repositoryIDs []string) error
	UpdateLocalRepositoryLastScanned(id string, scannedAt time.Time) error

	// Local git commits
	SaveNewLocalCommits(commits []models.LocalGitCommit) (int64, error)
	ListLocalCommitsByUserAndDateRange(userID string, since, until time.Time) ([]models.LocalGitCommit, error)
	GetLatestLocalCommitTime(repositoryID string) (*time.Time, error)

	// Planning tool integrations (Jira, Trello, etc.)
	GetPlanningIntegrationByUserAndProvider(userID, provider string) (*models.PlanningIntegration, error)
	ListPlanningIntegrationsByUserID(userID string) ([]models.PlanningIntegration, error)
	SavePlanningIntegration(integration *models.PlanningIntegration) error
	DeletePlanningIntegration(userID, provider string) error
	ListAllPlanningIntegrations() ([]models.PlanningIntegration, error)

	// Planning activities
	SaveNewPlanningActivities(activities []models.PlanningActivity) (int64, error)
	ListPlanningActivitiesByUserAndDateRange(userID string, since, until time.Time) ([]models.PlanningActivity, error)

	// Active repository
	UpsertActiveRepository(repo *models.ActiveRepository) error
	GetActiveRepositoryByUserID(userID string) (*models.ActiveRepository, error)
}

type activityRepository struct {
	db *gorm.DB
}

func NewActivityRepository(db *gorm.DB) ActivityRepository {
	return &activityRepository{db: db}
}

func (r *activityRepository) SaveLocalRepository(repo *models.LocalGitRepository) error {
	return r.db.Save(repo).Error
}

func (r *activityRepository) GetLocalRepositoryByIDAndUserID(id, userID string) (*models.LocalGitRepository, error) {
	var repo models.LocalGitRepository
	err := r.db.Where("id = ? AND user_id = ?", id, userID).First(&repo).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &repo, nil
}

func (r *activityRepository) ListLocalRepositoriesByUserID(userID string, trackedOnly bool) ([]models.LocalGitRepository, error) {
	query := r.db.Where("user_id = ?", userID)
	if trackedOnly {
		query = query.Where("is_tracked = ?", true)
	}

	var repos []models.LocalGitRepository
	if err := query.Order("name ASC").Find(&repos).Error; err != nil {
		return nil, err
	}
	return repos, nil
}

func (r *activityRepository) ListAllTrackedLocalRepositories() ([]models.LocalGitRepository, error) {
	var repos []models.LocalGitRepository
	if err := r.db.Where("is_tracked = ?", true).Order("user_id ASC, name ASC").Find(&repos).Error; err != nil {
		return nil, err
	}
	return repos, nil
}

func (r *activityRepository) DeleteLocalRepository(id, userID string) error {
	result := r.db.Where("id = ? AND user_id = ?", id, userID).Delete(&models.LocalGitRepository{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrLocalGitRepositoryNotFound
	}
	return nil
}

func (r *activityRepository) SetTrackedLocalRepositories(userID string, repositoryIDs []string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if len(repositoryIDs) > 0 {
			var count int64
			if err := tx.Model(&models.LocalGitRepository{}).
				Where("user_id = ? AND id IN ?", userID, repositoryIDs).
				Count(&count).Error; err != nil {
				return err
			}
			if count != int64(len(repositoryIDs)) {
				return ErrLocalGitRepositoryNotFound
			}
		}

		if err := tx.Model(&models.LocalGitRepository{}).
			Where("user_id = ?", userID).
			Update("is_tracked", false).Error; err != nil {
			return err
		}

		if len(repositoryIDs) == 0 {
			return nil
		}

		return tx.Model(&models.LocalGitRepository{}).
			Where("user_id = ? AND id IN ?", userID, repositoryIDs).
			Update("is_tracked", true).Error
	})
}

func (r *activityRepository) UpdateLocalRepositoryLastScanned(id string, scannedAt time.Time) error {
	return r.db.Model(&models.LocalGitRepository{}).
		Where("id = ?", id).
		Update("last_scanned_at", scannedAt).Error
}

func (r *activityRepository) SaveNewLocalCommits(commits []models.LocalGitCommit) (int64, error) {
	if len(commits) == 0 {
		return 0, nil
	}

	result := r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "local_git_repository_id"},
			{Name: "sha"},
		},
		DoNothing: true,
	}).Create(&commits)
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

func (r *activityRepository) ListLocalCommitsByUserAndDateRange(userID string, since, until time.Time) ([]models.LocalGitCommit, error) {
	var commits []models.LocalGitCommit
	if err := r.db.
		Where("user_id = ? AND committed_at >= ? AND committed_at <= ?", userID, since, until).
		Order("committed_at ASC").
		Find(&commits).Error; err != nil {
		return nil, err
	}
	return commits, nil
}

func (r *activityRepository) GetLatestLocalCommitTime(repositoryID string) (*time.Time, error) {
	var commit models.LocalGitCommit
	err := r.db.
		Where("local_git_repository_id = ?", repositoryID).
		Order("committed_at DESC").
		First(&commit).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &commit.CommittedAt, nil
}

func (r *activityRepository) GetPlanningIntegrationByUserAndProvider(userID, provider string) (*models.PlanningIntegration, error) {
	var integration models.PlanningIntegration
	err := r.db.Where("user_id = ? AND provider = ?", userID, provider).First(&integration).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &integration, nil
}

func (r *activityRepository) ListPlanningIntegrationsByUserID(userID string) ([]models.PlanningIntegration, error) {
	var integrations []models.PlanningIntegration
	if err := r.db.Where("user_id = ?", userID).Order("provider ASC").Find(&integrations).Error; err != nil {
		return nil, err
	}
	return integrations, nil
}

func (r *activityRepository) SavePlanningIntegration(integration *models.PlanningIntegration) error {
	return r.db.Save(integration).Error
}

func (r *activityRepository) DeletePlanningIntegration(userID, provider string) error {
	result := r.db.Where("user_id = ? AND provider = ?", userID, provider).Delete(&models.PlanningIntegration{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrPlanningIntegrationNotFound
	}
	return nil
}

func (r *activityRepository) ListAllPlanningIntegrations() ([]models.PlanningIntegration, error) {
	var integrations []models.PlanningIntegration
	if err := r.db.Order("user_id ASC, provider ASC").Find(&integrations).Error; err != nil {
		return nil, err
	}
	return integrations, nil
}

func (r *activityRepository) SaveNewPlanningActivities(activities []models.PlanningActivity) (int64, error) {
	if len(activities) == 0 {
		return 0, nil
	}

	result := r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "external_id"}},
		DoNothing: true,
	}).Create(&activities)
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

func (r *activityRepository) ListPlanningActivitiesByUserAndDateRange(userID string, since, until time.Time) ([]models.PlanningActivity, error) {
	var activities []models.PlanningActivity
	if err := r.db.
		Where("user_id = ? AND occurred_at >= ? AND occurred_at <= ?", userID, since, until).
		Order("occurred_at ASC").
		Find(&activities).Error; err != nil {
		return nil, err
	}
	return activities, nil
}

func (r *activityRepository) UpsertActiveRepository(repo *models.ActiveRepository) error {
	return r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"repository_name",
			"repository_path",
			"source",
			"updated_at",
		}),
	}).Create(repo).Error
}

func (r *activityRepository) GetActiveRepositoryByUserID(userID string) (*models.ActiveRepository, error) {
	var repo models.ActiveRepository
	err := r.db.Where("user_id = ?", userID).First(&repo).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &repo, nil
}

var (
	ErrLocalGitRepositoryNotFound = errors.New("local git repository not found")
	ErrPlanningIntegrationNotFound = errors.New("planning integration not found")
)
