package detect

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/codepigeon/codedoc/internal/scanner"
)

type Options struct {
	Files []scanner.FileInfo
}

type Result struct {
	Entrypoints []Entrypoint
	Frameworks  []Framework
	Endpoints   []Endpoint
	Models      []Model
	BuildTools  []BuildTool
}

type Entrypoint struct {
	Type        string
	Path        string
	Command     string
	Description string
}

type Framework struct {
	Name     string
	Language string
	Files    []string
}

type Endpoint struct {
	Method  string
	Path    string
	Handler string
	File    string
}

type Model struct {
	Name   string
	Fields []string
	File   string
}

type BuildTool struct {
	Type    string
	File    string
	Scripts []string
}

func Detect(ctx context.Context, opts Options) (*Result, error) {
	result := &Result{
		Entrypoints: []Entrypoint{},
		Frameworks:  []Framework{},
		Endpoints:   []Endpoint{},
		Models:      []Model{},
		BuildTools:  []BuildTool{},
	}

	for _, file := range opts.Files {
		detectEntrypoints(file, result)
		detectFrameworks(file, result)
		detectBuildTools(file, result)
		detectEndpoints(file, result)
		detectModels(file, result)
	}

	deduplicateResults(result)

	return result, nil
}

func detectEntrypoints(file scanner.FileInfo, result *Result) {
	base := filepath.Base(file.Path)
	dir := filepath.Dir(file.RelativePath)

	switch file.Language {
	case "go":
		if base == "main.go" || strings.Contains(dir, "cmd/") {
			content, err := os.ReadFile(file.Path)
			if err == nil && strings.Contains(string(content), "func main()") {
				result.Entrypoints = append(result.Entrypoints, Entrypoint{
					Type:        "go-binary",
					Path:        file.RelativePath,
					Command:     fmt.Sprintf("go run %s", file.RelativePath),
					Description: "Go main package",
				})
			}
		}

	case "python":
		if base == "__main__.py" || base == "main.py" || base == "app.py" {
			result.Entrypoints = append(result.Entrypoints, Entrypoint{
				Type:        "python-script",
				Path:        file.RelativePath,
				Command:     fmt.Sprintf("python %s", file.RelativePath),
				Description: "Python entrypoint",
			})
		}

	case "javascript", "typescript":
		if base == "index.js" || base == "index.ts" || base == "server.js" || base == "app.js" {
			result.Entrypoints = append(result.Entrypoints, Entrypoint{
				Type:        "node-script",
				Path:        file.RelativePath,
				Command:     fmt.Sprintf("node %s", file.RelativePath),
				Description: "Node.js entrypoint",
			})
		}

	case "dockerfile":
		result.Entrypoints = append(result.Entrypoints, Entrypoint{
			Type:        "docker",
			Path:        file.RelativePath,
			Command:     "docker build .",
			Description: "Docker container",
		})
	}
}

func detectFrameworks(file scanner.FileInfo, result *Result) {
	content, err := os.ReadFile(file.Path)
	if err != nil {
		return
	}

	contentStr := string(content)

	frameworkPatterns := map[string]map[string][]string{
		"go": {
			"gin":         {"github.com/gin-gonic/gin", "gin.New()", "gin.Default()"},
			"echo":        {"github.com/labstack/echo", "echo.New()"},
			"fiber":       {"github.com/gofiber/fiber", "fiber.New()"},
			"chi":         {"github.com/go-chi/chi", "chi.NewRouter()"},
			"gorilla/mux": {"github.com/gorilla/mux", "mux.NewRouter()"},
			"beego":       {"github.com/astaxie/beego", "beego.Run()"},
		},
		"python": {
			"flask":   {"from flask import", "Flask(__name__)"},
			"django":  {"from django", "django.contrib"},
			"fastapi": {"from fastapi import", "FastAPI()"},
			"tornado": {"import tornado", "tornado.web"},
			"pyramid": {"from pyramid", "pyramid.config"},
		},
		"javascript": {
			"express": {"require('express')", "require(\"express\")", "from 'express'"},
			"koa":     {"require('koa')", "from 'koa'"},
			"hapi":    {"require('@hapi/hapi')", "from '@hapi/hapi'"},
			"fastify": {"require('fastify')", "from 'fastify'"},
		},
		"typescript": {
			"express": {"from 'express'", "import express"},
			"nest":    {"@nestjs/", "from '@nestjs"},
			"next":    {"from 'next'", "import next"},
		},
	}

	if patterns, ok := frameworkPatterns[file.Language]; ok {
		for framework, indicators := range patterns {
			for _, indicator := range indicators {
				if strings.Contains(contentStr, indicator) {
					result.Frameworks = append(result.Frameworks, Framework{
						Name:     framework,
						Language: file.Language,
						Files:    []string{file.RelativePath},
					})
					break
				}
			}
		}
	}
}

