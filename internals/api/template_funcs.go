package api

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"html/template"
	"path/filepath"
	"strings"
)

// FuncMap defines the custom functions available in templates.
var funcMap = template.FuncMap{
	"add":        add,
	"subtract":   subtract,
	"multiply":   multiply,
	"min":        minInt, // Renamed to avoid conflict with potential built-ins if any
	"percentage": percentage,
	"sequence":   sequence,
	"truncate":   truncate,
	"md5":        calculateMD5,
}

func add(a, b int) int {
	return a + b
}

func subtract(a, b int) int {
	return a - b
}

func multiply(a, b int) int {
	return a * b
}

func minInt(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

// percentage calculates what percentage a is of total. Handles total being zero.
func percentage(a, total int64) string {
	if total == 0 {
		return "0.0" // Or maybe "N/A"
	}
	// Ensure floating point division
	p := (float64(a) / float64(total)) * 100.0
	// Format to one decimal place
	return fmt.Sprintf("%.1f", p)
}

// sequence generates a slice of integers from start to end (inclusive).
// Useful for pagination. Note: Template loops are 1-based usually.
func sequence(start, end int) []int {
	// Adjust if template loops expect 1-based start but Go is 0-based
	n := end - start + 1
	if n <= 0 {
		return []int{}
	}
	seq := make([]int, n)
	for i := 0; i < n; i++ {
		seq[i] = start + i
	}
	return seq
}

// truncate cuts a string to a max length, adding ellipsis if truncated.
func truncate(s string, length int) string {
	// Use rune count for proper multi-byte character handling
	runes := []rune(s)
	if len(runes) > length {
		return string(runes[:length]) + "..."
	}
	return s
}

// calculateMD5 computes the MD5 hash of a string and returns its hex representation.
// Commonly used for Gravatar URLs.
func calculateMD5(text string) string {
	hash := md5.Sum([]byte(strings.ToLower(strings.TrimSpace(text)))) // Gravatar uses lowercase email
	return hex.EncodeToString(hash[:])
}

// --- Helper function for parsing templates with the FuncMap ---

// ParseTemplates parses all template files from a directory, adding the FuncMap.
func ParseTemplates(templateDir string) (*template.Template, error) {
	templateDir = strings.TrimRight(templateDir, "/")

	// Create a new template with the function map
	tmpl := template.New("").Funcs(funcMap)

	// Get all HTML files
	pattern := fmt.Sprintf("%s/*.html", templateDir)
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("error finding templates: %w", err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no template files found in %s", templateDir)
	}

	// Parse all files at once
	tmpl, err = tmpl.ParseFiles(files...)
	if err != nil {
		return nil, fmt.Errorf("error parsing templates: %w", err)
	}

	return tmpl, nil
}
