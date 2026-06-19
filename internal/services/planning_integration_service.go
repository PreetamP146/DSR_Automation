package services

import (
	"dsr-automation/internal/dto"
	"dsr-automation/internal/models"
	"dsr-automation/internal/repository"
	"dsr-automation/pkg/planning"
	apperrors "dsr-automation/pkg/utils/errors"
	"errors"
	"strings"
)

type PlanningIntegrationService interface {
	ListProviders() []string
	Connect(userID string, req *dto.ConnectPlanningRequest) (*dto.PlanningIntegrationResponse, error)
	ListIntegrations(userID string) (*dto.ListPlanningIntegrationsResponse, error)
	GetIntegration(userID, provider string) (*dto.PlanningIntegrationResponse, error)
	Disconnect(userID, provider string) error
}

type planningIntegrationService struct {
	repo     repository.ActivityRepository
	registry *planning.Registry
}

func NewPlanningIntegrationService(repo repository.ActivityRepository, registry *planning.Registry) PlanningIntegrationService {
	return &planningIntegrationService{
		repo:     repo,
		registry: registry,
	}
}

func (s *planningIntegrationService) ListProviders() []string {
	return s.registry.List()
}

func (s *planningIntegrationService) Connect(userID string, req *dto.ConnectPlanningRequest) (*dto.PlanningIntegrationResponse, error) {
	providerName := strings.ToLower(strings.TrimSpace(req.Provider))
	provider, err := s.registry.Get(providerName)
	if err != nil {
		return nil, apperrors.ErrUnsupportedPlanningProvider
	}

	credentials, err := buildPlanningCredentials(providerName, req)
	if err != nil {
		return nil, err
	}

	account, err := provider.VerifyCredentials(credentials)
	if err != nil {
		return nil, apperrors.ErrInvalidPlanningCredentials
	}

	existing, err := s.repo.GetPlanningIntegrationByUserAndProvider(userID, providerName)
	if err != nil {
		return nil, apperrors.ErrFailedToSavePlanningIntegration
	}

	integration := &models.PlanningIntegration{
		UserID:      userID,
		Provider:    providerName,
		BaseURL:     credentials.BaseURL,
		Email:       credentials.Email,
		APIToken:    credentials.APIToken,
		APIKey:      credentials.APIKey,
		AccountID:   account.AccountID,
		DisplayName: account.DisplayName,
	}
	if existing != nil {
		integration.ID = existing.ID
		integration.LastSyncAt = existing.LastSyncAt
	}

	if err := s.repo.SavePlanningIntegration(integration); err != nil {
		return nil, apperrors.ErrFailedToSavePlanningIntegration
	}

	return toPlanningIntegrationResponse(integration), nil
}

func (s *planningIntegrationService) ListIntegrations(userID string) (*dto.ListPlanningIntegrationsResponse, error) {
	integrations, err := s.repo.ListPlanningIntegrationsByUserID(userID)
	if err != nil {
		return nil, apperrors.ErrFailedToListPlanningIntegrations
	}

	responses := make([]dto.PlanningIntegrationResponse, 0, len(integrations))
	for i := range integrations {
		responses = append(responses, *toPlanningIntegrationResponse(&integrations[i]))
	}

	return &dto.ListPlanningIntegrationsResponse{
		Integrations: responses,
		Total:        len(responses),
	}, nil
}

func (s *planningIntegrationService) GetIntegration(userID, provider string) (*dto.PlanningIntegrationResponse, error) {
	providerName := strings.ToLower(strings.TrimSpace(provider))
	integration, err := s.repo.GetPlanningIntegrationByUserAndProvider(userID, providerName)
	if err != nil {
		return nil, apperrors.ErrFailedToGetPlanningIntegration
	}
	if integration == nil {
		return nil, apperrors.ErrPlanningIntegrationNotFound
	}
	return toPlanningIntegrationResponse(integration), nil
}

func (s *planningIntegrationService) Disconnect(userID, provider string) error {
	providerName := strings.ToLower(strings.TrimSpace(provider))
	if err := s.repo.DeletePlanningIntegration(userID, providerName); err != nil {
		if errors.Is(err, repository.ErrPlanningIntegrationNotFound) {
			return apperrors.ErrPlanningIntegrationNotFound
		}
		return apperrors.ErrFailedToDeletePlanningIntegration
	}
	return nil
}

func buildPlanningCredentials(provider string, req *dto.ConnectPlanningRequest) (planning.Credentials, error) {
	baseURL := strings.TrimSuffix(strings.TrimSpace(req.BaseURL), "/")
	email := strings.TrimSpace(req.Email)
	apiToken := strings.TrimSpace(req.APIToken)
	apiKey := strings.TrimSpace(req.APIKey)

	switch provider {
	case planning.ProviderJira:
		if baseURL == "" || email == "" || apiToken == "" {
			return planning.Credentials{}, apperrors.ErrInvalidPlanningCredentials
		}
		return planning.Credentials{BaseURL: baseURL, Email: email, APIToken: apiToken}, nil

	case planning.ProviderTrello:
		if apiKey == "" || apiToken == "" {
			return planning.Credentials{}, apperrors.ErrInvalidPlanningCredentials
		}
		return planning.Credentials{APIKey: apiKey, APIToken: apiToken}, nil

	default:
		return planning.Credentials{}, apperrors.ErrUnsupportedPlanningProvider
	}
}

func toPlanningIntegrationResponse(integration *models.PlanningIntegration) *dto.PlanningIntegrationResponse {
	return &dto.PlanningIntegrationResponse{
		ID:          integration.ID,
		Provider:    integration.Provider,
		BaseURL:     integration.BaseURL,
		Email:       integration.Email,
		DisplayName: integration.DisplayName,
		LastSyncAt:  integration.LastSyncAt,
	}
}

func toPlanningCredentials(integration models.PlanningIntegration) planning.Credentials {
	return planning.Credentials{
		BaseURL:   integration.BaseURL,
		Email:     integration.Email,
		APIToken:  integration.APIToken,
		APIKey:    integration.APIKey,
		AccountID: integration.AccountID,
	}
}

func toPlanningAccount(integration models.PlanningIntegration) planning.Account {
	return planning.Account{
		AccountID:   integration.AccountID,
		DisplayName: integration.DisplayName,
	}
}
