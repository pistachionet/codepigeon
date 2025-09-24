package scanner

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type Options struct {
	Path         string
	MaxFiles     int
	IncludeTests bool
	Languages    []string
}

type Result struct {
	Files         []FileInfo
	TotalFiles    int
	TotalLines    int
	LanguageStats map[string]LanguageStat
	RepoMetadata  RepoMetadata
}

type FileInfo struct {
	Path         string
	RelativePath string
	Size         int64
	Lines        int
	Language     string
	IsTest       bool
	Imports      []string
	Hash         string
}

type LanguageStat struct {
	FileCount int
	Lines     int
	Percentage float64
}

type RepoMetadata struct {
	Name       string
	Path       string
	LastCommit CommitInfo
}

type CommitInfo struct {
	Hash    string
	Author  string
	Date    string
	Message string
}

var defaultIgnorePatterns = []string{
	".git",
	"vendor",
	"node_modules",
	"dist",
	"build",
	".codedoc-cache",
	"*.min.js",
	"*.min.css",
}

func Scan(ctx context.Context, opts Options) (*Result, error) {
	if opts.Path == "" {
		return nil, fmt.Errorf("path is required")
	}

	result := &Result{
		Files:         []FileInfo{},
		LanguageStats: make(map[string]LanguageStat),
	}

	result.RepoMetadata = getRepoMetadata(opts.Path)

	err := filepath.WalkDir(opts.Path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if d.IsDir() {
			if shouldIgnoreDir(path, opts.Path) {
				return filepath.SkipDir
			}
			return nil
		}

		if shouldIgnoreFile(path, opts) {
			return nil
		}

		if len(result.Files) >= opts.MaxFiles {
			return fmt.Errorf("reached max files limit")
		}

		fileInfo, err := processFile(path, opts.Path)
		if err != nil {
			return nil
		}

		if !opts.IncludeTests && fileInfo.IsTest {
			return nil
		}

		if !isLanguageSupported(fileInfo.Language, opts.Languages) {
			return nil
		}

		result.Files = append(result.Files, *fileInfo)
		updateLanguageStats(result, fileInfo)
		result.TotalLines += fileInfo.Lines

		return nil
	})

	if err != nil && !strings.Contains(err.Error(), "reached max files limit") {
		return nil, err
	}

	result.TotalFiles = len(result.Files)
	calculateLanguagePercentages(result)

	return result, nil
}

func shouldIgnoreDir(path, basePath string) bool {
	rel, err := filepath.Rel(basePath, path)
	if err != nil {
		return false
	}

	parts := strings.Split(rel, string(filepath.Separator))
	for _, part := range parts {
		for _, pattern := range defaultIgnorePatterns {
			if matched, _ := filepath.Match(pattern, part); matched {
				return true
			}
		}
	}
	return false
}

func shouldIgnoreFile(path string, opts Options) bool {
	base := filepath.Base(path)

	for _, pattern := range defaultIgnorePatterns {
		if matched, _ := filepath.Match(pattern, base); matched {
			return true
		}
	}

	info, err := os.Stat(path)
	if err != nil {
		return true
	}

	if info.Size() > 1024*1024 {
		return true
	}

	if !info.Mode().IsRegular() {
		return true
	}

	return false
}

func processFile(path, basePath string) (*FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	rel, _ := filepath.Rel(basePath, path)

	fileInfo := &FileInfo{
		Path:         path,
		RelativePath: rel,
		Size:         info.Size(),
		Lines:        countLines(content),
		Language:     detectLanguage(path),
		IsTest:       isTestFile(path),
		Imports:      extractImports(content, detectLanguage(path)),
		Hash:         hashFile(path, info),
	}

	return fileInfo, nil
}

func countLines(content []byte) int {
	if len(content) == 0 {
		return 0
	}

	lines := 1
	for _, b := range content {
		if b == '\n' {
			lines++
		}
	}
	return lines
}

