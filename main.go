package main

import (
	"fmt"
	"os"

	"github.com/creatorstation/toolbox/internal/appcron"
	"github.com/creatorstation/toolbox/internal/media"
	"github.com/creatorstation/toolbox/internal/misc"
	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
	"golang.org/x/exp/rand"
)

func a() {

}

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
		os.Exit(1)
	}

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

	appcron.SetupTranscriptionCron()
	appcron.SetupStoryTranscriptionCron()

	appcron.MountPostController(app.Group("/cron"))
	appcron.MountStoryController(app.Group("/cron"))

	app.Listen(":8080")
}
