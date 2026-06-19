package services

import (
	"context"
	"dsr-automation/internal/dto"
	"dsr-automation/internal/models"
	"dsr-automation/internal/repository"
	"dsr-automation/pkg/activity"
	"dsr-automation/pkg/ai"
	apperrors "dsr-automation/pkg/utils/errors"
	"fmt"
	"strings"
	"time"
)

type DSRService interface {
	GetActivity(userID, reportDate string, gitProjectIDs []string) (*dto.ListDSRActivityResponse, error)
	Generate(userID string, req *dto.GenerateDSRRequest) (*dto.GenerateDSRResponse, error)
	ListReports(userID, from, to string, limit, offset int) (*dto.ListDSRReportsResponse, error)
	GetReport(userID, reportID string) (*dto.DSRReportResponse, error)
}

type dsrService struct {
	dsrRepo      repository.DSRRepository
	gitRepo      repository.GitRepository
	commitRepo   repository.CommitRepository
	activityRepo repository.ActivityRepository
	summarizer   ai.Summarizer
}

func NewDSRService(
	dsrRepo repository.DSRRepository,
	gitRepo repository.GitRepository,
	commitRepo repository.CommitRepository,
	activityRepo repository.ActivityRepository,
	summarizer ai.Summarizer,
) DSRService {
	return &dsrService{
		dsrRepo:      dsrRepo,
		gitRepo:      gitRepo,
		commitRepo:   commitRepo,
		activityRepo: activityRepo,
		summarizer:   summarizer,
	}
}

func (s *dsrService) GetActivity(userID, reportDate string, gitProjectIDs []string) (*dto.ListDSRActivityResponse, error) {
	date, err := parseReportDate(reportDate)
	if err != nil {
		return nil, apperrors.ErrInvalidReportDate
	}

	projects, err := s.resolveProjects(userID, gitProjectIDs)
	if err != nil {
		return nil, err
	}

	summary, err := s.fetchAllActivity(userID, projects, date)
	if err != nil {
		return nil, err
	}

	return &dto.ListDSRActivityResponse{
		ReportDate:         date.Format("2006-01-02"),
		Activity:           summary,
		TotalRemoteCommits: summary.TotalRemoteCommits(),
		TotalLocalCommits:  len(summary.LocalCommits),
		TotalPlanningEvents: len(summary.PlanningActivities),
	}, nil
}

func (s *dsrService) Generate(userID string, req *dto.GenerateDSRRequest) (*dto.GenerateDSRResponse, error) {
	date, err := parseReportDate(req.ReportDate)
	if err != nil {
		return nil, apperrors.ErrInvalidReportDate
	}

	projects, err := s.resolveProjects(userID, req.GitProjectIDs)
	if err != nil {
		return nil, err
	}

	summary, err := s.fetchAllActivity(userID, projects, date)
	if err != nil {
		return nil, err
	}

	content, err := s.summarizer.GenerateDSR(context.Background(), date, summary)
	if err != nil {
		return nil, apperrors.ErrFailedToGenerateDSR
	}

	scopeProjectID := reportScopeProjectID(req.GitProjectIDs)
	existing, err := s.dsrRepo.GetByUserReportDateAndProject(userID, date, scopeProjectID)
	if err != nil {
		return nil, apperrors.ErrFailedToSaveDSR
	}

	report := &models.DSRReport{
		UserID:        userID,
		GitProjectID:  scopeProjectID,
		ReportDate:    date,
		YesterdayWork: content.YesterdayWork,
		TodayPlan:     content.TodayPlan,
		Blockers:      content.Blockers,
		Summary:       content.Summary,
		AIModel:       content.Model,
		PromptVersion: ai.PromptVersion,
	}
	if existing != nil {
		report.ID = existing.ID
	}

	if err := s.dsrRepo.Save(report); err != nil {
		return nil, apperrors.ErrFailedToSaveDSR
	}

	saved, err := s.dsrRepo.GetByIDAndUserID(report.ID, userID)
	if err != nil || saved == nil {
		return nil, apperrors.ErrFailedToSaveDSR
	}

	response := toDSRReportResponse(saved)
	return &dto.GenerateDSRResponse{
		Report:   response,
		Activity: summary,
	}, nil
}

