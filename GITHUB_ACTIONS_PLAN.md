# GitHub Actions CI/CD Pipeline Plan for CodePigeon v0.1

## Overview

Set up automated testing, building, and release workflows for the CodePigeon Go CLI project using GitHub Actions.

## Project Context

- **Language**: Go 1.24.4
- **Type**: CLI tool (single binary)
- **Current Version**: v0.1 (pre-release)
- **Build Tool**: Makefile with targets: build, test, lint, clean
- **Module**: github.com/codepigeon/codedoc
- **Branch**: Currently on `v_1` branch

---

## Phase 1: Basic CI Workflow

### File: `.github/workflows/ci.yml`

**Triggers**:
- Push to `main`, `v_1`, `develop` branches
- Pull requests to `main`, `v_1`

**Jobs**:

#### Job 1: Test & Build
```yaml
name: CI
on:
  push:
    branches: [main, v_1, develop]
  pull_request:
    branches: [main, v_1]

jobs:
  test:
    name: Test & Build
    runs-on: ubuntu-latest

    strategy:
      matrix:
        go-version: ['1.23', '1.24']

    steps:
    - uses: actions/checkout@v4

    - name: Set up Go
      uses: actions/setup-go@v5
      with:
        go-version: ${{ matrix.go-version }}
        cache: true

    - name: Download dependencies
      run: go mod download

    - name: Verify dependencies
      run: go mod verify

    - name: Run tests
      run: make test

    - name: Build
      run: make build

    - name: Test binary exists
      run: |
        test -f build/codedoc
        ./build/codedoc --help
```

**What it does**:
- ✅ Tests on multiple Go versions
- ✅ Caches Go modules for speed
- ✅ Runs your test suite
- ✅ Builds the binary
- ✅ Smoke test (binary runs)

---

## Phase 2: Linting & Code Quality

### Add to `.github/workflows/ci.yml`

#### Job 2: Lint
```yaml
  lint:
    name: Lint
    runs-on: ubuntu-latest

    steps:
    - uses: actions/checkout@v4

    - name: Set up Go
      uses: actions/setup-go@v5
      with:
        go-version: '1.24'
        cache: true

    - name: golangci-lint
      uses: golangci/golangci-lint-action@v4
      with:
        version: latest
        args: --timeout=5m

    - name: Check formatting
      run: |
        gofmt -l .
        test -z "$(gofmt -l .)"
```

**What it does**:
- ✅ Runs golangci-lint (meta-linter with 50+ linters)
- ✅ Checks code formatting
- ✅ Fails if code isn't properly formatted

---

## Phase 3: Test Coverage

### Add to test job in `.github/workflows/ci.yml`

```yaml
    - name: Run tests with coverage
      run: go test -v -race -coverprofile=coverage.out -covermode=atomic ./...

    - name: Upload coverage to Codecov
      uses: codecov/codecov-action@v4
      with:
        file: ./coverage.out
        flags: unittests
        name: codecov-umbrella
        fail_ci_if_error: false
```

**Setup Required**:
1. Sign up at https://codecov.io
2. Add repository
3. Add `CODECOV_TOKEN` to GitHub Secrets (if private repo)

**Badge for README**:
```markdown
[![codecov](https://codecov.io/gh/pistachionet/codepigeon/branch/main/graph/badge.svg)](https://codecov.io/gh/pistachionet/codepigeon)
```

---

## Phase 4: Release Automation

### File: `.github/workflows/release.yml`

**Triggers**:
- Push of version tags (e.g., `v0.1.0`, `v1.0.0`)

