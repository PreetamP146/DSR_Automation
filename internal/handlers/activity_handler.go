package handlers

import (
	"context"
	"dsr-automation/internal/dto"
	"dsr-automation/internal/services"
	apperrors "dsr-automation/pkg/utils/errors"
	"errors"

	"github.com/gofiber/fiber/v2"
)

type ActivityHandler interface {
	RegisterLocalRepository(c *fiber.Ctx) error
	ListLocalRepositories(c *fiber.Ctx) error
	UpdateTrackedLocalRepositories(c *fiber.Ctx) error
	DeleteLocalRepository(c *fiber.Ctx) error
	ConnectPlanning(c *fiber.Ctx) error
	ListPlanningProviders(c *fiber.Ctx) error
	ListPlanningIntegrations(c *fiber.Ctx) error
	GetPlanningIntegration(c *fiber.Ctx) error
	DisconnectPlanning(c *fiber.Ctx) error
	SetActiveRepository(c *fiber.Ctx) error
	GetActiveRepository(c *fiber.Ctx) error
	GetActivitySummary(c *fiber.Ctx) error
	SyncActivity(c *fiber.Ctx) error
}

type activityHandler struct {
	localGitService        services.LocalGitService
	planningService        services.PlanningIntegrationService
	activityTrackerService services.ActivityTrackerService
}

func NewActivityHandler(
	localGitService services.LocalGitService,
	planningService services.PlanningIntegrationService,
	activityTrackerService services.ActivityTrackerService,
) ActivityHandler {
	return &activityHandler{
		localGitService:        localGitService,
		planningService:        planningService,
		activityTrackerService: activityTrackerService,
	}
}

func (h *activityHandler) RegisterLocalRepository(c *fiber.Ctx) error {
	userID, err := authenticatedUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var req dto.RegisterLocalGitRepoRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if req.Path == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "path is required"})
	}

	response, err := h.localGitService.RegisterRepository(userID, &req)
	if err != nil {
		return activityError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(response)
}

func (h *activityHandler) ListLocalRepositories(c *fiber.Ctx) error {
	userID, err := authenticatedUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	trackedOnly, err := parseOptionalBoolQuery(c.Query("tracked_only"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "tracked_only must be true or false"})
	}

	response, err := h.localGitService.ListRepositories(userID, trackedOnly)
	if err != nil {
		return activityError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(response)
}

func (h *activityHandler) UpdateTrackedLocalRepositories(c *fiber.Ctx) error {
	userID, err := authenticatedUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var req dto.UpdateTrackedLocalReposRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if req.RepositoryIDs == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "repository_ids is required"})
	}

	response, err := h.localGitService.UpdateTrackedRepositories(userID, &req)
	if err != nil {
		return activityError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(response)
}

func (h *activityHandler) DeleteLocalRepository(c *fiber.Ctx) error {
	userID, err := authenticatedUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	repositoryID := c.Params("id")
	if repositoryID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "repository id is required"})
	}

	if err := h.localGitService.DeleteRepository(userID, repositoryID); err != nil {
		return activityError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *activityHandler) ListPlanningProviders(c *fiber.Ctx) error {
	_, err := authenticatedUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	return c.Status(fiber.StatusOK).JSON(dto.ListPlanningProvidersResponse{
		Providers: h.planningService.ListProviders(),
	})
}

func (h *activityHandler) ConnectPlanning(c *fiber.Ctx) error {
	userID, err := authenticatedUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var req dto.ConnectPlanningRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if req.Provider == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "provider is required"})
	}

	response, err := h.planningService.Connect(userID, &req)
	if err != nil {
		return activityError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(response)
}

func (h *activityHandler) ListPlanningIntegrations(c *fiber.Ctx) error {
	userID, err := authenticatedUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	response, err := h.planningService.ListIntegrations(userID)
	if err != nil {
		return activityError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(response)
}

func (h *activityHandler) GetPlanningIntegration(c *fiber.Ctx) error {
	userID, err := authenticatedUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	provider := c.Query("provider")
	if provider == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "provider query param is required"})
	}

	response, err := h.planningService.GetIntegration(userID, provider)
	if err != nil {
		return activityError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(response)
}

func (h *activityHandler) DisconnectPlanning(c *fiber.Ctx) error {
	userID, err := authenticatedUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	provider := c.Query("provider")
	if provider == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "provider query param is required"})
	}

	if err := h.planningService.Disconnect(userID, provider); err != nil {
		return activityError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *activityHandler) SetActiveRepository(c *fiber.Ctx) error {
	userID, err := authenticatedUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var req dto.SetActiveRepositoryRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if req.RepositoryName == "" || req.RepositoryPath == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "repository_name and repository_path are required",
		})
	}

	response, err := h.activityTrackerService.SetActiveRepository(userID, &req)
	if err != nil {
		return activityError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(response)
}

func (h *activityHandler) GetActiveRepository(c *fiber.Ctx) error {
	userID, err := authenticatedUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	response, err := h.activityTrackerService.GetActiveRepository(userID)
	if err != nil {
		return activityError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(response)
}

func (h *activityHandler) GetActivitySummary(c *fiber.Ctx) error {
	userID, err := authenticatedUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	response, err := h.activityTrackerService.GetActivitySummary(userID, c.Query("report_date"))
	if err != nil {
		return activityError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(response)
}

func (h *activityHandler) SyncActivity(c *fiber.Ctx) error {
	userID, err := authenticatedUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	response, err := h.activityTrackerService.SyncForUser(context.Background(), userID)
	if err != nil {
		return activityError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(response)
}

func activityError(c *fiber.Ctx, err error) error {
	status := fiber.StatusInternalServerError
	switch {
	case errors.Is(err, apperrors.ErrInvalidLocalGitPath),
		errors.Is(err, apperrors.ErrInvalidPlanningCredentials),
		errors.Is(err, apperrors.ErrUnsupportedPlanningProvider),
		errors.Is(err, apperrors.ErrInvalidActiveRepository),
		errors.Is(err, apperrors.ErrInvalidReportDate):
		status = fiber.StatusBadRequest
	case errors.Is(err, apperrors.ErrLocalGitRepositoryNotFound),
		errors.Is(err, apperrors.ErrPlanningIntegrationNotFound),
		errors.Is(err, apperrors.ErrActiveRepositoryNotFound),
		errors.Is(err, apperrors.ErrNoTrackedLocalGitRepositories):
		status = fiber.StatusNotFound
	}

	return c.Status(status).JSON(fiber.Map{
		"error": err.Error(),
	})
}
