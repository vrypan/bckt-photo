package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	exif "github.com/dsoprea/go-exif/v3"
	exifcommon "github.com/dsoprea/go-exif/v3/common"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type Config struct {
	ExifToTags map[string][]string `yaml:"exif_to_tags"`
	PostsDir   string              `yaml:"posts_dir"`
}

type PostFrontMatter struct {
	Title    string                 `yaml:"title,omitempty"`
	Date     time.Time              `yaml:"date"`
	Slug     string                 `yaml:"slug"`
	Tags     []string               `yaml:"tags,omitempty"`
	Type     string                 `yaml:"type"`
	Attached []string               `yaml:"attached"`
	Image    string                 `yaml:"image"`
	Thumb    string                 `yaml:"thumb"`
	Language string                 `yaml:"language,omitempty"`
	Extra    map[string]interface{} `yaml:",inline"`
}

var (
	imageFile  string
	title      string
	configFile string
	postsDir   string
	language   string
)

var rootCmd = &cobra.Command{
	Use:   "bckt-photo",
	Short: "Create bckt blog posts from images with EXIF data",
	Long:  `bckt-photo reads EXIF data from images and creates bckt-formatted blog posts with front matter.`,
	RunE:  run,
}

func init() {
	rootCmd.Flags().StringVarP(&imageFile, "image", "i", "", "Path to image file or directory (required)")
	rootCmd.Flags().StringVarP(&title, "title", "t", "", "Post title (optional, only used for single image)")
	rootCmd.Flags().StringVarP(&configFile, "config", "c", "bckt-photo.yaml", "Path to config file")
	rootCmd.Flags().StringVarP(&postsDir, "posts", "p", "posts", "Posts directory")
	rootCmd.Flags().StringVarP(&language, "lang", "l", "en", "Post language")
	rootCmd.MarkFlagRequired("image")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	// Load config
	config, err := loadConfig(configFile)
	if err != nil {
		fmt.Printf("Warning: Could not load config file: %v\n", err)
		config = &Config{
			ExifToTags: make(map[string][]string),
		}
	}

	// Override posts directory if specified in config
	if config.PostsDir != "" {
		postsDir = config.PostsDir
	}

	// Check if input is a directory
	fileInfo, err := os.Stat(imageFile)
	if err != nil {
		return fmt.Errorf("error accessing path: %w", err)
	}

	if fileInfo.IsDir() {
		// Process directory
		return processDirectory(imageFile, config)
	}

	// Process single file
	return processSinglePhoto(imageFile, "", config, title)
}

func processSinglePhoto(imagePath, relativeDir string, config *Config, photoTitle string) error {
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
	if err := createMarkdownFile(mdFile, photoTitle, postDate, slug, tags, attachedFiles, language, exifFields); err != nil {
		return fmt.Errorf("error creating markdown file: %w", err)
	}

	fmt.Printf("Post created successfully at: %s\n", postDir)
	return nil
}