```yaml
name: Release

on:
  push:
    tags:
      - 'v*.*.*'

permissions:
  contents: write

jobs:
  release:
    name: Create Release
    runs-on: ubuntu-latest

    steps:
    - uses: actions/checkout@v4
      with:
        fetch-depth: 0

    - name: Set up Go
      uses: actions/setup-go@v5
      with:
        go-version: '1.24'
        cache: true

    - name: Run tests
      run: make test

    - name: Run GoReleaser
      uses: goreleaser/goreleaser-action@v5
      with:
        version: latest
        args: release --clean
      env:
        GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

**Setup Required**: Create `.goreleaser.yml` (see Phase 5)

---

## Phase 5: GoReleaser Configuration

### File: `.goreleaser.yml`

```yaml
version: 2

before:
  hooks:
    - go mod tidy
    - go mod download

builds:
  - id: codedoc
    main: ./cmd/codedoc
    binary: codedoc
    env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w
      - -X main.version={{.Version}}
      - -X main.commit={{.Commit}}
      - -X main.date={{.Date}}

archives:
  - id: default
    format: tar.gz
    name_template: >-
      {{ .ProjectName }}_
      {{- .Version }}_
      {{- .Os }}_
      {{- .Arch }}
    format_overrides:
      - goos: windows
        format: zip
    files:
      - README.md
      - LICENSE

checksum:
  name_template: 'checksums.txt'

changelog:
  sort: asc
  filters:
    exclude:
      - '^docs:'
      - '^test:'
      - '^chore:'

release:
  github:
    owner: pistachionet
    name: codepigeon
  draft: false
  prerelease: auto
  name_template: "{{.ProjectName}} v{{.Version}}"
```

**What it does**:
- ✅ Builds for Linux, macOS, Windows (amd64 & arm64)
- ✅ Creates compressed archives
- ✅ Generates checksums
- ✅ Auto-generates changelog
- ✅ Creates GitHub Release with binaries

---

## Phase 6: Homebrew Release (Future)

### Add to `.goreleaser.yml` after initial release works:

```yaml
brews:
  - name: codedoc
    repository:
      owner: pistachionet
      name: homebrew-tap
    homepage: https://github.com/pistachionet/codepigeon
    description: "AI-powered codebase documentation generator"
    license: MIT
    install: |
      bin.install "codedoc"
```

**Setup Required**:
1. Create `pistachionet/homebrew-tap` repository
2. Add deploy key or use GITHUB_TOKEN

---

## Phase 7: Optional Enhancements

### Dependabot for Dependencies

**File**: `.github/dependabot.yml`

```yaml
version: 2
updates:
  - package-ecosystem: "gomod"
    directory: "/"
    schedule:
      interval: "weekly"
    open-pull-requests-limit: 5

  - package-ecosystem: "github-actions"
    directory: "/"
    schedule:
      interval: "weekly"
    open-pull-requests-limit: 5
```

### Security Scanning

**Add to `.github/workflows/ci.yml`**:

```yaml
  security:
    name: Security Scan
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4

    - name: Run Gosec Security Scanner
      uses: securego/gosec@master
      with:
        args: './...'

    - name: Run Trivy vulnerability scanner
      uses: aquasecurity/trivy-action@master
      with:
        scan-type: 'fs'
        scan-ref: '.'
        format: 'sarif'
        output: 'trivy-results.sarif'
