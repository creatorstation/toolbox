package media

// Import resty into your code and refer it as `resty`.
import "github.com/go-resty/resty/v2"

var client = resty.New()

func fetchMedia(mediaURI string) ([]byte, error) {
	resp, err := client.R().Get(mediaURI)
	if err != nil {
		return nil, err
	}
	return resp.Body(), nil
}