func (s *dsrService) ListReports(userID, from, to string, limit, offset int) (*dto.ListDSRReportsResponse, error) {
	fromDate, err := parseOptionalReportDate(from)
	if err != nil {
		return nil, apperrors.ErrInvalidReportDate
	}
	toDate, err := parseOptionalReportDate(to)
	if err != nil {
		return nil, apperrors.ErrInvalidReportDate
	}

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	total, err := s.dsrRepo.CountByUserID(userID, fromDate, toDate)
	if err != nil {
		return nil, apperrors.ErrFailedToListDSRReports
	}

	reports, err := s.dsrRepo.ListByUserID(userID, fromDate, toDate, limit, offset)
	if err != nil {
		return nil, apperrors.ErrFailedToListDSRReports
	}

	responses := make([]dto.DSRReportResponse, 0, len(reports))
	for _, report := range reports {
		responses = append(responses, toDSRReportResponse(&report))
	}

	return &dto.ListDSRReportsResponse{
		Reports: responses,
		Total:   int(total),
	}, nil
}

func (s *dsrService) GetReport(userID, reportID string) (*dto.DSRReportResponse, error) {
	report, err := s.dsrRepo.GetByIDAndUserID(reportID, userID)
	if err != nil {
		return nil, apperrors.ErrFailedToListDSRReports
	}
	if report == nil {
		return nil, apperrors.ErrDSRReportNotFound
	}

	response := toDSRReportResponse(report)
	return &response, nil
}

func (s *dsrService) resolveProjects(userID string, projectIDs []string) ([]models.GitProject, error) {
	if len(projectIDs) == 0 {
		projects, err := s.gitRepo.ListProjectsByUserID(userID, "", true)
		if err != nil {
			return nil, apperrors.ErrFailedToListGitProjects
		}
		return projects, nil
	}

	projects, err := s.gitRepo.GetProjectsByIDsAndUserID(userID, projectIDs)
	if err != nil {
		return nil, apperrors.ErrFailedToListGitProjects
	}
	if len(projects) != len(projectIDs) {
		return nil, apperrors.ErrGitProjectNotFound
	}

	for _, project := range projects {
		if !project.IsTracked {
			return nil, apperrors.ErrGitProjectNotFound
		}
	}

	return projects, nil
}

func (s *dsrService) fetchAllActivity(userID string, projects []models.GitProject, reportDate time.Time) (activity.Summary, error) {
	since, until := dayRange(reportDate)

	remoteGit, err := s.fetchRemoteGitActivity(projects, since, until)
	if err != nil {
		return activity.Summary{}, err
	}

	localCommits, err := s.activityRepo.ListLocalCommitsByUserAndDateRange(userID, since, until)
	if err != nil {
		return activity.Summary{}, apperrors.ErrFailedToFetchActivity
	}

	planningActivities, err := s.activityRepo.ListPlanningActivitiesByUserAndDateRange(userID, since, until)
	if err != nil {
		return activity.Summary{}, apperrors.ErrFailedToFetchActivity
	}

	activeRepo, err := s.activityRepo.GetActiveRepositoryByUserID(userID)
	if err != nil {
		return activity.Summary{}, apperrors.ErrFailedToFetchActivity
	}

	return activity.Summary{
		ReportDate:         reportDate.Format("2006-01-02"),
		RemoteGit:          remoteGit,
		LocalCommits:       toLocalCommitActivity(localCommits),
		PlanningActivities: toPlanningActivityItems(planningActivities),
		ActiveRepository:   toActiveRepoActivity(activeRepo),
	}, nil
}

