package main

import (
	"fmt"
	"os"
	"time"

	exif "github.com/dsoprea/go-exif/v3"
	exifcommon "github.com/dsoprea/go-exif/v3/common"
)

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
					fields[fieldName+"_friendly"] = friendly
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
