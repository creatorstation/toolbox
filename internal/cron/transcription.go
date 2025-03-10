package cron

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/creatorstation/toolbox/internal/db"
	"github.com/creatorstation/toolbox/internal/models"
	"github.com/creatorstation/toolbox/pkg/convert"
	"github.com/go-resty/resty/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/robfig/cron/v3"
)

type TranscriptionResponse struct {
	Text string `json:"text"`
}

func SetupTranscriptionCron() {
	db.Connect()

	istanbulLoc, err := time.LoadLocation("Europe/Istanbul")
	if err != nil {
		log.Fatalf("Failed to load timezone: %v", err)
	}

	c := cron.New(cron.WithLocation(istanbulLoc))

	// Schedule the job to run at 5 AM Istanbul time
	_, err = c.AddFunc("0 5 * * *", runTranscriptionJob)
	if err != nil {
		log.Fatalf("Failed to add cron job: %v", err)
	}

	c.Start()
	log.Println("Transcription cron job scheduled to run at 5 AM Istanbul time")
}

func MountController(router fiber.Router) {
	router.Post("/cron/transcription/run", func(c *fiber.Ctx) error {
		go runTranscriptionJob()
		return c.JSON(fiber.Map{
			"message": "Transcription job started",
		})
	})
}

// runTranscriptionJob runs the transcription job
func runTranscriptionJob() {
	log.Println("Starting transcription job")

	// Get posts that need transcription
	posts, err := getPostsForTranscription()
	if err != nil {
		log.Printf("Error getting posts: %v", err)
		return
	}

	log.Printf("Found %d posts to transcribe", len(posts))

	// Process each post
	for _, post := range posts {
		processPost(post)
	}

	log.Println("Transcription job completed")
}

// getPostsForTranscription gets posts that need transcription
func getPostsForTranscription() ([]models.InfluencerPost, error) {
	var posts []models.InfluencerPost

	// Query to get posts that need transcription
	result := db.GetDB().
		Table("n8n_influencer_posts p").
		Joins("LEFT JOIN n8n_influencer_accounts a ON p.account_id = a.id").
		Where("a.collect_stories = true").
		Where("p.transcription IS NULL").
		Where("p.video_url IS NOT NULL").
		Where("substring(p.video_url from 'oe=([0-9A-Fa-f]+)') IS NOT NULL").
		Where("('x' || substring(p.video_url from 'oe=([0-9A-Fa-f]+)'))::bit(32)::int >= EXTRACT(epoch FROM now())::int").
		Order("p.taken_at desc").
		Find(&posts)

	if result.Error != nil {
		return nil, result.Error
	}

	return posts, nil
}

// processPost processes a single post
func processPost(post models.InfluencerPost) {
	log.Printf("Processing post ID: %s", post.ID)

	// Get video content length without downloading
	contentLength, err := downloadVideo(post.VideoURL)
	if err != nil {
		log.Printf("Error checking video size for post ID %s: %v", post.ID, err)
		return
	}

	// Check if video is too large
	if contentLength > 100*1024*1024 {
		log.Printf("Video too large for post ID %s: %d bytes", post.ID, contentLength)
		return
	}

	var transcriptionText string

	// If video is small enough for direct transcription
	if contentLength < 29*1024*1024 {
		// Download and transcribe
		client := resty.New()
		resp, err := client.R().Get(post.VideoURL)
		if err != nil {
			log.Printf("Error downloading video for post ID %s: %v", post.ID, err)
			return
		}

		transcriptionText, err = transcribeAudio(resp.Body())
		if err != nil {
			log.Printf("Error transcribing audio for post ID %s: %v", post.ID, err)
			return
		}
	} else {
		// Convert to MP3 first
		mp3URL, err := convertToMP3(post.VideoURL)
		if err != nil {
			log.Printf("Error converting to MP3 for post ID %s: %v", post.ID, err)
			return
		}

		// Download MP3
		client := resty.New()
		resp, err := client.R().Get(mp3URL)
		if err != nil {
			log.Printf("Error downloading MP3 for post ID %s: %v", post.ID, err)
			return
		}

		// Transcribe MP3
		transcriptionText, err = transcribeAudio(resp.Body())
		if err != nil {
			log.Printf("Error transcribing MP3 for post ID %s: %v", post.ID, err)
			return
		}
	}

	// Update post with transcription
	err = updatePostTranscription(post.ID, transcriptionText)
	if err != nil {
		log.Printf("Error updating post ID %s: %v", post.ID, err)
		return
	}

	log.Printf("Successfully transcribed post ID: %s", post.ID)
}

// downloadVideo checks the size of a video without downloading it
func downloadVideo(url string) (int64, error) {
	client := resty.New()
	resp, err := client.R().Head(url)
	if err != nil {
		return 0, err
	}

	contentLengthStr := resp.Header().Get("Content-Length")
	contentLength, _ := strconv.ParseInt(contentLengthStr, 10, 64)

	return contentLength, nil
}

// convertToMP3 converts a video to MP3
func convertToMP3(videoURL string) (string, error) {
	// Download the video
	client := resty.New()
	resp, err := client.R().Get(videoURL)
	if err != nil {
		return "", fmt.Errorf("error downloading video: %v", err)
	}

	// Convert to MP3 using local function
	mp3Data, err := convert.ConvertMP4ToMP3(resp.Body())
	if err != nil {
		return "", fmt.Errorf("error converting to MP3: %v", err)
	}

	// Create a temporary file to store the MP3
	tempFile, err := os.CreateTemp("", "converted-*.mp3")
	if err != nil {
		return "", fmt.Errorf("error creating temp file: %v", err)
	}
	defer tempFile.Close()

	// Write MP3 data to the temp file
	_, err = tempFile.Write(mp3Data)
	if err != nil {
		os.Remove(tempFile.Name())
		return "", fmt.Errorf("error writing to temp file: %v", err)
	}

	// Return the path to the temp file
	return tempFile.Name(), nil
}

func transcribeAudio(audioData []byte) (string, error) {
	// Create a buffer to write the multipart form
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	// Add the model parameter
	err := w.WriteField("model", "ggml-large-v3-turbo")
	if err != nil {
		return "", err
	}

	// Create a form file for the audio data
	fw, err := w.CreateFormFile("file", "audio.mp3")
	if err != nil {
		return "", err
	}

	// Write the audio data to the form file
	_, err = io.Copy(fw, bytes.NewReader(audioData))
	if err != nil {
		return "", err
	}

	// Close the writer
	w.Close()

	// Create the request
	req, err := http.NewRequest("POST", "https://go-whisper-449168770512.us-central1.run.app/v1/audio/transcriptions", &b)
	if err != nil {
		return "", err
	}

	// Set the content type
	req.Header.Set("Content-Type", w.FormDataContentType())

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Read the response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to transcribe audio: %s", string(respBody))
	}

	// Parse the response
	var result TranscriptionResponse
	err = json.Unmarshal(respBody, &result)
	if err != nil {
		return "", err
	}

	return result.Text, nil
}

// updatePostTranscription updates the transcription for a post
func updatePostTranscription(postID string, transcription string) error {
	return db.GetDB().Table("n8n_influencer_posts").Where("id = ?", postID).Update("transcription", transcription).Error
}
