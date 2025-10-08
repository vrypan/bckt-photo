package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	"github.com/rwcarlsen/goexif/exif"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type Config struct {
	ExifToTags map[string]string `yaml:"exif_to_tags"`
	PostsDir   string            `yaml:"posts_dir"`
}

type PostFrontMatter struct {
	Title    string    `yaml:"title,omitempty"`
	Date     time.Time `yaml:"date"`
	Slug     string    `yaml:"slug"`
	Tags     []string  `yaml:"tags,omitempty"`
	Type     string    `yaml:"type"`
	Attached []string  `yaml:"attached"`
	Image    string    `yaml:"image"`
	Thumb    string    `yaml:"thumb"`
	Language string    `yaml:"language,omitempty"`
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
	rootCmd.Flags().StringVarP(&imageFile, "image", "i", "", "Path to image file (required)")
	rootCmd.Flags().StringVarP(&title, "title", "t", "", "Post title (optional)")
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
			ExifToTags: make(map[string]string),
		}
	}

	// Override posts directory if specified in config
	if config.PostsDir != "" {
		postsDir = config.PostsDir
	}

	// Read EXIF data
	exifData, err := readExifData(imageFile)
	if err != nil {
		return fmt.Errorf("error reading EXIF data: %w", err)
	}

	// Extract date from EXIF
	postDate := extractDate(exifData)

	// Generate slug from title or use timestamp
	slug := generateSlug(title, postDate)

	// Extract tags from EXIF based on config
	tags := extractTags(exifData, config.ExifToTags)

	// Create post directory structure
	postDir, err := createPostDirectory(postsDir, postDate, slug)
	if err != nil {
		return fmt.Errorf("error creating post directory: %w", err)
	}

	// Copy image to post directory
	imageName := filepath.Base(imageFile)
	destImage := filepath.Join(postDir, imageName)
	if err := copyFile(imageFile, destImage); err != nil {
		return fmt.Errorf("error copying image: %w", err)
	}

	// Create thumbnail
	ext := filepath.Ext(imageName)
	nameWithoutExt := strings.TrimSuffix(imageName, ext)
	thumbnailName := nameWithoutExt + "-thumb" + ext
	thumbnailPath := filepath.Join(postDir, thumbnailName)
	if err := createThumbnail(imageFile, thumbnailPath, 800, 800); err != nil {
		return fmt.Errorf("error creating thumbnail: %w", err)
	}

	// Create markdown file with front matter
	attachedFiles := []string{imageName, thumbnailName}
	mdFile := filepath.Join(postDir, slug+".md")
	if err := createMarkdownFile(mdFile, title, postDate, slug, tags, attachedFiles, language); err != nil {
		return fmt.Errorf("error creating markdown file: %w", err)
	}

	fmt.Printf("Post created successfully at: %s\n", postDir)
	fmt.Printf("Markdown file: %s\n", mdFile)
	return nil
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

func readExifData(imagePath string) (*exif.Exif, error) {
	f, err := os.Open(imagePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return exif.Decode(f)
}

func extractDate(x *exif.Exif) time.Time {
	if x == nil {
		return time.Now()
	}

	// Try to get DateTime
	dt, err := x.DateTime()
	if err == nil {
		return dt
	}

	// Fallback to current time
	return time.Now()
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

func extractTags(x *exif.Exif, exifToTags map[string]string) []string {
	if x == nil || len(exifToTags) == 0 {
		return nil
	}

	var tags []string
	for exifField, tagName := range exifToTags {
		tag, err := x.Get(exif.FieldName(exifField))
		if err == nil && tag != nil {
			value := tag.String()
			if value != "" {
				tags = append(tags, fmt.Sprintf("%s:%s", tagName, value))
			}
		}
	}

	return tags
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

func createMarkdownFile(path, title string, date time.Time, slug string, tags []string, attachedFiles []string, language string) error {
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
	}

	yamlData, err := yaml.Marshal(frontMatter)
	if err != nil {
		return err
	}

	content := fmt.Sprintf("---\n%s---\n\n", string(yamlData))

	return os.WriteFile(path, []byte(content), 0644)
}
