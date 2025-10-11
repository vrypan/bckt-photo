# bckt-photo

A command line tool written in Go that creates [bckt](https://github.com/vrypan/bckt) blog posts from image files using EXIF data.

## Features

- Extracts EXIF data from images (date, camera info, lens, etc.)
- **Supports both single files and directories** - recursively processes entire photo directories
- **Supports multiple image formats**: JPEG, PNG (with eXIf chunk), and more
- Creates bckt-formatted blog posts with proper directory structure
- **Preserves directory structure** when processing folders
- Automatically generates thumbnails (max 800x800 pixels)
- **Configurable EXIF field to tag mapping with priority fallbacks**
- **EXIF fields become individual frontmatter fields** for templates
- **Add custom tags to all posts** - perfect for batch processing (e.g., "summer2025", "vacation")
- **Path-based metadata templating** - generate titles and tags from directory structure and filenames
- Supports optional post titles
- Uses image date from EXIF data, falls back to current time
- **Modular codebase** - organized into focused modules for maintainability

## Installation

### Homebrew (macOS/Linux)

```bash
brew install vrypan/bckt-photo/bckt-photo
```

### Pre-built Binaries

Download the latest release for your platform from the [releases page](https://github.com/vrypan/bckt-photo/releases).

Extract and move the binary to your PATH:

```bash
# macOS/Linux
tar -xzf bckt-photo_*.tar.gz
sudo mv bckt-photo /usr/local/bin/
```

### Build from Source

```bash
git clone https://github.com/vrypan/bckt-photo.git
cd bckt-photo
go build -o bckt-photo
```

## Usage

### Single Image

```bash
bckt-photo -i /path/to/image.jpg [options]
```

### Directory (Batch Processing)

```bash
bckt-photo -i /path/to/photos [options]
```

When processing a directory, the tool will:
- Recursively find all image files (`.jpg`, `.jpeg`, `.png`, `.gif`, `.bmp`, `.tiff`, `.tif`, `.webp`)
- Preserve the directory structure under the posts directory
- Process each image individually
- Continue on errors (shows warning but keeps processing)

### Options

- `-i, --image` (required): Path to image file or directory
- `-t, --title`: Post title (literal or template using @keywords like @dir1, @basename)
- `-c, --config`: Path to config file (default: `bckt-photo.yaml`)
- `-p, --posts`: Posts directory (default: `posts`)
- `-l, --lang`: Post language (default: `en`)
- `-g, --tags`: Tags (literal or templates with @keywords, comma-separated or multiple flags)

### Examples

Create a post from a single image with a title:
```bash
bckt-photo -i photo.jpg -t "Beautiful Sunset"
```

Process an entire directory of photos:
```bash
bckt-photo -i ~/Photos/Vacation2025
```

Use a custom config and posts directory:
```bash
bckt-photo -i photo.jpg -c my-config.yaml -p /path/to/posts
```

Add custom tags to all processed photos:
```bash
# Single tag
bckt-photo -i ~/Photos/Summer --tags summer2025

# Multiple tags (comma-separated)
bckt-photo -i ~/Photos/Summer --tags summer2025,vacation,beach

# Multiple tags (using flag multiple times)
bckt-photo -i ~/Photos/Summer -g summer2025 -g vacation
```

Use path-based templates for titles and tags:
```bash
# Generate title from directory and filename (template auto-detected with @)
bckt-photo -i ~/Photos/2025/vacation/beach.jpg -t "@dir2 @dir1 - @basename"
# Result: "2025 vacation - beach"

# Generate tags from directory structure
bckt-photo -i ~/Photos/2025/vacation/ -g "@dir2" -g "@dir1"
# Each photo gets tags: "2025", "vacation"

# Combine directory parts in tags
bckt-photo -i ~/Photos/2025/vacation/ -g "@dir2/@dir1"
# Each photo gets tag: "2025/vacation"

# Mix literal and template tags
bckt-photo -i ~/Photos/2025/vacation/ -g summer -g "@dir1"
# Each photo gets tags: "summer", "vacation"
```

## Configuration

Copy `bckt-photo.yaml.example` to `bckt-photo.yaml` and customize:

```yaml
# Directory where posts will be created
posts_dir: posts

# Metadata templates - use @keywords to generate titles and tags from path
# Keywords: @dir1 (closest directory to file), @dir2, @dir3, ..., @filename, @basename, @ext
metadata:
  # Title template - generates post title from path components
  title: "@dir1 - @basename"

  # Tag templates - generates tags from path components (list)
  tags:
    - "@dir1"        # Tag from closest directory
    - "@dir2/@dir1"  # Tag from parent/child directories

# Map EXIF fields to frontmatter field names
# Format: fieldname: [list of EXIF tags to try in priority order]
# The first EXIF tag that exists and has a value will be used
exif_to_tags:
  # ISO speed - tries ISOSpeedRatings first, then PhotographicSensitivity, then ISO
  iso:
    - ISOSpeedRatings
    - PhotographicSensitivity
    - ISO

  # Aperture (f-number)
  aperture:
    - FNumber
    - ApertureValue

  # Shutter speed / Exposure time
  exposure:
    - ExposureTime
    - ShutterSpeedValue

  # Focal length in millimeters
  focal_length:
    - FocalLength
    - FocalLengthIn35mmFilm

  # Lens model - different manufacturers use different field names
  lens:
    - LensModel
    - Lens
    - LensInfo

  # Camera model
  camera:
    - Model
    - CameraModelName

  # Camera make/manufacturer
  make:
    - Make
```

### Priority Fallbacks

The config uses arrays to define priority order. For example, with the `iso` field above:
1. First tries to read `ISOSpeedRatings`
2. If not found or empty, tries `PhotographicSensitivity`
3. If not found or empty, tries `ISO`
4. Uses the first successful value

This ensures compatibility across different camera manufacturers and models.

### Special Field Names

**Important:** The following field names receive special formatting treatment and should not be renamed:

- **`aperture`**: Generates user-friendly format (e.g., `f/1.6`) in tags and creates an `aperture_friendly` field
- **`exposure`**: Generates user-friendly format (e.g., `1/63s`) in tags and creates an `exposure_friendly` field
- **`focal_length`**: Generates user-friendly format (e.g., `4.2mm`) in tags and creates a `focal_length_friendly` field

These fields store raw fractional values (e.g., `8/5`, `1/63`, `21/5`) in the main frontmatter fields, while the `_friendly` variants contain human-readable formats. Tags automatically use the friendly versions.

If you rename these fields in your config, you will lose the automatic friendly formatting, but the raw values will still be extracted correctly.

### Path-Based Metadata Templating

You can generate titles and tags dynamically from the file path and filename using template keywords. This is especially useful when batch processing photos organized in directories.

**Available Keywords:**
- `@dir1` - Closest directory to the file (parent directory)
- `@dir2` - Second-to-last directory
- `@dir3` - Third-to-last directory
- (and so on...)
- `@filename` - Full filename with extension (e.g., `beach.jpg`)
- `@basename` - Filename without extension (e.g., `beach`)
- `@ext` - File extension without dot (e.g., `jpg`)

**Note:** Directory numbering is reversed so `@dir1` is always the directory closest to the file, making templates more predictable.

**Example:**

For a file at `/photos/2025/vacation/beach/sunset.jpg`:
- `@dir1` = `beach`
- `@dir2` = `vacation`
- `@dir3` = `2025`
- `@filename` = `sunset.jpg`
- `@basename` = `sunset`
- `@ext` = `jpg`

**How it Works:**

The tool automatically detects if a title or tag contains the `@` character and treats it as a template. Otherwise, it's used as a literal value.

**Priority Order:**

Title is determined in this order (first match wins):
1. `-t, --title` flag (literal or template)
2. `metadata.title` from config file (always a template)
3. No title (default)

Tags are cumulative and applied in this order:
1. EXIF-based tags (from `exif_to_tags` config)
2. `-g, --tags` flags (literal or templates, applied to all files)
3. `metadata.tags` from config file (templates, applied to all files)

**Example Use Cases:**

```bash
# Use directory name as part of title
bckt-photo -i ~/Photos/2025/vacation/ -t "@dir2 - @basename"
# Results in titles like: "2025 - sunset", "2025 - beach"

# Tag by year and event
bckt-photo -i ~/Photos/2025/vacation/ -g "@dir2" -g "@dir1"
# Each photo gets tags: ["2025", "vacation"]

# Create hierarchical tags
bckt-photo -i ~/Photos/2025/summer/italy/ -g "@dir2/@dir1"
# Each photo gets tag: "summer/italy"

# Mix literal tags with template tags
bckt-photo -i ~/Photos/2025/vacation/ -g travel -g "@dir2" -g "@dir1"
# Each photo gets tags: ["travel", "2025", "vacation"]
```

## Output

### Single Image Output Structure

When processing a single image:

```
posts/
  └── photo-slug/
      ├── photo-slug.md
      ├── image.jpg
      └── image-thumb.jpg
```

### Directory Output Structure

When processing a directory, the original structure is preserved:

**Input:**
```
photos/
  └── 2025/
      ├── vacation/
      │   ├── beach.jpg
      │   └── sunset.jpg
      └── family/
          └── portrait.jpg
```

**Output:**
```
posts/
  └── 2025/
      ├── vacation/
      │   ├── photo-1728311400/
      │   │   ├── photo-1728311400.md
      │   │   ├── beach.jpg
      │   │   └── beach-thumb.jpg
      │   └── photo-1728311450/
      │       ├── photo-1728311450.md
      │       ├── sunset.jpg
      │       └── sunset-thumb.jpg
      └── family/
          └── photo-1728311500/
              ├── photo-1728311500.md
              ├── portrait.jpg
              └── portrait-thumb.jpg
```

## Frontmatter Format

The markdown file includes YAML front matter with:

**Standard Fields:**
- `title`: Post title (if provided)
- `date`: Extracted from image EXIF data (or current time)
- `slug`: Generated from title or timestamp
- `tags`: Array of values extracted from EXIF fields plus any custom tags (--tags flag)
- `type`: Set to "photo"
- `attached`: List containing original image and thumbnail filenames
- `image`: Original image filename
- `thumb`: Thumbnail filename
- `language`: Language code

**Dynamic EXIF Fields:**

Each EXIF field mapped in the config becomes its own frontmatter field. For example, with the config above:
- `iso`: ISO value (e.g., `400`)
- `aperture`: F-number (e.g., `5.6`)
- `exposure`: Shutter speed (e.g., `1/500`)
- `focal_length`: Focal length in mm (e.g., `200`)
- `lens`: Lens model name
- `camera`: Camera model name
- `make`: Camera manufacturer

### Example Output

Without custom tags:
```yaml
---
date: 2025-10-07 14:30:00
slug: photo-1728311400
tags:
  - ISO 400
  - f/5.6
  - 1/500s
  - 200.0mm
  - RF 100-500mm F4.5-7.1 L IS USM
  - Canon EOS R5
  - Canon
type: photo
attached:
  - image.jpg
  - image-thumb.jpg
image: image.jpg
thumb: image-thumb.jpg
language: en
iso: "400"
aperture: 28/5
aperture_friendly: f/5.6
exposure: 1/500
exposure_friendly: 1/500s
focal_length: 200/1
focal_length_friendly: 200.0mm
lens: RF 100-500mm F4.5-7.1 L IS USM
camera: Canon EOS R5
make: Canon
---
```

With custom tags (e.g., `--tags summer2025,vacation`):
```yaml
---
date: 2025-10-07 14:30:00
slug: photo-1728311400
tags:
  - ISO 400
  - f/5.6
  - 1/500s
  - 200.0mm
  - RF 100-500mm F4.5-7.1 L IS USM
  - Canon EOS R5
  - Canon
  - summer2025
  - vacation
type: photo
# ... rest of the fields
---
```

## Image Format Support

The tool uses the [dsoprea/go-exif](https://github.com/dsoprea/go-exif) library, which supports:
- **JPEG** files (standard EXIF)
- **PNG** files with eXIf chunks (including screenshots from some tools)
- Other formats supported by the library

If an image doesn't have EXIF data, the tool will show a warning but continue processing, using the current timestamp for the date.

## Dependencies

- [github.com/disintegration/imaging](https://github.com/disintegration/imaging) - Image processing and thumbnail generation
- [github.com/dsoprea/go-exif/v3](https://github.com/dsoprea/go-exif) - EXIF data extraction with multi-format support
- [github.com/spf13/cobra](https://github.com/spf13/cobra) - CLI framework
- [gopkg.in/yaml.v3](https://gopkg.in/yaml.v3) - YAML parsing

## License

MIT License