```

### Build Status Badges

**Add to `README.md`**:

```markdown
[![CI](https://github.com/pistachionet/codepigeon/actions/workflows/ci.yml/badge.svg)](https://github.com/pistachionet/codepigeon/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/codepigeon/codedoc)](https://goreportcard.com/report/github.com/codepigeon/codedoc)
[![codecov](https://codecov.io/gh/pistachionet/codepigeon/branch/main/graph/badge.svg)](https://codecov.io/gh/pistachionet/codepigeon)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
```

---

## Implementation Order (Recommended)

### Step 1: Basic CI (Start Here)
```bash
# Create directory structure
mkdir -p .github/workflows

# Create ci.yml with Phase 1 + Phase 2 (test, build, lint)
# Commit and push to test
git add .github/
git commit -m "Add basic CI workflow"
git push
```

### Step 2: Add Coverage (After CI works)
```bash
# Update ci.yml with Phase 3 (coverage)
# Sign up for Codecov
# Add badge to README
git commit -am "Add test coverage reporting"
git push
```

### Step 3: Release Setup (When ready for v0.1.0)
```bash
# Create .goreleaser.yml (Phase 5)
# Create release.yml workflow (Phase 4)
# Update main.go to include version variables
git commit -am "Add release automation"
git push

# Create first release
git tag v0.1.0
git push origin v0.1.0
```

### Step 4: Enhancements (Optional)
```bash
# Add dependabot.yml (Phase 7)
# Add security scanning (Phase 7)
# Add badges to README (Phase 7)
```

---

## Testing the Pipeline

### 1. Test CI Workflow
```bash
# Make a small change and push
echo "# Test CI" >> README.md
git add README.md
git commit -m "Test CI workflow"
git push
```

Go to: `https://github.com/pistachionet/codepigeon/actions`

### 2. Test Release Workflow
```bash
# Create a tag
git tag v0.1.0
git push origin v0.1.0
```

Check: `https://github.com/pistachionet/codepigeon/releases`

---

## Main.go Version Support

To support version info in releases, update `cmd/codedoc/main.go`:

```go
package main

import (
    // ... existing imports
)

var (
    version = "dev"
    commit  = "none"
    date    = "unknown"
)

func main() {
    // Add version flag
    versionFlag := flag.Bool("version", false, "Print version information")

    // ... existing flag parsing

    if *versionFlag {
        fmt.Printf("codedoc version %s\n", version)
        fmt.Printf("commit: %s\n", commit)
        fmt.Printf("built: %s\n", date)
        os.Exit(0)
    }

    // ... rest of main
}
```

---

## Expected Results

After full implementation:

✅ **Every push/PR**:
- Runs tests on multiple Go versions
- Lints code
- Builds binary
- Reports coverage

✅ **Every tag push**:
- Runs full test suite
- Builds binaries for 6 platforms
- Creates GitHub Release with:
  - Changelog
  - Downloadable binaries
  - Checksums

✅ **README shows**:
- Build status badge (green ✓)
- Coverage percentage
- Go Report Card score
- License badge

---

## Prompt for Claude Opus 4

Use this exact prompt with Claude to implement:

```
I have a Go CLI project called CodePigeon at /Users/navmisa/Documents/*pistachionet/1_ai_project/codepigeon

Please implement a GitHub Actions CI/CD pipeline following the plan in GITHUB_ACTIONS_PLAN.md

Start with:
1. Phase 1: Basic CI (test & build)
2. Phase 2: Linting
3. Phase 3: Test coverage (skip Codecov setup for now)

Create the files, explain what each part does, and help me test it.

After that works, we'll add:
4. Phase 4 & 5: Release automation with GoReleaser

Use the project structure:
- Go version: 1.24.4
- Module: github.com/codepigeon/codedoc
- Main branch: v_1
- Build command: make build
- Test command: make test
- Makefile already has all targets
```

---

## Troubleshooting

### Issue: Tests fail in CI but pass locally
**Solution**: Check Go version mismatch, ensure `go.mod` is correct

### Issue: Lint fails but code looks fine
**Solution**: Run `make fmt` locally before pushing

### Issue: GoReleaser fails
**Solution**: Check `.goreleaser.yml` syntax with `goreleaser check`

### Issue: Release not triggered
**Solution**: Ensure tag format is `v*.*.*` (e.g., `v0.1.0`)

---

## Cost & Performance

- **GitHub Actions**: Free for public repos (2000 min/month for private)
- **Average CI run time**: ~2-5 minutes
- **Average release time**: ~5-10 minutes
- **Codecov**: Free for open source

---

## Next Steps After Setup

1. Add more comprehensive tests (currently minimal)
2. Add integration tests (test on real repos)
3. Add performance benchmarks
4. Consider Docker image builds
5. Add PR preview deployments

---

*Generated for CodePigeon v0.1 - Pre-Release*