func detectLanguage(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	base := strings.ToLower(filepath.Base(path))

	languageMap := map[string]string{
		".go":         "go",
		".py":         "python",
		".js":         "javascript",
		".ts":         "typescript",
		".jsx":        "javascript",
		".tsx":        "typescript",
		".java":       "java",
		".c":          "c",
		".cpp":        "cpp",
		".cc":         "cpp",
		".h":          "c",
		".hpp":        "cpp",
		".rs":         "rust",
		".rb":         "ruby",
		".php":        "php",
		".cs":         "csharp",
		".swift":      "swift",
		".kt":         "kotlin",
		".scala":      "scala",
		".r":          "r",
		".m":          "objc",
		".mm":         "objc",
		".pl":         "perl",
		".sh":         "shell",
		".bash":       "shell",
		".zsh":        "shell",
		".fish":       "shell",
		".ps1":        "powershell",
		".lua":        "lua",
		".dart":       "dart",
		".elm":        "elm",
		".clj":        "clojure",
		".ex":         "elixir",
		".exs":        "elixir",
		".erl":        "erlang",
		".hrl":        "erlang",
		".fs":         "fsharp",
		".fsx":        "fsharp",
		".fsi":        "fsharp",
		".ml":         "ocaml",
		".mli":        "ocaml",
		".vim":        "vim",
		".yaml":       "yaml",
		".yml":        "yaml",
		".json":       "json",
		".xml":        "xml",
		".html":       "html",
		".htm":        "html",
		".css":        "css",
		".scss":       "scss",
		".sass":       "sass",
		".less":       "less",
		".sql":        "sql",
		".md":         "markdown",
		".markdown":   "markdown",
		".rst":        "rst",
		".tex":        "latex",
		".dockerfile": "dockerfile",
		".makefile":   "makefile",
		".cmake":      "cmake",
		".gradle":     "gradle",
		".proto":      "protobuf",
		".graphql":    "graphql",
		".vue":        "vue",
		".svelte":     "svelte",
	}

	if base == "dockerfile" || strings.HasPrefix(base, "dockerfile.") {
		return "dockerfile"
	}
	if base == "makefile" || base == "gnumakefile" {
		return "makefile"
	}
	if base == "cmakelists.txt" {
		return "cmake"
	}
	if base == "package.json" {
		return "json"
	}
	if base == "tsconfig.json" {
		return "json"
	}
	if base == "go.mod" || base == "go.sum" {
		return "go"
	}
	if base == "cargo.toml" || base == "cargo.lock" {
		return "rust"
	}
	if base == "requirements.txt" || base == "setup.py" || base == "pipfile" {
		return "python"
	}

	if lang, ok := languageMap[ext]; ok {
		return lang
	}

	return "unknown"
}

func isTestFile(path string) bool {
	base := filepath.Base(path)
	lower := strings.ToLower(base)

	testPatterns := []string{
		"_test.go",
		"_test.py",
		"_test.js",
		"_test.ts",
		".test.js",
		".test.ts",
		".spec.js",
		".spec.ts",
		"test_",
		"tests.",
		"spec.",
	}

	for _, pattern := range testPatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}

	if strings.Contains(path, "/test/") || strings.Contains(path, "/tests/") ||
		strings.Contains(path, "/__tests__/") || strings.Contains(path, "/spec/") {
		return true
	}

	return false
}

func extractImports(content []byte, language string) []string {
	return []string{}
}

func hashFile(path string, info os.FileInfo) string {
	return fmt.Sprintf("%s_%d_%d", path, info.Size(), info.ModTime().Unix())
}

func isLanguageSupported(language string, supported []string) bool {
	if len(supported) == 0 {
		return true
	}

	for _, lang := range supported {
		if strings.EqualFold(language, lang) {
			return true
		}
	}
	return false
}

func updateLanguageStats(result *Result, fileInfo *FileInfo) {
	stat := result.LanguageStats[fileInfo.Language]
	stat.FileCount++
	stat.Lines += fileInfo.Lines
	result.LanguageStats[fileInfo.Language] = stat
}

func calculateLanguagePercentages(result *Result) {
	if result.TotalLines == 0 {
		return
	}

	for lang, stat := range result.LanguageStats {
		stat.Percentage = float64(stat.Lines) / float64(result.TotalLines) * 100
		result.LanguageStats[lang] = stat
	}
}

func getRepoMetadata(path string) RepoMetadata {
	name := filepath.Base(path)

	metadata := RepoMetadata{
		Name: name,
		Path: path,
		LastCommit: CommitInfo{
			Hash:   "unknown",
			Author: "unknown",
			Date:   "unknown",
		},
	}

	return metadata
}