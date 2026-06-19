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

	if err := prepareLegacySchema(db); err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(
		&models.User{},
		&models.GitIntegration{},
		&models.GitProject{},
		&models.GitCommit{},
		&models.DSRReport{},
		&models.LocalGitRepository{},
		&models.LocalGitCommit{},
		&models.PlanningIntegration{},
		&models.PlanningActivity{},
		&models.ActiveRepository{},
	); err != nil {
		return nil, err
	}

	return db, nil
}

func prepareLegacySchema(db *gorm.DB) error {
	if !db.Migrator().HasTable("git_projects") {
		return nil
	}

	if db.Migrator().HasColumn("git_projects", "git_project_id") &&
		!db.Migrator().HasColumn("git_projects", "remote_project_id") {
		if err := db.Exec(`ALTER TABLE git_projects ADD COLUMN remote_project_id text`).Error; err != nil {
			return err
		}
		if err := db.Exec(`UPDATE git_projects SET remote_project_id = git_project_id WHERE remote_project_id IS NULL`).Error; err != nil {
			return err
		}
	}

	return nil
}
