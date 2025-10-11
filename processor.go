package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func processSinglePhoto(imagePath, relativeDir string, config *Config, photoTitle string, extraTags []string) error {
	// Extract path components for template expansion
	pathComponents := extractPathComponents(imagePath, postsDir)

	// Determine final title - check CLI flag first, then config
	finalTitle := ""
	if photoTitle != "" {
		// CLI title takes priority - expand if it contains @keywords
		if strings.Contains(photoTitle, "@") {
			finalTitle = expandTemplate(photoTitle, pathComponents)
		} else {
			finalTitle = photoTitle
		}
	} else if config.Metadata.Title != "" {
		// Fall back to config title template
		finalTitle = expandTemplate(config.Metadata.Title, pathComponents)
	}

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
	slug := generateSlug(finalTitle, postDate)

	// Extract tags from EXIF based on config (if available)
	tags := extractTags(exifData, config.ExifToTags)

	// Apply tags from CLI - expand templates or use as literals
	for _, tag := range extraTags {
		if strings.Contains(tag, "@") {
			// It's a template, expand it
			expandedTag := expandTemplate(tag, pathComponents)
			if expandedTag != "" {
				tags = append(tags, expandedTag)
			}
		} else {
			// Literal tag
			tags = append(tags, tag)
		}
	}

	// Apply tag templates from config
	for _, tagTemplate := range config.Metadata.Tags {
		expandedTag := expandTemplate(tagTemplate, pathComponents)
		if expandedTag != "" {
			tags = append(tags, expandedTag)
		}
	}

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
	if err := createMarkdownFile(mdFile, finalTitle, postDate, slug, tags, attachedFiles, language, exifFields, nil); err != nil {
		return fmt.Errorf("error creating markdown file: %w", err)
	}

	fmt.Printf("Post created successfully at: %s\n", postDir)
	return nil
}

func processDirectory(baseDir string, config *Config, title string, extraTags []string) error {
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
		if err := processSinglePhoto(path, relDir, config, title, extraTags); err != nil {
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

// extractPathComponents extracts components from a file path for template expansion
// Returns a map with keys: dir1 (closest to file), dir2, dir3, ..., filename, basename, ext
func extractPathComponents(filePath, baseDir string) map[string]string {
	components := make(map[string]string)

	// Get the filename
	filename := filepath.Base(filePath)
	components["filename"] = filename

	// Get extension and basename
	ext := filepath.Ext(filename)
	if ext != "" {
		components["ext"] = strings.TrimPrefix(ext, ".")
		components["basename"] = strings.TrimSuffix(filename, ext)
	} else {
		components["ext"] = ""
		components["basename"] = filename
	}

	// Get directory path relative to base
	dir := filepath.Dir(filePath)
	relDir, err := filepath.Rel(baseDir, dir)
	if err != nil || relDir == "." || relDir == "" {
		// No directory components
		return components
	}

	// Split directory into parts
	parts := strings.Split(filepath.ToSlash(relDir), "/")

	// Reverse the parts so dir1 is closest to the file
	for i := len(parts) - 1; i >= 0; i-- {
		dirNum := len(parts) - i
		key := fmt.Sprintf("dir%d", dirNum)
		components[key] = parts[i]
	}

	return components
}

// expandTemplate expands a template string by replacing @keywords with actual values
// Supports: @dir1, @dir2, @dir3, ..., @filename, @basename, @ext
func expandTemplate(template string, components map[string]string) string {
	result := template

	// Replace @keywords with their values
	for key, value := range components {
		placeholder := "@" + key
		result = strings.ReplaceAll(result, placeholder, value)
	}

	return result
}