func processDirectory(baseDir string, config *Config) error {
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
		if err := processSinglePhoto(path, relDir, config, ""); err != nil {
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

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func readExifData(imagePath string) (*exif.Ifd, error) {
	data, err := os.ReadFile(imagePath)
	if err != nil {
		return nil, err
	}

	rawExif, err := exif.SearchAndExtractExif(data)
	if err != nil {
		return nil, err
	}

	im, err := exifcommon.NewIfdMappingWithStandard()
	if err != nil {
		return nil, err
	}

	ti := exif.NewTagIndex()

	_, index, err := exif.Collect(im, ti, rawExif)
	if err != nil {
		return nil, err
	}

	return index.RootIfd, nil
}

func extractDate(ifd *exif.Ifd) time.Time {
	if ifd == nil {
		return time.Now()
	}

	// Try to get DateTime
	results, err := ifd.FindTagWithName("DateTime")
	if err != nil || len(results) == 0 {
		// Try DateTimeOriginal as fallback
		results, err = ifd.FindTagWithName("DateTimeOriginal")
		if err != nil || len(results) == 0 {
			return time.Now()
		}
	}

	ite := results[0]
	valueRaw, err := ite.Value()
	if err != nil {
		return time.Now()
	}

	// Parse EXIF datetime format: "2006:01:02 15:04:05"
	dateStr, ok := valueRaw.(string)
	if !ok {
		return time.Now()
	}

	dt, err := time.Parse("2006:01:02 15:04:05", dateStr)
	if err != nil {
		return time.Now()
	}

	return dt
}

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

func extractTags(ifd *exif.Ifd, exifToTags map[string][]string) []string {
	if ifd == nil || len(exifToTags) == 0 {
		return nil
	}

	// Use a map to deduplicate tags
	tagSet := make(map[string]bool)

	for fieldName, exifFields := range exifToTags {
		// Try each EXIF field in priority order
		for _, exifField := range exifFields {
			value := findExifValue(ifd, exifField)
			if value != "" {
				// Use friendly format for tags if available
				friendly := formatFriendlyValue(fieldName, value)
				if friendly != "" {
					tagSet[friendly] = true
				} else {
					tagSet[value] = true
				}
				break // Found a value, move to next tag
			}
		}
	}

	// Convert set to slice
	var tags []string
	for tag := range tagSet {
		tags = append(tags, tag)
	}

	return tags
}

func extractExifFields(ifd *exif.Ifd, exifToTags map[string][]string) map[string]interface{} {
	if ifd == nil || len(exifToTags) == 0 {
		return nil
	}

	fields := make(map[string]interface{})
	for fieldName, exifFields := range exifToTags {
		// Try each EXIF field in priority order until we find a value
		for _, exifField := range exifFields {
			value := findExifValue(ifd, exifField)
			if value != "" {
				fields[fieldName] = value

				// Add friendly version for specific fields
				friendly := formatFriendlyValue(fieldName, value)
				if friendly != "" && friendly != value {
					fields[fieldName+"-friendly"] = friendly
				}
				break // Found a value, move to next field
			}
		}
	}

	return fields
}

// findExifValue searches for an EXIF tag by name recursively through all IFDs
func findExifValue(ifd *exif.Ifd, tagName string) string {
	if ifd == nil {
		return ""
	}

	var foundValue string
	err := ifd.EnumerateTagsRecursively(func(i *exif.Ifd, ite *exif.IfdTagEntry) error {
		if ite.TagName() == tagName {
			valueRaw, err := ite.Value()
			if err == nil {
				foundValue = formatExifValue(valueRaw)
				if foundValue != "" {
					return fmt.Errorf("found") // Stop enumeration
				}
			}
		}
		return nil
	})

	// Ignore the "found" error we use to stop enumeration
	if err != nil && err.Error() != "found" {
		return ""
	}

	return foundValue
}

func formatExifValue(valueRaw interface{}) string {
	// Handle different types of EXIF values
	switch v := valueRaw.(type) {
	case []uint16:
		// Common for ISO, handle uint16 arrays
		if len(v) > 0 {
			return fmt.Sprintf("%d", v[0])
		}
	case []int:
		// Handle int arrays
		if len(v) > 0 {
			return fmt.Sprintf("%d", v[0])
		}
	case []string:
		// Handle string arrays
		if len(v) > 0 {
			return v[0]
		}
	case string:
		// Simple string value
		return v
	case []exifcommon.Rational:
		// Handle rational numbers (fractions) - common for aperture, exposure, focal length
		if len(v) > 0 {
			rational := v[0]
			if rational.Denominator == 0 {
				return ""
			}
			// Return as fraction
			return fmt.Sprintf("%d/%d", rational.Numerator, rational.Denominator)
		}
	case []exifcommon.SignedRational:
		// Handle signed rational numbers
		if len(v) > 0 {
			rational := v[0]
			if rational.Denominator == 0 {
				return ""
			}
			// Return as fraction
			return fmt.Sprintf("%d/%d", rational.Numerator, rational.Denominator)
		}
	default:
		// For other types, use default formatting
		return fmt.Sprintf("%v", valueRaw)
	}
	return ""
}

// formatFriendlyValue formats values in user-friendly format for specific field types
func formatFriendlyValue(fieldName, value string) string {
	// Parse fraction values (e.g., "8/5" or "21/5")
	var numerator, denominator int
	if _, err := fmt.Sscanf(value, "%d/%d", &numerator, &denominator); err == nil && denominator != 0 {
		decimalValue := float64(numerator) / float64(denominator)

		switch fieldName {
		case "aperture":
			// Format as f-stop: f/1.6
			return fmt.Sprintf("f/%.1f", decimalValue)
		case "focal_length":
			// Format as millimeters: 4.2mm
			return fmt.Sprintf("%.1fmm", decimalValue)
		case "exposure":
			// Format exposure time
			if numerator == 1 {
				// Already in 1/x format, just add 's'
				return fmt.Sprintf("%ss", value)
			}
			// For other fractions, show as is with 's'
			return fmt.Sprintf("%ss", value)
		}
	}

	// For non-fraction values or unrecognized field names, return original
	return value
}

func createPostDirectory(postsDir string, date time.Time, slug string) (string, error) {
	year := fmt.Sprintf("%04d", date.Year())
	yearDir := filepath.Join(postsDir, year)
	postDir := filepath.Join(yearDir, slug)

	if err := os.MkdirAll(postDir, 0755); err != nil {
		return "", err
	}

	return postDir, nil
}

func createPostDirectoryWithPath(postsDir, relativeDir, slug string) (string, error) {
	var postDir string
	if relativeDir != "" {
		postDir = filepath.Join(postsDir, relativeDir, slug)
	} else {
		postDir = filepath.Join(postsDir, slug)
	}

	if err := os.MkdirAll(postDir, 0755); err != nil {
		return "", err
	}

	return postDir, nil
}

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

func createMarkdownFile(path, title string, date time.Time, slug string, tags []string, attachedFiles []string, language string, exifFields map[string]interface{}) error {
	// Extract image and thumbnail names from attachedFiles
	imageName := ""
	thumbnailName := ""
	if len(attachedFiles) > 0 {
		imageName = attachedFiles[0]
	}
	if len(attachedFiles) > 1 {
		thumbnailName = attachedFiles[1]
	}

	frontMatter := PostFrontMatter{
		Title:    title,
		Date:     date,
		Slug:     slug,
		Tags:     tags,
		Type:     "photo",
		Attached: attachedFiles,
		Image:    imageName,
		Thumb:    thumbnailName,
		Language: language,
		Extra:    exifFields,
	}

	yamlData, err := yaml.Marshal(frontMatter)
	if err != nil {
		return err
	}

	content := fmt.Sprintf("---\n%s---\n\n", string(yamlData))

	return os.WriteFile(path, []byte(content), 0644)
}
