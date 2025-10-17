package report

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/codepigeon/codedoc/internal/detect"
	"github.com/codepigeon/codedoc/internal/scanner"
	"github.com/codepigeon/codedoc/internal/summarize"
)

type Options struct {
	RepoPath        string
	RepoURL         string
	ScanResult      *scanner.Result
	DetectionResult *detect.Result
	Summaries       *summarize.Result
	OutputFile      string
}

func Generate(ctx context.Context, opts Options) error {
	var builder strings.Builder

	writeHeader(&builder, opts)
	writeQuickstart(&builder, opts)
	writeArchitecture(&builder, opts)
	writeModules(&builder, opts)
	writeTopFiles(&builder, opts)
	writeEndpoints(&builder, opts)
	writeModels(&builder, opts)
	writeRisks(&builder, opts)

	content := builder.String()

	if err := os.WriteFile(opts.OutputFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write report: %w", err)
	}

	return nil
}

func writeHeader(builder *strings.Builder, opts Options) {
	repoName := opts.ScanResult.RepoMetadata.Name
	if repoName == "" {
		repoName = filepath.Base(opts.RepoPath)
	}

	builder.WriteString(fmt.Sprintf("# %s — Codebase Report\n\n", repoName))

	pathOrURL := opts.RepoPath
	if opts.RepoURL != "" {
		pathOrURL = opts.RepoURL
	}
	builder.WriteString(fmt.Sprintf("**Path/URL:** %s  \n", pathOrURL))

	commitInfo := getGitCommitInfo(opts.RepoPath)
	builder.WriteString(fmt.Sprintf("**Last Commit:** %s by %s on %s  \n",
		commitInfo.Hash, commitInfo.Author, commitInfo.Date))

	builder.WriteString("**Languages:** ")
	writeLanguageBreakdown(builder, opts.ScanResult.LanguageStats)
	builder.WriteString("  \n")

	builder.WriteString(fmt.Sprintf("**Size:** %d files, %d LOC\n\n",
		opts.ScanResult.TotalFiles, opts.ScanResult.TotalLines))
}

func writeLanguageBreakdown(builder *strings.Builder, stats map[string]scanner.LanguageStat) {
	type langStat struct {
		name       string
		percentage float64
	}

	languages := []langStat{}
	for name, stat := range stats {
		languages = append(languages, langStat{name: name, percentage: stat.Percentage})
	}

	sort.Slice(languages, func(i, j int) bool {
		return languages[i].percentage > languages[j].percentage
	})

	parts := []string{}
	for i, lang := range languages {
		if i >= 5 {
			break
		}
		parts = append(parts, fmt.Sprintf("%s %.1f%%", lang.name, lang.percentage))
	}

	builder.WriteString(strings.Join(parts, ", "))
}

func writeQuickstart(builder *strings.Builder, opts Options) {
	builder.WriteString("## Quickstart\n")

	if len(opts.Summaries.QuickstartSteps) > 0 {
		for _, step := range opts.Summaries.QuickstartSteps {
			builder.WriteString(fmt.Sprintf("- %s\n", step))
		}
	} else {
		builder.WriteString("- Clone the repository\n")
		builder.WriteString("- Install dependencies\n")
		builder.WriteString("- Run the application\n")
	}

	builder.WriteString("\n")
}

func writeArchitecture(builder *strings.Builder, opts Options) {
	builder.WriteString("## Architecture Overview\n")

	if opts.Summaries.ArchitectureSummary != "" {
		builder.WriteString(opts.Summaries.ArchitectureSummary)
	} else {
		builder.WriteString("Architecture overview not available (dry-run mode or LLM unavailable).")
	}

	builder.WriteString("\n\n")
}

