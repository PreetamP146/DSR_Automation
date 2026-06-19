package services

import (
	"context"
	"dsr-automation/internal/dto"
	"dsr-automation/internal/models"
	"dsr-automation/internal/repository"
	apperrors "dsr-automation/pkg/utils/errors"
	"strings"
)

type ActivityTrackerService interface {
	SetActiveRepository(userID string, req *dto.SetActiveRepositoryRequest) (*dto.ActiveRepositoryResponse, error)
	GetActiveRepository(userID string) (*dto.ActiveRepositoryResponse, error)
	GetActivitySummary(userID, reportDate string) (*dto.ActivitySummaryResponse, error)
	SyncAll(ctx context.Context) (*dto.SyncActivityResponse, error)
	SyncForUser(ctx context.Context, userID string) (*dto.SyncActivityResponse, error)
}

type activityTrackerService struct {
	repo         repository.ActivityRepository
	localGitSync LocalGitSyncService
	planningSync PlanningSyncService
}

func NewActivityTrackerService(
	repo repository.ActivityRepository,
	localGitSync LocalGitSyncService,
	planningSync PlanningSyncService,
) ActivityTrackerService {
	return &activityTrackerService{
		repo:         repo,
		localGitSync: localGitSync,
		planningSync: planningSync,
	}
}

func (s *activityTrackerService) SetActiveRepository(userID string, req *dto.SetActiveRepositoryRequest) (*dto.ActiveRepositoryResponse, error) {
	name := strings.TrimSpace(req.RepositoryName)
	path := strings.TrimSpace(req.RepositoryPath)
	if name == "" || path == "" {
		return nil, apperrors.ErrInvalidActiveRepository
	}

	source := strings.TrimSpace(req.Source)
	if source == "" {
		source = "local"
	}

	repo := &models.ActiveRepository{
		UserID:         userID,
		RepositoryName: name,
		RepositoryPath: path,
		Source:         source,
	}
	if err := s.repo.UpsertActiveRepository(repo); err != nil {
		return nil, apperrors.ErrFailedToSaveActiveRepository
	}

	saved, err := s.repo.GetActiveRepositoryByUserID(userID)
	if err != nil || saved == nil {
		return nil, apperrors.ErrFailedToSaveActiveRepository
	}

	return toActiveRepositoryResponse(saved), nil
}

func (s *activityTrackerService) GetActiveRepository(userID string) (*dto.ActiveRepositoryResponse, error) {
	repo, err := s.repo.GetActiveRepositoryByUserID(userID)
	if err != nil {
		return nil, apperrors.ErrFailedToGetActiveRepository
	}
	if repo == nil {
		return nil, apperrors.ErrActiveRepositoryNotFound
	}
	return toActiveRepositoryResponse(repo), nil
}

func (s *activityTrackerService) GetActivitySummary(userID, reportDate string) (*dto.ActivitySummaryResponse, error) {
	date, err := parseReportDate(reportDate)
	if err != nil {
		return nil, apperrors.ErrInvalidReportDate
	}

	since, until := dayRange(date)

	localCommits, err := s.repo.ListLocalCommitsByUserAndDateRange(userID, since, until)
	if err != nil {
		return nil, apperrors.ErrFailedToFetchActivity
	}

	planningActivities, err := s.repo.ListPlanningActivitiesByUserAndDateRange(userID, since, until)
	if err != nil {
		return nil, apperrors.ErrFailedToFetchActivity
	}

	activeRepo, err := s.repo.GetActiveRepositoryByUserID(userID)
	if err != nil {
		return nil, apperrors.ErrFailedToFetchActivity
	}

	response := &dto.ActivitySummaryResponse{
		ReportDate:         date.Format("2006-01-02"),
		LocalCommits:       toLocalGitCommitResponses(localCommits),
		PlanningActivities: toPlanningActivityResponses(planningActivities),
	}
	if activeRepo != nil {
		response.ActiveRepository = toActiveRepositoryResponse(activeRepo)
	}

	return response, nil
}

func (s *activityTrackerService) SyncAll(ctx context.Context) (*dto.SyncActivityResponse, error) {
	localResult, err := s.localGitSync.SyncAll(ctx)
	if err != nil {
		return nil, err
	}

	planningResult, err := s.planningSync.SyncAll(ctx)
	if err != nil {
		return nil, err
	}

	return &dto.SyncActivityResponse{
		LocalReposSynced:           localResult.LocalReposSynced,
		LocalCommitsAdded:          localResult.LocalCommitsAdded,
		PlanningIntegrationsSynced: planningResult.PlanningIntegrationsSynced,
		PlanningActivitiesAdded:    planningResult.PlanningActivitiesAdded,
	}, nil
}

func (s *activityTrackerService) SyncForUser(ctx context.Context, userID string) (*dto.SyncActivityResponse, error) {
	response := &dto.SyncActivityResponse{}

	localResult, err := s.localGitSync.SyncForUser(ctx, userID)
	if err != nil && !isNoTrackedLocalRepos(err) {
		return nil, err
	}
	if localResult != nil {
		response.LocalReposSynced = localResult.LocalReposSynced
		response.LocalCommitsAdded = localResult.LocalCommitsAdded
	}

	planningResult, err := s.planningSync.SyncForUser(ctx, userID)
	if err != nil && !isPlanningNotFound(err) {
		return nil, err
	}
	if planningResult != nil {
		response.PlanningIntegrationsSynced = planningResult.PlanningIntegrationsSynced
		response.PlanningActivitiesAdded = planningResult.PlanningActivitiesAdded
	}

	return response, nil
}

func isNoTrackedLocalRepos(err error) bool {
	return err == apperrors.ErrNoTrackedLocalGitRepositories
}

func isPlanningNotFound(err error) bool {
	return err == apperrors.ErrPlanningIntegrationNotFound
}

func toActiveRepositoryResponse(repo *models.ActiveRepository) *dto.ActiveRepositoryResponse {
	return &dto.ActiveRepositoryResponse{
		RepositoryName: repo.RepositoryName,
		RepositoryPath: repo.RepositoryPath,
		Source:         repo.Source,
		UpdatedAt:      repo.UpdatedAt,
	}
}

func toLocalGitCommitResponses(commits []models.LocalGitCommit) []dto.LocalGitCommitResponse {
	responses := make([]dto.LocalGitCommitResponse, 0, len(commits))
	for _, commit := range commits {
		responses = append(responses, dto.LocalGitCommitResponse{
			ID:             commit.ID,
			SHA:            commit.SHA,
			Message:        commit.Message,
			Branch:         commit.Branch,
			RepositoryName: commit.RepositoryName,
			Author:         commit.Author,
			CommittedAt:    commit.CommittedAt,
		})
	}
	return responses
}

func toPlanningActivityResponses(activities []models.PlanningActivity) []dto.PlanningActivityResponse {
	responses := make([]dto.PlanningActivityResponse, 0, len(activities))
	for _, item := range activities {
		responses = append(responses, dto.PlanningActivityResponse{
			ID:             item.ID,
			Provider:       item.Provider,
			ActivityType:   item.ActivityType,
			ItemKey:        item.ItemKey,
			ItemTitle:      item.ItemTitle,
			Status:         item.Status,
			PreviousStatus: item.PreviousStatus,
			Comment:        item.Comment,
			OccurredAt:     item.OccurredAt,
		})
	}
	return responses
}
