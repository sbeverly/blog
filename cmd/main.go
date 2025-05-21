package main

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Post represents the data for a single post page.
type Post struct {
	Title       string
	ContentHTML template.HTML
	Slug        string
}

// loadPosts reads all .html files from the contentDir, parses them into Post objects.
func loadPosts(contentDir string) ([]Post, error) {
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

func main() {
	fmt.Println("BUILDING SITE")

	contentDir := "content"
	outputDir := "public"

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		panic(fmt.Sprintf("Failed to create output directory %s: %v", outputDir, err))
	}

	var allPosts []Post

	postsPath := filepath.Join(contentDir, "posts")
	if _, err := os.Stat(postsPath); !os.IsNotExist(err) {
		postsContent, err := loadPosts(postsPath)
		if err != nil {
			panic(fmt.Sprintf("Failed to load posts from %s: %v", postsPath, err))
		}
		allPosts = append(allPosts, postsContent...)
	} else {
		fmt.Printf("Posts directory not found, skipping: %s\n", postsPath)
	}

	pagesPath := filepath.Join(contentDir, "pages")
	if _, err := os.Stat(pagesPath); !os.IsNotExist(err) {
		pagesContent, err := loadPosts(pagesPath)
		if err != nil {
			panic(fmt.Sprintf("Failed to load pages from %s: %v", pagesPath, err))
		}
		allPosts = append(allPosts, pagesContent...)
	} else {
		fmt.Printf("Pages directory not found, skipping: %s\n", pagesPath)
	}

	if len(allPosts) == 0 {
		fmt.Println("No content found in 'content/posts' or 'content/pages' directories.")
	}

	tmplPath := "templates"
	baseTmpl := filepath.Join(tmplPath, "base.html")
	headerTmpl := filepath.Join(tmplPath, "header.html")
	postPageTmpl := filepath.Join(tmplPath, "post.html")

	templates, err := template.ParseFiles(baseTmpl, headerTmpl, postPageTmpl)
	if err != nil {
		panic(fmt.Sprintf("Failed to parse templates: %v", err))
	}

	for _, postData := range allPosts {
		fileName := strings.TrimPrefix(postData.Slug, "/") + ".html"
		if postData.Slug == "/" {
			fileName = "index.html"
		}
		filePath := filepath.Join(outputDir, fileName)

		file, err := os.Create(filePath)
		if err != nil {
			fmt.Printf("Warning: Failed to create file %s: %v\n", filePath, err)
			continue
		}

		// defer file.Close() // Closed explicitly below to catch errors sooner in loop

		executeErr := templates.ExecuteTemplate(file, "base", postData)
		closeErr := file.Close()

		if executeErr != nil {
			fmt.Printf("Warning: Failed to execute template for %s: %v\n", filePath, executeErr)
		} else if closeErr != nil {
			fmt.Printf("Warning: Failed to close file %s: %v\n", filePath, closeErr)
		} else {
			fmt.Printf("Successfully generated %s\n", filePath)
		}
	}

	fmt.Println("SITE BUILD COMPLETE")

	publicFs := http.FileServer(http.Dir(outputDir))

	staticDir := "static"
	staticFs := http.FileServer(http.Dir(staticDir))
	http.Handle("/static/", http.StripPrefix("/static/", staticFs))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		diskFilePath := filepath.Join(outputDir, r.URL.Path)

		_, err := os.Stat(diskFilePath)
		if os.IsNotExist(err) {

			r.URL.Path = r.URL.Path + ".html"
		}

		publicFs.ServeHTTP(w, r)
	})

	port := "8080"
	fmt.Printf("Starting server on http://localhost:%s\n", port)
	serverErr := http.ListenAndServe(":"+port, nil)
	if serverErr != nil {
		panic(fmt.Sprintf("Failed to start server: %v", serverErr))
	}
}
