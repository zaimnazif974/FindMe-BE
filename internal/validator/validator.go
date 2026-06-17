package validator

import (
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strings"
)

const MaxPhotoSize int64 = 5 << 20

var allowedMIMEs = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
}

func Coordinates(latitude, longitude float64) error {
	if latitude < -90 || latitude > 90 {
		return fmt.Errorf("latitude must be between -90 and 90")
	}
	if longitude < -180 || longitude > 180 {
		return fmt.Errorf("longitude must be between -180 and 180")
	}
	return nil
}

func Photo(file *multipart.FileHeader) error {
	if file.Size <= 0 || file.Size > MaxPhotoSize {
		return fmt.Errorf("photo must be between 1 byte and 5 MB")
	}
	contentType := strings.ToLower(file.Header.Get("Content-Type"))
	if !allowedMIMEs[contentType] {
		return fmt.Errorf("photo type must be jpg, jpeg, png, or webp")
	}
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
		return fmt.Errorf("photo extension must be jpg, jpeg, png, or webp")
	}
	return nil
}

func SafeFilename(name string) string {
	name = filepath.Base(name)
	var b strings.Builder
	for _, r := range name {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '.' || r == '-' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	if b.Len() == 0 {
		return "photo"
	}
	return b.String()
}
