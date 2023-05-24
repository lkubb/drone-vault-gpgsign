package plugin

import (
	"encoding/base64"
	"fmt"
	"os"

	"github.com/bmatcuk/doublestar/v4"
)

func findFiles(include, exclude []string) ([]string, error) {
	var excludes, files []string

	// glob excludes
	for _, pattern := range exclude {
		excludeMatches, err := doublestar.FilepathGlob(pattern)
		if err != nil {
			return files, fmt.Errorf("Failed to match excludes: %w")
		}
		for _, file := range excludeMatches {
			if _, err := os.Stat(file); err == nil {
				excludes = append(excludes, file)
			}
		}
	}

	// glob files
	for _, pattern := range include {
		includeMatches, err := doublestar.FilepathGlob(pattern)
		if err != nil {
			return files, fmt.Errorf("Failed to match files: %w")
		}
		for _, file := range includeMatches {
			excluded := false
			for _, exclude := range excludes {
				if file == exclude {
					excluded = true
					break
				}
			}
			if excluded {
				continue
			}
			if _, err := os.Stat(file); err == nil {
				files = append(files, file)
			}
		}
	}
	return files, nil
}

func loadFileBase64(file string) (string, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}