func writeModules(builder *strings.Builder, opts Options) {
	builder.WriteString("## Key Modules / Directories\n")
	builder.WriteString("| Module | Summary |\n")
	builder.WriteString("|---|---|\n")

	modules := []string{}
	for module := range opts.Summaries.ModuleSummaries {
		modules = append(modules, module)
	}
	sort.Strings(modules)

	if len(modules) == 0 {
		modules = identifyModulesFromScan(opts.ScanResult)
	}

	for _, module := range modules {
		summary := opts.Summaries.ModuleSummaries[module]
		if summary == "" {
			summary = fmt.Sprintf("Module containing %s functionality", getModuleType(module))
		}
		builder.WriteString(fmt.Sprintf("| /%s | %s |\n", module, summary))
	}

	builder.WriteString("\n")
}

func writeTopFiles(builder *strings.Builder, opts Options) {
	builder.WriteString("## Top Files\n")

	files := []string{}
	for path := range opts.Summaries.FileSummaries {
		files = append(files, path)
	}
	sort.Strings(files)

	if len(files) == 0 {
		files = selectTopFilesForReport(opts.ScanResult.Files, 5)
	}

	for _, path := range files {
		summary := opts.Summaries.FileSummaries[path]

		builder.WriteString(fmt.Sprintf("### %s\n", path))

		if summary.Summary != "" {
			builder.WriteString(fmt.Sprintf("**Role.** %s\n\n", summary.Summary))
		} else {
			builder.WriteString("**Role.** File summary not available.\n\n")
		}

		if len(summary.Functions) > 0 {
			builder.WriteString("**Key functions/classes**\n")
			for _, fn := range summary.Functions {
				builder.WriteString(fmt.Sprintf("- %s\n", fn))
			}
			builder.WriteString("\n")
		}
	}
}

func writeEndpoints(builder *strings.Builder, opts Options) {
	builder.WriteString("## HTTP Endpoints (detected)\n")

	if len(opts.DetectionResult.Endpoints) > 0 {
		builder.WriteString("| Method | Path | Handler/File |\n")
		builder.WriteString("|---|---|---|\n")

		count := 0
		for _, endpoint := range opts.DetectionResult.Endpoints {
			builder.WriteString(fmt.Sprintf("| %s | %s | %s |\n",
				endpoint.Method, endpoint.Path, endpoint.File))
			count++
			if count >= 20 {
				break
			}
		}
	} else {
		builder.WriteString("No HTTP endpoints detected.\n")
	}

	builder.WriteString("\n")
}

func writeModels(builder *strings.Builder, opts Options) {
	builder.WriteString("## Data Models (detected)\n")

	if len(opts.DetectionResult.Models) > 0 {
		builder.WriteString("| Model | Fields | File |\n")
		builder.WriteString("|---|---|---|\n")

		for _, model := range opts.DetectionResult.Models {
			fields := strings.Join(model.Fields[:min(5, len(model.Fields))], ", ")
			if len(model.Fields) > 5 {
				fields += ", ..."
			}
			builder.WriteString(fmt.Sprintf("| %s | %s | %s |\n",
				model.Name, fields, model.File))
		}
	} else {
		builder.WriteString("No data models detected.\n")
	}

	builder.WriteString("\n")
}

func writeRisks(builder *strings.Builder, opts Options) {
	builder.WriteString("## Notable Risks / TODOs\n")

	risks := identifyRisks(opts)

	if len(risks) > 0 {
		for _, risk := range risks {
			builder.WriteString(fmt.Sprintf("- %s\n", risk))
		}
	} else {
		builder.WriteString("- No significant risks detected\n")
	}

	builder.WriteString("\n")
}

func getGitCommitInfo(repoPath string) scanner.CommitInfo {
	info := scanner.CommitInfo{
		Hash:   "unknown",
		Author: "unknown",
		Date:   time.Now().Format("2006-01-02"),
	}

	cmd := exec.Command("git", "log", "-1", "--format=%H|%an|%ad", "--date=short")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return info
	}

	parts := strings.Split(strings.TrimSpace(string(output)), "|")
	if len(parts) >= 3 {
		info.Hash = parts[0][:7]
		info.Author = parts[1]
		info.Date = parts[2]
	}

	return info
}

