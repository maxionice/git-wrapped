# git-wrapped

A year-in-review for your git repository — like Spotify Wrapped, but for commits.
Run it inside any repo and get a colorful terminal summary of your coding year.

## Build

```sh
go build -o git-wrapped .
```

## Usage

```sh
git-wrapped                 # your commits in the current year
git-wrapped --year 2025     # a specific year
git-wrapped --year 0        # all time
git-wrapped --all                # every author, not just you
git-wrapped --author jane@x.com  # a specific author by email
git-wrapped --no-color           # plain output (also honors NO_COLOR)
git-wrapped --json               # machine-readable JSON instead of the report
```

The `--json` output is stable, snake_case, and pipe-friendly:

```sh
git-wrapped --year 2025 --json | jq '.weekend_share, .top_files[0].label'
```

By default it filters to your commits using `git config user.email`. Pass
`--all` to summarize the whole team, or `--author <email>` to focus on someone
specific. `--all` and `--author` are mutually exclusive.

## What it shows

- Total commits, lines added/removed, net lines, files touched, active days
- First → last commit span, longest daily commit streak, and busiest single day
- Commit distribution by weekday and by month (bar charts)
- Hour-of-day sparkline and peak coding time (hour + weekday)
- A GitHub-style contribution heatmap for the year
- Most-changed files and top file types

## Tests

```sh
go test ./...
```
