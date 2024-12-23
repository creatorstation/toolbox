package video

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func Thumbnail(input []byte) ([]byte, error) {

	tempDir, err := os.MkdirTemp("", "thumbnail_temp")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary directory: %v", err)
	}

	defer os.RemoveAll(tempDir)

	inputPath := filepath.Join(tempDir, "input.mp4")
	outputPath := filepath.Join(tempDir, "output.png")

	if err := os.WriteFile(inputPath, input, 0644); err != nil {
		return nil, fmt.Errorf("failed to write input video to temp file: %v", err)
	}

	cmd := exec.Command(
		"ffmpeg",
		"-i", inputPath,
		"-ss", "00:00:01.000",
		"-vframes", "1",
		"-y",
		outputPath,
	)

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg command failed: %v", err)
	}

	thumbnail, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read thumbnail from temp file: %v", err)
	}

	return thumbnail, nil
}
