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

const sitesRootDirConst = "sites"
const tmplPathConst = "templates"

func buildSite(targetSiteName string, sitesRootDir string, tmplPath string) error {
	fmt.Printf("STARTING SITE BUILD FOR: %s\n", targetSiteName)

	// Use targetSiteName for the rest of the build logic
	siteName := targetSiteName
	siteSourceDir := filepath.Join(sitesRootDir, siteName)

	// Validate the target site directory
	siteInfo, err := os.Stat(siteSourceDir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("site directory '%s' does not exist", siteSourceDir)
		}
		return fmt.Errorf("failed to access site directory '%s': %w", siteSourceDir, err)
	}
	if !siteInfo.IsDir() {
		return fmt.Errorf("path '%s' is not a directory", siteSourceDir)
	}

	baseTmpl := filepath.Join(tmplPath, "base.html")
	headerTmpl := filepath.Join(tmplPath, "header.html")
	postTmpl := filepath.Join(tmplPath, "post.html")
	pageTmpl := filepath.Join(tmplPath, "page.html")
	postListTmpl := filepath.Join(tmplPath, "post_list.html")

	parsedTemplates, err := template.ParseFiles(baseTmpl, headerTmpl, postTmpl, pageTmpl, postListTmpl)
	if err != nil {
		return fmt.Errorf("failed to parse templates: %w", err)
	}

	siteContentDir := filepath.Join(siteSourceDir, "content")
	siteStaticSourceDir := filepath.Join(siteSourceDir, "static")
	// siteOutputDir is now relative to the project root, not siteSourceDir
	siteOutputDir := filepath.Join("public", siteName) // Output for this site

	if err := os.MkdirAll(siteOutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory %s for site %s: %w", siteOutputDir, siteName, err)
	}

	// Copy static assets for the site
	siteStaticOutputDir := filepath.Join(siteOutputDir, "static")
	if _, err := os.Stat(siteStaticSourceDir); !os.IsNotExist(err) {
		fmt.Printf("Copying static assets for site %s from %s to %s\n", siteName, siteStaticSourceDir, siteStaticOutputDir)
		if err := static.CopyAll(siteStaticSourceDir, siteStaticOutputDir); err != nil {
			return fmt.Errorf("failed to copy static assets for site %s: %w", siteName, err)
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
			return fmt.Errorf("failed to load posts for site %s from %s: %w", siteName, postsContentPath, err)
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
			return fmt.Errorf("failed to load pages for site %s from %s: %w", siteName, pagesContentPath, err)
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
		if itemData.Slug == "/" {
			fileName = "index.html"
		}
		filePath := filepath.Join(siteOutputDir, fileName)

		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			fmt.Printf("Warning: Failed to create directory for %s: %v\n", filePath, err)
			continue
		}

		file, err := os.Create(filePath)
		if err != nil {
			fmt.Printf("Warning: Failed to create file %s for site %s: %v\n", filePath, siteName, err)
			continue
		}

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
	fmt.Printf("SITE BUILD FINISHED FOR: %s\n", siteName)
	return nil
}

func runServer(targetSiteName string, sitesRootDir string) error {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		originalURLPath := r.URL.Path

		resourcePath := r.URL.Path
		// sitePublicRoot is now relative to the project root
		sitePublicRoot := filepath.Join("public", targetSiteName)

		if _, err := os.Stat(sitePublicRoot); os.IsNotExist(err) {
			http.NotFound(w, r)
			return
		}

		fsPathInSite := filepath.Join(sitePublicRoot, resourcePath)
		siteFs := http.FileServer(http.Dir(sitePublicRoot))
		r.URL.Path = resourcePath

		s, err := os.Stat(fsPathInSite)
		if err == nil {
			if s.IsDir() && !strings.HasSuffix(resourcePath, "/") {
				http.Redirect(w, r, originalURLPath+"/", http.StatusMovedPermanently)
				r.URL.Path = originalURLPath
				return
			}
			siteFs.ServeHTTP(w, r)
			r.URL.Path = originalURLPath
			return
		}

		if os.IsNotExist(err) {
			fsPathWithHtmlInSite := fsPathInSite + ".html"
			sHtml, errHtml := os.Stat(fsPathWithHtmlInSite)
			if errHtml == nil && (sHtml != nil && !sHtml.IsDir()) {
				r.URL.Path = resourcePath + ".html"
				siteFs.ServeHTTP(w, r)
				r.URL.Path = originalURLPath
				return
			}
		}

		site404Path := filepath.Join(sitePublicRoot, "404.html")
		r.URL.Path = originalURLPath
		if _, statErr := os.Stat(site404Path); statErr == nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
			http.ServeFile(w, r, site404Path)
		} else {
			http.NotFound(w, r)
		}
	})

	port := ":8080"
	// sitePublicDirForServer is now relative to the project root
	sitePublicDirForServer := filepath.Join("public", targetSiteName)
	fmt.Printf("Starting server on http://127.0.0.1%s (serving site '%s' from %s)\n", port, targetSiteName, sitePublicDirForServer)
	serverErr := http.ListenAndServe("127.0.0.1"+port, nil)
	if serverErr != nil {
		return fmt.Errorf("failed to start server: %w", serverErr)
	}
	return nil
}

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: go run cmd/main.go <action> <site_name>")
		fmt.Println("Actions: build, serve")
		os.Exit(1)
	}
	action := os.Args[1]
	targetSiteName := os.Args[2]

	switch action {
	case "build":
		err := buildSite(targetSiteName, sitesRootDirConst, tmplPathConst)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error building site %s: %v\n", targetSiteName, err)
			os.Exit(1)
		}
		fmt.Printf("Site %s built successfully.\n", targetSiteName)
	case "serve":
		err := buildSite(targetSiteName, sitesRootDirConst, tmplPathConst)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error building site %s before serving: %v\n", targetSiteName, err)
			os.Exit(1)
		}
		fmt.Printf("Site %s built successfully. Starting server...\n", targetSiteName)
		err = runServer(targetSiteName, sitesRootDirConst)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error running server for site %s: %v\n", targetSiteName, err)
			os.Exit(1)
		}
	default:
		fmt.Printf("Unknown action: %s. Available actions: build, serve\n", action)
		os.Exit(1)
	}
}
