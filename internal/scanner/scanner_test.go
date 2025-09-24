package scanner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"main.go", "go"},
		{"app.py", "python"},
		{"index.js", "javascript"},
		{"style.css", "css"},
		{"Dockerfile", "dockerfile"},
		{"Makefile", "makefile"},
		{"README.md", "markdown"},
		{"unknown.xyz", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := detectLanguage(tt.path)
			if result != tt.expected {
				t.Errorf("detectLanguage(%s) = %s, want %s", tt.path, result, tt.expected)
			}
		})
	}
}

func TestIsTestFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"scanner_test.go", true},
		{"test_scanner.py", true},
		{"scanner.test.js", true},
		{"scanner.spec.ts", true},
		{"scanner.go", false},
		{"main.py", false},
		{"app.js", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := isTestFile(tt.path)
			if result != tt.expected {
				t.Errorf("isTestFile(%s) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestCountLines(t *testing.T) {
	tests := []struct {
		content  string
		expected int
	}{
		{"", 0},
		{"single line", 1},
		{"line one\nline two", 2},
		{"line one\nline two\nline three\n", 4},
		{"line one\r\nline two", 2},
	}

	for i, tt := range tests {
		t.Run(string(rune(i)), func(t *testing.T) {
			result := countLines([]byte(tt.content))
			if result != tt.expected {
				t.Errorf("countLines(%q) = %d, want %d", tt.content, result, tt.expected)
			}
		})
	}
}

func TestShouldIgnoreDir(t *testing.T) {
	basePath := "/project"
	tests := []struct {
		path     string
		expected bool
	}{
		{"/project/.git", true},
		{"/project/node_modules", true},
		{"/project/vendor", true},
		{"/project/src", false},
		{"/project/internal", false},
		{"/project/.codedoc-cache", true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := shouldIgnoreDir(tt.path, basePath)
			if result != tt.expected {
				t.Errorf("shouldIgnoreDir(%s, %s) = %v, want %v", tt.path, basePath, result, tt.expected)
			}
		})
	}
}

func TestScanWithFixture(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "scanner-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	testFiles := map[string]string{
		"main.go":       "package main\n\nfunc main() {\n\t// Main function\n}\n",
		"util.go":       "package main\n\nfunc Helper() {}\n",
		"main_test.go":  "package main\n\nimport \"testing\"\n\nfunc TestMain(t *testing.T) {}\n",
		"README.md":     "# Test Project\n\nThis is a test.\n",
		"go.mod":        "module test\n\ngo 1.22\n",
	}

	for name, content := range testFiles {
		path := filepath.Join(tempDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	if err := os.MkdirAll(filepath.Join(tempDir, "node_modules", "pkg"), 0755); err != nil {
		t.Fatal(err)
	}
	ignoredFile := filepath.Join(tempDir, "node_modules", "pkg", "index.js")
	if err := os.WriteFile(ignoredFile, []byte("// Should be ignored"), 0644); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	opts := Options{
		Path:         tempDir,
		MaxFiles:     100,
		IncludeTests: false,
		Languages:    []string{"go", "markdown"},
	}

	result, err := Scan(ctx, opts)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	expectedFiles := 4
	if result.TotalFiles != expectedFiles {
		t.Errorf("Expected %d files, got %d", expectedFiles, result.TotalFiles)
	}

	hasGoFiles := false
	hasMarkdown := false
	for _, file := range result.Files {
		if file.Language == "go" {
			hasGoFiles = true
		}
		if file.Language == "markdown" {
			hasMarkdown = true
		}
		if file.IsTest {
			t.Error("Test file should not be included when IncludeTests=false")
		}
		if filepath.Base(file.Path) == "index.js" {
			t.Error("node_modules file should be ignored")
		}
	}

	if !hasGoFiles {
		t.Error("Expected to find Go files")
	}
	if !hasMarkdown {
		t.Error("Expected to find Markdown files")
	}

	if _, ok := result.LanguageStats["go"]; !ok {
		t.Error("Expected Go in language stats")
	}
}

func TestLanguageSupport(t *testing.T) {
	tests := []struct {
		language  string
		supported []string
		expected  bool
	}{
		{"go", []string{"go", "python"}, true},
		{"python", []string{"go", "python"}, true},
		{"ruby", []string{"go", "python"}, false},
		{"go", []string{}, true},
		{"anything", []string{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.language, func(t *testing.T) {
			result := isLanguageSupported(tt.language, tt.supported)
			if result != tt.expected {
				t.Errorf("isLanguageSupported(%s, %v) = %v, want %v",
					tt.language, tt.supported, result, tt.expected)
			}
		})
	}
}