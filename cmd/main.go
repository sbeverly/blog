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
	Type        string // "post" or "page"
}

// PostListPageData holds data for the all posts listing page.
type PostListPageData struct {
	Title string       // For the <title> tag of the page
	Posts []RenderItem // List of posts to display
	Type  string       // "post_list"
}

func main() {
	fmt.Println("STARTING SITE BUILDS")

	sitesRootDir := "sites" // Root directory for all sites

	siteEntries, err := os.ReadDir(sitesRootDir)
	if err != nil {
		panic(fmt.Sprintf("Failed to read sites directory %s: %v", sitesRootDir, err))
	}

	tmplPath := "templates"
	baseTmpl := filepath.Join(tmplPath, "base.html")
	headerTmpl := filepath.Join(tmplPath, "header.html")
	postTmpl := filepath.Join(tmplPath, "post.html")
	pageTmpl := filepath.Join(tmplPath, "page.html")
	postListTmpl := filepath.Join(tmplPath, "post_list.html")

	parsedTemplates, err := template.ParseFiles(baseTmpl, headerTmpl, postTmpl, pageTmpl, postListTmpl)
	if err != nil {
		panic(fmt.Sprintf("Failed to parse templates: %v", err))
	}

	foundSites := false
	for _, entry := range siteEntries {
		if !entry.IsDir() {
			continue
		}
		foundSites = true
		siteName := entry.Name()
		fmt.Printf("BUILDING SITE: %s\n", siteName)

		siteSourceDir := filepath.Join(sitesRootDir, siteName)
		siteContentDir := filepath.Join(siteSourceDir, "content")
		siteStaticSourceDir := filepath.Join(siteSourceDir, "static")
		siteOutputDir := filepath.Join(siteSourceDir, "public") // Output for this site

		if err := os.MkdirAll(siteOutputDir, 0755); err != nil {
			panic(fmt.Sprintf("Failed to create output directory %s for site %s: %v", siteOutputDir, siteName, err))
		}

		// Copy static assets for the site
		siteStaticOutputDir := filepath.Join(siteOutputDir, "static")
		if _, err := os.Stat(siteStaticSourceDir); !os.IsNotExist(err) {
			fmt.Printf("Copying static assets for site %s from %s to %s\n", siteName, siteStaticSourceDir, siteStaticOutputDir)
			if err := static.CopyAll(siteStaticSourceDir, siteStaticOutputDir); err != nil {
				panic(fmt.Sprintf("Failed to copy static assets for site %s: %v", siteName, err))
			}
		} else {
			fmt.Printf("Source static directory %s for site %s not found, skipping static asset copy.\n", siteStaticSourceDir, siteName)
		}

		var allRenderItems []RenderItem
		var postItemsForListPage []RenderItem

		// Load Posts for the site
		postsContentPath := filepath.Join(siteContentDir, "posts")
		if _, err := os.Stat(postsContentPath); !os.IsNotExist(err) {
			loadedPosts, err := posts.Load(postsContentPath)
			if err != nil {
				panic(fmt.Sprintf("Failed to load posts for site %s from %s: %v", siteName, postsContentPath, err))
			}
			for _, p := range loadedPosts {
				renderItem := RenderItem{
					Title:       p.Title,
					ContentHTML: p.ContentHTML,
					Slug:        p.Slug,
					Type:        "post",
				}
				allRenderItems = append(allRenderItems, renderItem)
				postItemsForListPage = append(postItemsForListPage, renderItem)
			}
		} else {
			fmt.Printf("Posts directory not found for site %s, skipping: %s\n", siteName, postsContentPath)
		}

		// Load Pages for the site
		pagesContentPath := filepath.Join(siteContentDir, "pages")
		if _, err := os.Stat(pagesContentPath); !os.IsNotExist(err) {
			loadedPages, err := pages.Load(pagesContentPath)
			if err != nil {
				panic(fmt.Sprintf("Failed to load pages for site %s from %s: %v", siteName, pagesContentPath, err))
			}
			for _, p := range loadedPages {
				allRenderItems = append(allRenderItems, RenderItem{
					Title:       p.Title,
					ContentHTML: p.ContentHTML,
					Slug:        p.Slug,
					Type:        "page",
				})
			}
		} else {
			fmt.Printf("Pages directory not found for site %s, skipping: %s\n", siteName, pagesContentPath)
		}

		if len(allRenderItems) == 0 {
			fmt.Printf("No content found in '%s' or '%s' for site %s.\n", postsContentPath, pagesContentPath, siteName)
		}

		// Generate individual post and page files for the site
		for _, itemData := range allRenderItems {
			fileName := strings.TrimPrefix(itemData.Slug, "/") + ".html"
			if itemData.Slug == "/" { // This case might need review based on how slugs like "/" are generated
				fileName = "index.html"
			}
			filePath := filepath.Join(siteOutputDir, fileName)

			// Ensure parent directory exists for nested slugs if any
			if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
				fmt.Printf("Warning: Failed to create directory for %s: %v\n", filePath, err)
				continue
			}
			
			file, err := os.Create(filePath)
			if err != nil {
				fmt.Printf("Warning: Failed to create file %s for site %s: %v\n", filePath, siteName, err)
				continue
			}

			// defer file.Close() // Closed explicitly below to catch errors sooner in loop

			executeErr := parsedTemplates.ExecuteTemplate(file, "base", itemData)
			closeErr := file.Close()

			if executeErr != nil {
				fmt.Printf("Warning: Failed to execute template for %s (site %s): %v\n", filePath, siteName, executeErr)
			} else if closeErr != nil {
				fmt.Printf("Warning: Failed to close file %s (site %s): %v\n", filePath, siteName, closeErr)
			} else {
				fmt.Printf("Successfully generated %s for site %s\n", filePath, siteName)
			}
		}

		// Generate the post list page for the site if there are posts
		if len(postItemsForListPage) > 0 {
			postListData := PostListPageData{
				Title: "All Posts",
				Posts: postItemsForListPage,
				Type:  "post_list",
			}
			postListFilePath := filepath.Join(siteOutputDir, "posts.html")
			postListFile, createErr := os.Create(postListFilePath)
			if createErr != nil {
				fmt.Printf("Warning: Failed to create post list file %s for site %s: %v\n", postListFilePath, siteName, createErr)
			} else {
				// defer postListFile.Close() // Closed explicitly below
				executeErr := parsedTemplates.ExecuteTemplate(postListFile, "base", postListData)
				closeErr := postListFile.Close()

				if executeErr != nil {
					fmt.Printf("Warning: Failed to execute template for %s (site %s): %v\n", postListFilePath, siteName, executeErr)
				} else if closeErr != nil {
					fmt.Printf("Warning: Failed to close file %s (site %s): %v\n", postListFilePath, siteName, closeErr)
				} else {
					fmt.Printf("Successfully generated %s for site %s\n", postListFilePath, siteName)
				}
			}
		}
		fmt.Printf("SITE BUILD COMPLETE: %s\n", siteName)
	}

	if !foundSites {
		fmt.Printf("No sites found in %s directory.\n", sitesRootDir)
	}
	fmt.Println("ALL SITE BUILDS FINISHED")

	// The http.Handle for "/static/" is no longer needed here,
	// as publicFs will serve files from outputDir (e.g., "public/static/...").

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		originalURLPath := r.URL.Path // Store for restoration
		reqPath := originalURLPath
		var siteName, resourcePath string

		// Extract siteName and resourcePath from the URL
		// e.g., /site1/foo/bar -> siteName="site1", resourcePath="/foo/bar"
		// e.g., /site1 -> siteName="site1", resourcePath="/"
		// e.g., / -> special handling or error, depends on desired behavior for root
		trimmedPath := strings.TrimPrefix(reqPath, "/")
		if idx := strings.Index(trimmedPath, "/"); idx != -1 {
			siteName = trimmedPath[:idx]
			resourcePath = "/" + trimmedPath[idx+1:]
		} else if trimmedPath != "" { // Path is just "/sitename"
			siteName = trimmedPath
			resourcePath = "/" // Serve index.html from the site's root
		} else { // Path is "/"
			// This case needs to be decided:
			// 1. Serve a default site?
			// 2. Serve a landing page listing all sites?
			// 3. Return a 404 or specific error?
			// For now, let's assume it's an error or requires specific site in URL.
			http.NotFound(w, r) // Or a custom "choose a site" page
			return
		}
		
		// Basic validation for siteName to prevent path traversal or invalid names
		if siteName == "" || siteName == "." || siteName == ".." || strings.Contains(siteName, "/") {
			 http.NotFound(w,r) // Or a more specific error
			 return
		}

		sitePublicRoot := filepath.Join(sitesRootDir, siteName, "public")

		// Check if the site's public directory exists
		if _, err := os.Stat(sitePublicRoot); os.IsNotExist(err) {
			http.NotFound(w, r) // Site does not exist
			return
		}

		// Construct the path in the filesystem relative to the site's public directory
		fsPathInSite := filepath.Join(sitePublicRoot, resourcePath)
		siteFs := http.FileServer(http.Dir(sitePublicRoot))
		r.URL.Path = resourcePath // Crucial: http.FileServer expects path relative to its root

		// Check if the direct path exists (file or directory)
		s, err := os.Stat(fsPathInSite)
		if err == nil {
			// If it's a directory, http.FileServer will look for index.html within it.
			// If it's a file, http.FileServer will serve it.
			if s.IsDir() && !strings.HasSuffix(resourcePath, "/") {
				// Redirect to path with trailing slash for directories
				http.Redirect(w, r, originalURLPath+"/", http.StatusMovedPermanently)
				r.URL.Path = originalURLPath // Restore for next potential handler if redirect is not followed by client
				return
			}
			siteFs.ServeHTTP(w, r)
			r.URL.Path = originalURLPath // Restore for next potential handler
			return
		}

		// If direct path doesn't exist, try with .html extension (for clean URLs like /about)
		if os.IsNotExist(err) {
			fsPathWithHtmlInSite := fsPathInSite + ".html"
			sHtml, errHtml := os.Stat(fsPathWithHtmlInSite)
			// Ensure it's a file and not a directory
			if errHtml == nil && (sHtml != nil && !sHtml.IsDir()) {
				// Adjust r.URL.Path for siteFs to serve the .html file
				r.URL.Path = resourcePath + ".html"
				siteFs.ServeHTTP(w, r)
				r.URL.Path = originalURLPath // Restore for next potential handler
				return
			}
		}

		// If neither path nor path.html exists, serve site-specific 404.html or generic 404
		site404Path := filepath.Join(sitePublicRoot, "404.html")
		r.URL.Path = originalURLPath // Restore before serving 404 or generic NotFound
		if _, statErr := os.Stat(site404Path); statErr == nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8") // Ensure correct content type for 404.html
			w.WriteHeader(http.StatusNotFound)
			http.ServeFile(w, r, site404Path) // Serve the site-specific 404.html
		} else {
			http.NotFound(w, r) // Fallback to generic Go http.NotFound
		}
	})

	port := "8080"
	fmt.Printf("Starting server on http://localhost:%s (serving from site-specific public directories within %s)\n", port, sitesRootDir)
	serverErr := http.ListenAndServe(":"+port, nil)
	if serverErr != nil {
		panic(fmt.Sprintf("Failed to start server: %v", serverErr))
	}
}
