package static

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// CopyAll copies all files and directories from srcDir to destDir.
// It preserves file permissions for copied files.
func CopyAll(srcDir, destDir string) error {
	// Ensure the destination directory for static assets exists
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination static directory %s: %w", destDir, err)
	}

	return filepath.WalkDir(srcDir, func(srcPath string, d os.DirEntry, err error) error {
		if err != nil {
			return err // Error from WalkDir itself
		}

		// Determine the corresponding destination path
		relPath, err := filepath.Rel(srcDir, srcPath)
		if err != nil {
			return fmt.Errorf("failed to get relative path for %s: %w", srcPath, err)
		}
		destPath := filepath.Join(destDir, relPath)

		if d.IsDir() {
			// Create directory in destination, mirroring source permissions
			info, err := d.Info()
			if err != nil {
				return fmt.Errorf("failed to get info for directory %s: %w", srcPath, err)
			}
			if err := os.MkdirAll(destPath, info.Mode().Perm()); err != nil { // Use Perm() for mode
				return fmt.Errorf("failed to create directory %s: %w", destPath, err)
			}
		} else {
			// It's a file, copy it
			if err := copyFile(srcPath, destPath); err != nil {
				return fmt.Errorf("failed to copy file from %s to %s: %w", srcPath, destPath, err)
			}
		}
		return nil
	})
}

// copyFile copies a single file from src to dest, preserving permissions.
func copyFile(src, dest string) error {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	// Ensure the destination directory for the file exists
	destFileDir := filepath.Dir(dest)
	if err := os.MkdirAll(destFileDir, 0755); err != nil { // Create parent dir if not exists
		return fmt.Errorf("failed to create directory for destination file %s: %w", dest, err)
	}

	destination, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer destination.Close()

	if _, err := io.Copy(destination, source); err != nil {
		return err
	}

	// Preserve permissions
	if err := os.Chmod(dest, sourceFileStat.Mode().Perm()); err != nil { // Use Perm() for mode
		// Log warning, but don't fail the whole copy for a chmod error
		fmt.Printf("Warning: Failed to set permissions on %s: %v\n", dest, err)
	}
	fmt.Printf("Successfully copied static asset %s to %s\n", src, dest)
	return nil
}
