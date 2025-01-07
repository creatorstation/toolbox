package misc

import (
	"os"
	"os/exec"
)

// ReSavePPTX takes a path to a PPTX file and overwrites it with a new PPTX file.
func ReSavePPTX(path string) error {
	if err := exec.Command("unoconv", "-f", "pptx", "--output", path+"_new", path).Run(); err != nil {
		return err
	}

	if err := os.Remove(path); err != nil {
		return err
	}

	if err := os.Rename(path+"_new.pptx", path); err != nil {
		return err
	}

	return nil
}

// EmbedVideos takes a path to a PPTX file and embeds videos into it.
func EmbedVideos(path string) error {
	if err := ReSavePPTX(path); err != nil {
		return err
	}

	if err := exec.Command("./PPTXCreator", path, path+"_embed").Run(); err != nil {
		return err
	}

	if err := os.Remove(path); err != nil {
		return err
	}

	if err := os.Rename(path+"_embed", path); err != nil {
		return err
	}

	return nil
}
