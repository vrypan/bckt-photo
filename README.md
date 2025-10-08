# bckt-photo

A command line tool written in Go that creates [bckt](https://github.com/vrypan/bckt) blog posts from image files using EXIF data.

## Features

- Extracts EXIF data from images (date, camera info, lens, etc.)
- Creates bckt-formatted blog posts with proper directory structure
- Automatically generates thumbnails (max 800x800 pixels)
- Configurable EXIF field to tag mapping
- Supports optional post titles
- Uses image date as post date

## Installation

```bash
go build -o bckt-photo
```

## Usage

```bash
bckt-photo -i /path/to/image.jpg [options]
```

### Options

- `-i, --image` (required): Path to the image file
- `-t, --title`: Post title (optional)
- `-c, --config`: Path to config file (default: `bckt-photo.yaml`)
- `-p, --posts`: Posts directory (default: `posts`)
- `-l, --lang`: Post language (default: `en`)

### Examples

Create a post with a title:
```bash
bckt-photo -i photo.jpg -t "Beautiful Sunset"
```

Create a post without a title:
```bash
bckt-photo -i photo.jpg
```

Use a custom config and posts directory:
```bash
bckt-photo -i photo.jpg -c my-config.yaml -p /path/to/posts
```

## Configuration

Copy `bckt-photo.yaml.example` to `bckt-photo.yaml` and customize:

```yaml
# Directory where posts will be created
posts_dir: posts

# Map EXIF fields to tag names
exif_to_tags:
  Make: camera-make
  Model: camera-model
  ISOSpeedRatings: iso
  FNumber: aperture
  ExposureTime: shutter-speed
  FocalLength: focal-length
  LensModel: lens
```

## Output

The tool creates a bckt post structure:

```
posts/
  └── YYYY/
      └── post-slug/
          ├── post-slug.md
          ├── image.jpg
          └── image-thumb.jpg
```

The markdown file includes YAML front matter with:
- `title`: Post title (if provided)
- `date`: Extracted from image EXIF data
- `slug`: Generated from title or timestamp
- `tags`: List of tags generated from EXIF data based on config
- `type`: Set to "photo"
- `attached`: List containing original image and thumbnail filenames
- `image`: Original image filename
- `thumb`: Thumbnail filename
- `language`: Language code
