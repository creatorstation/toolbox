package media

import (
	v "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

type MediaURLBody struct {
	MediaURI string `json:"media_uri"`
}

func (b MediaURLBody) Validate() error {
	return v.ValidateStruct(&b,
		v.Field(&b.MediaURI, v.Required, is.URL),
	)
}
