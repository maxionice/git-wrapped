package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"time"
)

func main() {
	var (
		year    = flag.Int("year", time.Now().Year(), "calendar year to summarize (0 = all time)")
		all     = flag.Bool("all", false, "include every author (default: only your commits)")
		author  = flag.String("author", "", "summarize a specific author by email (overrides the default)")
		top     = flag.Int("top", 5, "number of entries to show in ranked lists")
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

	email, err := resolveFilter(*all, *author, currentUserEmail())
	if err != nil {
		fatal(err.Error())
	}

	commits, err := collectCommits(*year, email)
	if err != nil {
		fatal(err.Error())
	}

	stats := computeStats(commits, *year, *top)
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

// resolveFilter decides which author-email filter to apply from the flags. An
// empty result means "all authors". currentEmail is consulted only when neither
// --all nor --author is given.
func resolveFilter(all bool, author, currentEmail string) (string, error) {
	switch {
	case all && author != "":
		return "", errors.New("--all and --author are mutually exclusive")
	case all:
		return "", nil
	case author != "":
		return author, nil
	case currentEmail != "":
		return currentEmail, nil
	default:
		return "", errors.New("git user.email is not set; pass --all or --author <email>")
	}
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
