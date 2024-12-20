package convert

import (
	"bytes"
	"fmt"
	"os"
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

func JPEG(input []byte, isHEIC bool) []byte {
	if isHEIC {
		// Handle HEIC using heif-convert

		// 1. Create a temporary input file for HEIC data
		inFile, err := os.CreateTemp("", "heic-input-*.heic")
		if err != nil {
			fmt.Println("error creating temp input file:", err)
			return nil
		}
		defer os.Remove(inFile.Name())

		// Write the input bytes to the temp file
		_, err = inFile.Write(input)
		inFile.Close()
		if err != nil {
			fmt.Println("error writing to temp input file:", err)
			return nil
		}

		// 2. Create a temporary output file for the converted JPEG
		outFile, err := os.CreateTemp("", "heic-output-*.jpg")
		if err != nil {
			fmt.Println("error creating temp output file:", err)
			return nil
		}
		outName := outFile.Name()
		outFile.Close()
		defer os.Remove(outName)

		// 3. Run heif-convert: heif-convert input.heic output.jpg
		cmd := exec.Command("heif-convert", inFile.Name(), outName)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		err = cmd.Run()
		if err != nil {
			fmt.Println("heif-convert error:", err)
			fmt.Println("details:", stderr.String())
			return nil
		}

		// 4. Read the converted output file
		outBytes, err := os.ReadFile(outName)
		if err != nil {
			fmt.Println("error reading output file:", err)
			return nil
		}

		// Return the JPEG data from the heif-convert output
		return outBytes

	} else {
		// Use ffmpeg as originally done for non-HEIC formats
		args := []string{
			"-i", "pipe:0",
			"-f", "image2",
			"-vframes", "1",
			"-vcodec", "mjpeg",
			"pipe:1",
		}

		cmd := exec.Command("ffmpeg", args...)
		fmt.Println(cmd.String())

		cmd.Stdin = bytes.NewReader(input)

		var out bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &stderr

		err := cmd.Run()
		if err != nil {
			fmt.Println("ffmpeg error:", err)
			fmt.Println("details:", stderr.String())
			return nil
		}

		return out.Bytes()
	}
}
