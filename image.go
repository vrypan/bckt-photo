package main

import (
	"io"
	"os"

	"github.com/disintegration/imaging"
)

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return err
	}

	return destFile.Sync()
}

func createThumbnail(src, dst string, maxWidth, maxHeight int) error {
	// Open the image
	img, err := imaging.Open(src)
	if err != nil {
		return err
	}

	// Get current dimensions
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Only resize if image is larger than max dimensions
	if width > maxWidth || height > maxHeight {
		img = imaging.Fit(img, maxWidth, maxHeight, imaging.Lanczos)
	}

	// Save the thumbnail
	return imaging.Save(img, dst)
}