func detectBuildTools(file scanner.FileInfo, result *Result) {
	base := filepath.Base(file.Path)

	switch strings.ToLower(base) {
	case "makefile", "gnumakefile":
		content, _ := os.ReadFile(file.Path)
		scripts := extractMakefileTargets(string(content))
		result.BuildTools = append(result.BuildTools, BuildTool{
			Type:    "make",
			File:    file.RelativePath,
			Scripts: scripts,
		})

	case "package.json":
		content, _ := os.ReadFile(file.Path)
		scripts := extractPackageJsonScripts(string(content))
		result.BuildTools = append(result.BuildTools, BuildTool{
			Type:    "npm",
			File:    file.RelativePath,
			Scripts: scripts,
		})

	case "go.mod":
		result.BuildTools = append(result.BuildTools, BuildTool{
			Type:    "go",
			File:    file.RelativePath,
			Scripts: []string{"go build", "go test", "go run"},
		})

	case "cargo.toml":
		result.BuildTools = append(result.BuildTools, BuildTool{
			Type:    "cargo",
			File:    file.RelativePath,
			Scripts: []string{"cargo build", "cargo test", "cargo run"},
		})

	case "requirements.txt", "setup.py", "pipfile":
		result.BuildTools = append(result.BuildTools, BuildTool{
			Type:    "pip",
			File:    file.RelativePath,
			Scripts: []string{"pip install -r requirements.txt"},
		})

	case "docker-compose.yml", "docker-compose.yaml":
		result.BuildTools = append(result.BuildTools, BuildTool{
			Type:    "docker-compose",
			File:    file.RelativePath,
			Scripts: []string{"docker-compose up", "docker-compose build"},
		})
	}
}

func detectEndpoints(file scanner.FileInfo, result *Result) {
	content, err := os.ReadFile(file.Path)
	if err != nil {
		return
	}

	contentStr := string(content)
	endpoints := []Endpoint{}

	switch file.Language {
	case "go":
		endpoints = extractGoEndpoints(contentStr, file.RelativePath)
	case "python":
		endpoints = extractPythonEndpoints(contentStr, file.RelativePath)
	case "javascript", "typescript":
		endpoints = extractJSEndpoints(contentStr, file.RelativePath)
	}

	result.Endpoints = append(result.Endpoints, endpoints...)
}

func detectModels(file scanner.FileInfo, result *Result) {
	content, err := os.ReadFile(file.Path)
	if err != nil {
		return
	}

	contentStr := string(content)
	models := []Model{}

	switch file.Language {
	case "go":
		models = extractGoModels(contentStr, file.RelativePath)
	case "python":
		models = extractPythonModels(contentStr, file.RelativePath)
	case "javascript", "typescript":
		models = extractJSModels(contentStr, file.RelativePath)
	}

	result.Models = append(result.Models, models...)
}

func extractMakefileTargets(content string) []string {
	targets := []string{}
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasSuffix(line, ":") && !strings.HasPrefix(line, "#") {
			target := strings.TrimSuffix(line, ":")
			if idx := strings.Index(target, ":"); idx > 0 {
				target = target[:idx]
			}
			if target != "" && !strings.HasPrefix(target, ".") {
				targets = append(targets, target)
			}
		}
	}

	return targets
}

func extractPackageJsonScripts(content string) []string {
	scripts := []string{}

	if idx := strings.Index(content, "\"scripts\""); idx >= 0 {
		start := strings.Index(content[idx:], "{")
		if start < 0 {
			return scripts
		}
		start += idx

		end := strings.Index(content[start:], "}")
		if end < 0 {
			return scripts
		}
		end += start

		scriptSection := content[start:end]
		lines := strings.Split(scriptSection, "\n")

		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.Contains(line, "\":") {
				parts := strings.Split(line, "\"")
				if len(parts) >= 2 {
					script := parts[1]
					if script != "" && script != "scripts" {
						scripts = append(scripts, script)
					}
				}
			}
		}
	}

	return scripts
}

func extractGoEndpoints(content, file string) []Endpoint {
	endpoints := []Endpoint{}
	patterns := []string{
		".Get(",
		".Post(",
		".Put(",
		".Delete(",
		".Patch(",
		".Handle(",
		".HandleFunc(",
	}

	for _, pattern := range patterns {
		if strings.Contains(content, pattern) {
		}
	}

	return endpoints
}

func extractPythonEndpoints(content, file string) []Endpoint {
	endpoints := []Endpoint{}
	patterns := []string{
		"@app.route(",
		"@app.get(",
		"@app.post(",
		"@app.put(",
		"@app.delete(",
		"@router.get(",
		"@router.post(",
	}

	for _, pattern := range patterns {
		if strings.Contains(content, pattern) {
		}
	}

	return endpoints
}

func extractJSEndpoints(content, file string) []Endpoint {
	endpoints := []Endpoint{}
	patterns := []string{
		"app.get(",
		"app.post(",
		"app.put(",
		"app.delete(",
		"router.get(",
		"router.post(",
	}

	for _, pattern := range patterns {
		if strings.Contains(content, pattern) {
		}
	}

	return endpoints
}

func extractGoModels(content, file string) []Model {
	models := []Model{}
	return models
}

func extractPythonModels(content, file string) []Model {
	models := []Model{}
	return models
}

func extractJSModels(content, file string) []Model {
	models := []Model{}
	return models
}

func deduplicateResults(result *Result) {
	frameworkMap := make(map[string]Framework)
	for _, fw := range result.Frameworks {
		key := fmt.Sprintf("%s-%s", fw.Language, fw.Name)
		if existing, ok := frameworkMap[key]; ok {
			existing.Files = append(existing.Files, fw.Files...)
			frameworkMap[key] = existing
		} else {
			frameworkMap[key] = fw
		}
	}

	result.Frameworks = []Framework{}
	for _, fw := range frameworkMap {
		result.Frameworks = append(result.Frameworks, fw)
	}
}
