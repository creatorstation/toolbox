package cron

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"strconv"
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

	istanbulLoc, err := time.LoadLocation("Europe/Istanbul")
	if err != nil {
		log.Fatalf("Failed to load timezone: %v", err)
	}

	c := cron.New(cron.WithLocation(istanbulLoc))

	// Schedule the job to run at 5 AM Istanbul time
	_, err = c.AddFunc("0 5 * * *", runStoryTranscriptionJob)
	if err != nil {
		log.Fatalf("Failed to add story transcription cron job: %v", err)
	}

	c.Start()
	log.Println("Story transcription cron job scheduled to run at 5 AM Istanbul time")
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

	log.Printf("Found %d stories to transcribe", len(stories))

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

	var transcriptionText string

	// If video is small enough for direct transcription
	if contentLength < 29*1024*1024 {
		// Download and transcribe
		client := resty.New()
		resp, err := client.R().Get(videoURL)
		if err != nil {
			log.Printf("Error downloading video for story ID %s: %v", story.StoryID, err)
			updateStoryDownloaded(story.StoryID, false)
			return
		}

		transcriptionText, err = transcribeStoryAudio(resp.Body())
		if err != nil {
			log.Printf("Error transcribing audio for story ID %s: %v", story.StoryID, err)
			return
		}
	} else {
		// Convert to MP3 first
		mp3URL, err := convertStoryToMP3(videoURL)
		if err != nil {
			log.Printf("Error converting to MP3 for story ID %s: %v", story.StoryID, err)
			return
		}

		// Download MP3
		client := resty.New()
		resp, err := client.R().Get(mp3URL)
		if err != nil {
			log.Printf("Error downloading MP3 for story ID %s: %v", story.StoryID, err)
			return
		}

		// Transcribe MP3
		transcriptionText, err = transcribeStoryAudio(resp.Body())
		if err != nil {
			log.Printf("Error transcribing MP3 for story ID %s: %v", story.StoryID, err)
			return
		}
	}

	// Update story with transcription
	err = updateStoryTranscription(story.StoryID, transcriptionText)
	if err != nil {
		log.Printf("Error updating story ID %s: %v", story.StoryID, err)
		return
	}

	log.Printf("Successfully transcribed story ID: %s", story.StoryID)
}

// getVideoSize checks the size of a video without downloading it
func getVideoSize(url string) (int64, error) {
	client := resty.New()
	resp, err := client.R().Head(url)
	if err != nil {
		return 0, err
	}

	contentLengthStr := resp.Header().Get("Content-Length")
	contentLength, _ := strconv.ParseInt(contentLengthStr, 10, 64)

	return contentLength, nil
}

// convertStoryToMP3 converts a video to MP3 using the external service
func convertStoryToMP3(videoURL string) (string, error) {
	client := resty.New()

	// Prepare request to the MP4 to MP3 conversion service
	resp, err := client.R().
		SetHeader("x-api-key", "gufdviesw87ry4t5uhgfdijresw839r457tuy3r29iewjokdfruhgtby5438i9e").
		SetBody(map[string]string{
			"media_uri": videoURL,
		}).
		Post("https://cs-toolbox-449168770512.us-central1.run.app/media/mp4-to-mp3")

	if err != nil {
		return "", fmt.Errorf("error calling MP4 to MP3 service: %v", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return "", fmt.Errorf("MP4 to MP3 service returned status code %d", resp.StatusCode())
	}

	// Parse the response to get the MP3 URL
	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return "", fmt.Errorf("error parsing MP3 service response: %v", err)
	}

	mp3URL, ok := result["url"].(string)
	if !ok {
		return "", fmt.Errorf("MP3 URL not found in response")
	}

	return mp3URL, nil
}

// transcribeStoryAudio transcribes audio data
func transcribeStoryAudio(audioData []byte) (string, error) {
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
	req, err := http.NewRequest("POST", "https://go-whisper-2-449168770512.us-central1.run.app/v1/audio/transcriptions", &b)
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
