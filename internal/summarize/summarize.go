package summarize

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/codepigeon/codedoc/internal/detect"
	"github.com/codepigeon/codedoc/internal/llm"
	"github.com/codepigeon/codedoc/internal/scanner"
)

type Options struct {
	ScanResult      *scanner.Result
	DetectionResult *detect.Result
	MaxLinesPerFile int
	LLMProvider     llm.Provider
	RedactSecrets   bool
}

type Result struct {
	ArchitectureSummary string
	ModuleSummaries     map[string]string
	FileSummaries       map[string]FileSummary
	QuickstartSteps     []string
}

type FileSummary struct {
	Path       string
	Summary    string
	Functions  []string
	Cached     bool
	TokensUsed int
}

func Summarize(ctx context.Context, opts Options) (*Result, error) {
	result := &Result{
		ModuleSummaries: make(map[string]string),
		FileSummaries:   make(map[string]FileSummary),
		QuickstartSteps: []string{},
	}

	if opts.LLMProvider == nil {
		opts.LLMProvider = llm.NewNoOpProvider()
	}

	if err := summarizeArchitecture(ctx, opts, result); err != nil {
		return nil, fmt.Errorf("architecture summary failed: %w", err)
	}

	if err := summarizeModules(ctx, opts, result); err != nil {
		return nil, fmt.Errorf("module summary failed: %w", err)
	}

	if err := summarizeTopFiles(ctx, opts, result); err != nil {
		return nil, fmt.Errorf("file summary failed: %w", err)
	}

	if err := generateQuickstart(ctx, opts, result); err != nil {
		return nil, fmt.Errorf("quickstart generation failed: %w", err)
	}

	return result, nil
}

func summarizeArchitecture(ctx context.Context, opts Options, result *Result) error {
	context := buildArchitectureContext(opts)

	request := llm.SummarizeRequest{
		Type:    llm.SummaryTypeArchitecture,
		Context: context,
		Constraints: llm.Constraints{
			MaxWords: 180,
		},
	}

	response, err := opts.LLMProvider.Summarize(ctx, request)
	if err != nil {
		return err
	}

	result.ArchitectureSummary = response.Summary
	return nil
}

func buildArchitectureContext(opts Options) string {
	var parts []string

	parts = append(parts, fmt.Sprintf("Repository: %s", opts.ScanResult.RepoMetadata.Name))
	parts = append(parts, fmt.Sprintf("Total files: %d", opts.ScanResult.TotalFiles))
	parts = append(parts, fmt.Sprintf("Total lines: %d", opts.ScanResult.TotalLines))

	parts = append(parts, "\nLanguages:")
	for lang, stat := range opts.ScanResult.LanguageStats {
		parts = append(parts, fmt.Sprintf("- %s: %.1f%% (%d files, %d lines)",
			lang, stat.Percentage, stat.FileCount, stat.Lines))
	}

	if len(opts.DetectionResult.Frameworks) > 0 {
		parts = append(parts, "\nFrameworks detected:")
		for _, fw := range opts.DetectionResult.Frameworks {
			parts = append(parts, fmt.Sprintf("- %s (%s)", fw.Name, fw.Language))
		}
	}

	if len(opts.DetectionResult.BuildTools) > 0 {
		parts = append(parts, "\nBuild tools:")
		for _, tool := range opts.DetectionResult.BuildTools {
			parts = append(parts, fmt.Sprintf("- %s (%s)", tool.Type, tool.File))
		}
	}

	if len(opts.DetectionResult.Entrypoints) > 0 {
		parts = append(parts, "\nEntrypoints:")
		for _, ep := range opts.DetectionResult.Entrypoints {
			parts = append(parts, fmt.Sprintf("- %s: %s", ep.Type, ep.Path))
		}
	}

	dirStructure := buildDirectoryStructure(opts.ScanResult.Files)
	parts = append(parts, "\nKey directories:")
	parts = append(parts, dirStructure...)

	return strings.Join(parts, "\n")
}

func buildDirectoryStructure(files []scanner.FileInfo) []string {
	dirCounts := make(map[string]int)
	for _, file := range files {
		dir := filepath.Dir(file.RelativePath)
		if dir != "." {
			parts := strings.Split(dir, string(filepath.Separator))
			for i := range parts {
				subDir := strings.Join(parts[:i+1], string(filepath.Separator))
				dirCounts[subDir]++
			}
		}
	}

	topDirs := []string{}
	for dir, count := range dirCounts {
		depth := strings.Count(dir, string(filepath.Separator))
		if depth <= 2 && count >= 2 {
			topDirs = append(topDirs, fmt.Sprintf("- /%s (%d files)", dir, count))
		}
	}

	if len(topDirs) > 10 {
		topDirs = topDirs[:10]
	}

	return topDirs
}

func summarizeModules(ctx context.Context, opts Options, result *Result) error {
	modules := identifyKeyModules(opts.ScanResult.Files)

	for _, module := range modules {
		context := buildModuleContext(module, opts.ScanResult.Files)

		request := llm.SummarizeRequest{
			Type:    llm.SummaryTypeModule,
			Context: context,
			Constraints: llm.Constraints{
				MaxWords: 80,
			},
		}

		response, err := opts.LLMProvider.Summarize(ctx, request)
		if err != nil {
			continue
		}

		result.ModuleSummaries[module] = response.Summary
	}

	return nil
}

