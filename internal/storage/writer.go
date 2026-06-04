package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func CreateDirIfNotExist(dirPath string) error {
	return os.MkdirAll(dirPath, 0o755)
}

func AppendContentToNote(notePath, newContent, heading, delimiter string, reversedOrder bool) error {
	if err := CreateDirIfNotExist(filepath.Dir(notePath)); err != nil {
		return fmt.Errorf("create directory for note: %w", err)
	}

	existing, err := os.ReadFile(notePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read note: %w", err)
	}
	body := string(existing)

	// New file: write content directly (with heading prefix if specified).
	if body == "" {
		if heading != "" {
			newContent = heading + "\n" + newContent
		}
		return os.WriteFile(notePath, []byte(newContent), 0o644)
	}

	// Existing file with heading specified and found in content.
	if heading != "" {
		if idx := strings.Index(body, heading); idx >= 0 {
			afterHeading := idx + len(heading)
			before := body[:afterHeading]
			after := strings.TrimLeft(body[afterHeading:], "\n")

			var result string
			if reversedOrder {
				if after == "" {
					result = before + "\n" + newContent
				} else {
					result = before + "\n" + newContent + delimiter + after
				}
			} else {
				if after == "" {
					result = before + "\n" + newContent
				} else {
					result = before + "\n" + after + delimiter + newContent
				}
			}
			return os.WriteFile(notePath, []byte(result), 0o644)
		}
		// Heading not found: prepend heading to new content, then fall through.
		newContent = heading + "\n" + newContent
	}

	// No heading, or heading not found in existing content.
	var result string
	if reversedOrder {
		result = newContent + delimiter + body
	} else {
		result = body + delimiter + newContent
	}
	return os.WriteFile(notePath, []byte(result), 0o644)
}

var fileNameInvalidChars = regexp.MustCompile(`[\\/:*?"<>|\n\r]`)

func SanitizeFileName(name string) string {
	return fileNameInvalidChars.ReplaceAllString(name, "")
}

var filePathInvalidChars = regexp.MustCompile(`[\\:*?"<>|\n\r]`)

func SanitizeFilePath(path string) string {
	return filePathInvalidChars.ReplaceAllString(path, "")
}

func GetUniqueFilePath(basePath string, date time.Time, extension string, existingPaths map[string]bool) string {
	if !fileExists(basePath) && !existingPaths[basePath] {
		return basePath
	}

	dir := filepath.Dir(basePath)
	name := strings.TrimSuffix(filepath.Base(basePath), "."+extension)
	suffix := date.Format("20060102150405.000")
	suffix = strings.ReplaceAll(suffix, ".", "")

	candidate := filepath.Join(dir, fmt.Sprintf("%s - %s.%s", name, suffix, extension))
	for fileExists(candidate) || existingPaths[candidate] {
		date = date.Add(time.Millisecond)
		suffix = date.Format("20060102150405.000")
		suffix = strings.ReplaceAll(suffix, ".", "")
		candidate = filepath.Join(dir, fmt.Sprintf("%s - %s.%s", name, suffix, extension))
	}
	return candidate
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
