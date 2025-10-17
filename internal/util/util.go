package util

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func GitCloneShallow(repoURL, targetDir string) error {
	cmd := exec.Command("git", "clone", "--depth", "1", repoURL, targetDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone failed: %w", err)
	}

	return nil
}

func IsGitRepo(path string) bool {
	gitDir := filepath.Join(path, ".git")
	info, err := os.Stat(gitDir)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func IsDirectory(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func NormalizeRepoURL(url string) string {
	url = strings.TrimSpace(url)
	url = strings.TrimSuffix(url, ".git")

	if strings.HasPrefix(url, "git@") {
		url = strings.Replace(url, ":", "/", 1)
		url = strings.Replace(url, "git@", "https://", 1)
	}

	return url
}

func GetRepoNameFromURL(url string) string {
	url = NormalizeRepoURL(url)

	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		name := parts[len(parts)-1]
		name = strings.TrimSuffix(name, ".git")
		return name
	}

	return "unknown"
}

func SafeTruncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func CleanPath(path string) string {
	path = filepath.Clean(path)
	path = strings.ReplaceAll(path, "\\", "/")
	return path
}

func GetFileExtension(path string) string {
	ext := filepath.Ext(path)
	if ext != "" && len(ext) > 1 {
		return ext[1:]
	}
	return ""
}

func CountNonEmptyLines(content []byte) int {
	lines := strings.Split(string(content), "\n")
	count := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count
}

func BytesToHumanReadable(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d bytes", bytes)
	}
}

func EnsureDir(path string) error {
	if !IsDirectory(path) {
		return os.MkdirAll(path, 0o755)
	}
	return nil
}

func RemoveDir(path string) error {
	return os.RemoveAll(path)
}
