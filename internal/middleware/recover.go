package middleware

import (
	"log"

	"github.com/gofiber/fiber/v2"
)

func Recover() fiber.Handler {
	return func(c *fiber.Ctx) error {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("panic recovered: %v", r)
				_ = c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "internal server error",
				})
			}
		}()
		return c.Next()
	}
}
