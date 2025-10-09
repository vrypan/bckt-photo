package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

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

func createMarkdownFile(path, title string, date time.Time, slug string, tags []string, attachedFiles []string, language string, exifFields map[string]interface{}, extraTags []string) error {
	// Extract image and thumbnail names from attachedFiles
	imageName := ""
	thumbnailName := ""
	if len(attachedFiles) > 0 {
		imageName = attachedFiles[0]
	}
	if len(attachedFiles) > 1 {
		thumbnailName = attachedFiles[1]
	}

	// Merge extra tags with EXIF-extracted tags
	allTags := make([]string, 0, len(tags)+len(extraTags))
	allTags = append(allTags, tags...)
	allTags = append(allTags, extraTags...)

	frontMatter := PostFrontMatter{
		Title:    title,
		Date:     date,
		Slug:     slug,
		Tags:     allTags,
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
