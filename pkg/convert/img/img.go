package img

import (
	"bytes"
	"fmt"
	"image/jpeg"

	"github.com/sunshineplan/imgconv"
)

func Downscale(imageData *[]byte, maxMPXS float64) (*[]byte, error) {
	reader := bytes.NewReader(*imageData)

	img, err := jpeg.Decode(reader)
	if err != nil {
		return nil, fmt.Errorf("error decoding JPEG: %v", err)
	}

	bounds := img.Bounds()
	width := bounds.Max.X - bounds.Min.X
	height := bounds.Max.Y - bounds.Min.Y
	currentMPXS := float64(width*height) / 1000000.0

	if currentMPXS <= maxMPXS {
		return imageData, nil
	}

	ratio := maxMPXS / currentMPXS
	newWidth := int(float64(width) * ratio)
	newHeight := int(float64(height) * ratio)

	resized := imgconv.Resize(img, &imgconv.ResizeOption{
		Width:  newWidth,
		Height: newHeight,
	})

	var buf bytes.Buffer

	if err := jpeg.Encode(&buf, resized, nil); err != nil {
		return nil, fmt.Errorf("error encoding JPEG: %v", err)
	}

	b := buf.Bytes()
	return &b, nil
}
