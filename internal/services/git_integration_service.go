package services

import (
	"dsr-automation/internal/dto"
	"dsr-automation/internal/models"
	"dsr-automation/internal/repository"
	apperrors "dsr-automation/pkg/utils/errors"
	"dsr-automation/pkg/github"
	"dsr-automation/pkg/gitlab"
	"errors"
	"net/url"
	"strconv"
	"strings"
)

type GitIntegrationService interface {
	Connect(userID string, req *dto.ConnectGitRequest) (*dto.ConnectGitResponse, error)
	ListIntegrations(userID string) (*dto.ListGitIntegrationsResponse, error)
	ListProjects(userID, provider string, trackedOnly bool) (*dto.ListGitProjectsResponse, error)
	UpdateTrackedProjects(userID string, req *dto.UpdateTrackedProjectsRequest) (*dto.UpdateTrackedProjectsResponse, error)
	Sync(userID string, req *dto.SyncGitRequest) (*dto.SyncGitResponse, error)
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

	baseURL := normalizeBaseURL(req.Provider, req.BaseURL)
	if baseURL == "" {
		return nil, apperrors.ErrInvalidBaseURL
	}

	gitUserID, gitUsername, projects, err := s.fetchRemoteProjects(req.Provider, baseURL, req.AccessToken, userID)
	if err != nil {
		return nil, err
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

	if err := s.repo.SyncProjects(integration.ID, userID, projects); err != nil {
		return nil, apperrors.ErrFailedToSaveGitProjects
	}

	savedProjects, err := s.repo.ListProjectsByUserID(userID, req.Provider, false)
	if err != nil {
		return nil, apperrors.ErrFailedToListGitProjects
	}

	return &dto.ConnectGitResponse{
		ID:             integration.ID,
		Provider:       integration.Provider,
		BaseURL:        integration.BaseURL,
		Username:       integration.GitUsername,
		ProjectsSynced: len(savedProjects),
		Projects:       toGitProjectResponses(savedProjects),
	}, nil
}

func (s *gitIntegrationService) ListIntegrations(userID string) (*dto.ListGitIntegrationsResponse, error) {
	integrations, err := s.repo.ListIntegrationsByUserID(userID)
	if err != nil {
		return nil, apperrors.ErrFailedToListGitIntegrations
	}

	responses := make([]dto.GitIntegrationResponse, 0, len(integrations))
	for _, integration := range integrations {
		total, tracked, err := s.repo.CountProjectsByIntegrationID(integration.ID)
		if err != nil {
			return nil, apperrors.ErrFailedToListGitIntegrations
		}

		responses = append(responses, dto.GitIntegrationResponse{
			ID:             integration.ID,
			Provider:       integration.Provider,
			BaseURL:        integration.BaseURL,
			Username:       integration.GitUsername,
			ProjectsSynced: int(total),
			TrackedCount:   int(tracked),
		})
	}

	return &dto.ListGitIntegrationsResponse{Integrations: responses}, nil
}

func (s *gitIntegrationService) ListProjects(userID, provider string, trackedOnly bool) (*dto.ListGitProjectsResponse, error) {
	if provider != "" && provider != "gitlab" && provider != "github" {
		return nil, apperrors.ErrUnsupportedGitProvider
	}

	projects, err := s.repo.ListProjectsByUserID(userID, provider, trackedOnly)
	if err != nil {
		return nil, apperrors.ErrFailedToListGitProjects
	}

	responses := toGitProjectResponses(projects)
	return &dto.ListGitProjectsResponse{
		Projects: responses,
		Total:    len(responses),
	}, nil
}

func (s *gitIntegrationService) UpdateTrackedProjects(userID string, req *dto.UpdateTrackedProjectsRequest) (*dto.UpdateTrackedProjectsResponse, error) {
	if err := s.repo.SetTrackedProjects(userID, req.ProjectIDs); err != nil {
		if errors.Is(err, repository.ErrGitProjectNotFound) {
			return nil, apperrors.ErrGitProjectNotFound
		}
		return nil, apperrors.ErrFailedToUpdateTrackedProjects
	}

	projects, err := s.repo.ListProjectsByUserID(userID, "", true)
	if err != nil {
		return nil, apperrors.ErrFailedToListGitProjects
	}

	responses := toGitProjectResponses(projects)
	return &dto.UpdateTrackedProjectsResponse{
		TrackedCount: len(responses),
		Projects:     responses,
	}, nil
}

func (s *gitIntegrationService) Sync(userID string, req *dto.SyncGitRequest) (*dto.SyncGitResponse, error) {
	if req.Provider != "gitlab" && req.Provider != "github" {
		return nil, apperrors.ErrUnsupportedGitProvider
	}

	integration, err := s.repo.GetByUserAndProvider(userID, req.Provider)
	if err != nil {
		return nil, apperrors.ErrFailedToSyncGitIntegration
	}
	if integration == nil {
		return nil, apperrors.ErrGitIntegrationNotFound
	}

	gitUserID, gitUsername, projects, err := s.fetchRemoteProjects(req.Provider, integration.BaseURL, integration.AccessToken, userID)
	if err != nil {
		return nil, err
	}

	integration.GitUserID = gitUserID
	integration.GitUsername = gitUsername
	if err := s.repo.Save(integration); err != nil {
		return nil, apperrors.ErrFailedToSyncGitIntegration
	}

	if err := s.repo.SyncProjects(integration.ID, userID, projects); err != nil {
		return nil, apperrors.ErrFailedToSaveGitProjects
	}

	savedProjects, err := s.repo.ListProjectsByUserID(userID, req.Provider, false)
	if err != nil {
		return nil, apperrors.ErrFailedToListGitProjects
	}

	return &dto.SyncGitResponse{
		ID:             integration.ID,
		Provider:       integration.Provider,
		BaseURL:        integration.BaseURL,
		Username:       integration.GitUsername,
		ProjectsSynced: len(savedProjects),
		Projects:       toGitProjectResponses(savedProjects),
	}, nil
}

func (s *gitIntegrationService) fetchRemoteProjects(provider, baseURL, accessToken, userID string) (string, string, []models.GitProject, error) {
	if provider == "github" {
		baseURL = normalizeGitHubBaseURL(baseURL)
	}

	switch provider {
	case "gitlab":
		user, err := s.gitlabClient.VerifyToken(baseURL, accessToken)
		if err != nil {
			return "", "", nil, apperrors.ErrInvalidGitAccessToken
		}

		gitProjects, err := s.gitlabClient.ListMemberProjects(baseURL, accessToken)
		if err != nil {
			return "", "", nil, apperrors.ErrFailedToFetchGitProjects
		}

		projects := make([]models.GitProject, 0, len(gitProjects))
		for _, project := range gitProjects {
			projects = append(projects, models.GitProject{
				UserID:            userID,
				RemoteProjectID:   strconv.Itoa(project.ID),
				Name:              project.Name,
				Path:              project.Path,
				PathWithNamespace: project.PathWithNamespace,
				WebURL:            project.WebURL,
				Description:       project.Description,
				DefaultBranch:     project.DefaultBranch,
			})
		}

		return strconv.Itoa(user.ID), user.Username, projects, nil

	case "github":
		user, err := s.githubClient.VerifyToken(baseURL, accessToken)
		if err != nil {
			return "", "", nil, apperrors.ErrInvalidGitAccessToken
		}

		repos, err := s.githubClient.ListMemberRepos(baseURL, accessToken)
		if err != nil {
			return "", "", nil, apperrors.ErrFailedToFetchGitProjects
		}

		projects := make([]models.GitProject, 0, len(repos))
		for _, repo := range repos {
			projects = append(projects, models.GitProject{
				UserID:            userID,
				RemoteProjectID:   strconv.Itoa(repo.ID),
				Name:              repo.Name,
				Path:              repo.Name,
				PathWithNamespace: repo.FullName,
				WebURL:            repo.HTMLURL,
				Description:       repo.Description,
				DefaultBranch:     repo.DefaultBranch,
			})
		}

		return strconv.Itoa(user.ID), user.Login, projects, nil

	default:
		return "", "", nil, apperrors.ErrUnsupportedGitProvider
	}
}

func toGitProjectResponses(projects []models.GitProject) []dto.GitProjectResponse {
	responses := make([]dto.GitProjectResponse, 0, len(projects))
	for _, project := range projects {
		responses = append(responses, dto.GitProjectResponse{
			ID:                project.ID,
			GitIntegrationID:  project.GitIntegrationID,
			Provider:          project.GitIntegration.Provider,
			GitProjectID:      project.RemoteProjectID,
			Name:              project.Name,
			Path:              project.Path,
			PathWithNamespace: project.PathWithNamespace,
			WebURL:            project.WebURL,
			Description:       project.Description,
			DefaultBranch:     project.DefaultBranch,
			IsTracked:         project.IsTracked,
		})
	}
	return responses
}

func normalizeBaseURL(provider, raw string) string {
	raw = strings.TrimSuffix(strings.TrimSpace(raw), "/")
	if raw == "" {
		return ""
	}

	if provider != "github" {
		return raw
	}

	return normalizeGitHubBaseURL(raw)
}

func normalizeGitHubBaseURL(raw string) string {
	switch raw {
	case "https://github.com", "http://github.com":
		return "https://api.github.com"
	case "https://api.github.com", "http://api.github.com":
		return "https://api.github.com"
	}

	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return raw
	}

	// GitHub Enterprise Server uses /api/v3 on the host root.
	if parsed.Host != "github.com" && parsed.Host != "api.github.com" && !strings.Contains(parsed.Path, "/api/") {
		return raw + "/api/v3"
	}

	return raw
}
