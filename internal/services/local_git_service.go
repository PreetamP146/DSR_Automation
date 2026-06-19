package services

import (
	"dsr-automation/internal/dto"
	"dsr-automation/internal/models"
	"dsr-automation/internal/repository"
	apperrors "dsr-automation/pkg/utils/errors"
	"dsr-automation/pkg/localgit"
	"errors"
	"path/filepath"
	"strings"
)

type LocalGitService interface {
	RegisterRepository(userID string, req *dto.RegisterLocalGitRepoRequest) (*dto.LocalGitRepositoryResponse, error)
	ListRepositories(userID string, trackedOnly bool) (*dto.ListLocalGitReposResponse, error)
	UpdateTrackedRepositories(userID string, req *dto.UpdateTrackedLocalReposRequest) (*dto.ListLocalGitReposResponse, error)
	DeleteRepository(userID, repositoryID string) error
}

type localGitService struct {
	repo     repository.ActivityRepository
	scanner  localgit.Scanner
}

func NewLocalGitService(repo repository.ActivityRepository, scanner localgit.Scanner) LocalGitService {
	return &localGitService{
		repo:    repo,
		scanner: scanner,
	}
}

func (s *localGitService) RegisterRepository(userID string, req *dto.RegisterLocalGitRepoRequest) (*dto.LocalGitRepositoryResponse, error) {
	absPath, err := filepath.Abs(strings.TrimSpace(req.Path))
	if err != nil {
		return nil, apperrors.ErrInvalidLocalGitPath
	}
	if !s.scanner.IsGitRepository(absPath) {
		return nil, apperrors.ErrInvalidLocalGitPath
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		name, err = s.scanner.RepositoryName(absPath)
		if err != nil {
			return nil, apperrors.ErrInvalidLocalGitPath
		}
	}

	repo := &models.LocalGitRepository{
		UserID:    userID,
		Name:      name,
		Path:      absPath,
		IsTracked: true,
	}
	if err := s.repo.SaveLocalRepository(repo); err != nil {
		return nil, apperrors.ErrFailedToSaveLocalGitRepository
	}

	return toLocalGitRepositoryResponse(repo), nil
}

func (s *localGitService) ListRepositories(userID string, trackedOnly bool) (*dto.ListLocalGitReposResponse, error) {
	repos, err := s.repo.ListLocalRepositoriesByUserID(userID, trackedOnly)
	if err != nil {
		return nil, apperrors.ErrFailedToListLocalGitRepositories
	}

	responses := make([]dto.LocalGitRepositoryResponse, 0, len(repos))
	for i := range repos {
		responses = append(responses, *toLocalGitRepositoryResponse(&repos[i]))
	}

	return &dto.ListLocalGitReposResponse{
		Repositories: responses,
		Total:        len(responses),
	}, nil
}

func (s *localGitService) UpdateTrackedRepositories(userID string, req *dto.UpdateTrackedLocalReposRequest) (*dto.ListLocalGitReposResponse, error) {
	if err := s.repo.SetTrackedLocalRepositories(userID, req.RepositoryIDs); err != nil {
		if errors.Is(err, repository.ErrLocalGitRepositoryNotFound) {
			return nil, apperrors.ErrLocalGitRepositoryNotFound
		}
		return nil, apperrors.ErrFailedToUpdateTrackedLocalRepos
	}
	return s.ListRepositories(userID, true)
}

func (s *localGitService) DeleteRepository(userID, repositoryID string) error {
	if err := s.repo.DeleteLocalRepository(repositoryID, userID); err != nil {
		if errors.Is(err, repository.ErrLocalGitRepositoryNotFound) {
			return apperrors.ErrLocalGitRepositoryNotFound
		}
		return apperrors.ErrFailedToDeleteLocalGitRepository
	}
	return nil
}

func toLocalGitRepositoryResponse(repo *models.LocalGitRepository) *dto.LocalGitRepositoryResponse {
	return &dto.LocalGitRepositoryResponse{
		ID:            repo.ID,
		Name:          repo.Name,
		Path:          repo.Path,
		IsTracked:     repo.IsTracked,
		LastScannedAt: repo.LastScannedAt,
	}
}
