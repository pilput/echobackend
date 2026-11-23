package storage

import (
	"bytes"
	"errors"
	"image"

	"github.com/gen2brain/webp" // also registers the "webp" decoder with image.Decode
	"github.com/kovidgoyal/imaging"

	_ "image/jpeg" // register decoders for image.DecodeConfig
	_ "image/png"
)

const (
	// webpQuality is the lossy WebP quality (0-100): visually close to the
	// source, several times smaller than phone JPEGs.
	webpQuality = 80
	// maxDecodeDimension refuses to decode anything larger (decompression-bomb
	// guard); checked from the header before any pixel is allocated.
	maxDecodeDimension = 10_000
)

// optimizeSlots caps concurrent decode/re-encode work: each image can take a
// few hundred ms of CPU and tens of MB of RAM.
var optimizeSlots = make(chan struct{}, 2)

// OptimizeImage shrinks an uploaded image before it is stored: it decodes it,
// applies the EXIF orientation, downscales so the longest side fits
// maxDimension, and re-encodes it as lossy WebP (which also strips metadata
// such as EXIF GPS). GIFs are returned untouched so animations survive. When
// the image cannot be decoded, or the result would not be smaller, the
// original data, contentType and ext are returned.
//
// contentType and ext are what DetectAllowedImage reported for data.
func OptimizeImage(data []byte, contentType, ext string, maxDimension int) ([]byte, string, string) {
	if contentType == "image/gif" {
		return data, contentType, ext
	}

	optimizeSlots <- struct{}{}
	defer func() { <-optimizeSlots }()

	out, err := optimizeToWebP(data, maxDimension)
	if err != nil {
		log.Warn("image optimize: keeping original", "content_type", contentType, "error", err)
		return data, contentType, ext
	}
	if len(out) >= len(data) {
		return data, contentType, ext
	}
	return out, "image/webp", ".webp"
}

func optimizeToWebP(data []byte, maxDimension int) ([]byte, error) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	if cfg.Width > maxDecodeDimension || cfg.Height > maxDecodeDimension {
		return nil, errImageTooLarge
	}

	// Phone photos are often stored sideways with an EXIF rotation tag; the
	// re-encoded file has no EXIF, so bake the rotation into the pixels.
	img, err := imaging.Decode(bytes.NewReader(data), imaging.AutoOrientation(true))
	if err != nil {
		return nil, err
	}
	// Fit only ever downscales.
	img = imaging.Fit(img, maxDimension, maxDimension, imaging.Lanczos)

	var buf bytes.Buffer
	if err := webp.Encode(&buf, img, webp.Options{Quality: webpQuality}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

var errImageTooLarge = errors.New("image dimensions exceed decode limit")
