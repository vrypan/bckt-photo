package main

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ExifToTags map[string][]string `yaml:"exif_to_tags"`
	PostsDir   string              `yaml:"posts_dir"`
	Metadata   MetadataConfig      `yaml:"metadata"`
}

type MetadataConfig struct {
	Title string   `yaml:"title"` // Template string with @keywords
	Tags  []string `yaml:"tags"`  // List of template strings with @keywords
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
