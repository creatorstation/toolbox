package appcron

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"

	"github.com/creatorstation/toolbox/pkg/convert"
	"github.com/go-resty/resty/v2"
)

// TranscribeAudio transcribes audio data using the specified server URL
func TranscribeAudio(audioData []byte, serverURL string) (string, error) {
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
	req, err := http.NewRequest("POST", serverURL, &b)
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

func getVideoSize(url string) (int64, error) {
	client := resty.New()
	resp, err := client.R().SetHeader("User-Agent", "toolbox-getVideoSize").Head(url)
	if err != nil {
		return 0, err
	}

	contentLengthStr := resp.Header().Get("Content-Length")
	contentLength, _ := strconv.ParseInt(contentLengthStr, 10, 64)

	return contentLength, nil
}

func convertToMP3(videoURL string) (string, error) {
	// Download the video
	client := resty.New()
	resp, err := client.R().SetHeader("User-Agent", "toolbox-convertToMP3").Get(videoURL)
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
