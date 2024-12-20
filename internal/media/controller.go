package media

// Import resty into your code and refer it as `resty`.
import (
	"bytes"
	"fmt"
	"log"

	"github.com/creatorstation/toolbox/pkg/convert"
	"github.com/creatorstation/toolbox/pkg/img"
	"github.com/gofiber/fiber/v2"
)

func MountController(router fiber.Router) {
	router.Post("/mp4-to-mp3", ConvertMP4ToMP3)
	router.Post("/resize-image", ResizeImage)
}

func ConvertMP4ToMP3(c *fiber.Ctx) error {
	var body MediaURLBody
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if err := body.Validate(); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	log.Printf("Converting MP4 to MP3: %s", body.MediaURI)

	mp4, err := fetchMedia(body.MediaURI)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Convert the MP4 file to MP3.
	mp3, err := convert.ConvertMP4ToMP3(mp4)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Return the MP3 file.
	c.Context().SetContentType("audio/mpeg")

	log.Printf("MP4 Size: %dMB | MP3 Size: %dMB", len(mp4)/1024/1024, len(mp3)/1024/1024)

	return c.Status(fiber.StatusOK).Send(mp3)
}

func ResizeImage(c *fiber.Ctx) error {
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

	contentType := file.Header.Get("Content-Type")
	jpegImage := convert.JPEG(buf.Bytes(), contentType == "image/heic")

	downscaleTo := 23.0

	if contentType == "image/heic" {
		downscaleTo = 5.0
	}

	resized, err := img.Downscale(&jpegImage, downscaleTo)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	fmt.Println("Before", len(jpegImage), "After", len(*resized))

	c.Response().Header.SetContentType("image/jpeg")
	return c.Status(fiber.StatusOK).Send(*resized)
}
