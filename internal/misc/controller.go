package misc

import "github.com/gofiber/fiber/v2"

func MountController(router fiber.Router) {
	router.Post("/slides-to-pptx", ConvertSlidesToPPTX)
}

func ConvertSlidesToPPTX(c *fiber.Ctx) error {
	return c.SendString("Hello, World!")
}
