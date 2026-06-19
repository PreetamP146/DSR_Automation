package services

import (
	"context"
	"dsr-automation/internal/dto"
	"dsr-automation/internal/models"
	"dsr-automation/internal/repository"
	"dsr-automation/pkg/planning"
	apperrors "dsr-automation/pkg/utils/errors"
	"log"
	"time"
)

type PlanningSyncService interface {
	SyncAll(ctx context.Context) (*dto.SyncActivityResponse, error)
	SyncForUser(ctx context.Context, userID string) (*dto.SyncActivityResponse, error)
}

type planningSyncService struct {
	repo         repository.ActivityRepository
	registry     *planning.Registry
	lookbackDays int
}

func NewPlanningSyncService(repo repository.ActivityRepository, registry *planning.Registry, lookbackDays int) PlanningSyncService {
	if lookbackDays <= 0 {
		lookbackDays = 30
	}
	return &planningSyncService{
		repo:         repo,
		registry:     registry,
		lookbackDays: lookbackDays,
	}
}

func (s *planningSyncService) SyncAll(ctx context.Context) (*dto.SyncActivityResponse, error) {
	integrations, err := s.repo.ListAllPlanningIntegrations()
	if err != nil {
		return nil, apperrors.ErrFailedToSyncPlanningActivity
	}

	var integrationsSynced int
	var activitiesAdded int64
	for _, integration := range integrations {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		added, err := s.syncIntegration(ctx, integration)
		if err != nil {
			log.Printf("planning sync failed for user %s provider %s: %v", integration.UserID, integration.Provider, err)
			continue
		}
		integrationsSynced++
		activitiesAdded += added
	}

	return &dto.SyncActivityResponse{
		PlanningIntegrationsSynced: integrationsSynced,
		PlanningActivitiesAdded:    int(activitiesAdded),
	}, nil
}

func (s *planningSyncService) SyncForUser(ctx context.Context, userID string) (*dto.SyncActivityResponse, error) {
	integrations, err := s.repo.ListPlanningIntegrationsByUserID(userID)
	if err != nil {
		return nil, apperrors.ErrFailedToSyncPlanningActivity
	}
	if len(integrations) == 0 {
		return nil, apperrors.ErrPlanningIntegrationNotFound
	}

	var integrationsSynced int
	var activitiesAdded int64
	for _, integration := range integrations {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		added, err := s.syncIntegration(ctx, integration)
		if err != nil {
			log.Printf("planning sync failed for user %s provider %s: %v", userID, integration.Provider, err)
			continue
		}
		integrationsSynced++
		activitiesAdded += added
	}

	if integrationsSynced == 0 {
		return nil, apperrors.ErrFailedToSyncPlanningActivity
	}

	return &dto.SyncActivityResponse{
		PlanningIntegrationsSynced: integrationsSynced,
		PlanningActivitiesAdded:    int(activitiesAdded),
	}, nil
}

func (s *planningSyncService) syncIntegration(ctx context.Context, integration models.PlanningIntegration) (int64, error) {
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}

	provider, err := s.registry.Get(integration.Provider)
	if err != nil {
		return 0, err
	}

	since := syncSince(integration.LastSyncAt, s.lookbackDays)
	events, err := provider.FetchActivities(
		toPlanningCredentials(integration),
		toPlanningAccount(integration),
		since,
	)
	if err != nil {
		return 0, err
	}

	activities := make([]models.PlanningActivity, 0, len(events))
	for _, event := range events {
		activities = append(activities, models.PlanningActivity{
			UserID:                integration.UserID,
			PlanningIntegrationID: integration.ID,
			Provider:              integration.Provider,
			ActivityType:          event.ActivityType,
			ItemKey:               event.ItemKey,
			ItemTitle:             event.ItemTitle,
			Status:                event.Status,
			PreviousStatus:        event.PreviousStatus,
			Comment:               event.Comment,
			ExternalID:            event.ExternalID,
			OccurredAt:            event.OccurredAt,
		})
	}

	added, err := s.repo.SaveNewPlanningActivities(activities)
	if err != nil {
		return 0, err
	}

	now := time.Now().UTC()
	integration.LastSyncAt = &now
	if err := s.repo.SavePlanningIntegration(&integration); err != nil {
		return added, err
	}

	return added, nil
}

func syncSince(lastSync *time.Time, lookbackDays int) time.Time {
	defaultSince := time.Now().UTC().AddDate(0, 0, -lookbackDays)
	if lastSync == nil {
		return defaultSince
	}
	since := lastSync.Add(-1 * time.Hour)
	if since.Before(defaultSince) {
		return defaultSince
	}
	return since
}
