package services

import (
	"dsr-automation/internal/dto"
	"dsr-automation/internal/models"
	"dsr-automation/internal/repository"
	apperrors "dsr-automation/pkg/utils/errors"
	"dsr-automation/pkg/github"
	"dsr-automation/pkg/gitlab"
	"strconv"
	"strings"
)

type GitIntegrationService interface {
	Connect(userID string, req *dto.ConnectGitRequest) (*dto.ConnectGitResponse, error)
}

type gitIntegrationService struct {
	repo         repository.GitRepository
	gitlabClient gitlab.Client
	githubClient github.Client
}

func NewGitIntegrationService(repo repository.GitRepository, gitlabClient gitlab.Client, githubClient github.Client) GitIntegrationService {
	return &gitIntegrationService{
		repo:         repo,
		gitlabClient: gitlabClient,
		githubClient: githubClient,
	}
}

func (s *gitIntegrationService) Connect(userID string, req *dto.ConnectGitRequest) (*dto.ConnectGitResponse, error) {
	if req.Provider != "gitlab" && req.Provider != "github" {
		return nil, apperrors.ErrUnsupportedGitProvider
	}

	baseURL := normalizeBaseURL(req.BaseURL)
	if baseURL == "" {
		return nil, apperrors.ErrInvalidBaseURL
	}

	var gitUserID, gitUsername string
	var projects []models.GitProject

	switch req.Provider {
	case "gitlab":
		user, err := s.gitlabClient.VerifyToken(baseURL, req.AccessToken)
		if err != nil {
			return nil, apperrors.ErrInvalidGitAccessToken
		}
		gitUserID = strconv.Itoa(user.ID)
		gitUsername = user.Username

		gitProjects, err := s.gitlabClient.ListMemberProjects(baseURL, req.AccessToken)
		if err != nil {
			return nil, apperrors.ErrFailedToFetchGitProjects
		}
		for _, p := range gitProjects {
			projects = append(projects, models.GitProject{
				UserID:            userID,
				GitProjectID:      strconv.Itoa(p.ID),
				Name:              p.Name,
				Path:              p.Path,
				PathWithNamespace: p.PathWithNamespace,
				WebURL:            p.WebURL,
				Description:       p.Description,
				DefaultBranch:     p.DefaultBranch,
			})
		}

	case "github":
		user, err := s.githubClient.VerifyToken(baseURL, req.AccessToken)
		if err != nil {
			return nil, apperrors.ErrInvalidGitAccessToken
		}
		gitUserID = strconv.Itoa(user.ID)
		gitUsername = user.Login

		repos, err := s.githubClient.ListMemberRepos(baseURL, req.AccessToken)
		if err != nil {
			return nil, apperrors.ErrFailedToFetchGitProjects
		}
		for _, r := range repos {
			projects = append(projects, models.GitProject{
				UserID:            userID,
				GitProjectID:      strconv.Itoa(r.ID),
				Name:              r.Name,
				Path:              r.Name,
				PathWithNamespace: r.FullName,
				WebURL:            r.HTMLURL,
				Description:       r.Description,
				DefaultBranch:     r.DefaultBranch,
			})
		}
	}

	existing, err := s.repo.GetByUserAndProvider(userID, req.Provider)
	if err != nil {
		return nil, apperrors.ErrFailedToSaveGitIntegration
	}

	integration := &models.GitIntegration{
		UserID:      userID,
		Provider:    req.Provider,
		BaseURL:     baseURL,
		AccessToken: req.AccessToken,
		GitUsername: gitUsername,
		GitUserID:   gitUserID,
	}
	if existing != nil {
		integration.ID = existing.ID
	}

	if err := s.repo.Save(integration); err != nil {
		return nil, apperrors.ErrFailedToSaveGitIntegration
	}

	for i := range projects {
		projects[i].GitIntegrationID = integration.ID
	}

	if err := s.repo.ReplaceProjects(integration.ID, userID, projects); err != nil {
		return nil, apperrors.ErrFailedToSaveGitProjects
	}

	projectResponses := make([]dto.GitProjectResponse, 0, len(projects))
	for _, p := range projects {
		projectResponses = append(projectResponses, dto.GitProjectResponse{
			ID:                p.ID,
			GitProjectID:      p.GitProjectID,
			Name:              p.Name,
			Path:              p.Path,
			PathWithNamespace: p.PathWithNamespace,
			WebURL:            p.WebURL,
			Description:       p.Description,
			DefaultBranch:     p.DefaultBranch,
		})
	}

	return &dto.ConnectGitResponse{
		ID:             integration.ID,
		Provider:       integration.Provider,
		BaseURL:        integration.BaseURL,
		Username:       integration.GitUsername,
		ProjectsSynced: len(projectResponses),
		Projects:       projectResponses,
	}, nil
}

func normalizeBaseURL(raw string) string {
	return strings.TrimSuffix(strings.TrimSpace(raw), "/")
}
