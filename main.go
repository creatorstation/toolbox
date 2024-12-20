package main

import (
	"github.com/creatorstation/toolbox/internal/media"
	"github.com/gofiber/fiber/v2"
)

func main() {
	app := fiber.New(fiber.Config{
		//1 GB
		BodyLimit: 1024 * 1024 * 1024,
	})

	media.MountController(app.Group("/media"))

	app.Listen(":8080")
}