func identifyKeyModules(files []scanner.FileInfo) []string {
	dirFiles := make(map[string]int)
	for _, file := range files {
		dir := filepath.Dir(file.RelativePath)
		if dir != "." {
			dirFiles[dir]++
		}
	}

	modules := []string{}
	for dir, count := range dirFiles {
		depth := strings.Count(dir, string(filepath.Separator))
		if depth <= 2 && count >= 3 {
			modules = append(modules, dir)
		}
	}

	if len(modules) > 10 {
		modules = modules[:10]
	}

	return modules
}

func buildModuleContext(module string, files []scanner.FileInfo) string {
	var parts []string
	parts = append(parts, fmt.Sprintf("Module: %s", module))

	moduleFiles := []scanner.FileInfo{}
	for _, file := range files {
		if strings.HasPrefix(file.RelativePath, module) {
			moduleFiles = append(moduleFiles, file)
		}
	}

	langCounts := make(map[string]int)
	totalLines := 0
	for _, file := range moduleFiles {
		langCounts[file.Language]++
		totalLines += file.Lines
	}

	parts = append(parts, fmt.Sprintf("Files: %d", len(moduleFiles)))
	parts = append(parts, fmt.Sprintf("Lines: %d", totalLines))

	parts = append(parts, "Languages:")
	for lang, count := range langCounts {
		parts = append(parts, fmt.Sprintf("- %s: %d files", lang, count))
	}

	parts = append(parts, "\nKey files:")
	for i, file := range moduleFiles {
		if i >= 10 {
			break
		}
		parts = append(parts, fmt.Sprintf("- %s (%d lines)", filepath.Base(file.RelativePath), file.Lines))
	}

	return strings.Join(parts, "\n")
}

func summarizeTopFiles(ctx context.Context, opts Options, result *Result) error {
	topFiles := selectTopFiles(opts.ScanResult.Files, 10)

	for _, file := range topFiles {
		context, err := buildFileContext(file, opts.MaxLinesPerFile, opts.RedactSecrets)
		if err != nil {
			continue
		}

		summaryRequest := llm.SummarizeRequest{
			Type:    llm.SummaryTypeFile,
			Context: context,
			Constraints: llm.Constraints{
				MaxWords: 120,
			},
			CacheKey: file.Hash,
		}

		summaryResponse, err := opts.LLMProvider.Summarize(ctx, summaryRequest)
		if err != nil {
			continue
		}

		functionsRequest := llm.SummarizeRequest{
			Type:    llm.SummaryTypeFunction,
			Context: context,
			Constraints: llm.Constraints{
				MaxBullets: 8,
			},
			CacheKey: file.Hash + "-functions",
		}

		functionsResponse, err := opts.LLMProvider.Summarize(ctx, functionsRequest)
		if err != nil {
			functionsResponse.Summary = ""
		}

		functions := []string{}
		if functionsResponse.Summary != "" {
			for _, line := range strings.Split(functionsResponse.Summary, "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "-") || strings.HasPrefix(line, "*") {
					functions = append(functions, strings.TrimSpace(line[1:]))
				}
			}
		}

		result.FileSummaries[file.RelativePath] = FileSummary{
			Path:       file.RelativePath,
			Summary:    summaryResponse.Summary,
			Functions:  functions,
			Cached:     summaryResponse.Cached,
			TokensUsed: summaryResponse.Tokens + functionsResponse.Tokens,
		}
	}

	return nil
}

func selectTopFiles(files []scanner.FileInfo, limit int) []scanner.FileInfo {
	selected := []scanner.FileInfo{}

	priority := []scanner.FileInfo{}
	regular := []scanner.FileInfo{}

	for _, file := range files {
		if file.IsTest {
			continue
		}

		base := filepath.Base(file.RelativePath)
		if base == "main.go" || base == "main.py" || base == "index.js" || base == "app.py" ||
			base == "server.js" || base == "Makefile" || base == "package.json" ||
			base == "requirements.txt" || base == "go.mod" {
			priority = append(priority, file)
		} else {
			regular = append(regular, file)
		}
	}

	selected = append(selected, priority...)

	remaining := limit - len(selected)
	if remaining > 0 && len(regular) > 0 {
		if len(regular) > remaining {
			regular = regular[:remaining]
		}
		selected = append(selected, regular...)
	}

	if len(selected) > limit {
		selected = selected[:limit]
	}

	return selected
}

func buildFileContext(file scanner.FileInfo, maxLines int, redactSecrets bool) (string, error) {
	content, err := os.ReadFile(file.Path)
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(content), "\n")
	if len(lines) > maxLines {
		lines = extractKeyLines(lines, maxLines)
	}

	text := strings.Join(lines, "\n")
	if redactSecrets {
		text = redactSecretsFromText(text)
	}

	context := fmt.Sprintf("File: %s\n", file.RelativePath)
	context += fmt.Sprintf("Language: %s\n", file.Language)
	context += fmt.Sprintf("Total lines: %d\n", file.Lines)
	context += fmt.Sprintf("Size: %d bytes\n", file.Size)
	context += "\nContent sample:\n"
	context += text

	return context, nil
}

