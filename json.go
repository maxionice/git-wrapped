package main

import (
	"encoding/json"
)

// jsonReport is the machine-readable view of a Stats summary. Field names are
// stable and snake_case so the output is convenient to consume from scripts.
type jsonReport struct {
	Repo         string `json:"repo"`
	Year         *int   `json:"year"` // null for all-time
	Author       string `json:"author"`
	TotalCommits int    `json:"total_commits"`
	LinesAdded   int    `json:"lines_added"`
	LinesRemoved int    `json:"lines_removed"`
	NetLines     int    `json:"net_lines"`
	FilesTouched int    `json:"files_touched"`
	ActiveDays   int    `json:"active_days"`

	FirstCommit string `json:"first_commit,omitempty"` // YYYY-MM-DD
	LastCommit  string `json:"last_commit,omitempty"`

	LongestStreak int    `json:"longest_streak_days"`
	StreakStart   string `json:"streak_start,omitempty"`
	StreakEnd     string `json:"streak_end,omitempty"`

	BusiestDay      string `json:"busiest_day,omitempty"`
	BusiestDayCount int    `json:"busiest_day_commits"`

	PeakHour    int `json:"peak_hour"`
	PeakWeekday int `json:"peak_weekday"` // 0=Sunday

	ByWeekday [7]int  `json:"by_weekday"`
	ByHour    [24]int `json:"by_hour"`
	ByMonth   [12]int `json:"by_month"`

	TopFiles      []Count `json:"top_files"`
	TopExtensions []Count `json:"top_extensions"`
	TopCoAuthors  []Count `json:"top_co_authors"`

	CommitsPerActiveDay float64 `json:"commits_per_active_day"`
	WeekendShare        float64 `json:"weekend_share"` // fraction of commits on Sat/Sun
}

// toJSONReport projects Stats into the stable, serialization-friendly shape.
func toJSONReport(s Stats, repo string) jsonReport {
	r := jsonReport{
		Repo:            repo,
		Author:          s.Author,
		TotalCommits:    s.TotalCommits,
		LinesAdded:      s.TotalAdded,
		LinesRemoved:    s.TotalRemoved,
		NetLines:        s.TotalAdded - s.TotalRemoved,
		FilesTouched:    s.FilesTouched,
		ActiveDays:      s.ActiveDays,
		LongestStreak:   s.LongestStreak,
		BusiestDayCount: s.BusiestDayCount,
		ByWeekday:       s.ByWeekday,
		ByHour:          s.ByHour,
		ByMonth:         s.ByMonth,
		TopFiles:        s.TopFiles,
		TopExtensions:   s.TopExtensions,
		TopCoAuthors:    s.TopCoAuthors,
	}
	if s.Year != 0 {
		y := s.Year
		r.Year = &y
	}
	if !s.First.IsZero() {
		r.FirstCommit = s.First.Format("2006-01-02")
		r.LastCommit = s.Last.Format("2006-01-02")
	}
	if s.LongestStreak > 0 {
		r.StreakStart = s.StreakStart.Format("2006-01-02")
		r.StreakEnd = s.StreakEnd.Format("2006-01-02")
	}
	if s.BusiestDayCount > 0 {
		r.BusiestDay = s.BusiestDay.Format("2006-01-02")
	}
	r.PeakHour, _ = argmax(s.ByHour[:])
	r.PeakWeekday, _ = argmax(s.ByWeekday[:])

	if s.ActiveDays > 0 {
		r.CommitsPerActiveDay = round2(float64(s.TotalCommits) / float64(s.ActiveDays))
	}
	if s.TotalCommits > 0 {
		weekend := s.ByWeekday[0] + s.ByWeekday[6]
		r.WeekendShare = round2(float64(weekend) / float64(s.TotalCommits))
	}
	return r
}

// round2 rounds to two decimal places without pulling in math just for this.
func round2(f float64) float64 {
	return float64(int(f*100+0.5)) / 100
}

// renderJSON returns the indented JSON encoding of the report.
func renderJSON(s Stats, repo string) (string, error) {
	out, err := json.MarshalIndent(toJSONReport(s, repo), "", "  ")
	if err != nil {
		return "", err
	}
	return string(out) + "\n", nil
}
