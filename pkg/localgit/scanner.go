package localgit

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Commit struct {
	SHA            string
	Message        string
	Branch         string
	RepositoryName string
	Author         string
	CommittedAt    time.Time
}

type Scanner interface {
	IsGitRepository(path string) bool
	ScanCommits(repoPath string, since time.Time) ([]Commit, error)
	RepositoryName(repoPath string) (string, error)
}

type scanner struct{}

func NewScanner() Scanner {
	return &scanner{}
}

func (s *scanner) IsGitRepository(path string) bool {
	gitDir := filepath.Join(path, ".git")
	info, err := os.Stat(gitDir)
	return err == nil && info.IsDir()
}

func (s *scanner) RepositoryName(repoPath string) (string, error) {
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return "", err
	}
	return filepath.Base(absPath), nil
}

func (s *scanner) ScanCommits(repoPath string, since time.Time) ([]Commit, error) {
	if !s.IsGitRepository(repoPath) {
		return nil, fmt.Errorf("path is not a git repository: %s", repoPath)
	}

	sinceArg := since.UTC().Format(time.RFC3339)
	output, err := runGit(repoPath, "log",
		fmt.Sprintf("--since=%s", sinceArg),
		"--pretty=format:%H\x01%s\x01%an\x01%aI\x01%D",
	)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(output) == "" {
		return nil, nil
	}

	repoName, err := s.RepositoryName(repoPath)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	commits := make([]Commit, 0, len(lines))
	for _, line := range lines {
		parts := strings.Split(line, "\x01")
		if len(parts) < 4 {
			continue
		}

		committedAt, err := time.Parse(time.RFC3339, parts[3])
		if err != nil {
			continue
		}

		branch := parseBranchFromDecorations(parts)
		commits = append(commits, Commit{
			SHA:         parts[0],
			Message:     parts[1],
			Author:      parts[2],
			CommittedAt: committedAt.UTC(),
			Branch:      branch,
			RepositoryName: repoName,
		})
	}

	return commits, nil
}

func parseBranchFromDecorations(parts []string) string {
	if len(parts) < 5 {
		return "unknown"
	}
	decorations := strings.TrimSpace(parts[4])
	if decorations == "" {
		return "unknown"
	}

	for _, segment := range strings.Split(decorations, ", ") {
		segment = strings.TrimSpace(segment)
		if strings.HasPrefix(segment, "HEAD -> ") {
			return strings.TrimPrefix(segment, "HEAD -> ")
		}
		if !strings.HasPrefix(segment, "tag:") && !strings.HasPrefix(segment, "origin/") {
			return segment
		}
	}

	for _, segment := range strings.Split(decorations, ", ") {
		segment = strings.TrimSpace(segment)
		if strings.HasPrefix(segment, "origin/") {
			return strings.TrimPrefix(segment, "origin/")
		}
	}

	return "unknown"
}

func runGit(repoPath string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = repoPath
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", err
	}
	return string(out), nil
}
