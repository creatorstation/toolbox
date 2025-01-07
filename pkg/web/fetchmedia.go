package web

import (
	"fmt"

	"github.com/go-resty/resty/v2"
)

var client = resty.New()

func FetchMedia(mediaURI string) ([]byte, error) {
	resp, err := client.R().Get(mediaURI)
	if err != nil {
		return nil, err
	}

	if resp.IsError() {
		return nil, fmt.Errorf("failed to fetch media: %s, %s", resp.Status(), resp.String())
	}

	return resp.Body(), nil
}
