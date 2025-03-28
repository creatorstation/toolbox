package convert

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
)

func ConvertMP4ToMP3(input []byte) ([]byte, error) {
	cmd := exec.Command(
		"ffmpeg",
		"-i", "pipe:0",
		"-f", "mp3",
		"pipe:1",
		"-y",
	)

	cmd.Stdin = bytes.NewReader(input)

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("ffmpeg error: %v, details: %s", err, stderr.String())
	}

	// Return the MP3 data.
	return out.Bytes(), nil
}

func JPEG(input []byte, isHEIC bool) []byte {
	if isHEIC {
		inFile, err := os.CreateTemp("", "heic-input-*.heic")
		if err != nil {
			fmt.Println("error creating temp input file:", err)
			return nil
		}
		defer os.Remove(inFile.Name())

		_, err = inFile.Write(input)
		inFile.Close()
		if err != nil {
			fmt.Println("error writing to temp input file:", err)
			return nil
		}

		outFile, err := os.CreateTemp("", "heic-output-*.jpg")
		if err != nil {
			fmt.Println("error creating temp output file:", err)
			return nil
		}
		outName := outFile.Name()
		outFile.Close()
		defer os.Remove(outName)

		cmd := exec.Command("heif-convert", inFile.Name(), outName)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		err = cmd.Run()
		if err != nil {
			fmt.Println("heif-convert error:", err)
			fmt.Println("details:", stderr.String())
			return nil
		}

		outBytes, err := os.ReadFile(outName)
		if err != nil {
			fmt.Println("error reading output file:", err)
			return nil
		}

		return outBytes

	} else {
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

func ConvertQuicktimeToMP4(input []byte) ([]byte, error) {
	cmd := exec.Command(
		"ffmpeg",
		"-i", "pipe:0",
		"-c:v", "libx264",
		"-c:a", "aac",
		"-movflags", "frag_keyframe+empty_moov",
		"-f", "mp4",
		"pipe:1",
		"-y",
	)

	cmd.Stdin = bytes.NewReader(input)

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("ffmpeg error: %v, details: %s", err, stderr.String())
	}

	return out.Bytes(), nil
}
