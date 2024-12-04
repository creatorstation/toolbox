package main

import (
	"github.com/creatorstation/toolbox/internal/media"
	"github.com/gofiber/fiber/v2"
)

func main() {
	app := fiber.New()

	media.MountController(app.Group("/media"))

	app.Listen(":8080")
}
