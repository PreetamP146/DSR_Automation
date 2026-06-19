package handlers

import (
	"dsr-automation/internal/dto"
	"dsr-automation/internal/services"
	apperrors "dsr-automation/pkg/utils/errors"
	"errors"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type DSRHandler interface {
	GetActivity(c *fiber.Ctx) error
	Generate(c *fiber.Ctx) error
	ListReports(c *fiber.Ctx) error
	GetReport(c *fiber.Ctx) error
}

type dsrHandler struct {
	dsrService services.DSRService
}

func NewDSRHandler(dsrService services.DSRService) DSRHandler {
	return &dsrHandler{dsrService: dsrService}
}

func (h *dsrHandler) GetActivity(c *fiber.Ctx) error {
	userID, err := authenticatedUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	reportDate := c.Query("report_date")
	if reportDate == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "report_date is required",
		})
	}

	projectIDs := parseCSVQuery(c.Query("git_project_ids"))
	response, err := h.dsrService.GetActivity(userID, reportDate, projectIDs)
	if err != nil {
		return dsrError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(response)
}

func (h *dsrHandler) Generate(c *fiber.Ctx) error {
	userID, err := authenticatedUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var req dto.GenerateDSRRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.ReportDate == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "report_date is required",
		})
	}

	response, err := h.dsrService.Generate(userID, &req)
	if err != nil {
		return dsrError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(response)
}

func (h *dsrHandler) ListReports(c *fiber.Ctx) error {
	userID, err := authenticatedUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	limit, err := parsePositiveIntQuery(c.Query("limit"), 20)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "limit must be a positive integer"})
	}
	offset, err := parseNonNegativeIntQuery(c.Query("offset"), 0)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "offset must be a non-negative integer"})
	}

	response, err := h.dsrService.ListReports(userID, c.Query("from"), c.Query("to"), limit, offset)
	if err != nil {
		return dsrError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(response)
}

func (h *dsrHandler) GetReport(c *fiber.Ctx) error {
	userID, err := authenticatedUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	reportID := c.Params("id")
	if reportID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "report id is required",
		})
	}

	response, err := h.dsrService.GetReport(userID, reportID)
	if err != nil {
		return dsrError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(response)
}

func parseCSVQuery(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	ids := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			ids = append(ids, part)
		}
	}
	return ids
}

func parsePositiveIntQuery(value string, defaultValue int) (int, error) {
	if strings.TrimSpace(value) == "" {
		return defaultValue, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return 0, errors.New("invalid limit")
	}
	return parsed, nil
}

func parseNonNegativeIntQuery(value string, defaultValue int) (int, error) {
	if strings.TrimSpace(value) == "" {
		return defaultValue, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return 0, errors.New("invalid offset")
	}
	return parsed, nil
}

func dsrError(c *fiber.Ctx, err error) error {
	status := fiber.StatusInternalServerError
	switch {
	case errors.Is(err, apperrors.ErrInvalidReportDate):
		status = fiber.StatusBadRequest
	case errors.Is(err, apperrors.ErrNoTrackedGitProjects),
		errors.Is(err, apperrors.ErrGitProjectNotFound):
		status = fiber.StatusNotFound
	case errors.Is(err, apperrors.ErrDSRReportNotFound):
		status = fiber.StatusNotFound
	case errors.Is(err, apperrors.ErrInvalidGitAccessToken):
		status = fiber.StatusUnauthorized
	}

	return c.Status(status).JSON(fiber.Map{
		"error": err.Error(),
	})
}
