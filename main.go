package main

import (
	"fmt"

	"github.com/creatorstation/toolbox/internal/cron"
	"github.com/creatorstation/toolbox/internal/media"
	"github.com/creatorstation/toolbox/internal/misc"
	"github.com/gofiber/fiber/v2"
	"golang.org/x/exp/rand"
)

func main() {
	app := fiber.New(fiber.Config{
		//1 GB
		BodyLimit: 1024 * 1024 * 1024,
	})

	app.Get("/health", func(c *fiber.Ctx) error {
		randNum := rand.Intn(100)

		return c.JSON(fiber.Map{
			"status": "ok",
			"rand":   randNum,
		})
	})

	media.MountController(app.Group("/media"))
	misc.MountController(app.Group("/misc"))

	fmt.Println("Server is running on :8080")

	cron.SetupTranscriptionCron()
	cron.SetupStoryTranscriptionCron()

	cron.MountPostController(app.Group("/cron"))
	cron.MountStoryController(app.Group("/cron"))

	app.Listen(":8080")
}
