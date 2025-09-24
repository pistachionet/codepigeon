Prompt for Claude Code

You are an expert Go engineer and technical writer. Build a single-binary Go CLI that scans a local or simple public Git repo, analyzes the codebase, and generates a concise Markdown report named CODEBASE_REPORT.md. This is an MVP: no server, no vectors, no DB—just filesystem scanning + targeted LLM summarization with strict word limits and caching.

Objectives

- Input: path to repo (preferred) or shallow-clone from --repo-url.
- Output: CODEBASE_REPORT.md with:
  - Repo metadata & language breakdown
  - Quickstart (run/build/test)
  - Architecture overview (concise)
  - Key modules/directories summary
  - Top files summary (and one-line function/class bullets)
  - Endpoints & models (if detected)
  - Notable risks/TODOs (lint-like heuristics)
- Keep everything short, scannable, useful.

Tech Constraints

- Language: Go 1.22+
- Single module repo; no external DB.
- CLI name: codedoc.
- LLM provider abstraction in internal/llm with default Anthropic (Claude) via ANTHROPIC_API_KEY. Provide an alternate no-LLM/dry-run path that outputs the skeleton with computed stats only.
- Caching: file-level cache in ./.codedoc-cache/ keyed by file hash so repeated runs don’t re-summarize unchanged files.

Project Layout

cmd/codedoc/main.go
internal/scanner/      # walk repo, ignore globs, language detect, LOC
internal/detect/       # entrypoints, frameworks, scripts, routes, models
internal/summarize/    # prompt builders, budgeter, markdown composers
internal/llm/          # interface + anthropic impl + rate limit + cache
internal/report/       # assemble final CODEBASE_REPORT.md
internal/util/         # hashing, glob, safe io

CLI Spec

codedoc generate \
  --path ./my-repo \
  [--repo-url https://github.com/user/repo.git] \
  [--out CODEBASE_REPORT.md] \
  [--max-files 200] [--max-lines-per-file 1000] \
  [--include-tests=false] \
  [--dry-run] [--lang go,py,ts,js] \
  [--redact-secrets=true]

- Default is --path. If --repo-url is provided and --path absent, shallow clone (depth=1) to a temp dir and clean up after.
- Ignore patterns by default: .git/, vendor/, node_modules/, dist/, build/, *.min.*, binaries > 1MB.
- Language support (MVP): Go, Python, TS/JS, Markdown, YAML, Dockerfile. Gracefully skip others.

Heuristics & Detection

- Languages & LOC: per file extension; fast LOC (non-blank lines).
- Entrypoints:
  - Go: package main with func main, cmd/*/main.go
  - Py: __main__, CLI in if __name__ == "__main__":
  - Node: bin/, "start"/"build" scripts in package.json
  - Dockerfile, docker-compose, Makefile targets
- Framework hints:
  - Go: gorilla/mux, chi, gin, echo
  - Py: flask, fastapi, django
  - Node: express, next, nest
- HTTP routes (best-effort): look for router registrations (e.g., r.Get("/x"...), app.get("/x",...), FastAPI decorators).
- Data models: ORM structs/classes; SQL migrations in migrations/ or db/.

Summarization Rules (strict, concise)

- All AI text must be bounded and actionable.

Architecture Overview (≤ 180 words):
- What the project does, main components, data flow, key dependencies/frameworks.

Module/Directory Summary (≤ 80 words each):
- Purpose, noteworthy submodules, cross-deps.

File Summary (≤ 120 words per file):
- Role, key responsibilities, important imports, side-effects.

Function/Class Bullets:
- One line each: Name — purpose; key inputs → outputs; side effects (if any).
- Cap max 8 bullets per file.

Quickstart (≤ 8 bullets total):
- How to run, test, build (derive from Makefile/package.json/go.mod, etc.).

Endpoints:
- Table with METHOD | PATH | Handler/File if detected (limit 20).

Models:
- Table with Model | Fields (top 5) | File if detected.

Risks/TODOs (≤ 10 bullets):
- E.g., missing tests, mixed frameworks, large God files, secrets in repo, outdated lockfiles, duplicated patterns.

Prompting Strategy (internal/summarize)

- Never send entire huge files. If > max-lines-per-file, extract:
  - Top header/comments, imports, top-level declarations, and the largest N exported symbols.
- Build compact prompts:
  - System: "You are a senior software engineer writing concise internal docs."
  - Instruction: specify section type + word limits + bullet style.
  - Provide minimal, representative code slices (not whole file) and a file context header {path, language, LOC, imports, symbols}.
- Temperature low. Enforce limits in prompts (e.g., "Do not exceed 120 words.")

Security & Privacy

- Default redact obvious secrets (API keys, tokens) in displayed code snippets: replace with ***.
- Never transmit whole repo externally in one call. Keep each request to minimal context.
- Support --dry-run to disable LLM and just generate skeleton + stats.

Caching & Rate Limiting

- Cache per file hash (sha256(path+size+mtime)).
- Simple token budget: skip AI on files > --max-lines-per-file unless --force.
- Backoff on 429/5xx; configurable LLM_MAX_QPS.

Report Markdown Template (exact sections)

# {Repo Name} — Codebase Report

**Path/URL:** {path-or-url}  
**Last Commit:** {hash} by {author} on {date}  
**Languages:** {pie: Go 60%, TS 30%, MD 10%}  
**Size:** {files} files, {loc} LOC

## Quickstart
- {bullets … ≤8}

## Architecture Overview
{≤180 words}

## Key Modules / Directories
| Module | Summary (≤80w) |
|---|---|
| /cmd/api | … |
| /internal/core | … |

## Top Files
### {path/to/file.go}
**Role.** {≤120w}

**Key functions/classes**
- Foo() — …
- Bar() — …

### {path/to/other.ts}
…

## HTTP Endpoints (detected)
| Method | Path | Handler/File |
|---|---|---|
| GET | /health | health.go:Check |

## Data Models (detected)
| Model | Fields | File |
|---|---|---|
| User | id, email, … | models/user.go |

## Notable Risks / TODOs
- …

Implementation Tasks

- internal/scanner
  - Walk with ignore globs; collect files, sizes, LOC, imports (lightweight regex per lang).
  - Detect languages %, entrypoints, frameworks, scripts.
- internal/llm
  - Interface: Summarize(ctx, kind, constraints, context) (string, error)
  - Anthropic impl using ANTHROPIC_API_KEY; simple rate limiter; file-hash cache.
- internal/summarize
  - Builders: BuildArchitectureSummary, SummarizeModule(dir), SummarizeFile(file) (+ function bullets).
- internal/detect
  - Endpoints/models heuristics.
- internal/report
  - Compose final Markdown; write to --out.
- cmd/codedoc/main.go
  - Flags, shallow clone (if --repo-url), invoke pipeline, print where file written.
- Tests
  - Unit tests for scanner/detect.
  - Integration: run on a small fixture repo; compare golden CODEBASE_REPORT.md (allow ±5% variance).

Acceptance Criteria

- Running codedoc generate --path ./fixtures/tiny-repo produces CODEBASE_REPORT.md with all sections populated, within word limits, and no LLM calls when --dry-run is set.
- Re-running with no changes uses cache (near-instant).
- On a medium repo (~1k files), respects --max-files cap and still outputs a coherent report.

Please scaffold the project, implement the modules, add a Makefile (make build, make test, make run PATH=...), and provide a short README with usage examples.
