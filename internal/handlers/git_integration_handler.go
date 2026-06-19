package handlers

import (
	"context"
	"dsr-automation/internal/dto"
	"dsr-automation/internal/services"
	apperrors "dsr-automation/pkg/utils/errors"
	"errors"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
)

var errUnauthorized = errors.New("unauthorized")

type GitIntegrationHandler interface {
	Connect(c *fiber.Ctx) error
	ListIntegrations(c *fiber.Ctx) error
	ListProjects(c *fiber.Ctx) error
	UpdateTrackedProjects(c *fiber.Ctx) error
	Sync(c *fiber.Ctx) error
	SyncCommits(c *fiber.Ctx) error
}

type gitIntegrationHandler struct {
	gitService        services.GitIntegrationService
	commitSyncService services.CommitSyncService
}

func NewGitIntegrationHandler(gitService services.GitIntegrationService, commitSyncService services.CommitSyncService) GitIntegrationHandler {
	return &gitIntegrationHandler{
		gitService:        gitService,
		commitSyncService: commitSyncService,
	}
}

func (h *gitIntegrationHandler) Connect(c *fiber.Ctx) error {
	userID, err := authenticatedUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var req dto.ConnectGitRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.Provider == "" || req.BaseURL == "" || req.AccessToken == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "provider, base_url, and access_token are required",
		})
	}
	req.AccessToken = strings.TrimSpace(req.AccessToken)

	response, err := h.gitService.Connect(userID, &req)
	if err != nil {
		return gitIntegrationError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(response)
}

func (h *gitIntegrationHandler) ListIntegrations(c *fiber.Ctx) error {
	userID, err := authenticatedUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	response, err := h.gitService.ListIntegrations(userID)
	if err != nil {
		return gitIntegrationError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(response)
}

func (h *gitIntegrationHandler) ListProjects(c *fiber.Ctx) error {
	userID, err := authenticatedUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	provider := c.Query("provider")
	trackedOnly, err := parseOptionalBoolQuery(c.Query("tracked_only"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "tracked_only must be true or false",
		})
	}

	response, err := h.gitService.ListProjects(userID, provider, trackedOnly)
	if err != nil {
		return gitIntegrationError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(response)
}

func (h *gitIntegrationHandler) UpdateTrackedProjects(c *fiber.Ctx) error {
	userID, err := authenticatedUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var req dto.UpdateTrackedProjectsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.ProjectIDs == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "project_ids is required",
		})
	}

	response, err := h.gitService.UpdateTrackedProjects(userID, &req)
	if err != nil {
		return gitIntegrationError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(response)
}

func (h *gitIntegrationHandler) Sync(c *fiber.Ctx) error {
	userID, err := authenticatedUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var req dto.SyncGitRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.Provider == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "provider is required",
		})
	}

	response, err := h.gitService.Sync(userID, &req)
	if err != nil {
		return gitIntegrationError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(response)
}

func (h *gitIntegrationHandler) SyncCommits(c *fiber.Ctx) error {
	userID, err := authenticatedUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	response, err := h.commitSyncService.SyncForUser(context.Background(), userID)
	if err != nil {
		return gitIntegrationError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(response)
}

func authenticatedUserID(c *fiber.Ctx) (string, error) {
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return "", errUnauthorized
	}
	return userID, nil
}

func parseOptionalBoolQuery(value string) (bool, error) {
	if value == "" {
		return false, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, err
	}
	return parsed, nil
}

func gitIntegrationError(c *fiber.Ctx, err error) error {
	status := fiber.StatusInternalServerError
	switch {
	case errors.Is(err, apperrors.ErrUnsupportedGitProvider),
		errors.Is(err, apperrors.ErrInvalidBaseURL):
		status = fiber.StatusBadRequest
	case errors.Is(err, apperrors.ErrInvalidGitAccessToken):
		status = fiber.StatusBadRequest
	case errors.Is(err, apperrors.ErrGitIntegrationNotFound),
		errors.Is(err, apperrors.ErrGitProjectNotFound),
		errors.Is(err, apperrors.ErrNoTrackedGitProjects):
		status = fiber.StatusNotFound
	}

	return c.Status(status).JSON(fiber.Map{
		"error": err.Error(),
	})
}
