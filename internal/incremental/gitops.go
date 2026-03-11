package incremental

import (
	"os/exec"
	"strings"
	"time"
)

const gitTimeout = 30 * time.Second

func runGit(repoRoot string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// GetChangedFiles returns files changed since base ref.
func GetChangedFiles(repoRoot, base string) []string {
	out, err := runGit(repoRoot, "diff", "--name-only", base)
	if err != nil {
		// Fallback: try diff against cached
		out, err = runGit(repoRoot, "diff", "--name-only", "--cached")
		if err != nil {
			return nil
		}
	}
	return splitLines(out)
}

// GetStagedAndUnstaged returns all modified files (staged + unstaged + untracked).
func GetStagedAndUnstaged(repoRoot string) []string {
	out, err := runGit(repoRoot, "status", "--porcelain")
	if err != nil {
		return nil
	}
	var files []string
	for _, line := range strings.Split(out, "\n") {
		if len(line) > 3 {
			entry := strings.TrimSpace(line[3:])
			if idx := strings.Index(entry, " -> "); idx >= 0 {
				entry = entry[idx+4:]
			}
			if entry != "" {
				files = append(files, entry)
			}
		}
	}
	return files
}

// GetAllTrackedFiles returns all files tracked by git.
func GetAllTrackedFiles(repoRoot string) []string {
	out, err := runGit(repoRoot, "ls-files")
	if err != nil {
		return nil
	}
	return splitLines(out)
}

func splitLines(s string) []string {
	var lines []string
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}
