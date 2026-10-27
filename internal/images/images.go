// Package images turns uploads into avatars and space icons: square PNGs of at most 256 px. Re-encoding keeps
// them small, strips metadata such as GPS positions and guarantees that only real images are served.
package images

import (
	"bytes"
	"errors"
	"image"
	"image/png"

	// Decoders for image.Decode
	_ "image/gif"
	_ "image/jpeg"

	"golang.org/x/image/draw"
)

const (
	MaxUploadBytes = 5 << 20
	// GIF (maybe animated) and WebP cannot be re-encoded without losing quality, so they are kept up to this size
	maxOriginalBytes = 1 << 20
	Size             = 256
	// A tiny PNG can claim gigapixel dimensions: refuse before decoding
	maxPixels = 50_000_000
)

var (
	ErrUnsupported = errors.New("Only PNG, JPEG, GIF and WebP images are supported")
	ErrTooLarge    = errors.New("The image is too large")
	ErrUnreadable  = errors.New("The image could not be read")
)

// Normalize returns the content type and bytes to store.
func Normalize(data []byte) (string, []byte, error) {
	format := detect(data)
	switch {
	case format == "":
		return "", nil, ErrUnsupported
	case (format == "gif" || format == "webp") && len(data) <= maxOriginalBytes:
		return "image/" + format, data, nil
	case format == "webp":
		return "", nil, errors.New("WebP images must be smaller than 1 MB")
	}

	config, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return "", nil, ErrUnreadable
	}
	if int64(config.Width)*int64(config.Height) > maxPixels {
		return "", nil, ErrTooLarge
	}
	source, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return "", nil, ErrUnreadable
	}
	var out bytes.Buffer
	if err := png.Encode(&out, squared(source)); err != nil {
		return "", nil, err
	}
	return "image/png", out.Bytes(), nil
}

// squared center-crops to a square and scales down to at most Size pixels (never up).
func squared(source image.Image) image.Image {
	bounds := source.Bounds()
	side := min(bounds.Dx(), bounds.Dy())
	crop := image.Rect(0, 0, side, side).Add(image.Pt(
		bounds.Min.X+(bounds.Dx()-side)/2,
		bounds.Min.Y+(bounds.Dy()-side)/2,
	))
	target := min(side, Size)
	scaled := image.NewNRGBA(image.Rect(0, 0, target, target))
	draw.CatmullRom.Scale(scaled, scaled.Bounds(), source, crop, draw.Src, nil)
	return scaled
}

// detect looks at the content: declared content types and file names are not trustworthy.
func detect(data []byte) string {
	switch {
	case bytes.HasPrefix(data, []byte("\x89PNG\r\n\x1a\n")):
		return "png"
	case bytes.HasPrefix(data, []byte{0xff, 0xd8, 0xff}):
		return "jpeg"
	case bytes.HasPrefix(data, []byte("GIF87a")), bytes.HasPrefix(data, []byte("GIF89a")):
		return "gif"
	case len(data) >= 12 && bytes.Equal(data[:4], []byte("RIFF")) && bytes.Equal(data[8:12], []byte("WEBP")):
		return "webp"
	}
	return ""
}
