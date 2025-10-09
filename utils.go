package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

func generateSlug(title string, date time.Time) string {
	if title != "" {
		// Convert title to slug
		slug := strings.ToLower(title)
		slug = strings.ReplaceAll(slug, " ", "-")
		// Remove non-alphanumeric characters except hyphens
		var result []rune
		for _, r := range slug {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
				result = append(result, r)
			}
		}
		return string(result)
	}

	// Use timestamp as slug
	return fmt.Sprintf("photo-%d", date.Unix())
}

func isImageFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	imageExtensions := []string{".jpg", ".jpeg", ".png", ".gif", ".bmp", ".tiff", ".tif", ".webp"}

	for _, imgExt := range imageExtensions {
		if ext == imgExt {
			return true
		}
	}

	return false
}
