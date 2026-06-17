package handlers

import (
	"dsr-automation/internal/dto"
	"dsr-automation/internal/services"
	apperrors "dsr-automation/pkg/utils/errors"
	"errors"

	"github.com/gofiber/fiber/v2"
)

type GitIntegrationHandler interface {
	Connect(c *fiber.Ctx) error
}

type gitIntegrationHandler struct {
	gitService services.GitIntegrationService
}

func NewGitIntegrationHandler(gitService services.GitIntegrationService) GitIntegrationHandler {
	return &gitIntegrationHandler{gitService: gitService}
}

func (h *gitIntegrationHandler) Connect(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
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

	response, err := h.gitService.Connect(userID, &req)
	if err != nil {
		status := fiber.StatusInternalServerError
		switch {
		case errors.Is(err, apperrors.ErrUnsupportedGitProvider),
			errors.Is(err, apperrors.ErrInvalidBaseURL):
			status = fiber.StatusBadRequest
		case errors.Is(err, apperrors.ErrInvalidGitAccessToken):
			status = fiber.StatusUnauthorized
		}
		return c.Status(status).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(response)
}