func identifyModulesFromScan(scanResult *scanner.Result) []string {
	dirFiles := make(map[string]int)
	for _, file := range scanResult.Files {
		dir := filepath.Dir(file.RelativePath)
		if dir != "." && !strings.HasPrefix(dir, ".") {
			parts := strings.Split(dir, string(filepath.Separator))
			if len(parts) <= 2 {
				dirFiles[dir]++
			}
		}
	}

	modules := []string{}
	for dir, count := range dirFiles {
		if count >= 2 {
			modules = append(modules, dir)
		}
	}

	sort.Strings(modules)
	if len(modules) > 10 {
		modules = modules[:10]
	}

	return modules
}

func getModuleType(module string) string {
	lower := strings.ToLower(module)

	if strings.Contains(lower, "cmd") {
		return "command-line interface"
	}
	if strings.Contains(lower, "internal") {
		return "internal"
	}
	if strings.Contains(lower, "pkg") {
		return "public package"
	}
	if strings.Contains(lower, "api") {
		return "API"
	}
	if strings.Contains(lower, "web") {
		return "web interface"
	}
	if strings.Contains(lower, "test") {
		return "testing"
	}
	if strings.Contains(lower, "doc") {
		return "documentation"
	}
	if strings.Contains(lower, "util") || strings.Contains(lower, "common") {
		return "utility"
	}
	if strings.Contains(lower, "model") || strings.Contains(lower, "entity") {
		return "data model"
	}
	if strings.Contains(lower, "service") || strings.Contains(lower, "handler") {
		return "business logic"
	}

	return "application"
}

func selectTopFilesForReport(files []scanner.FileInfo, limit int) []string {
	paths := []string{}

	for _, file := range files {
		if !file.IsTest && file.Lines > 10 {
			paths = append(paths, file.RelativePath)
		}
		if len(paths) >= limit {
			break
		}
	}

	return paths
}

func identifyRisks(opts Options) []string {
	risks := []string{}

	if opts.ScanResult.TotalFiles > 1000 {
		risks = append(risks, fmt.Sprintf("Large codebase with %d files may benefit from modularization",
			opts.ScanResult.TotalFiles))
	}

	testCount := 0
	for _, file := range opts.ScanResult.Files {
		if file.IsTest {
			testCount++
		}
	}

	if float64(testCount)/float64(opts.ScanResult.TotalFiles) < 0.1 {
		risks = append(risks, "Low test coverage (less than 10% test files)")
	}

	for _, file := range opts.ScanResult.Files {
		if file.Lines > 1000 {
			risks = append(risks, fmt.Sprintf("Large file: %s (%d lines) - consider splitting",
				file.RelativePath, file.Lines))
			break
		}
	}

	hasTests := false
	hasDocs := false
	hasCI := false

	for _, file := range opts.ScanResult.Files {
		base := filepath.Base(file.RelativePath)
		if strings.Contains(base, "test") {
			hasTests = true
		}
		if base == "README.md" || base == "CONTRIBUTING.md" {
			hasDocs = true
		}
		if strings.Contains(file.RelativePath, ".github/workflows") ||
		   base == ".gitlab-ci.yml" || base == "Jenkinsfile" {
			hasCI = true
		}
	}

	if !hasTests {
		risks = append(risks, "No test files detected")
	}
	if !hasDocs {
		risks = append(risks, "Missing README.md documentation")
	}
	if !hasCI {
		risks = append(risks, "No CI/CD configuration detected")
	}

	if len(opts.DetectionResult.Frameworks) > 3 {
		risks = append(risks, fmt.Sprintf("Multiple frameworks detected (%d) - consider consolidation",
			len(opts.DetectionResult.Frameworks)))
	}

	foundLockFile := false
	for _, file := range opts.ScanResult.Files {
		base := filepath.Base(file.RelativePath)
		if base == "package-lock.json" || base == "go.sum" || base == "Gemfile.lock" ||
		   base == "yarn.lock" || base == "poetry.lock" || base == "Cargo.lock" {
			foundLockFile = true
			break
		}
	}

	if !foundLockFile && len(opts.DetectionResult.BuildTools) > 0 {
		risks = append(risks, "Missing dependency lock file")
	}

	if len(risks) > 10 {
		risks = risks[:10]
	}

	return risks
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}