package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rdo34/blt/internal/app"
	"github.com/rdo34/blt/internal/model"
	"github.com/rdo34/blt/internal/store"
)

// runCLI parses CLI subcommands. Returns (handled, exitCode).
func runCLI(args []string) (bool, int) {
	if len(args) == 0 {
		return false, 0
	}
	switch args[0] {
	case "help", "-h", "--help":
		printHelp()
		return true, 0
	case "list":
		return true, cliList(args[1:])
	case "add":
		return true, cliAdd(args[1:])
	case "delete":
		return true, cliDelete(args[1:])
	case "complete":
		return true, cliComplete(args[1:])
	case "migrate":
		return true, cliMigrate(args[1:])
	case "schedule":
		return true, cliSchedule(args[1:])
	case "edit":
		return true, cliEdit(args[1:])
	default:
		// Not a CLI subcommand; fall back to TUI
		return false, 0
	}
}

func newAppWithContext(span, dateStr, dataDir string) (*app.App, error) {
	var st *store.FSStore
	var err error
	if dataDir != "" {
		st, err = store.NewFSStore(dataDir)
	} else {
		st, err = store.NewDefaultFSStore()
	}
	if err != nil {
		return nil, err
	}
	a := app.New(st)
	// period
	switch strings.ToLower(span) {
	case "week":
		_ = a.SetPeriod(model.PeriodWeek)
	case "month":
		_ = a.SetPeriod(model.PeriodMonth)
	default:
		_ = a.SetPeriod(model.PeriodDay)
	}
	// date
	if dateStr != "" {
		if d, err := time.Parse("2006-01-02", dateStr); err == nil {
			_ = a.JumpToDate(d)
		}
	} else {
		_ = a.LoadDay(time.Now())
	}
	return a, nil
}

func parseTypesCSV(s string) []model.BulletType {
	if s == "" {
		return nil
	}
	var out []model.BulletType
	for _, part := range strings.Split(s, ",") {
		switch strings.TrimSpace(strings.ToLower(part)) {
		case "task":
			out = append(out, model.Task)
		case "event":
			out = append(out, model.Event)
		case "note":
			out = append(out, model.Note)
		case "important":
			out = append(out, model.HighlightImportant)
		case "inspiration":
			out = append(out, model.HighlightInspiration)
		case "done", "completed":
			out = append(out, model.Done)
		case "migrated":
			out = append(out, model.Migrated)
		case "scheduled":
			out = append(out, model.Scheduled)
		}
	}
	return out
}

