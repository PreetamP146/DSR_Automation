package services

import (
	"context"
	"dsr-automation/internal/dto"
	"dsr-automation/internal/models"
	"dsr-automation/internal/repository"
	apperrors "dsr-automation/pkg/utils/errors"
	"dsr-automation/pkg/github"
	"dsr-automation/pkg/gitlab"
	"fmt"
	"log"
	"strings"
	"time"
)

type CommitSyncService interface {
	SyncAll(ctx context.Context) (*dto.SyncCommitsResponse, error)
	SyncForUser(ctx context.Context, userID string) (*dto.SyncCommitsResponse, error)
}

type commitSyncService struct {
	gitRepo      repository.GitRepository
	commitRepo   repository.CommitRepository
	githubClient github.Client
	gitlabClient gitlab.Client
	lookbackDays int
}

func NewCommitSyncService(
	gitRepo repository.GitRepository,
	commitRepo repository.CommitRepository,
	githubClient github.Client,
	gitlabClient gitlab.Client,
	lookbackDays int,
) CommitSyncService {
	if lookbackDays <= 0 {
		lookbackDays = 30
	}
	return &commitSyncService{
		gitRepo:      gitRepo,
		commitRepo:   commitRepo,
		githubClient: githubClient,
		gitlabClient: gitlabClient,
		lookbackDays: lookbackDays,
	}
}

func (s *commitSyncService) SyncAll(ctx context.Context) (*dto.SyncCommitsResponse, error) {
	projects, err := s.gitRepo.ListAllTrackedProjects()
	if err != nil {
		return nil, apperrors.ErrFailedToSyncCommits
	}
	return s.syncProjects(ctx, projects)
}

func (s *commitSyncService) SyncForUser(ctx context.Context, userID string) (*dto.SyncCommitsResponse, error) {
	projects, err := s.gitRepo.ListProjectsByUserID(userID, "", true)
	if err != nil {
		return nil, apperrors.ErrFailedToSyncCommits
	}
	if len(projects) == 0 {
		return nil, apperrors.ErrNoTrackedGitProjects
	}
	return s.syncProjects(ctx, projects)
}

func (s *commitSyncService) syncProjects(ctx context.Context, projects []models.GitProject) (*dto.SyncCommitsResponse, error) {
	var commitsAdded int64
	projectsSynced := 0

	for _, project := range projects {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		added, err := s.syncProject(ctx, project)
		if err != nil {
			log.Printf("commit sync failed for project %s: %v", project.ID, err)
			continue
		}
		projectsSynced++
		commitsAdded += added
	}

	return &dto.SyncCommitsResponse{
		ProjectsSynced: projectsSynced,
		CommitsAdded:   int(commitsAdded),
	}, nil
}

func (s *commitSyncService) syncProject(ctx context.Context, project models.GitProject) (int64, error) {
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}

	until := time.Now().UTC()
	since, err := s.syncSince(project.ID, until)
	if err != nil {
		return 0, err
	}

	remoteCommits, err := s.fetchRemoteCommits(&project.GitIntegration, &project, since, until)
	if err != nil {
		return 0, err
	}

	toSave := make([]models.GitCommit, 0, len(remoteCommits))
	for _, commit := range remoteCommits {
		toSave = append(toSave, models.GitCommit{
			GitProjectID: project.ID,
			UserID:       project.UserID,
			SHA:          commit.SHA,
			Message:      commit.Message,
			Author:       commit.Author,
			CommittedAt:  commit.Date,
			URL:          commit.URL,
		})
	}

	return s.commitRepo.SaveNew(toSave)
}

func (s *commitSyncService) syncSince(gitProjectID string, until time.Time) (time.Time, error) {
	defaultSince := until.AddDate(0, 0, -s.lookbackDays)
	latest, err := s.commitRepo.GetLatestCommittedAt(gitProjectID)
	if err != nil {
		return time.Time{}, err
	}
	if latest == nil {
		return defaultSince, nil
	}

	since := latest.Add(-1 * time.Hour)
	if since.Before(defaultSince) {
		return defaultSince, nil
	}
	return since, nil
}

type remoteCommit struct {
	SHA     string
	Message string
	Author  string
	Date    time.Time
	URL     string
}

func (s *commitSyncService) fetchRemoteCommits(integration *models.GitIntegration, project *models.GitProject, since, until time.Time) ([]remoteCommit, error) {
	switch integration.Provider {
	case "github":
		owner, repo, err := splitNamespace(project.PathWithNamespace)
		if err != nil {
			return nil, err
		}
		commits, err := s.githubClient.ListCommits(
			integration.BaseURL,
			integration.AccessToken,
			owner,
			repo,
			integration.GitUsername,
			since,
			until,
		)
		if err != nil {
			return nil, err
		}
		return mapGitHubRemoteCommits(commits), nil

	case "gitlab":
		commits, err := s.gitlabClient.ListCommits(
			integration.BaseURL,
			integration.AccessToken,
			project.GitProjectID,
			integration.GitUsername,
			since,
			until,
		)
		if err != nil {
			return nil, err
		}
		return mapGitLabRemoteCommits(commits), nil

	default:
		return nil, fmt.Errorf("unsupported provider: %s", integration.Provider)
	}
}

func mapGitHubRemoteCommits(commits []github.Commit) []remoteCommit {
	mapped := make([]remoteCommit, 0, len(commits))
	for _, commit := range commits {
		mapped = append(mapped, remoteCommit{
			SHA:     commit.SHA,
			Message: commit.Message,
			Author:  commit.Author,
			Date:    commit.Date,
			URL:     commit.URL,
		})
	}
	return mapped
}

func mapGitLabRemoteCommits(commits []gitlab.Commit) []remoteCommit {
	mapped := make([]remoteCommit, 0, len(commits))
	for _, commit := range commits {
		mapped = append(mapped, remoteCommit{
			SHA:     commit.SHA,
			Message: commit.Message,
			Author:  commit.Author,
			Date:    commit.Date,
			URL:     commit.URL,
		})
	}
	return mapped
}

func splitNamespace(pathWithNamespace string) (string, string, error) {
	parts := strings.SplitN(pathWithNamespace, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid repository namespace: %s", pathWithNamespace)
	}
	return parts[0], parts[1], nil
}
