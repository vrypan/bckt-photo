package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func processSinglePhoto(imagePath, relativeDir string, config *Config, photoTitle string, extraTags []string) error {
	// Read EXIF data (optional - may not exist for some images)
	exifData, err := readExifData(imagePath)
	if err != nil {
		fmt.Printf("Warning: Could not read EXIF data from %s: %v\n", filepath.Base(imagePath), err)
		// Continue processing without EXIF data
		exifData = nil
	}

	// Extract date from EXIF (or use current time as fallback)
	postDate := extractDate(exifData)

	// Generate slug from title or use timestamp
	slug := generateSlug(photoTitle, postDate)

	// Extract tags from EXIF based on config (if available)
	tags := extractTags(exifData, config.ExifToTags)

	// Extract EXIF fields for frontmatter
	exifFields := extractExifFields(exifData, config.ExifToTags)

	// Create post directory structure
	postDir, err := createPostDirectoryWithPath(postsDir, relativeDir, slug)
	if err != nil {
		return fmt.Errorf("error creating post directory: %w", err)
	}

	// Copy image to post directory
	imageName := filepath.Base(imagePath)
	destImage := filepath.Join(postDir, imageName)
	if err := copyFile(imagePath, destImage); err != nil {
		return fmt.Errorf("error copying image: %w", err)
	}

	// Create thumbnail
	ext := filepath.Ext(imageName)
	nameWithoutExt := strings.TrimSuffix(imageName, ext)
	thumbnailName := nameWithoutExt + "-thumb" + ext
	thumbnailPath := filepath.Join(postDir, thumbnailName)
	if err := createThumbnail(imagePath, thumbnailPath, 800, 800); err != nil {
		return fmt.Errorf("error creating thumbnail: %w", err)
	}

	// Create markdown file with front matter
	attachedFiles := []string{imageName, thumbnailName}
	mdFile := filepath.Join(postDir, slug+".md")
	if err := createMarkdownFile(mdFile, photoTitle, postDate, slug, tags, attachedFiles, language, exifFields, extraTags); err != nil {
		return fmt.Errorf("error creating markdown file: %w", err)
	}

	fmt.Printf("Post created successfully at: %s\n", postDir)
	return nil
}

func processDirectory(baseDir string, config *Config, extraTags []string) error {
	fmt.Printf("Processing directory: %s\n", baseDir)

	// Walk the directory tree
	err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if file is an image
		if !isImageFile(path) {
			return nil
		}

		// Calculate relative path from base directory
		relPath, err := filepath.Rel(baseDir, path)
		if err != nil {
			return fmt.Errorf("error calculating relative path: %w", err)
		}

		// Get directory part (without filename)
		relDir := filepath.Dir(relPath)
		if relDir == "." {
			relDir = ""
		}

		// Process this image
		fmt.Printf("Processing: %s\n", path)
		if err := processSinglePhoto(path, relDir, config, "", extraTags); err != nil {
			fmt.Printf("Error processing %s: %v\n", path, err)
			// Continue processing other files
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("error walking directory: %w", err)
	}

	fmt.Println("Directory processing complete")
	return nil
}
