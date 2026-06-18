package main

import (
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Stats is the aggregated "wrapped" summary computed from a set of commits.
type Stats struct {
	Year         int
	Author       string // display name of the dominant author (or "everyone")
	TotalCommits int
	TotalAdded   int
	TotalRemoved int
	FilesTouched int // unique file paths touched across all commits

	First time.Time
	Last  time.Time

	ByWeekday [7]int  // commits per weekday, Sunday=0
	ByHour    [24]int // commits per hour of day
	ByMonth   [12]int // commits per calendar month, Jan=0

	TopFiles      []Count // most frequently changed files
	TopExtensions []Count // most common file extensions
	TopCoAuthors  []Count // most frequent Co-authored-by collaborators

	LongestStreak int       // longest run of consecutive days with >=1 commit
	StreakStart   time.Time // first day of that streak
	StreakEnd     time.Time // last day of that streak
	ActiveDays    int       // distinct calendar days with >=1 commit

	BusiestDay      time.Time // calendar day with the most commits
	BusiestDayCount int       // commits on that day

	DayCounts map[string]int // "2006-01-02" -> commit count, for the heatmap
}

// Count is a label with an associated tally, used for ranked lists.
type Count struct {
	Label string `json:"label"`
	N     int    `json:"count"`
}

// computeStats folds a slice of commits into a Stats summary. The commits need
// not be sorted; computeStats orders what it needs internally.
func computeStats(commits []Commit, year int) Stats {
	s := Stats{Year: year}
	if len(commits) == 0 {
		return s
	}

	uniqueFiles := map[string]bool{}
	fileCounts := map[string]int{}
	extCounts := map[string]int{}
	authorCounts := map[string]int{}
	coAuthorCounts := map[string]int{} // display name -> commits collaborated on
	coAuthorKeyToName := map[string]string{}
	dayCounts := map[string]int{} // "2006-01-02" -> commit count

	for _, c := range commits {
		s.TotalCommits++
		s.TotalAdded += c.Added
		s.TotalRemoved += c.Removed

		local := c.Date.Local()
		s.ByWeekday[int(local.Weekday())]++
		s.ByHour[local.Hour()]++
		s.ByMonth[int(local.Month())-1]++
		dayCounts[local.Format("2006-01-02")]++

		if s.First.IsZero() || local.Before(s.First) {
			s.First = local
		}
		if local.After(s.Last) {
			s.Last = local
		}

		authorCounts[c.Author]++

		// Count each distinct co-author at most once per commit, keyed by a
		// normalized identity so "Jane <j@x>" and "jane <J@X>" coalesce.
		seenCo := map[string]bool{}
		for _, raw := range c.CoAuthors {
			name, key := parseIdentity(raw)
			if seenCo[key] {
				continue
			}
			seenCo[key] = true
			coAuthorCounts[key]++
			if _, ok := coAuthorKeyToName[key]; !ok {
				coAuthorKeyToName[key] = name
			}
		}

		for _, f := range c.Files {
			uniqueFiles[f] = true
			fileCounts[f]++
			if ext := extensionOf(f); ext != "" {
				extCounts[ext]++
			}
		}
	}

	s.FilesTouched = len(uniqueFiles)
	s.ActiveDays = len(dayCounts)
	s.TopFiles = topN(fileCounts, 5)
	s.TopExtensions = topN(extCounts, 5)
	s.TopCoAuthors = topNNamed(coAuthorCounts, coAuthorKeyToName, 5)
	s.Author = topLabel(authorCounts)
	s.LongestStreak, s.StreakStart, s.StreakEnd = longestStreak(dayCounts)
	s.DayCounts = dayCounts
	s.BusiestDay, s.BusiestDayCount = busiestDay(dayCounts)

	return s
}

// busiestDay returns the calendar day with the most commits, ties broken toward
// the earlier date for stable output.
func busiestDay(dayCounts map[string]int) (time.Time, int) {
	bestDay, bestN := "", 0
	for d, n := range dayCounts {
		if n > bestN || (n == bestN && d < bestDay) {
			bestDay, bestN = d, n
		}
	}
	t, _ := time.Parse("2006-01-02", bestDay)
	return t, bestN
}

// extensionOf returns the lowercased file extension without the dot, or "" when
// the path has none. Dotfiles like ".gitignore" are treated as having no ext.
func extensionOf(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	if ext == "" || ext == base {
		return ""
	}
	return strings.ToLower(strings.TrimPrefix(ext, "."))
}

// topN returns the highest-count entries, ties broken alphabetically for stable
// output. At most n entries are returned.
func topN(m map[string]int, n int) []Count {
	out := make([]Count, 0, len(m))
	for k, v := range m {
		out = append(out, Count{Label: k, N: v})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].N != out[j].N {
			return out[i].N > out[j].N
		}
		return out[i].Label < out[j].Label
	})
	if len(out) > n {
		out = out[:n]
	}
	return out
}

// parseIdentity splits a "Name <email>" string into a display name and a
// normalized identity key. The key prefers the lowercased email; when no email
// is present it falls back to the lowercased name, so the same person counts
// once regardless of case.
func parseIdentity(raw string) (name, key string) {
	raw = strings.TrimSpace(raw)
	if lt := strings.LastIndex(raw, "<"); lt >= 0 {
		if gt := strings.Index(raw[lt:], ">"); gt >= 0 {
			email := strings.TrimSpace(raw[lt+1 : lt+gt])
			name = strings.TrimSpace(raw[:lt])
			if name == "" {
				name = email
			}
			if email != "" {
				return name, strings.ToLower(email)
			}
			return name, strings.ToLower(name)
		}
	}
	return raw, strings.ToLower(raw)
}

// topNNamed ranks counts keyed by identity, resolving each key to its display
// name via names. Ties break alphabetically by display name for stable output.
func topNNamed(counts map[string]int, names map[string]string, n int) []Count {
	out := make([]Count, 0, len(counts))
	for k, v := range counts {
		out = append(out, Count{Label: names[k], N: v})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].N != out[j].N {
			return out[i].N > out[j].N
		}
		return out[i].Label < out[j].Label
	})
	if len(out) > n {
		out = out[:n]
	}
	return out
}

// topLabel returns the single highest-count key, or "" for an empty map.
func topLabel(m map[string]int) string {
	best, bestN := "", -1
	for k, v := range m {
		if v > bestN || (v == bestN && k < best) {
			best, bestN = k, v
		}
	}
	return best
}

// longestStreak finds the longest run of consecutive calendar days present in
// the set, returning its length and inclusive bounds.
func longestStreak(dayCounts map[string]int) (int, time.Time, time.Time) {
	if len(dayCounts) == 0 {
		return 0, time.Time{}, time.Time{}
	}
	days := make([]time.Time, 0, len(dayCounts))
	for d := range dayCounts {
		t, err := time.Parse("2006-01-02", d)
		if err == nil {
			days = append(days, t)
		}
	}
	sort.Slice(days, func(i, j int) bool { return days[i].Before(days[j]) })

	bestLen, curLen := 1, 1
	bestStart, bestEnd := days[0], days[0]
	curStart := days[0]
	for i := 1; i < len(days); i++ {
		if days[i].Sub(days[i-1]) == 24*time.Hour {
			curLen++
		} else {
			curLen = 1
			curStart = days[i]
		}
		if curLen > bestLen {
			bestLen, bestStart, bestEnd = curLen, curStart, days[i]
		}
	}
	return bestLen, bestStart, bestEnd
}
