package posts

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
)

// Post represents the data for a single post.
type Post struct {
	Title       string
	ContentHTML template.HTML
	Slug        string
}

// Load reads all .html files from the contentDir, parses them into Post objects.
func Load(contentDir string) ([]Post, error) {
	var loadedPosts []Post
	files, err := os.ReadDir(contentDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read content directory %s: %w", contentDir, err)
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".html") {
			filePath := filepath.Join(contentDir, file.Name())
			contentBytes, err := os.ReadFile(filePath)
			if err != nil {
				fmt.Printf("Warning: Failed to read file %s: %v\n", filePath, err)
				continue // Skip this file
			}

			baseName := strings.TrimSuffix(file.Name(), ".html")
			slug := "/" + baseName
			// Derive title from slug (e.g., "/my-post" -> "My Post")
			title := strings.ReplaceAll(strings.TrimPrefix(slug, "/"), "-", " ")
			title = strings.Title(title) //nolint:staticcheck // SA1019: strings.Title is deprecated, but good enough for this example
			if title == "" {
				title = "Untitled Post" // Fallback
			}

			loadedPosts = append(loadedPosts, Post{
				Title:       title,
				ContentHTML: template.HTML(contentBytes),
				Slug:        slug,
			})
		}
	}
	return loadedPosts, nil
}
