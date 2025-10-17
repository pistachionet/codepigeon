package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/codepigeon/codedoc/internal/detect"
	"github.com/codepigeon/codedoc/internal/llm"
	"github.com/codepigeon/codedoc/internal/report"
	"github.com/codepigeon/codedoc/internal/scanner"
	"github.com/codepigeon/codedoc/internal/summarize"
	"github.com/codepigeon/codedoc/internal/util"
)

// Version information set by GoReleaser
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
	builtBy = "unknown"
)

type Config struct {
	Path            string
	RepoURL         string
	OutputFile      string
	MaxFiles        int
	MaxLinesPerFile int
	IncludeTests    bool
	DryRun          bool
	Languages       []string
	RedactSecrets   bool
	Force           bool
}

func main() {
	config := parseFlags()

	if err := validateConfig(config); err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	ctx := context.Background()
	if err := runGenerate(ctx, config); err != nil {
		log.Fatalf("Generation failed: %v", err)
	}
}

func parseFlags() *Config {
	config := &Config{}

	generateCmd := flag.NewFlagSet("generate", flag.ExitOnError)
	generateCmd.StringVar(&config.Path, "path", "", "Path to repository to analyze")
	generateCmd.StringVar(&config.RepoURL, "repo-url", "", "Git repository URL to clone and analyze")
	generateCmd.StringVar(&config.OutputFile, "out", "CODEBASE_REPORT.md", "Output file name")
	generateCmd.IntVar(&config.MaxFiles, "max-files", 200, "Maximum number of files to process")
	generateCmd.IntVar(&config.MaxLinesPerFile, "max-lines-per-file", 1000, "Maximum lines per file to process")
	generateCmd.BoolVar(&config.IncludeTests, "include-tests", false, "Include test files in analysis")
	generateCmd.BoolVar(&config.DryRun, "dry-run", false, "Generate report without LLM calls")
	generateCmd.BoolVar(&config.RedactSecrets, "redact-secrets", true, "Redact potential secrets from output")
	generateCmd.BoolVar(&config.Force, "force", false, "Force re-analysis of cached files")

	langDefault := "go,py,ts,js,md,yaml,dockerfile"
	langUsage := "Comma-separated list of languages to analyze"
	var langString string
	generateCmd.StringVar(&langString, "lang", langDefault, langUsage)

	// Check for version flag first
	if len(os.Args) > 1 && (os.Args[1] == "-v" || os.Args[1] == "--version" || os.Args[1] == "version") {
		fmt.Printf("codedoc version %s\n", version)
		fmt.Printf("  commit: %s\n", commit)
		fmt.Printf("  built at: %s\n", date)
		fmt.Printf("  built by: %s\n", builtBy)
		os.Exit(0)
	}

	if len(os.Args) < 2 {
		fmt.Println("Usage: codedoc generate [flags]")
		fmt.Println("       codedoc version")
		generateCmd.PrintDefaults()
		os.Exit(1)
	}

	if os.Args[1] != "generate" {
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		fmt.Println("Usage: codedoc generate [flags]")
		fmt.Println("       codedoc version")
		os.Exit(1)
	}

	generateCmd.Parse(os.Args[2:])

	config.Languages = parseLanguages(langString)

	return config
}

func parseLanguages(langString string) []string {
	if langString == "" {
		return []string{"go", "py", "ts", "js", "md", "yaml", "dockerfile"}
	}

	languages := []string{}
	for _, lang := range splitAndTrim(langString, ",") {
		if lang != "" {
			languages = append(languages, lang)
		}
	}
	return languages
}

func splitAndTrim(s, sep string) []string {
	parts := []string{}
	for _, part := range stringSlice(s, sep) {
		trimmed := stringTrim(part)
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}

func stringSlice(s, sep string) []string {
	if s == "" {
		return []string{}
	}

	result := []string{}
	start := 0
	for i := 0; i < len(s); i++ {
		if i+len(sep) <= len(s) && s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
			i = start - 1
		}
	}
	result = append(result, s[start:])
	return result
}

func stringTrim(s string) string {
	start := 0
	end := len(s)

	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}

	return s[start:end]
}

func validateConfig(config *Config) error {
	if config.Path == "" && config.RepoURL == "" {
		return fmt.Errorf("either --path or --repo-url must be specified")
	}

	if config.Path != "" && config.RepoURL != "" {
		return fmt.Errorf("cannot specify both --path and --repo-url")
	}

	if config.MaxFiles <= 0 {
		return fmt.Errorf("--max-files must be positive")
	}

	if config.MaxLinesPerFile <= 0 {
		return fmt.Errorf("--max-lines-per-file must be positive")
	}

	return nil
}

func runGenerate(ctx context.Context, config *Config) error {
	startTime := time.Now()

	repoPath := config.Path

	if config.RepoURL != "" {
		clonedPath, cleanupFunc, err := cloneRepository(config.RepoURL)
		if err != nil {
			return fmt.Errorf("failed to clone repository: %w", err)
		}
		defer cleanupFunc()
		repoPath = clonedPath
	}

	fmt.Printf("Analyzing repository: %s\n", repoPath)

	scanOpts := scanner.Options{
		Path:         repoPath,
		MaxFiles:     config.MaxFiles,
		IncludeTests: config.IncludeTests,
		Languages:    config.Languages,
	}

	scanResult, err := scanner.Scan(ctx, scanOpts)
	if err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	fmt.Printf("Scanned %d files (%d lines)\n", len(scanResult.Files), scanResult.TotalLines)

	detectOpts := detect.Options{
		Files: scanResult.Files,
	}

	detectionResult, err := detect.Detect(ctx, detectOpts)
	if err != nil {
		return fmt.Errorf("detection failed: %w", err)
	}

	var llmProvider llm.Provider
	if !config.DryRun {
		llmProvider, err = llm.NewAnthropicProvider(llm.AnthropicConfig{
			CacheDir: filepath.Join(repoPath, ".codedoc-cache"),
			Force:    config.Force,
		})
		if err != nil {
			return fmt.Errorf("failed to create LLM provider: %w", err)
		}
	}

	summarizeOpts := summarize.Options{
		ScanResult:      scanResult,
		DetectionResult: detectionResult,
		MaxLinesPerFile: config.MaxLinesPerFile,
		LLMProvider:     llmProvider,
		RedactSecrets:   config.RedactSecrets,
	}

	summaries, err := summarize.Summarize(ctx, summarizeOpts)
	if err != nil {
		return fmt.Errorf("summarization failed: %w", err)
	}

	reportOpts := report.Options{
		RepoPath:        repoPath,
		RepoURL:         config.RepoURL,
		ScanResult:      scanResult,
		DetectionResult: detectionResult,
		Summaries:       summaries,
		OutputFile:      config.OutputFile,
	}

	if err := report.Generate(ctx, reportOpts); err != nil {
		return fmt.Errorf("report generation failed: %w", err)
	}

	elapsed := time.Since(startTime)
	fmt.Printf("\nReport generated: %s\n", config.OutputFile)
	fmt.Printf("Time elapsed: %s\n", elapsed.Round(time.Second))

	return nil
}

func cloneRepository(repoURL string) (string, func(), error) {
	tempDir, err := os.MkdirTemp("", "codedoc-*")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temp dir: %w", err)
	}

	cleanupFunc := func() {
		os.RemoveAll(tempDir)
	}

	if err := util.GitCloneShallow(repoURL, tempDir); err != nil {
		cleanupFunc()
		return "", nil, err
	}

	return tempDir, cleanupFunc, nil
}
