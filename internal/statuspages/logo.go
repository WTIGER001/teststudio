package statuspages

import (
	"bytes"
	"encoding/base64"
	"errors"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"net/url"
	"strings"
)

func validateLogo(value string) error {
	if value == "" {
		return nil
	}
	if len(value) > 256*1024 {
		return errors.New("the logo must be at most 256 KiB after resizing")
	}
	for _, kind := range []string{"png", "jpeg"} {
		prefix := "data:image/" + kind + ";base64,"
		if !strings.HasPrefix(value, prefix) {
			continue
		}
		data, err := base64.StdEncoding.Strict().DecodeString(strings.TrimPrefix(value, prefix))
		if err != nil {
			return errors.New("the uploaded logo is not a valid image")
		}
		config, format, err := image.DecodeConfig(bytes.NewReader(data))
		if err != nil || format != kind || config.Width < 1 || config.Height < 1 || config.Width > 1600 || config.Height > 1600 {
			return errors.New("upload a PNG or JPEG logo no larger than 1600 by 1600 pixels")
		}
		if _, _, err = image.Decode(bytes.NewReader(data)); err != nil {
			return errors.New("the uploaded logo is incomplete or invalid")
		}
		return nil
	}
	address, err := url.Parse(value)
	if err != nil || len(value) > 2048 || address.Scheme != "https" || address.Hostname() == "" || address.User != nil || strings.ContainsAny(value, "\r\n\t ") || address.Opaque != "" {
		return errors.New("use a public HTTPS image URL without embedded credentials, or upload a PNG or JPEG logo")
	}
	return nil
}
