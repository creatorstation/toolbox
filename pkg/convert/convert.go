package convert

import (
	"bytes"
	"fmt"
	"os/exec"
)

func ConvertMP4ToMP3(input []byte) ([]byte, error) {
	// Initialize the FFmpeg command.
	// -i pipe:0      : Read input from stdin.
	// -f mp3         : Specify the output format as MP3.
	// pipe:1          : Write output to stdout.
	// -y             : Overwrite output files without asking.
	cmd := exec.Command(
		"ffmpeg",
		"-i", "pipe:0",
		"-f", "mp3",
		"pipe:1",
		"-y",
	)

	// Set the input for FFmpeg to be the provided MP4 byte array.
	cmd.Stdin = bytes.NewReader(input)

	// Buffers to capture FFmpeg's stdout and stderr.
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	// Run the FFmpeg command.
	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("ffmpeg error: %v, details: %s", err, stderr.String())
	}

	// Return the MP3 data.
	return out.Bytes(), nil
}
