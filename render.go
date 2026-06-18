package main

import (
	"fmt"
	"strings"
	"time"
)

// palette holds the ANSI escape codes used for output. When color is disabled
// every field is the empty string, so format strings render plainly.
type palette struct {
	reset, bold, dim          string
	green, red, cyan, magenta string
	yellow                    string
}

func newPalette(color bool) palette {
	if !color {
		return palette{}
	}
	return palette{
		reset:   "\x1b[0m",
		bold:    "\x1b[1m",
		dim:     "\x1b[2m",
		green:   "\x1b[32m",
		red:     "\x1b[31m",
		cyan:    "\x1b[36m",
		magenta: "\x1b[35m",
		yellow:  "\x1b[33m",
	}
}

var weekdayNames = [7]string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
var monthNames = [12]string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}

// render writes the full wrapped report for s to a string.
func render(s Stats, repo string, p palette) string {
	var b strings.Builder

	scope := "in " + s.Year2()
	title := fmt.Sprintf("  Git Wrapped — %s %s", repo, scope)
	b.WriteString(p.magenta + p.bold + strings.Repeat("─", len(title)+2) + p.reset + "\n")
	b.WriteString(p.magenta + p.bold + title + p.reset + "\n")
	if s.Author != "" {
		b.WriteString(p.dim + "  for " + s.Author + p.reset + "\n")
	}
	b.WriteString(p.magenta + p.bold + strings.Repeat("─", len(title)+2) + p.reset + "\n\n")

	if s.TotalCommits == 0 {
		b.WriteString(p.dim + "  No commits found for this range. Try --year or --all.\n" + p.reset)
		return b.String()
	}

	// Headline numbers.
	line := func(label, val string) {
		b.WriteString(fmt.Sprintf("  %s%-18s%s %s\n", p.dim, label, p.reset, val))
	}
	line("Commits", p.bold+commaInt(s.TotalCommits)+p.reset)
	line("Lines added", p.green+"+"+commaInt(s.TotalAdded)+p.reset)
	line("Lines removed", p.red+"-"+commaInt(s.TotalRemoved)+p.reset)
	line("Net lines", signedInt(s.TotalAdded-s.TotalRemoved, p))
	line("Files touched", commaInt(s.FilesTouched))
	line("Active days", fmt.Sprintf("%d", s.ActiveDays))
	if s.ActiveDays > 0 {
		line("Avg commits/day", fmt.Sprintf("%.1f", float64(s.TotalCommits)/float64(s.ActiveDays)))
	}
	if s.TotalCommits > 0 {
		weekend := 100 * (s.ByWeekday[0] + s.ByWeekday[6]) / s.TotalCommits
		line("Weekend commits", fmt.Sprintf("%d%%", weekend))
	}
	if !s.First.IsZero() {
		span := fmt.Sprintf("%s → %s", s.First.Format("Jan 2"), s.Last.Format("Jan 2"))
		line("First → last", span)
	}
	if s.LongestStreak > 1 {
		streak := fmt.Sprintf("%s%d days%s  %s(%s → %s)%s",
			p.yellow, s.LongestStreak, p.reset,
			p.dim, s.StreakStart.Format("Jan 2"), s.StreakEnd.Format("Jan 2"), p.reset)
		line("Longest streak", streak)
	}
	if s.BusiestDayCount > 0 {
		busiest := fmt.Sprintf("%s%d %s%s %son %s%s",
			p.yellow, s.BusiestDayCount, plural(s.BusiestDayCount, "commit"), p.reset,
			p.dim, s.BusiestDay.Format("Mon, Jan 2"), p.reset)
		line("Busiest day", busiest)
	}
	b.WriteString("\n")

	// When are you coding?
	b.WriteString(p.cyan + p.bold + "  By weekday\n" + p.reset)
	wkLabels := make([]string, 7)
	wkVals := make([]int, 7)
	for i := 0; i < 7; i++ {
		wkLabels[i] = weekdayNames[i]
		wkVals[i] = s.ByWeekday[i]
	}
	writeBars(&b, wkLabels, wkVals, p, p.cyan)
	b.WriteString("\n")

	b.WriteString(p.cyan + p.bold + "  By month\n" + p.reset)
	mLabels := make([]string, 12)
	mVals := make([]int, 12)
	for i := 0; i < 12; i++ {
		mLabels[i] = monthNames[i]
		mVals[i] = s.ByMonth[i]
	}
	writeBars(&b, mLabels, mVals, p, p.cyan)
	b.WriteString("\n")

	// Hour-of-day sparkline (00..23).
	b.WriteString(p.cyan + p.bold + "  By hour\n" + p.reset)
	b.WriteString("    " + p.cyan + sparkline(s.ByHour[:]) + p.reset + "\n")
	b.WriteString("    " + p.dim + "0  3  6  9  12 15 18 21 " + p.reset + "\n\n")

	// Peak hour & day callouts.
	peakHour, _ := argmax(s.ByHour[:])
	peakDay, _ := argmax(s.ByWeekday[:])
	b.WriteString(fmt.Sprintf("  %sPeak time%s   %02d:00–%02d:00 on %ss\n\n",
		p.dim, p.reset, peakHour, (peakHour+1)%24, fullWeekday(peakDay)))

	// Contribution heatmap (only for a single calendar year — all-time would be
	// unreadably wide).
	if s.Year != 0 && len(s.DayCounts) > 0 {
		b.WriteString(p.cyan + p.bold + "  Contribution heatmap\n" + p.reset)
		b.WriteString(heatmap(s.Year, s.DayCounts, p))
		b.WriteString("\n")
	}

	// Top files & extensions.
	if len(s.TopFiles) > 0 {
		b.WriteString(p.cyan + p.bold + "  Most-changed files\n" + p.reset)
		for _, c := range s.TopFiles {
			b.WriteString(fmt.Sprintf("    %s%3d×%s  %s\n", p.yellow, c.N, p.reset, c.Label))
		}
		b.WriteString("\n")
	}
	if len(s.TopExtensions) > 0 {
		var parts []string
		for _, c := range s.TopExtensions {
			parts = append(parts, fmt.Sprintf("%s.%s%s %d", p.bold, c.Label, p.reset, c.N))
		}
		b.WriteString(p.dim + "  Top file types  " + p.reset + strings.Join(parts, "   ") + "\n")
	}

	if len(s.TopCoAuthors) > 0 {
		b.WriteString("\n" + p.cyan + p.bold + "  Top collaborators\n" + p.reset)
		for _, c := range s.TopCoAuthors {
			b.WriteString(fmt.Sprintf("    %s%3d×%s  %s\n", p.yellow, c.N, p.reset, c.Label))
		}
	}

	return b.String()
}

