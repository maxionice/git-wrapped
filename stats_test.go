package main

import (
	"testing"
	"time"
)

func TestExtensionOf(t *testing.T) {
	cases := map[string]string{
		"main.go":         "go",
		"src/App.TSX":     "tsx",
		"path/to/readme":  "",
		".gitignore":      "",
		"archive.tar.gz":  "gz",
		"dir/.env":        "",
		"Makefile":        "",
		"weird.name.JSON": "json",
	}
	for in, want := range cases {
		if got := extensionOf(in); got != want {
			t.Errorf("extensionOf(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCommaInt(t *testing.T) {
	cases := map[int]string{
		0: "0", 42: "42", 1000: "1,000", -1000: "-1,000",
		1234567: "1,234,567", 999: "999",
	}
	for in, want := range cases {
		if got := commaInt(in); got != want {
			t.Errorf("commaInt(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestTopN(t *testing.T) {
	m := map[string]int{"a": 3, "b": 5, "c": 5, "d": 1}
	got := topN(m, 2)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	// b and c tie at 5; alphabetical tiebreak puts b first.
	if got[0].Label != "b" || got[1].Label != "c" {
		t.Errorf("got %+v, want [b c]", got)
	}
}

func TestLongestStreak(t *testing.T) {
	hits := map[string]int{
		"2026-06-10": 1,
		"2026-06-11": 4,
		"2026-06-12": 2,
		"2026-06-15": 1, // gap, new run of 1
		"2026-01-01": 9,
	}
	n, start, end := longestStreak(hits)
	if n != 3 {
		t.Errorf("streak = %d, want 3", n)
	}
	if start.Format("2006-01-02") != "2026-06-10" || end.Format("2006-01-02") != "2026-06-12" {
		t.Errorf("bounds = %s..%s, want 06-10..06-12", start.Format("2006-01-02"), end.Format("2006-01-02"))
	}
}

func TestBusiestDay(t *testing.T) {
	counts := map[string]int{
		"2026-06-10": 2,
		"2026-06-11": 7,
		"2026-06-12": 7, // ties with 06-11; earlier date wins
	}
	day, n := busiestDay(counts)
	if n != 7 || day.Format("2006-01-02") != "2026-06-11" {
		t.Errorf("busiestDay = %s/%d, want 2026-06-11/7", day.Format("2006-01-02"), n)
	}
}

func TestComputeStatsBasic(t *testing.T) {
	mk := func(day, hour int, added, removed int, files ...string) Commit {
		return Commit{
			Date:    time.Date(2026, 6, day, hour, 0, 0, 0, time.Local),
			Author:  "Max",
			Email:   "max@example.com",
			Added:   added,
			Removed: removed,
			Files:   files,
		}
	}
	commits := []Commit{
		mk(1, 9, 10, 2, "a.go", "b.go"),
		mk(2, 9, 5, 1, "a.go"),
		mk(3, 14, 0, 3, "c.py"),
	}
	s := computeStats(commits, 2026, 5)
	if s.TotalCommits != 3 {
		t.Errorf("commits = %d, want 3", s.TotalCommits)
	}
	if s.TotalAdded != 15 || s.TotalRemoved != 6 {
		t.Errorf("added/removed = %d/%d, want 15/6", s.TotalAdded, s.TotalRemoved)
	}
	if s.FilesTouched != 3 {
		t.Errorf("filesTouched = %d, want 3", s.FilesTouched)
	}
	if s.ActiveDays != 3 {
		t.Errorf("activeDays = %d, want 3", s.ActiveDays)
	}
	if len(s.TopFiles) == 0 || s.TopFiles[0].Label != "a.go" || s.TopFiles[0].N != 2 {
		t.Errorf("topFiles[0] = %+v, want a.go x2", s.TopFiles)
	}
}

func TestParseIdentity(t *testing.T) {
	cases := []struct {
		raw, name, key string
	}{
		{"Jane Doe <jane@x.com>", "Jane Doe", "jane@x.com"},
		{"Jane Doe <JANE@X.com>", "Jane Doe", "jane@x.com"},
		{"  Bob  <bob@y.io> ", "Bob", "bob@y.io"},
		{"<solo@z.dev>", "solo@z.dev", "solo@z.dev"},
		{"NoEmail Name", "NoEmail Name", "noemail name"},
	}
	for _, c := range cases {
		name, key := parseIdentity(c.raw)
		if name != c.name || key != c.key {
			t.Errorf("parseIdentity(%q) = (%q,%q), want (%q,%q)", c.raw, name, key, c.name, c.key)
		}
	}
}

func TestCoAuthorAggregation(t *testing.T) {
	commits := []Commit{
		{Date: time.Date(2026, 6, 1, 9, 0, 0, 0, time.Local), Author: "Max",
			CoAuthors: []string{"Jane Doe <jane@x.com>", "Bob <bob@y.io>"}},
		{Date: time.Date(2026, 6, 2, 9, 0, 0, 0, time.Local), Author: "Max",
			// Same Jane (different case) plus a duplicate within one commit.
			CoAuthors: []string{"jane doe <JANE@X.com>", "jane doe <jane@x.com>"}},
	}
	s := computeStats(commits, 2026, 5)
	if len(s.TopCoAuthors) != 2 {
		t.Fatalf("co-authors = %+v, want 2 distinct", s.TopCoAuthors)
	}
	// Jane appears in both commits (deduped within commit 2) => 2; Bob => 1.
	if s.TopCoAuthors[0].N != 2 || s.TopCoAuthors[0].Label != "Jane Doe" {
		t.Errorf("top co-author = %+v, want Jane Doe x2", s.TopCoAuthors[0])
	}
	if s.TopCoAuthors[1].N != 1 || s.TopCoAuthors[1].Label != "Bob" {
		t.Errorf("second co-author = %+v, want Bob x1", s.TopCoAuthors[1])
	}
}

func TestComputeStatsEmpty(t *testing.T) {
	s := computeStats(nil, 2026, 5)
	if s.TotalCommits != 0 || s.FilesTouched != 0 {
		t.Errorf("empty stats not zero: %+v", s)
	}
}

func TestComputeStatsTopBounds(t *testing.T) {
	mk := func(files ...string) Commit {
		return Commit{Date: time.Date(2026, 6, 1, 9, 0, 0, 0, time.Local), Files: files}
	}
	commits := []Commit{mk("a.go"), mk("b.go"), mk("c.go"), mk("d.go")}

	if got := computeStats(commits, 2026, 2); len(got.TopFiles) != 2 {
		t.Errorf("top=2 gave %d files, want 2", len(got.TopFiles))
	}
	// top below 1 is clamped to 1, not treated as "unbounded" or zero.
	if got := computeStats(commits, 2026, 0); len(got.TopFiles) != 1 {
		t.Errorf("top=0 gave %d files, want clamped to 1", len(got.TopFiles))
	}
}
