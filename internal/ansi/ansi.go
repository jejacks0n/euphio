package ansi

import (
	"euphio/internal/app"
	"euphio/internal/assets"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	ResetSeq = "\x1b[0m"
)

// RenderArt loads an ANSI/art file (with overrides), processes it (SAUCE, CP437, Templates), and writes it to the
// writer.
// It handles file lookup, extension resolution (.utf8ans, .ans, .asc), and fallback to embedded assets.
func RenderArt(w io.Writer, artName string, isUTF8 bool) error {
	// Determine possible file extensions
	extensions := []string{}
	if isUTF8 {
		extensions = append(extensions, ".utf8ans")
	}
	extensions = append(extensions, ".ans", ".asc")

	// Load the art file
	data, ext, err := LoadArt(artName, extensions)
	if err != nil {
		return fmt.Errorf("art not found: %s (checked extensions: %v)", artName, extensions)
	}

	// Remove SAUCE record
	cleanData := StripSauce(data)

	// Render templates first, as they might contain CP437 characters or UTF-8 depending on the source
	renderedData, err := RenderTemplate(cleanData, nil)
	if err != nil {
		return fmt.Errorf("failed to render template for %s: %w", artName, err)
	}

	var s string
	if isUTF8 {
		if ext == ".ans" {
			s = DecodeCP437(renderedData)
		} else {
			// If the file is already probably UTF-8, just use it as is
			s = string(renderedData)
		}
	} else {
		s = string(renderedData)
	}

	// Normalize line endings
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\n", "\r\n")

	finalBytes := []byte(s)
	if ext == ".utf8ans" {
	}

	finalBytes = append(finalBytes, []byte(ResetSeq)...)
	_, err = w.Write(finalBytes)
	return err
}

// LoadArt attempts to find and load an art file.
// It checks the configured Ansi path first, then falls back to embedded assets in "config/ansi/".
// Returns data, extension, error.
func LoadArt(name string, exts []string) ([]byte, string, error) {
	// Try configured art path (overrides)
	if app.Config != nil && app.Config.Paths.Ansi != "" {
		for _, ext := range exts {
			fullPath := filepath.Join(app.Config.Paths.Ansi, name+ext)
			data, err := os.ReadFile(fullPath)
			if err == nil {
				app.Logger.Debug("Loaded art from disk", "path", fullPath)
				return data, ext, nil
			}
		}
	}

	// Try embedded assets
	for _, ext := range exts {
		fullPath := filepath.Join("config/ansi", name+ext)
		data, err := assets.FS.ReadFile(fullPath)
		if err == nil {
			if app.Logger != nil {
				app.Logger.Debug("Loaded art from assets", "path", fullPath)
			}
			return data, ext, nil
		}
	}

	return nil, "", fmt.Errorf("art not found: %s", name)
}
