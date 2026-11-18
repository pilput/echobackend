package storage

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
)

// FileUploader is the write side of object storage used for image uploads.
type FileUploader interface {
	Save(ctx context.Context, path string, file io.Reader, contentType string) error
}

// DetectAllowedImage sniffs data and reports its content type and file
// extension when it is a JPEG, PNG, WebP or GIF image.
func DetectAllowedImage(data []byte) (contentType string, ext string, ok bool) {
	contentType = http.DetectContentType(data)
	switch contentType {
	case "image/jpeg":
		return contentType, ".jpg", true
	case "image/png":
		return contentType, ".png", true
	case "image/webp":
		return contentType, ".webp", true
	case "image/gif":
		return contentType, ".gif", true
	default:
		return "", "", false
	}
}

// RandomImageObjectKey generates a random object key under the given storage
// prefix (e.g. "posts/images" or "users/avatars").
func RandomImageObjectKey(prefix, ext string) (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%s%s", prefix, hex.EncodeToString(b), ext), nil
}
