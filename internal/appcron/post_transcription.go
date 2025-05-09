package appcron

import (
	"log"
	"os"
	"strings"

	"github.com/creatorstation/toolbox/internal/db"
	"github.com/creatorstation/toolbox/internal/models"
	"github.com/go-resty/resty/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/robfig/cron/v3"
)

type TranscriptionResponse struct {
	Text string `json:"text"`
}

func SetupPostTranscriptionCron() {
	db.ConnectPG()

	c := cron.New()

	// Schedule the job to run every 6 hours
	_, err := c.AddFunc("0 */6 * * *", runPostTranscriptionJob)
	if err != nil {
		log.Fatalf("Failed to add cron job: %v", err)
	}

	c.Start()
	log.Println("Post transcription cron job scheduled to run every 6 hours")
}

func MountPostController(router fiber.Router) {
	router.Post("/run-post-transcription", func(c *fiber.Ctx) error {
		go runPostTranscriptionJob()
		return c.JSON(fiber.Map{
			"message": "Post transcription job started",
		})
	})
}

// runTranscriptionJob runs the transcription job
func runPostTranscriptionJob() {
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
	result := db.GetPGDB().
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
	contentLength, err := getVideoSize(post.VideoURL)
	if err != nil {
		log.Printf("Error checking video size for post ID %s: %v", post.ID, err)
		return
	}

	// Separate check for > 500MB
	if contentLength > 500*1024*1024 {
		f, ferr := os.OpenFile("large_media_ids.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if ferr == nil {
			defer f.Close()
			f.WriteString(post.ID + "\n")
		}
		log.Printf("Post ID %s is larger than 500MB, skipping.", post.ID)
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
		resp, err := client.R().SetHeader("User-Agent", "toolbox-processPost").Get(post.VideoURL)
		if err != nil {
			log.Printf("Error downloading video for post ID %s: %v", post.ID, err)
			return
		}

		transcriptionText, err = TranscribeAudio(resp.Body(), "https://go-whisper-449168770512.us-central1.run.app/v1/audio/transcriptions")

		if err != nil {
			log.Printf("Error transcribing audio for post ID %s: %v", post.ID, err)
			return
		}
	} else {
		mp3FilePath, err := convertToMP3(post.VideoURL)
		if err != nil {
			log.Printf("Error converting to MP3 for post ID %s: %v", post.ID, err)
			return
		}

		mp3Data, err := os.ReadFile(mp3FilePath)
		if err != nil {
			log.Printf("Error reading MP3 file for post ID %s: %v", post.ID, err)
			return
		}
		defer os.Remove(mp3FilePath)

		transcriptionText, err = TranscribeAudio(mp3Data, "https://go-whisper-449168770512.us-central1.run.app/v1/audio/transcriptions")
		if err != nil {
			log.Printf("Error transcribing MP3 for post ID %s: %v", post.ID, err)
			return
		}
	}

	// Replace "Altyazı M.K." with a dot character
	transcriptionText = strings.Replace(transcriptionText, "Altyazı M.K.", ".", -1)

	// Update post with transcription
	err = updatePostTranscription(post.ID, transcriptionText)
	if err != nil {
		log.Printf("Error updating post ID %s: %v", post.ID, err)
		return
	}

	log.Printf("Successfully transcribed post ID: %s", post.ID)
}

// updatePostTranscription updates the transcription for a post
func updatePostTranscription(postID string, transcription string) error {
	return db.GetPGDB().Table("n8n_influencer_posts").Where("id = ?", postID).Update("transcription", transcription).Error
}
