package repository

import (
	"dsr-automation/internal/models"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CommitRepository interface {
	SaveNew(commits []models.GitCommit) (int64, error)
	ListByProjectIDsAndDateRange(projectIDs []string, since, until time.Time) ([]models.GitCommit, error)
	GetLatestCommittedAt(gitProjectID string) (*time.Time, error)
}

type commitRepository struct {
	db *gorm.DB
}

func NewCommitRepository(db *gorm.DB) CommitRepository {
	return &commitRepository{db: db}
}

func (r *commitRepository) SaveNew(commits []models.GitCommit) (int64, error) {
	if len(commits) == 0 {
		return 0, nil
	}

	result := r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "git_project_id"},
			{Name: "sha"},
		},
		DoNothing: true,
	}).Create(&commits)
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

func (r *commitRepository) ListByProjectIDsAndDateRange(projectIDs []string, since, until time.Time) ([]models.GitCommit, error) {
	if len(projectIDs) == 0 {
		return nil, nil
	}

	var commits []models.GitCommit
	if err := r.db.
		Where("git_project_id IN ? AND committed_at >= ? AND committed_at <= ?", projectIDs, since, until).
		Order("committed_at ASC").
		Find(&commits).Error; err != nil {
		return nil, err
	}
	return commits, nil
}

func (r *commitRepository) GetLatestCommittedAt(gitProjectID string) (*time.Time, error) {
	var commit models.GitCommit
	err := r.db.
		Where("git_project_id = ?", gitProjectID).
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