func extractKeyLines(lines []string, maxLines int) []string {
	if len(lines) <= maxLines {
		return lines
	}

	result := []string{}

	headerLines := 0
	for i, line := range lines {
		if i >= 50 {
			break
		}
		result = append(result, line)
		headerLines++
		if strings.Contains(line, "func ") || strings.Contains(line, "class ") ||
			strings.Contains(line, "def ") || strings.Contains(line, "interface ") {
			break
		}
	}

	remaining := maxLines - headerLines
	if remaining > 0 {
		skip := (len(lines) - headerLines) / remaining
		if skip < 1 {
			skip = 1
		}

		for i := headerLines; i < len(lines) && len(result) < maxLines; i += skip {
			result = append(result, lines[i])
		}
	}

	return result
}

func redactSecretsFromText(text string) string {
	patterns := []string{
		`(api[_-]?key|api[_-]?secret|access[_-]?token|auth[_-]?token|private[_-]?key)[\s]*[:=][\s]*["']?[\w\-]+["']?`,
		`(password|passwd|pwd)[\s]*[:=][\s]*["']?[\w\-]+["']?`,
		`[a-zA-Z0-9]{40}`,
		`sk-[a-zA-Z0-9]{48}`,
		`ghp_[a-zA-Z0-9]{36}`,
	}

	for _, pattern := range patterns {
		text = redactPattern(text, pattern)
	}

	return text
}

func redactPattern(text, pattern string) string {
	return text
}

func generateQuickstart(ctx context.Context, opts Options, result *Result) error {
	context := buildQuickstartContext(opts)

	request := llm.SummarizeRequest{
		Type:    llm.SummaryTypeQuickstart,
		Context: context,
		Constraints: llm.Constraints{
			MaxBullets: 8,
		},
	}

	response, err := opts.LLMProvider.Summarize(ctx, request)
	if err != nil {
		result.QuickstartSteps = generateDefaultQuickstart(opts)
		return nil
	}

	steps := []string{}
	for _, line := range strings.Split(response.Summary, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "-") || strings.HasPrefix(line, "*") ||
			strings.HasPrefix(line, "•") || (len(line) > 2 && line[1] == '.') {
			step := strings.TrimSpace(line)
			if len(step) > 2 {
				step = step[2:]
			}
			steps = append(steps, strings.TrimSpace(step))
		}
	}

	result.QuickstartSteps = steps
	return nil
}

func buildQuickstartContext(opts Options) string {
	var parts []string

	parts = append(parts, fmt.Sprintf("Project: %s", opts.ScanResult.RepoMetadata.Name))

	if len(opts.DetectionResult.BuildTools) > 0 {
		parts = append(parts, "\nBuild tools found:")
		for _, tool := range opts.DetectionResult.BuildTools {
			parts = append(parts, fmt.Sprintf("- %s: %s", tool.Type, tool.File))
			if len(tool.Scripts) > 0 {
				parts = append(parts, fmt.Sprintf("  Scripts: %s", strings.Join(tool.Scripts[:min(3, len(tool.Scripts))], ", ")))
			}
		}
	}

	if len(opts.DetectionResult.Entrypoints) > 0 {
		parts = append(parts, "\nEntrypoints:")
		for _, ep := range opts.DetectionResult.Entrypoints {
			parts = append(parts, fmt.Sprintf("- %s: %s", ep.Description, ep.Command))
		}
	}

	return strings.Join(parts, "\n")
}

func generateDefaultQuickstart(opts Options) []string {
	steps := []string{}

	steps = append(steps, "Clone the repository")

	for _, tool := range opts.DetectionResult.BuildTools {
		switch tool.Type {
		case "npm":
			steps = append(steps, "Install dependencies: npm install")
			if contains(tool.Scripts, "build") {
				steps = append(steps, "Build the project: npm run build")
			}
			if contains(tool.Scripts, "test") {
				steps = append(steps, "Run tests: npm test")
			}
			if contains(tool.Scripts, "start") {
				steps = append(steps, "Start the application: npm start")
			}

		case "go":
			steps = append(steps, "Download dependencies: go mod download")
			steps = append(steps, "Build the project: go build")
			steps = append(steps, "Run tests: go test ./...")

		case "make":
			if contains(tool.Scripts, "build") {
				steps = append(steps, "Build the project: make build")
			}
			if contains(tool.Scripts, "test") {
				steps = append(steps, "Run tests: make test")
			}
			if contains(tool.Scripts, "run") {
				steps = append(steps, "Run the application: make run")
			}

		case "pip":
			steps = append(steps, "Install dependencies: pip install -r requirements.txt")

		case "docker-compose":
			steps = append(steps, "Start services: docker-compose up")
		}
	}

	if len(steps) == 1 {
		steps = append(steps, "Check documentation for setup instructions")
	}

	return steps
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