func (s *dsrService) fetchRemoteGitActivity(projects []models.GitProject, since, until time.Time) ([]activity.RemoteProject, error) {
	projectIDs := make([]string, 0, len(projects))
	for _, project := range projects {
		projectIDs = append(projectIDs, project.ID)
	}

	storedCommits, err := s.commitRepo.ListByProjectIDsAndDateRange(projectIDs, since, until)
	if err != nil {
		return nil, apperrors.ErrFailedToFetchGitActivity
	}

	commitsByProject := make(map[string][]activity.RemoteCommit, len(projects))
	for _, commit := range storedCommits {
		commitsByProject[commit.GitProjectID] = append(commitsByProject[commit.GitProjectID], activity.RemoteCommit{
			SHA:     commit.SHA,
			Message: commit.Message,
			Author:  commit.Author,
			Date:    commit.CommittedAt,
			URL:     commit.URL,
		})
	}

	result := make([]activity.RemoteProject, 0, len(projects))
	for _, project := range projects {
		commits := commitsByProject[project.ID]
		if commits == nil {
			commits = []activity.RemoteCommit{}
		}
		result = append(result, activity.RemoteProject{
			ProjectID:         project.ID,
			ProjectName:       project.Name,
			PathWithNamespace: project.PathWithNamespace,
			Provider:          project.GitIntegration.Provider,
			Commits:           commits,
		})
	}

	return result, nil
}

func toLocalCommitActivity(commits []models.LocalGitCommit) []activity.LocalCommit {
	result := make([]activity.LocalCommit, 0, len(commits))
	for _, commit := range commits {
		result = append(result, activity.LocalCommit{
			SHA:            commit.SHA,
			Message:        commit.Message,
			Branch:         commit.Branch,
			RepositoryName: commit.RepositoryName,
			Author:         commit.Author,
			Date:           commit.CommittedAt,
		})
	}
	return result
}

func toPlanningActivityItems(activities []models.PlanningActivity) []activity.PlanningItem {
	result := make([]activity.PlanningItem, 0, len(activities))
	for _, item := range activities {
		result = append(result, activity.PlanningItem{
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
	return result
}

func toActiveRepoActivity(repo *models.ActiveRepository) *activity.ActiveRepo {
	if repo == nil {
		return nil
	}
	return &activity.ActiveRepo{
		RepositoryName: repo.RepositoryName,
		RepositoryPath: repo.RepositoryPath,
		Source:         repo.Source,
		UpdatedAt:      repo.UpdatedAt,
	}
}

func parseReportDate(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, fmt.Errorf("empty report date")
	}
	date, err := time.Parse("2006-01-02", value)
	if err != nil {
		return time.Time{}, err
	}
	return time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC), nil
}

func parseOptionalReportDate(value string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	date, err := parseReportDate(value)
	if err != nil {
		return nil, err
	}
	return &date, nil
}

func dayRange(reportDate time.Time) (time.Time, time.Time) {
	start := time.Date(reportDate.Year(), reportDate.Month(), reportDate.Day(), 0, 0, 0, 0, time.UTC)
	end := start.Add(24*time.Hour - time.Nanosecond)
	return start, end
}

func reportScopeProjectID(projectIDs []string) *string {
	if len(projectIDs) == 1 {
		id := projectIDs[0]
		return &id
	}
	return nil
}

func toDSRReportResponse(report *models.DSRReport) dto.DSRReportResponse {
	response := dto.DSRReportResponse{
		ID:            report.ID,
		ReportDate:    report.ReportDate.Format("2006-01-02"),
		GitProjectID:  report.GitProjectID,
		YesterdayWork: report.YesterdayWork,
		TodayPlan:     report.TodayPlan,
		Blockers:      report.Blockers,
		Summary:       report.Summary,
		AIModel:       report.AIModel,
		PromptVersion: report.PromptVersion,
		CreatedAt:     report.CreatedAt,
		UpdatedAt:     report.UpdatedAt,
	}
	if report.GitProject != nil {
		name := report.GitProject.Name
		response.GitProjectName = &name
	}
	return response
}
