package misc

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/creatorstation/toolbox/pkg/str"
	"github.com/gofiber/fiber/v2"
)

func MountController(router fiber.Router) {
	router.Post("/slides-to-pptx", ConvertSlidesToPPTX)
	router.Get("/agi-screenshot", GetAGIScreenshot)
	router.Get("/agi-screenshot-tab4", GetAGIScreenshotTab4)
}

func ConvertSlidesToPPTX(c *fiber.Ctx) error {
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	fileContent, err := file.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	defer fileContent.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(fileContent)

	// creating a temporary directory in case of multiple files are being processed with the same name
	tempDir, err := os.MkdirTemp("", "slides-to-pptx-*")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// generating a string for the path of the pptx file
	fileName := str.RandomString(10)
	pptxPath := filepath.Join(tempDir, fileName)

	os.WriteFile(pptxPath, buf.Bytes(), 0644)

	if err := EmbedVideos(pptxPath); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	respFile, err := os.ReadFile(pptxPath)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if err := os.RemoveAll(tempDir); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	c.Context().SetContentType("application/vnd.openxmlformats-officedocument.presentationml.presentation")
	return c.Status(fiber.StatusOK).Send(respFile)
}

var screenshotCache *ScreenshotCache

func GetAGIScreenshot(c *fiber.Ctx) error {
	username := c.Query("username")
	elementID := c.Query("elementId")

	if username == "" || elementID == "" {
		return c.Status(fiber.StatusBadRequest).SendString("Missing username or elementId parameter")
	}

	cacheKey := fmt.Sprintf("%s_%s", username, elementID)

	if imgBytes, found := getCachedScreenshot(cacheKey); found {
		c.Set("Content-Type", "image/png")
		return c.Send(imgBytes)
	}

	imgBytes, err := takeScreenshot(username, elementID)
	if err != nil {
		log.Printf("Screenshot error: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to capture screenshot")
	}

	saveToCache(cacheKey, imgBytes)

	c.Set("Content-Type", "image/png")
	return c.Send(imgBytes)
}

func GetAGIScreenshotTab4(c *fiber.Ctx) error {
	username := c.Query("username")
	elementID := c.Query("elementId")

	if username == "" || elementID == "" {
		return c.Status(fiber.StatusBadRequest).SendString("Missing username or elementId parameter")
	}

	cacheKey := fmt.Sprintf("%s_%s_tab4", username, elementID)

	if imgBytes, found := getCachedScreenshot(cacheKey); found {
		c.Set("Content-Type", "image/png")
		return c.Send(imgBytes)
	}

	imgBytes, err := takeScreenshotTab4(username, elementID)
	if err != nil {
		log.Printf("Screenshot tab4 error: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to capture screenshot")
	}

	saveToCache(cacheKey, imgBytes)

	c.Set("Content-Type", "image/png")
	return c.Send(imgBytes)
}