func parseTagsCSV(s string) []string {
	if s == "" {
		return nil
	}
	var out []string
	for _, p := range strings.Split(s, ",") {
		t := strings.TrimSpace(p)
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}

func cliList(args []string) int {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	span := fs.String("timespan", "day", "day|week|month")
	dateStr := fs.String("date", "", "YYYY-MM-DD (defaults to today)")
	types := fs.String("type", "", "comma-separated types")
	tags := fs.String("tags", "", "comma-separated tags")
	text := fs.String("text", "", "text filter")
	jsonOut := fs.Bool("json", false, "output JSON")
	dataDir := fs.String("data-dir", "", "override data directory")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	a, err := newAppWithContext(*span, *dateStr, *dataDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if *text != "" {
		a.SetTextFilter(*text)
	}
	if *types != "" {
		a.SetTypeFilter(parseTypesCSV(*types))
	}
	if *tags != "" {
		a.SetTagFilter(parseTagsCSV(*tags))
	}
	_ = a.Refresh()
	vis := a.Visible()
	if *jsonOut {
		// Emit JSON array of entries
		type J struct {
			Date string
			ID   string
			Type string
			Text string
			Tags []string
		}
		out := make([]J, 0, len(vis))
		for _, e := range vis {
			out = append(out, J{Date: e.Date.Format("2006-01-02"), ID: e.Item.ID, Type: string(e.Item.Type), Text: e.Item.Text, Tags: e.Item.Tags})
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(out)
		return 0
	}
	for _, e := range vis {
		// compute absolute index within the day (ignores filters)
		dayItems, _ := a.Store.LoadDay(e.Date)
		abs := 0
		for i := range dayItems {
			if dayItems[i].ID == e.Item.ID {
				abs = i
				break
			}
		}
		label := formatBullet(e.Item)
		if a.Period != model.PeriodDay {
			fmt.Printf("%s %d. %s\n", e.Date.Format("2006-01-02"), abs, label)
		} else {
			fmt.Printf("%d. %s\n", abs, label)
		}
	}
	return 0
}

func cliAdd(args []string) int {
	fs := flag.NewFlagSet("add", flag.ContinueOnError)
	dataDir := fs.String("data-dir", "", "override data directory")
	dateStr := fs.String("date", "", "YYYY-MM-DD (defaults to today)")
	typ := fs.String("type", "task", "task|event|note|important|inspiration")
	note := fs.String("note", "", "shortcut for --type note with given text")
	text := fs.String("text", "", "bullet text")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	t := strings.TrimSpace(*text)
	if *note != "" {
		t = *note
		*typ = "note"
	}
	if strings.TrimSpace(t) == "" {
		fmt.Fprintln(os.Stderr, "missing --text or --note")
		return 2
	}
	st, err := func() (*store.FSStore, error) {
		if *dataDir != "" {
			return store.NewFSStore(*dataDir)
		}
		return store.NewDefaultFSStore()
	}()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	d := time.Now()
	if *dateStr != "" {
		if parsed, perr := time.Parse("2006-01-02", *dateStr); perr == nil {
			d = parsed
		}
	}
	b := model.Bullet{Text: t, CreatedAt: time.Now()}
	switch strings.ToLower(*typ) {
	case "task":
		b.Type = model.Task
	case "event":
		b.Type = model.Event
	case "note":
		b.Type = model.Note
	case "important":
		b.Type = model.HighlightImportant
	case "inspiration":
		b.Type = model.HighlightInspiration
	default:
		fmt.Fprintln(os.Stderr, "invalid --type")
		return 2
	}
	if err := st.Append(d, b); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func cliDelete(args []string) int {
	fs := flag.NewFlagSet("delete", flag.ContinueOnError)
	dateStr := fs.String("date", "", "YYYY-MM-DD (required)")
	dataDir := fs.String("data-dir", "", "override data directory")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "missing index")
		return 2
	}
	if strings.TrimSpace(*dateStr) == "" {
		fmt.Fprintln(os.Stderr, "--date is required for delete")
		return 2
	}
	idx := parseIndex(fs.Arg(0))
	a, err := newAppWithContext("day", *dateStr, *dataDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := a.DeleteDayIndex(parseDate(*dateStr), idx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func cliComplete(args []string) int {
	fs := flag.NewFlagSet("complete", flag.ContinueOnError)
	dateStr := fs.String("date", "", "YYYY-MM-DD (required)")
	dataDir := fs.String("data-dir", "", "override data directory")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "missing index")
		return 2
	}
	if strings.TrimSpace(*dateStr) == "" {
		fmt.Fprintln(os.Stderr, "--date is required for complete")
		return 2
	}
	idx := parseIndex(fs.Arg(0))
	a, err := newAppWithContext("day", *dateStr, *dataDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := a.CompleteDayIndex(parseDate(*dateStr), idx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func cliMigrate(args []string) int {
	fs := flag.NewFlagSet("migrate", flag.ContinueOnError)
	dateStr := fs.String("date", "", "YYYY-MM-DD (required)")
	dataDir := fs.String("data-dir", "", "override data directory")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "missing index")
		return 2
	}
	if strings.TrimSpace(*dateStr) == "" {
		fmt.Fprintln(os.Stderr, "--date is required for migrate")
		return 2
	}
	idx := parseIndex(fs.Arg(0))
	a, err := newAppWithContext("day", *dateStr, *dataDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := a.MigrateDayIndex(parseDate(*dateStr), idx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func cliSchedule(args []string) int {
	fs := flag.NewFlagSet("schedule", flag.ContinueOnError)
	dateStr := fs.String("date", "", "YYYY-MM-DD (required)")
	to := fs.String("to", "", "YYYY-MM-DD target date (required)")
	dataDir := fs.String("data-dir", "", "override data directory")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "missing index")
		return 2
	}
	if *to == "" {
		fmt.Fprintln(os.Stderr, "--to is required")
		return 2
	}
	if strings.TrimSpace(*dateStr) == "" {
		fmt.Fprintln(os.Stderr, "--date is required for schedule")
		return 2
	}
	target, err := time.Parse("2006-01-02", *to)
	if err != nil {
		fmt.Fprintln(os.Stderr, "invalid --to date")
		return 2
	}
	idx := parseIndex(fs.Arg(0))
	a, err := newAppWithContext("day", *dateStr, *dataDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := a.ScheduleDayIndex(parseDate(*dateStr), idx, target); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func cliEdit(args []string) int {
	fs := flag.NewFlagSet("edit", flag.ContinueOnError)
	dateStr := fs.String("date", "", "YYYY-MM-DD (required)")
	newText := fs.String("set", "", "new text (required)")
	dataDir := fs.String("data-dir", "", "override data directory")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "missing index")
		return 2
	}
	if strings.TrimSpace(*newText) == "" {
		fmt.Fprintln(os.Stderr, "--set is required")
		return 2
	}
	if strings.TrimSpace(*dateStr) == "" {
		fmt.Fprintln(os.Stderr, "--date is required for edit")
		return 2
	}
	idx := parseIndex(fs.Arg(0))
	a, err := newAppWithContext("day", *dateStr, *dataDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := a.UpdateDayIndexText(parseDate(*dateStr), idx, *newText); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func parseIndex(s string) int {
	// best-effort parse; on failure, return -1 which will no-op in app methods
	var i int
	_, _ = fmt.Sscanf(s, "%d", &i)
	return i
}

func parseDate(s string) time.Time {
	d, _ := time.Parse("2006-01-02", s)
	return d
}

func formatBullet(b model.Bullet) string {
	prefix := "•"
	switch b.Type {
	case model.Done:
		prefix = "✓"
	case model.Migrated:
		prefix = "»"
	case model.Scheduled:
		prefix = "⧗"
	case model.Event:
		prefix = "◇"
	case model.Note:
		prefix = "–"
	case model.HighlightImportant:
		prefix = "!"
	case model.HighlightInspiration:
		prefix = "★"
	}
	tagStr := ""
	if len(b.Tags) > 0 {
		parts := make([]string, 0, len(b.Tags))
		for _, t := range b.Tags {
			t = strings.TrimSpace(t)
			if t == "" {
				continue
			}
			if strings.HasPrefix(t, "#") {
				parts = append(parts, t)
			} else {
				parts = append(parts, "#"+t)
			}
		}
		if len(parts) > 0 {
			tagStr = "  [" + strings.Join(parts, " ") + "]"
		}
	}
	return prefix + " " + b.Text + tagStr
}

func printHelp() {
	fmt.Println("blt CLI")
	fmt.Println("\nUsage:")
	fmt.Println("  blt list [--timespan day|week|month] [--date YYYY-MM-DD] [--type ...] [--tags ...] [--text ...] [--json]")
	fmt.Println("  blt add [--type task|event|note|important|inspiration] --text \"...\" | --note \"...\"")
	fmt.Println("  blt delete <index> --date YYYY-MM-DD [--data-dir PATH]")
	fmt.Println("  blt complete <index> --date YYYY-MM-DD [--data-dir PATH]")
	fmt.Println("  blt migrate  <index> --date YYYY-MM-DD [--data-dir PATH]")
	fmt.Println("  blt schedule <index> --date YYYY-MM-DD --to YYYY-MM-DD [--data-dir PATH]")
	fmt.Println("  blt edit     <index> --date YYYY-MM-DD --set \"new text\" [--data-dir PATH]")
	fmt.Println("\nContext flags:")
	fmt.Println("  --timespan day|week|month   --date YYYY-MM-DD   --type t1,t2   --tags tag1,tag2   --text query   --data-dir path")
	fmt.Println("\nNotes:")
	fmt.Println("  Indexes shown by 'list' are absolute per day and unaffected by filters.")
}
