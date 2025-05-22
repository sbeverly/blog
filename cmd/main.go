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
	"github.com/sbeverly/blog/cmd/static"
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

	// Copy static assets
	sourceStaticDir := "static" // Assuming your source static files are in a directory named "static"
	targetStaticDir := filepath.Join(outputDir, "static")
	if _, err := os.Stat(sourceStaticDir); !os.IsNotExist(err) {
		fmt.Printf("Copying static assets from %s to %s\n", sourceStaticDir, targetStaticDir)
		if err := static.CopyAll(sourceStaticDir, targetStaticDir); err != nil {
			panic(fmt.Sprintf("Failed to copy static assets: %v", err))
		}
	} else {
		fmt.Printf("Source static directory %s not found, skipping static asset copy.\n", sourceStaticDir)
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
	pageTmpl := filepath.Join(tmplPath, "page.html") // Added page.html

	templates, err := template.ParseFiles(baseTmpl, headerTmpl, postPageTmpl, pageTmpl) // Added pageTmpl
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

	// The http.Handle for "/static/" is no longer needed here,
	// as publicFs will serve files from outputDir (e.g., "public/static/...").

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Construct the path in the filesystem relative to the outputDir
		fsPath := filepath.Join(outputDir, r.URL.Path)

		// Check if the direct path exists
		s, err := os.Stat(fsPath)
		if err == nil {
			// If it's a directory, http.FileServer will look for index.html within it.
			// If it's a file, http.FileServer will serve it.
			// This handles cases like "/" (serves public/index.html)
			// and "/my-first-post.html" (serves the file directly).
			if s.IsDir() && !strings.HasSuffix(r.URL.Path, "/") {
				// For a directory accessed without a trailing slash (e.g., /foo),
				// http.FileServer will typically issue a redirect to /foo/.
				// This is standard behavior and generally what we want.
			}
			publicFs.ServeHTTP(w, r)
			return
		}

		// If direct path doesn't exist, try with .html extension (for clean URLs like /about)
		if os.IsNotExist(err) {
			fsPathWithHtml := fsPath + ".html"
			sHtml, errHtml := os.Stat(fsPathWithHtml)
			// Ensure it's a file and not a directory
			if errHtml == nil && (sHtml != nil && !sHtml.IsDir()) {
				// Adjust r.URL.Path for publicFs to serve the .html file
				r.URL.Path = r.URL.Path + ".html"
				publicFs.ServeHTTP(w, r)
				return
			}
		}

		// If neither path nor path.html exists (or path.html is a dir), serve 404
		// w.WriteHeader(http.StatusNotFound)
		// Serve the custom 404.html page
		// Note: http.ServeFile uses the original r.URL.Path for content negotiation,
		// but we are explicitly serving 404.html here.
		http.ServeFile(w, r, filepath.Join(outputDir, "404.html"))
	})

	port := "8080"
	fmt.Printf("Starting server on http://localhost:%s\n", port)
	serverErr := http.ListenAndServe(":"+port, nil)
	if serverErr != nil {
		panic(fmt.Sprintf("Failed to start server: %v", serverErr))
	}
}
