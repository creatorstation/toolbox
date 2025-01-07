package media

// Import resty into your code and refer it as `resty`.
import (
	"bytes"
	"fmt"
	"log"

	"github.com/creatorstation/toolbox/pkg/convert"
	"github.com/creatorstation/toolbox/pkg/img"
	"github.com/creatorstation/toolbox/pkg/video"
	"github.com/creatorstation/toolbox/pkg/web"
	"github.com/gofiber/fiber/v2"
)

func MountController(router fiber.Router) {
	router.Post("/mp4-to-mp3", ConvertMP4ToMP3)
	router.Post("/resize-image", ResizeImage)
	router.Post("/quicktime-to-mp4", ConvertQuicktimeToMP4)
	router.Post("/thumbnail", GenerateThumbnail)
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

	mp4, err := web.FetchMedia(body.MediaURI)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	mp3, err := convert.ConvertMP4ToMP3(mp4)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	c.Context().SetContentType("audio/mpeg")
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

	isHEIF := file.Header.Get("Content-Type") == "image/heif" || file.Header.Get("Content-Type") == "image/heic"
	jpegImage := convert.JPEG(buf.Bytes(), isHEIF)

	downscaleTo := 23.0

	if isHEIF {
		downscaleTo = 5.0
	}

	resized, err := img.Downscale(&jpegImage, downscaleTo)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	fmt.Println("Before", len(jpegImage), "After", len(*resized))

	c.Context().SetContentType("image/jpeg")
	return c.Status(fiber.StatusOK).Send(*resized)
}

func ConvertQuicktimeToMP4(c *fiber.Ctx) error {
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

	mp4, err := convert.ConvertQuicktimeToMP4(buf.Bytes())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	c.Context().SetContentType("video/mp4")
	return c.Status(fiber.StatusOK).Send(mp4)
}

func GenerateThumbnail(c *fiber.Ctx) error {
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

	thumbnail, err := video.Thumbnail(buf.Bytes())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	c.Context().SetContentType("image/jpeg")
	return c.Status(fiber.StatusOK).Send(thumbnail)
}
