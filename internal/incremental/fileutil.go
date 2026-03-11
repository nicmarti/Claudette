package incremental

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DefaultIgnorePatterns are paths to skip during builds.
var DefaultIgnorePatterns = []string{
	".claudette/**",
	"node_modules/**",
	".git/**",
	"__pycache__/**",
	"*.pyc",
	".venv/**",
	"venv/**",
	"dist/**",
	"build/**",
	".next/**",
	"target/**",
	"*.min.js",
	"*.min.css",
	"*.map",
	"*.lock",
	"package-lock.json",
	"yarn.lock",
	"*.db",
	"*.sqlite",
	"*.db-journal",
	"*.db-wal",
}

// FindRepoRoot walks up from start to find the nearest .git directory.
func FindRepoRoot(start string) string {
	if start == "" {
		start, _ = os.Getwd()
	}
	current, _ := filepath.Abs(start)
	for {
		if _, err := os.Stat(filepath.Join(current, ".git")); err == nil {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return ""
}

// FindProjectRoot returns the git repo root or the current directory.
func FindProjectRoot(start string) string {
	root := FindRepoRoot(start)
	if root != "" {
		return root
	}
	if start != "" {
		return start
	}
	cwd, _ := os.Getwd()
	return cwd
}

// GetDBPath returns the database path for a repository.
func GetDBPath(repoRoot string) string {
	crgDir := filepath.Join(repoRoot, ".claudette")
	dbPath := filepath.Join(crgDir, "graph.db")

	os.MkdirAll(crgDir, 0o755)

	// Auto-create .gitignore
	gitignore := filepath.Join(crgDir, ".gitignore")
	if _, err := os.Stat(gitignore); os.IsNotExist(err) {
		os.WriteFile(gitignore, []byte("*\n"), 0o644)
	}

	return dbPath
}

// LoadIgnorePatterns loads ignore patterns from .claudetteignore file.
func LoadIgnorePatterns(repoRoot string) []string {
	patterns := make([]string, len(DefaultIgnorePatterns))
	copy(patterns, DefaultIgnorePatterns)

	ignoreFile := filepath.Join(repoRoot, ".claudetteignore")
	data, err := os.ReadFile(ignoreFile)
	if err != nil {
		return patterns
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			patterns = append(patterns, line)
		}
	}
	return patterns
}

// ShouldIgnore checks if a path matches any ignore pattern.
func ShouldIgnore(path string, patterns []string) bool {
	for _, p := range patterns {
		matched, _ := filepath.Match(p, path)
		if matched {
			return true
		}
		// Also check if any path component matches
		if strings.Contains(p, "**") {
			// Simple glob: convert ** to a prefix match
			prefix := strings.TrimSuffix(p, "/**")
			if strings.HasPrefix(path, prefix+"/") || strings.HasPrefix(path, prefix+"\\") {
				return true
			}
		}
	}
	return false
}

// IsBinary checks if a file appears to be binary.
func IsBinary(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return true
	}
	defer f.Close()
	buf := make([]byte, 8192)
	n, err := f.Read(buf)
	if err != nil {
		return true
	}
	for i := 0; i < n; i++ {
		if buf[i] == 0 {
			return true
		}
	}
	return false
}

// FileHash computes the SHA-256 hash of a file.
func FileHash(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h), nil
}
