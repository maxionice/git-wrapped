package main

import (
	"bufio"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Commit is a single commit with its file-change stats.
type Commit struct {
	Hash    string
	Date    time.Time
	Author  string
	Email   string
	Added   int
	Removed int
	Files   []string
}

// fieldSep is the byte we ask git to emit between pretty-format fields. Using a
// control byte avoids collisions with anything a human would put in a name or
// commit subject.
const fieldSep = "\x1f"

// gitInstalled reports whether a git binary is on PATH.
func gitInstalled() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

// insideRepo reports whether the current working directory is inside a git work tree.
func insideRepo() bool {
	out, err := exec.Command("git", "rev-parse", "--is-inside-work-tree").Output()
	return err == nil && strings.TrimSpace(string(out)) == "true"
}

// repoName returns the basename of the repository's top-level directory.
func repoName() string {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "this repo"
	}
	return filepath.Base(strings.TrimSpace(string(out)))
}

// currentUserEmail returns the configured git user.email (may be empty).
func currentUserEmail() string {
	out, _ := exec.Command("git", "config", "user.email").Output()
	return strings.TrimSpace(string(out))
}

// collectCommits runs `git log` and parses commits. When year != 0 the range is
// limited to that calendar year. When email is non-empty only commits by that
// author email are kept.
func collectCommits(year int, email string) ([]Commit, error) {
	format := strings.Join([]string{"%H", "%aI", "%aN", "%aE"}, fieldSep)
	args := []string{
		"log",
		"--no-merges",
		"--numstat",
		"--date=iso-strict",
		"--pretty=format:\x1e" + format, // \x1e (record sep) marks a commit header line
	}
	if year != 0 {
		args = append(args,
			"--since", fmt.Sprintf("%d-01-01T00:00:00", year),
			"--until", fmt.Sprintf("%d-01-01T00:00:00", year+1),
		)
	}

	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git log failed: %w", err)
	}

	var commits []Commit
	var cur *Commit
	flush := func() {
		if cur == nil {
			return
		}
		if email == "" || strings.EqualFold(cur.Email, email) {
			commits = append(commits, *cur)
		}
		cur = nil
	}

	sc := bufio.NewScanner(strings.NewReader(string(out)))
	sc.Buffer(make([]byte, 0, 1024*1024), 16*1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "\x1e") {
			flush()
			fields := strings.Split(strings.TrimPrefix(line, "\x1e"), fieldSep)
			if len(fields) < 4 {
				continue
			}
			t, perr := time.Parse(time.RFC3339, fields[1])
			if perr != nil {
				continue
			}
			cur = &Commit{
				Hash:   fields[0],
				Date:   t,
				Author: fields[2],
				Email:  fields[3],
			}
			continue
		}
		// numstat line: "<added>\t<removed>\t<path>" (binary files use "-")
		if cur == nil {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) != 3 {
			continue
		}
		added, _ := strconv.Atoi(parts[0]) // "-" -> 0
		removed, _ := strconv.Atoi(parts[1])
		cur.Added += added
		cur.Removed += removed
		cur.Files = append(cur.Files, parts[2])
	}
	flush()
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return commits, nil
}