// writeBars renders a labelled horizontal bar chart scaled to the max value.
func writeBars(b *strings.Builder, labels []string, vals []int, p palette, color string) {
	const width = 32
	max := 0
	for _, v := range vals {
		if v > max {
			max = v
		}
	}
	if max == 0 {
		max = 1
	}
	for i, v := range vals {
		n := v * width / max
		if v > 0 && n == 0 {
			n = 1
		}
		bar := strings.Repeat("█", n)
		b.WriteString(fmt.Sprintf("    %s%-4s%s %s%s%s %s%d%s\n",
			p.dim, labels[i], p.reset, color, bar, p.reset, p.dim, v, p.reset))
	}
}

// sparkline renders a slice of counts as a single row of block characters,
// scaled so the largest value uses the tallest block.
func sparkline(vals []int) string {
	blocks := []rune(" ▁▂▃▄▅▆▇█")
	max := 0
	for _, v := range vals {
		if v > max {
			max = v
		}
	}
	out := make([]rune, len(vals))
	for i, v := range vals {
		idx := 0
		if max > 0 {
			idx = v * (len(blocks) - 1) / max
			if v > 0 && idx == 0 {
				idx = 1
			}
		}
		out[i] = blocks[idx]
	}
	return string(out)
}

// heatmap renders a GitHub-style contribution grid for the given year: seven
// rows (Sun..Sat) by up to 53 week columns, each cell shaded by commit count.
func heatmap(year int, dayCounts map[string]int, p palette) string {
	shades := []string{"·", "░", "▒", "▓", "█"}
	colors := []string{p.dim, p.dim, p.green, p.green, p.green}

	// Find the max count to scale shade levels.
	max := 0
	for _, n := range dayCounts {
		if n > max {
			max = n
		}
	}
	level := func(n int) int {
		if n == 0 || max == 0 {
			return 0
		}
		l := 1 + n*(len(shades)-2)/max
		if l > len(shades)-1 {
			l = len(shades) - 1
		}
		return l
	}

	// Grid start: the Sunday on or before Jan 1.
	jan1 := time.Date(year, time.January, 1, 0, 0, 0, 0, time.Local)
	start := jan1.AddDate(0, 0, -int(jan1.Weekday()))
	end := time.Date(year, time.December, 31, 0, 0, 0, 0, time.Local)

	type cell struct {
		in  bool
		lvl int
	}
	var weeks [][7]cell
	var col [7]cell
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		wd := int(d.Weekday())
		if d.Year() == year {
			col[wd] = cell{in: true, lvl: level(dayCounts[d.Format("2006-01-02")])}
		}
		if wd == 6 {
			weeks = append(weeks, col)
			col = [7]cell{}
		}
	}
	if col != ([7]cell{}) {
		weeks = append(weeks, col)
	}

	var b strings.Builder
	rowLabels := [7]string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
	for wd := 0; wd < 7; wd++ {
		b.WriteString("    " + p.dim + rowLabels[wd] + p.reset + " ")
		for _, wk := range weeks {
			c := wk[wd]
			if !c.in {
				b.WriteString(" ")
				continue
			}
			b.WriteString(colors[c.lvl] + shades[c.lvl] + p.reset)
		}
		b.WriteString("\n")
	}
	b.WriteString("    " + p.dim + "less " + strings.Join(shades, "") + " more" + p.reset + "\n")
	return b.String()
}

// argmax returns the index and value of the largest element. For an empty slice
// it returns (0, 0).
func argmax(vals []int) (int, int) {
	bi, bv := 0, 0
	for i, v := range vals {
		if v > bv {
			bi, bv = i, v
		}
	}
	return bi, bv
}

func fullWeekday(i int) string {
	return time.Weekday(i).String()
}

func signedInt(n int, p palette) string {
	if n >= 0 {
		return p.green + "+" + commaInt(n) + p.reset
	}
	return p.red + "-" + commaInt(-n) + p.reset
}

// plural returns word, suffixed with "s" unless n == 1.
func plural(n int, word string) string {
	if n == 1 {
		return word
	}
	return word + "s"
}

// commaInt formats an integer with thousands separators.
func commaInt(n int) string {
	s := fmt.Sprintf("%d", n)
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	var out []byte
	for i, c := range []byte(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, c)
	}
	if neg {
		return "-" + string(out)
	}
	return string(out)
}

// Year2 renders the year for the title, or "all time" when unset.
func (s Stats) Year2() string {
	if s.Year == 0 {
		return "all time"
	}
	return fmt.Sprintf("%d", s.Year)
}
