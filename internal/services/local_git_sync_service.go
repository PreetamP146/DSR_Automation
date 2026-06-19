package services

import (
	"context"
	"dsr-automation/internal/dto"
	"dsr-automation/internal/models"
	"dsr-automation/internal/repository"
	apperrors "dsr-automation/pkg/utils/errors"
	"dsr-automation/pkg/localgit"
	"log"
	"time"
)

type LocalGitSyncService interface {
	SyncAll(ctx context.Context) (*dto.SyncActivityResponse, error)
	SyncForUser(ctx context.Context, userID string) (*dto.SyncActivityResponse, error)
}

type localGitSyncService struct {
	repo         repository.ActivityRepository
	scanner      localgit.Scanner
	lookbackDays int
}

func NewLocalGitSyncService(repo repository.ActivityRepository, scanner localgit.Scanner, lookbackDays int) LocalGitSyncService {
	if lookbackDays <= 0 {
		lookbackDays = 30
	}
	return &localGitSyncService{
		repo:         repo,
		scanner:      scanner,
		lookbackDays: lookbackDays,
	}
}

func (s *localGitSyncService) SyncAll(ctx context.Context) (*dto.SyncActivityResponse, error) {
	repos, err := s.repo.ListAllTrackedLocalRepositories()
	if err != nil {
		return nil, apperrors.ErrFailedToSyncLocalGitActivity
	}
	return s.syncRepositories(ctx, repos)
}

func (s *localGitSyncService) SyncForUser(ctx context.Context, userID string) (*dto.SyncActivityResponse, error) {
	repos, err := s.repo.ListLocalRepositoriesByUserID(userID, true)
	if err != nil {
		return nil, apperrors.ErrFailedToSyncLocalGitActivity
	}
	if len(repos) == 0 {
		return nil, apperrors.ErrNoTrackedLocalGitRepositories
	}
	return s.syncRepositories(ctx, repos)
}

func (s *localGitSyncService) syncRepositories(ctx context.Context, repos []models.LocalGitRepository) (*dto.SyncActivityResponse, error) {
	var reposSynced int
	var commitsAdded int64

	for _, repo := range repos {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		added, err := s.syncRepository(ctx, repo)
		if err != nil {
			log.Printf("local git sync failed for repository %s: %v", repo.ID, err)
			continue
		}
		reposSynced++
		commitsAdded += added
	}

	return &dto.SyncActivityResponse{
		LocalReposSynced:  reposSynced,
		LocalCommitsAdded: int(commitsAdded),
	}, nil
}

func (s *localGitSyncService) syncRepository(ctx context.Context, repo models.LocalGitRepository) (int64, error) {
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}

	until := time.Now().UTC()
	since, err := s.syncSince(repo.ID, until)
	if err != nil {
		return 0, err
	}

	commits, err := s.scanner.ScanCommits(repo.Path, since)
	if err != nil {
		return 0, err
	}

	toSave := make([]models.LocalGitCommit, 0, len(commits))
	for _, commit := range commits {
		if commit.CommittedAt.After(until) {
			continue
		}
		toSave = append(toSave, models.LocalGitCommit{
			LocalGitRepositoryID: repo.ID,
			UserID:               repo.UserID,
			SHA:                  commit.SHA,
			Message:              commit.Message,
			Branch:               commit.Branch,
			RepositoryName:       commit.RepositoryName,
			Author:               commit.Author,
			CommittedAt:          commit.CommittedAt,
		})
	}

	added, err := s.repo.SaveNewLocalCommits(toSave)
	if err != nil {
		return 0, err
	}

	now := time.Now().UTC()
	if err := s.repo.UpdateLocalRepositoryLastScanned(repo.ID, now); err != nil {
		return added, err
	}

	return added, nil
}

func (s *localGitSyncService) syncSince(repositoryID string, until time.Time) (time.Time, error) {
	defaultSince := until.AddDate(0, 0, -s.lookbackDays)
	latest, err := s.repo.GetLatestLocalCommitTime(repositoryID)
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
