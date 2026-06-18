package main

import (
	"flag"
	"fmt"
	"os"
	"time"
)

func main() {
	var (
		year    = flag.Int("year", time.Now().Year(), "calendar year to summarize (0 = all time)")
		all     = flag.Bool("all", false, "include every author (default: only your commits)")
		noColor = flag.Bool("no-color", false, "disable ANSI colors")
		asJSON  = flag.Bool("json", false, "emit machine-readable JSON instead of the report")
	)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "git-wrapped — a year-in-review for your git repository.\n\n")
		fmt.Fprintf(os.Stderr, "Usage: git-wrapped [flags]\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if !gitInstalled() {
		fatal("git is not installed or not on your PATH.")
	}
	if !insideRepo() {
		fatal("not inside a git repository — cd into one and try again.")
	}

	email := ""
	if !*all {
		email = currentUserEmail()
		if email == "" {
			fatal("git user.email is not set; pass --all to summarize every author.")
		}
	}

	commits, err := collectCommits(*year, email)
	if err != nil {
		fatal(err.Error())
	}

	stats := computeStats(commits, *year)
	if *all {
		stats.Author = "everyone"
	}

	if *asJSON {
		out, err := renderJSON(stats, repoName())
		if err != nil {
			fatal(err.Error())
		}
		fmt.Print(out)
		return
	}

	p := newPalette(!*noColor && colorEnabled())
	fmt.Print(render(stats, repoName(), p))
}

func fatal(msg string) {
	fmt.Fprintln(os.Stderr, "git-wrapped: "+msg)
	os.Exit(1)
}

// colorEnabled reports whether ANSI color should be used, honoring the NO_COLOR
// convention and skipping when stdout is not a terminal.
func colorEnabled() bool {
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		return false
	}
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}
