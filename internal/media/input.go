package media

import (
	v "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

type ConvertMP4ToMP3Body struct {
	MediaURI string `json:"media_uri"`
}

func (b ConvertMP4ToMP3Body) Validate() error {
	return v.ValidateStruct(&b,
		v.Field(&b.MediaURI, v.Required, is.URL),
	)
}
