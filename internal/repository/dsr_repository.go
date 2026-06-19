package repository

import (
	"dsr-automation/internal/models"
	"errors"
	"time"

	"gorm.io/gorm"
)

type DSRRepository interface {
	Save(report *models.DSRReport) error
	GetByIDAndUserID(id, userID string) (*models.DSRReport, error)
	GetByUserReportDateAndProject(userID string, reportDate time.Time, gitProjectID *string) (*models.DSRReport, error)
	ListByUserID(userID string, from, to *time.Time, limit, offset int) ([]models.DSRReport, error)
	CountByUserID(userID string, from, to *time.Time) (int64, error)
}

type dsrRepository struct {
	db *gorm.DB
}

func NewDSRRepository(db *gorm.DB) DSRRepository {
	return &dsrRepository{db: db}
}

func (r *dsrRepository) Save(report *models.DSRReport) error {
	return r.db.Save(report).Error
}

func (r *dsrRepository) GetByIDAndUserID(id, userID string) (*models.DSRReport, error) {
	var report models.DSRReport
	err := r.db.
		Preload("GitProject").
		Where("id = ? AND user_id = ?", id, userID).
		First(&report).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &report, nil
}

func (r *dsrRepository) GetByUserReportDateAndProject(userID string, reportDate time.Time, gitProjectID *string) (*models.DSRReport, error) {
	query := r.db.Where("user_id = ? AND report_date = ?", userID, reportDate)
	if gitProjectID == nil {
		query = query.Where("git_project_id IS NULL")
	} else {
		query = query.Where("git_project_id = ?", *gitProjectID)
	}

	var report models.DSRReport
	err := query.First(&report).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &report, nil
}

func (r *dsrRepository) ListByUserID(userID string, from, to *time.Time, limit, offset int) ([]models.DSRReport, error) {
	query := r.db.
		Preload("GitProject").
		Where("user_id = ?", userID).
		Order("report_date DESC, created_at DESC")

	if from != nil {
		query = query.Where("report_date >= ?", *from)
	}
	if to != nil {
		query = query.Where("report_date <= ?", *to)
	}
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	var reports []models.DSRReport
	if err := query.Find(&reports).Error; err != nil {
		return nil, err
	}
	return reports, nil
}

func (r *dsrRepository) CountByUserID(userID string, from, to *time.Time) (int64, error) {
	query := r.db.Model(&models.DSRReport{}).Where("user_id = ?", userID)
	if from != nil {
		query = query.Where("report_date >= ?", *from)
	}
	if to != nil {
		query = query.Where("report_date <= ?", *to)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}
