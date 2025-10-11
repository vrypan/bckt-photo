package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Version information (set by GoReleaser)
	version = "dev"
	commit  = "none"
	date    = "unknown"

	// CLI flags
	imageFile  string
	title      string
	configFile string
	postsDir   string
	language   string
	extraTags  []string
)

var rootCmd = &cobra.Command{
	Use:     "bckt-photo",
	Short:   "Create bckt blog posts from images with EXIF data",
	Long:    `bckt-photo reads EXIF data from images and creates bckt-formatted blog posts with front matter.`,
	RunE:    run,
	Version: version,
}

func init() {
	rootCmd.Flags().StringVarP(&imageFile, "image", "i", "", "Path to image file or directory (required)")
	rootCmd.Flags().StringVarP(&title, "title", "t", "", "Post title (or template using @keywords like @dir1, @basename)")
	rootCmd.Flags().StringVarP(&configFile, "config", "c", "bckt-photo.yaml", "Path to config file")
	rootCmd.Flags().StringVarP(&postsDir, "posts", "p", "posts", "Posts directory")
	rootCmd.Flags().StringVarP(&language, "lang", "l", "en", "Post language")
	rootCmd.Flags().StringSliceVarP(&extraTags, "tags", "g", []string{}, "Tags (literal or templates with @keywords, comma-separated or multiple flags)")
	rootCmd.MarkFlagRequired("image")

	// Custom version template
	rootCmd.SetVersionTemplate(fmt.Sprintf("bckt-photo version %s (commit: %s, built: %s)\n", version, commit, date))
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
		return processDirectory(imageFile, config, title, extraTags)
	}

	// Process single file
	return processSinglePhoto(imageFile, "", config, title, extraTags)
}
