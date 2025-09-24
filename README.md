# CodeDoc - AI-Powered Codebase Documentation Generator

CodeDoc is a lightweight Go CLI tool that analyzes your codebase and generates comprehensive documentation using AI. It scans local repositories or public Git repos, detects frameworks and patterns, and produces a concise Markdown report with architectural insights.

## Features

- 🚀 **Fast Scanning**: Efficiently walks through codebases with smart ignore patterns
- 🤖 **AI-Powered Summaries**: Uses Claude (Anthropic) for intelligent code understanding
- 📊 **Language Detection**: Automatically identifies and reports language breakdown
- 🔍 **Framework Detection**: Recognizes popular frameworks and build tools
- 💾 **Smart Caching**: File-level caching prevents redundant API calls
- 🔒 **Security Conscious**: Redacts secrets and sensitive information
- 🎯 **Focused Output**: Strict word limits ensure concise, scannable reports

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/codepigeon/codedoc.git
cd codedoc

# Build the binary
make build

# Or install globally
make install
```

### Prerequisites

- Go 1.22 or higher
- Git (for repository cloning features)
- Anthropic API key (for AI summaries)

## Usage

### Basic Usage

```bash
# Analyze current directory
codedoc generate --path .

# Analyze a specific project
codedoc generate --path /path/to/project

# Clone and analyze a GitHub repo
codedoc generate --repo-url https://github.com/user/repo.git
```

### Command Line Options

```bash
codedoc generate [flags]

Flags:
  --path string              Path to repository to analyze
  --repo-url string          Git repository URL to clone and analyze
  --out string               Output file name (default: CODEBASE_REPORT.md)
  --max-files int            Maximum number of files to process (default: 200)
  --max-lines-per-file int   Maximum lines per file to process (default: 1000)
  --include-tests            Include test files in analysis
  --dry-run                  Generate report without LLM calls
  --lang string              Comma-separated list of languages (default: go,py,ts,js,md,yaml,dockerfile)
  --redact-secrets           Redact potential secrets from output (default: true)
  --force                    Force re-analysis of cached files
```

### Environment Variables

```bash
# Set your Anthropic API key
export ANTHROPIC_API_KEY=your_api_key_here
```

## Examples

### Dry Run (No AI)
Generate a basic report without using the LLM:

```bash
codedoc generate --path ./myproject --dry-run
```

### Focused Analysis
Analyze only specific languages:

```bash
codedoc generate --path ./myproject --lang go,python
```

### Large Codebase
Handle large repositories with file limits:

```bash
codedoc generate --path ./large-repo --max-files 500 --max-lines-per-file 2000
```

### Force Fresh Analysis
Bypass cache for updated summaries:

```bash
codedoc generate --path ./myproject --force
```

## Output Format

CodeDoc generates a structured Markdown report (`CODEBASE_REPORT.md`) with:

- **Repository metadata**: Path, last commit, language breakdown
- **Quickstart guide**: Build/run/test instructions
- **Architecture overview**: High-level system description (≤180 words)
- **Key modules**: Directory structure and purpose (≤80 words each)
- **Top files**: Important files with function summaries (≤120 words each)
- **HTTP endpoints**: Detected API routes and handlers
- **Data models**: Identified database/domain models
- **Risks & TODOs**: Code quality issues and improvements

## Project Structure

```
codedoc/
├── cmd/codedoc/          # CLI entry point
├── internal/
│   ├── scanner/          # File system traversal and analysis
│   ├── detect/           # Framework and pattern detection
│   ├── llm/              # LLM provider interface (Anthropic)
│   ├── summarize/        # Content summarization logic
│   ├── report/           # Markdown report generation
│   └── util/             # Common utilities
├── fixtures/             # Test repositories
└── Makefile              # Build automation
```

## Development

### Building

```bash
# Build binary
make build

# Run tests
make test

# Format code
make fmt

# Run linters
make lint

# Full build (clean, deps, fmt, lint, test, build)
make all
```

### Testing

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Test with a fixture repo
make demo
```

### Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests and linters
5. Submit a pull request

## Configuration

### Cache Directory

CodeDoc stores API response caches in `.codedoc-cache/` within the analyzed repository. This directory is automatically ignored during scanning.

### Ignore Patterns

Default ignore patterns:
- `.git/`, `vendor/`, `node_modules/`
- `dist/`, `build/`, `.codedoc-cache/`
- Minified files (`*.min.js`, `*.min.css`)
- Binary files > 1MB

## Limitations

- **MVP Scope**: This is an MVP focused on essential features
- **Language Support**: Best results with Go, Python, JavaScript/TypeScript
- **API Rate Limits**: Respects Anthropic API rate limits with built-in throttling
- **File Size**: Large files are truncated to key sections for analysis

## Roadmap

Future enhancements:
- [ ] Support for more LLM providers (OpenAI, local models)
- [ ] Configuration file support (`.codedoc.yml`)
- [ ] Interactive mode with progress indicators
- [ ] HTML/JSON output formats
- [ ] Plugin system for custom detectors
- [ ] Incremental analysis for large codebases

## License

MIT License - See LICENSE file for details

## Support

For issues, questions, or suggestions:
- Open an issue on GitHub
- Check existing issues for solutions
- Read the [CLAUDE.md](CLAUDE.md) design document for technical details

## Acknowledgments

Built with:
- Go standard library
- Anthropic Claude API for intelligent summaries
- Community feedback and contributions

---

*CodeDoc - Understanding codebases, one summary at a time*