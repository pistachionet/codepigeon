# CodeDoc v0.1 - Codebase Documentation Generator

CodeDoc is a lightweight Go CLI tool that analyzes your codebase and generates structured documentation reports. This first version focuses on local repository scanning with basic analysis capabilities.

## Version 1.0 Features (Current)

- 📁 **Local Repository Scanning**: Analyze codebases from local filesystem paths
- 📊 **Language Detection**: Identifies programming languages and calculates LOC (lines of code)
- 🔍 **Smart File Walking**: Respects common ignore patterns (node_modules, .git, vendor, etc.)
- 📝 **Structured Reports**: Generates CODEBASE_REPORT.md with consistent formatting
- 🏃 **Dry Run Mode**: Generate skeleton reports without external dependencies
- 🎯 **File Limits**: Control analysis scope with max-files and max-lines-per-file options

## What's NOT in v1.0 (Coming Soon)

- ❌ **No LLM Integration**: AI-powered summaries not yet implemented
- ❌ **No Git Cloning**: --repo-url flag present but not functional
- ❌ **No Caching**: File-level caching system not implemented
- ❌ **No Framework Detection**: Framework/library detection pending
- ❌ **No Endpoint Detection**: API route detection not available
- ❌ **No Model Detection**: Data model identification not implemented

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

## Usage

### Basic Usage (v1.0)

```bash
# Analyze current directory
codedoc generate --path .

# Analyze a specific project
codedoc generate --path /path/to/project

# Dry run mode (skeleton report only)
codedoc generate --path . --dry-run
```

### Command Line Options (v1.0)

```bash
codedoc generate [flags]

Available Flags:
  --path string              Path to repository to analyze (required)
  --out string               Output file name (default: CODEBASE_REPORT.md)
  --max-files int            Maximum number of files to process (default: 200)
  --max-lines-per-file int   Maximum lines per file to process (default: 1000)
  --include-tests            Include test files in analysis (default: false)
  --dry-run                  Generate skeleton report only (default: false)
  --lang string              Languages to analyze (default: go,py,ts,js,md,yaml,dockerfile)

Flags Present but Not Functional in v1.0:
  --repo-url string          (Not implemented)
  --redact-secrets           (Not implemented)
  --force                    (Not implemented)
```

## Examples (v1.0)

### Basic Analysis
Generate a report for current directory:

```bash
codedoc generate --path .
```

### Dry Run Mode
Generate skeleton report structure:

```bash
codedoc generate --path ./myproject --dry-run
```

### Language Filtering
Analyze only specific languages:

```bash
codedoc generate --path ./myproject --lang go,python
```

### File Limits
Control analysis scope:

```bash
codedoc generate --path ./large-repo --max-files 100
```

## Output Format (v1.0)

CodeDoc generates a structured Markdown report (`CODEBASE_REPORT.md`) with:

### Currently Implemented:
- **Repository metadata**: Path, language breakdown, file count, total LOC
- **Language statistics**: Percentage breakdown by file type
- **File listing**: Organized list of analyzed files

### Report Sections (Skeleton Only in v1.0):
- **Quickstart**: Placeholder section
- **Architecture Overview**: Placeholder section
- **Key Modules/Directories**: Basic directory listing
- **Top Files**: File paths without summaries
- **HTTP Endpoints**: Empty table structure
- **Data Models**: Empty table structure
- **Notable Risks/TODOs**: Placeholder section

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

## Configuration (v1.0)

### Ignore Patterns

Default ignore patterns (hardcoded):
- `.git/`, `vendor/`, `node_modules/`
- `dist/`, `build/`, `.codedoc-cache/`
- Minified files (`*.min.js`, `*.min.css`)
- Binary files and common build artifacts

### Supported Languages

File extensions recognized in v1.0:
- **Go**: `.go`
- **Python**: `.py`
- **JavaScript/TypeScript**: `.js`, `.ts`, `.jsx`, `.tsx`
- **Markdown**: `.md`
- **YAML**: `.yaml`, `.yml`
- **Dockerfile**: `Dockerfile`, `.dockerfile`

## Limitations (v1.0)

- **No AI Integration**: All summaries are placeholders
- **Local Only**: No remote repository support
- **Basic Analysis**: Simple file counting and language detection only
- **No Caching**: Every run performs full analysis
- **Limited Detection**: No framework or pattern recognition

## Roadmap to v2.0

Next version will include:
- [ ] Anthropic Claude API integration for intelligent summaries
- [ ] Git repository cloning support
- [ ] File-level caching system
- [ ] Framework and library detection
- [ ] Endpoint and route detection
- [ ] Data model identification
- [ ] Secret redaction
- [ ] Risk and TODO identification

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
