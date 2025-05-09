package appcron

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/creatorstation/toolbox/internal/db"
	"github.com/go-resty/resty/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/robfig/cron/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Story represents a story document in MongoDB
type Story struct {
	StoryID       string `json:"story_id" bson:"story_id"`
	InstAccount   string `json:"inst_account" bson:"inst_account"`
	PublishedAt   int64  `json:"published_at" bson:"published_at"`
	MediaType     string `json:"media_type" bson:"media_type"`
	HasAudio      bool   `json:"has_audio,omitempty" bson:"has_audio,omitempty"`
	Downloaded    bool   `json:"downloaded,omitempty" bson:"downloaded,omitempty"`
	Transcription string `json:"transcription,omitempty" bson:"transcription,omitempty"`
}

// SetupStoryTranscriptionCron sets up the cron job for story transcription
func SetupStoryTranscriptionCron() {
	// Connect to MongoDB
	db.ConnectMongo()

	c := cron.New()

	// Schedule the job to run every 6 hours
	_, err := c.AddFunc("0 */6 * * *", runStoryTranscriptionJob)
	if err != nil {
		log.Fatalf("Failed to add story transcription cron job: %v", err)
	}

	c.Start()
	log.Println("Story transcription cron job scheduled to run every 6 hours")
}

// MountStoryController mounts the story transcription controller
func MountStoryController(router fiber.Router) {
	router.Post("/run-story-transcription", func(c *fiber.Ctx) error {
		go runStoryTranscriptionJob()
		return c.JSON(fiber.Map{
			"message": "Story transcription job started",
		})
	})
}

// runStoryTranscriptionJob runs the story transcription job
func runStoryTranscriptionJob() {
	log.Println("Starting story transcription job")

	// Get stories that need transcription
	stories, err := getStoriesForTranscription()
	if err != nil {
		log.Printf("Error getting stories: %v", err)
		return
	}

	log.Printf("Found %d stories to transcribe", len(stories))

	// Process each story
	for _, story := range stories {
		processStory(story)
	}

	log.Println("Story transcription job completed")
}

// getStoriesForTranscription gets stories that need transcription
func getStoriesForTranscription() ([]Story, error) {
	collection := db.GetMongoDB().Collection("stories")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Query to get stories that need transcription
	filter := bson.M{
		"media_type":    "v",
		"transcription": bson.M{"$exists": false},
		"downloaded":    true,
		"has_audio":     true,
	}

	opts := options.Find().SetSort(bson.D{bson.E{Key: "published_at", Value: -1}})

	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var stories []Story
	if err = cursor.All(ctx, &stories); err != nil {
		return nil, err
	}

	return stories, nil
}

// processStory processes a single story
func processStory(story Story) {
	log.Printf("Processing story ID: %s", story.StoryID)

	// Construct the video URL
	videoURL := fmt.Sprintf("https://5f3ed6f04fb403797d015bbb315fcd55013fef40b95218be44a32bca98c22e7.s3.us-east-1.amazonaws.com/story_harvester/s/%s/v/%s",
		story.InstAccount, story.StoryID)

	// Get video content length without downloading
	contentLength, err := getVideoSize(videoURL)
	if err != nil {
		log.Printf("Error checking video size for story ID %s: %v", story.StoryID, err)
		updateStoryDownloaded(story.StoryID, false)
		return
	}

	if contentLength > 300*1024*1024 {
		f, ferr := os.OpenFile("large_media_ids.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if ferr == nil {
			defer f.Close()
			f.WriteString(story.StoryID + "\n")
		}
		log.Printf("Story ID %s is larger than 500MB, skipping.", story.StoryID)
		return
	}

	var transcriptionText string

	// If video is small enough for direct transcription
	if contentLength < 29*1024*1024 {
		// Download and transcribe
		client := resty.New()
		resp, err := client.R().SetHeader("User-Agent", "toolbox-processStory").Get(videoURL)
		if err != nil {
			log.Printf("Error downloading video for story ID %s: %v", story.StoryID, err)
			updateStoryDownloaded(story.StoryID, false)
			return
		}

		transcriptionText, err = TranscribeAudio(resp.Body(), "https://go-whisper-2-449168770512.us-central1.run.app/v1/audio/transcriptions")
		if err != nil {
			log.Printf("Error transcribing audio for story ID %s: %v", story.StoryID, err)
			return
		}
	} else {
		// Convert to MP3 first
		mp3FilePath, err := convertToMP3(videoURL)
		if err != nil {
			log.Printf("Error converting to MP3 for story ID %s: %v", story.StoryID, err)
			return
		}

		// Read the MP3 file
		mp3Data, err := os.ReadFile(mp3FilePath)
		if err != nil {
			log.Printf("Error reading MP3 file for story ID %s: %v", story.StoryID, err)
			return
		}

		// Clean up the temporary file
		defer os.Remove(mp3FilePath)

		// Transcribe MP3
		transcriptionText, err = TranscribeAudio(mp3Data, "https://go-whisper-2-449168770512.us-central1.run.app/v1/audio/transcriptions")
		if err != nil {
			log.Printf("Error transcribing MP3 for story ID %s: %v", story.StoryID, err)
			return
		}
	}

	// Replace "Altyazı M.K." with a dot character
	transcriptionText = strings.Replace(transcriptionText, "Altyazı M.K.", ".", -1)

	// Update story with transcription
	err = updateStoryTranscription(story.StoryID, transcriptionText)
	if err != nil {
		log.Printf("Error updating story ID %s: %v", story.StoryID, err)
		return
	}

	log.Printf("Successfully transcribed story ID: %s", story.StoryID)
}

// updateStoryTranscription updates the transcription for a story
func updateStoryTranscription(storyID string, transcription string) error {
	collection := db.GetMongoDB().Collection("stories")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{"story_id": storyID}
	update := bson.M{"$set": bson.M{"transcription": transcription}}

	_, err := collection.UpdateOne(ctx, filter, update)
	return err
}

// updateStoryDownloaded updates the downloaded status for a story
func updateStoryDownloaded(storyID string, downloaded bool) error {
	collection := db.GetMongoDB().Collection("stories")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{"story_id": storyID}
	update := bson.M{"$set": bson.M{"downloaded": downloaded}}

	_, err := collection.UpdateOne(ctx, filter, update)
	return err
}
