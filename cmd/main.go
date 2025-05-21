package main

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/sbeverly/blog/cmd/pages"
	"github.com/sbeverly/blog/cmd/posts"
)

// RenderItem represents common data structure for rendering any content (post or page).
type RenderItem struct {
	Title       string
	ContentHTML template.HTML
	Slug        string
}

func main() {
	fmt.Println("BUILDING SITE")

	contentDir := "content"
	outputDir := "public"

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		panic(fmt.Sprintf("Failed to create output directory %s: %v", outputDir, err))
	}

	var allRenderItems []RenderItem

	// Load Posts
	postsContentPath := filepath.Join(contentDir, "posts")
	if _, err := os.Stat(postsContentPath); !os.IsNotExist(err) {
		loadedPosts, err := posts.Load(postsContentPath)
		if err != nil {
			panic(fmt.Sprintf("Failed to load posts from %s: %v", postsContentPath, err))
		}
		for _, p := range loadedPosts {
			allRenderItems = append(allRenderItems, RenderItem{
				Title:       p.Title,
				ContentHTML: p.ContentHTML,
				Slug:        p.Slug,
			})
		}
	} else {
		fmt.Printf("Posts directory not found, skipping: %s\n", postsContentPath)
	}

	// Load Pages
	pagesContentPath := filepath.Join(contentDir, "pages")
	if _, err := os.Stat(pagesContentPath); !os.IsNotExist(err) {
		loadedPages, err := pages.Load(pagesContentPath)
		if err != nil {
			panic(fmt.Sprintf("Failed to load pages from %s: %v", pagesContentPath, err))
		}
		for _, p := range loadedPages {
			allRenderItems = append(allRenderItems, RenderItem{
				Title:       p.Title,
				ContentHTML: p.ContentHTML,
				Slug:        p.Slug,
			})
		}
	} else {
		fmt.Printf("Pages directory not found, skipping: %s\n", pagesContentPath)
	}

	if len(allRenderItems) == 0 {
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

	for _, itemData := range allRenderItems {
		fileName := strings.TrimPrefix(itemData.Slug, "/") + ".html"
		if itemData.Slug == "/" { // This case might need review based on how slugs like "/" are generated
			fileName = "index.html"
		}
		filePath := filepath.Join(outputDir, fileName)

		file, err := os.Create(filePath)
		if err != nil {
			fmt.Printf("Warning: Failed to create file %s: %v\n", filePath, err)
			continue
		}

		// defer file.Close() // Closed explicitly below to catch errors sooner in loop

		executeErr := templates.ExecuteTemplate(file, "base", itemData)
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
