package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"text/tabwriter"

	"imv/internal/appcore"
	"imv/internal/store"
)

const defaultDBPath = appcore.DefaultDBPath

const (
	scanUsage   = "imv scan <folder> [--db " + defaultDBPath + "] [--workers 4] [--rescan] [--fail-on-error] [--json]"
	showUsage   = "imv show <path-or-id> [--db " + defaultDBPath + "] [--raw] [--json]"
	searchUsage = "imv search [--db " + defaultDBPath + "] [--tag <tag>] [--source nai|comfyui] [--format png|webp] [--has-workflow] [--q <text>] [--limit 50] [--long] [--json]"
	tagsUsage   = "imv tags [--db " + defaultDBPath + "] [--source nai|comfyui|generic|unknown] [--q <text>] [--limit 100] [--json]"
	statsUsage  = "imv stats [--db " + defaultDBPath + "] [--json]"
	exportUsage = "imv export --out <file.json> [--db " + defaultDBPath + "] [--pretty]"
	moveUsage   = "imv move --tag <tag> --to <folder> [--db " + defaultDBPath + "] [--apply] [--conflict skip|rename] [--json]"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := runWithIOContext(ctx, os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	return runWithIO(args, os.Stdout, os.Stderr)
}

func runWithIO(args []string, stdout, stderr io.Writer) error {
	return runWithIOContext(context.Background(), args, stdout, stderr)
}

func runWithIOContext(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		printUsage(stdout)
		return nil
	}

	switch args[0] {
	case "scan":
		return runScan(ctx, args[1:], stdout, stderr)
	case "show":
		return runShow(ctx, args[1:], stdout, stderr)
	case "search":
		return runSearch(ctx, args[1:], stdout, stderr)
	case "tags":
		return runTags(ctx, args[1:], stdout, stderr)
	case "stats":
		return runStats(ctx, args[1:], stdout, stderr)
	case "export":
		return runExport(ctx, args[1:], stdout, stderr)
	case "move":
		return runMove(ctx, args[1:], stdout, stderr)
	case "help":
		return runHelp(ctx, args[1:], stdout, stderr)
	case "-h", "--help":
		printUsage(stdout)
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runScan(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fs := newFlagSet("scan", stderr)
	dbPath := fs.String("db", defaultDBPath, "SQLite database path")
	workers := fs.Int("workers", 4, "number of scanner workers")
	rescan := fs.Bool("rescan", false, "rescan unchanged files")
	failOnError := fs.Bool("fail-on-error", false, "return an error when any file fails to scan")
	asJSON := fs.Bool("json", false, "print JSON")
	handled, err := parseCommandFlags(fs, args, stdout, scanUsage)
	if err != nil {
		return err
	}
	if handled {
		return nil
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: %s", scanUsage)
	}
	if *workers <= 0 {
		return fmt.Errorf("--workers must be greater than zero")
	}

	result, err := appcore.New(*dbPath).Scan(ctx, appcore.ScanRequest{
		Root:    fs.Arg(0),
		Workers: *workers,
		Rescan:  *rescan,
	}, nil)
	if err != nil {
		return err
	}
	if *asJSON {
		if err := printJSON(stdout, scanOutput(result), true); err != nil {
			return err
		}
	} else {
		fmt.Fprintf(stdout, "scanned=%d indexed=%d skipped=%d errors=%d\n", result.Scanned, result.Indexed, result.Skipped, len(result.Errors))
		for _, scanErr := range result.Errors {
			fmt.Fprintf(stderr, "%s: %s\n", scanErr.Path, scanErr.Error)
		}
	}
	return scanCommandError(result, *failOnError)
}

func scanCommandError(result appcore.ScanResult, failOnError bool) error {
	if failOnError && len(result.Errors) > 0 {
		return fmt.Errorf("scan completed with %d file errors", len(result.Errors))
	}
	return nil
}

func runShow(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fs := newFlagSet("show", stderr)
	dbPath := fs.String("db", defaultDBPath, "SQLite database path")
	raw := fs.Bool("raw", false, "include raw metadata")
	asJSON := fs.Bool("json", false, "print JSON")
	handled, err := parseCommandFlags(fs, args, stdout, showUsage)
	if err != nil {
		return err
	}
	if handled {
		return nil
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: %s", showUsage)
	}

	detail, err := appcore.New(*dbPath).GetImage(ctx, appcore.GetImageRequest{
		Ref:        fs.Arg(0),
		IncludeRaw: *raw,
	})
	if err != nil {
		return err
	}
	record := detail.Record
	if *asJSON {
		return printJSON(stdout, record, true)
	}
	fmt.Fprintf(stdout, "%d %s\n", record.ID, record.Path)
	fmt.Fprintf(stdout, "format=%s size=%d dimensions=%s\n", record.Format, record.Size, dimensions(record))
	if tags := tagsText(record.Tags, true); tags != "" {
		fmt.Fprintf(stdout, "tags=%s\n", tags)
	}
	for _, meta := range record.Metadata {
		fmt.Fprintf(stdout, "[%s]\npositive=%s\n", meta.Source, meta.PositivePrompt)
		if meta.NegativePrompt != "" {
			fmt.Fprintf(stdout, "negative=%s\n", meta.NegativePrompt)
		}
		if settings := keyValueText(meta.Settings); settings != "" {
			fmt.Fprintf(stdout, "settings=%s\n", settings)
		}
		if workflow := workflowText(meta.WorkflowSummary); workflow != "" {
			fmt.Fprintf(stdout, "workflow=%s\n", workflow)
		}
	}
	return nil
}

func runSearch(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fs := newFlagSet("search", stderr)
	dbPath := fs.String("db", defaultDBPath, "SQLite database path")
	tag := fs.String("tag", "", "normalized tag to search")
	source := fs.String("source", "", "metadata source")
	query := fs.String("q", "", "text query")
	format := fs.String("format", "", "image format: png or webp")
	hasWorkflow := fs.Bool("has-workflow", false, "only show records with workflow metadata")
	long := fs.Bool("long", false, "do not shorten prompt and tag previews")
	limit := fs.Int("limit", 50, "maximum records")
	asJSON := fs.Bool("json", false, "print JSON")
	handled, err := parseCommandFlags(fs, args, stdout, searchUsage)
	if err != nil {
		return err
	}
	if handled {
		return nil
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("usage: %s", searchUsage)
	}
	if *limit <= 0 {
		return fmt.Errorf("--limit must be greater than zero")
	}

	records, err := appcore.New(*dbPath).Search(ctx, appcore.SearchRequest{
		Tag:         *tag,
		Source:      *source,
		Query:       *query,
		Format:      *format,
		HasWorkflow: *hasWorkflow,
		Limit:       *limit,
	})
	if err != nil {
		return err
	}
	if *asJSON {
		return printJSON(stdout, records, true)
	}
	return printSearchTable(stdout, records, *long)
}

func runTags(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fs := newFlagSet("tags", stderr)
	dbPath := fs.String("db", defaultDBPath, "SQLite database path")
	source := fs.String("source", "", "metadata source")
	query := fs.String("q", "", "tag text query")
	limit := fs.Int("limit", 100, "maximum tags")
	asJSON := fs.Bool("json", false, "print JSON")
	handled, err := parseCommandFlags(fs, args, stdout, tagsUsage)
	if err != nil {
		return err
	}
	if handled {
		return nil
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("usage: %s", tagsUsage)
	}
	if *limit <= 0 {
		return fmt.Errorf("--limit must be greater than zero")
	}

	tags, err := appcore.New(*dbPath).Tags(ctx, appcore.TagsRequest{
		Source: *source,
		Query:  *query,
		Limit:  *limit,
	})
	if err != nil {
		return err
	}
	if *asJSON {
		return printJSON(stdout, tags, true)
	}
	return printTagsTable(stdout, tags)
}

func runStats(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fs := newFlagSet("stats", stderr)
	dbPath := fs.String("db", defaultDBPath, "SQLite database path")
	asJSON := fs.Bool("json", false, "print JSON")
	handled, err := parseCommandFlags(fs, args, stdout, statsUsage)
	if err != nil {
		return err
	}
	if handled {
		return nil
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("usage: %s", statsUsage)
	}

	stats, err := appcore.New(*dbPath).Stats(ctx)
	if err != nil {
		return err
	}
	if *asJSON {
		return printJSON(stdout, stats, true)
	}
	return printStats(stdout, stats)
}

func runExport(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fs := newFlagSet("export", stderr)
	dbPath := fs.String("db", defaultDBPath, "SQLite database path")
	out := fs.String("out", "", "JSON output path")
	pretty := fs.Bool("pretty", false, "pretty-print JSON")
	handled, err := parseCommandFlags(fs, args, stdout, exportUsage)
	if err != nil {
		return err
	}
	if handled {
		return nil
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("usage: %s", exportUsage)
	}
	if *out == "" {
		return fmt.Errorf("usage: %s", exportUsage)
	}

	_, err = appcore.New(*dbPath).Export(ctx, appcore.ExportRequest{Out: *out, Pretty: *pretty})
	return err
}

func runMove(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fs := newFlagSet("move", stderr)
	dbPath := fs.String("db", defaultDBPath, "SQLite database path")
	tag := fs.String("tag", "", "tag to move")
	to := fs.String("to", "", "destination root")
	apply := fs.Bool("apply", false, "apply file moves")
	conflict := fs.String("conflict", "skip", "conflict behavior: skip or rename")
	asJSON := fs.Bool("json", false, "print JSON")
	handled, err := parseCommandFlags(fs, args, stdout, moveUsage)
	if err != nil {
		return err
	}
	if handled {
		return nil
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("usage: %s", moveUsage)
	}
	if *tag == "" || *to == "" {
		return fmt.Errorf("usage: %s", moveUsage)
	}

	service := appcore.New(*dbPath)
	var plans []appcore.MovePlan
	if *apply {
		plans, err = service.ApplyMove(ctx, appcore.MoveRequest{Tag: *tag, To: *to, Conflict: *conflict})
	} else {
		plans, err = service.PlanMove(ctx, appcore.MoveRequest{Tag: *tag, To: *to, Conflict: *conflict})
	}
	if err != nil {
		return err
	}
	if *asJSON {
		return printJSON(stdout, plans, true)
	}
	for _, plan := range plans {
		fmt.Fprintf(stdout, "%s\t%s -> %s\t%s\n", plan.Status, plan.SourcePath, plan.DestinationPath, plan.Reason)
	}
	return nil
}

func runHelp(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		printUsage(stdout)
		return nil
	}
	if len(args) != 1 {
		return fmt.Errorf("usage: imv help [command]")
	}
	if args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
		printUsage(stdout)
		return nil
	}
	return runWithIOContext(ctx, []string{args[0], "--help"}, stdout, stderr)
}

func parseCommandFlags(fs *flag.FlagSet, args []string, stdout io.Writer, usage string) (bool, error) {
	err := parseInterspersed(fs, args)
	if errors.Is(err, flag.ErrHelp) {
		printCommandUsage(stdout, usage, fs)
		return true, nil
	}
	return false, err
}

func printCommandUsage(stdout io.Writer, usage string, fs *flag.FlagSet) {
	fmt.Fprintf(stdout, "Usage: %s\n\nOptions:\n", usage)
	fs.SetOutput(stdout)
	fs.PrintDefaults()
}

func parseInterspersed(fs *flag.FlagSet, args []string) error {
	var flags []string
	var positional []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			positional = append(positional, args[i+1:]...)
			break
		}
		if strings.HasPrefix(arg, "-") && arg != "-" {
			flags = append(flags, arg)
			if !strings.Contains(arg, "=") && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") && flagNeedsValue(fs, strings.TrimLeft(arg, "-")) {
				flags = append(flags, args[i+1])
				i++
			}
			continue
		}
		positional = append(positional, arg)
	}
	return fs.Parse(append(flags, positional...))
}

func flagNeedsValue(fs *flag.FlagSet, name string) bool {
	flag := fs.Lookup(name)
	if flag == nil {
		return false
	}
	return flag.DefValue != "false" && flag.DefValue != "true"
}

func newFlagSet(name string, stderr io.Writer) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {}
	return fs
}

func printUsage(stdout io.Writer) {
	fmt.Fprintln(stdout, strings.TrimSpace(`imv - AI image metadata indexer

Commands:
  scan    index PNG/WebP files
  show    show one indexed record
  search  search indexed records
  tags    summarize indexed prompt tags
  stats   summarize the current index
  export  write deterministic JSON export
  move    plan or apply tag-based moves

Run "imv help <command>" for command-specific options.`))
}

func printJSON(stdout io.Writer, v any, pretty bool) error {
	data, err := marshal(v, pretty)
	if err != nil {
		return err
	}
	fmt.Fprintln(stdout, string(data))
	return nil
}

func marshal(v any, pretty bool) ([]byte, error) {
	if pretty {
		return json.MarshalIndent(v, "", "  ")
	}
	return json.Marshal(v)
}

func mustJSON(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(data)
}

type scanErrorOutput struct {
	Path  string `json:"path"`
	Error string `json:"error"`
}

type scanResultOutput struct {
	Scanned int               `json:"scanned"`
	Indexed int               `json:"indexed"`
	Skipped int               `json:"skipped"`
	Errors  []scanErrorOutput `json:"errors"`
}

func scanOutput(result appcore.ScanResult) scanResultOutput {
	out := scanResultOutput{
		Scanned: result.Scanned,
		Indexed: result.Indexed,
		Skipped: result.Skipped,
	}
	for _, item := range result.Errors {
		out.Errors = append(out.Errors, scanErrorOutput{Path: item.Path, Error: item.Error})
	}
	return out
}

func printSearchTable(stdout io.Writer, records []store.ImageRecord, long bool) error {
	tw := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tSOURCE\tFORMAT\tDIMENSIONS\tPATH\tPROMPT\tTAGS")
	for _, record := range records {
		fmt.Fprintf(tw, "%d\t%s\t%s\t%s\t%s\t%s\t%s\n",
			record.ID,
			recordSource(record),
			record.Format,
			dimensions(record),
			record.Path,
			preview(recordPrompt(record), 72, long),
			preview(tagsText(record.Tags, long), 48, long),
		)
	}
	return tw.Flush()
}

func printTagsTable(stdout io.Writer, tags []store.TagSummary) error {
	tw := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "TAG\tCOUNT\tSOURCES\tEXAMPLE")
	for _, tag := range tags {
		fmt.Fprintf(tw, "%s\t%d\t%s\t%s\n", tag.Tag, tag.Count, strings.Join(tag.Sources, ","), tag.Example)
	}
	return tw.Flush()
}

func printStats(stdout io.Writer, stats store.Stats) error {
	fmt.Fprintf(stdout, "files=%d workflow_files=%d\n", stats.TotalFiles, stats.WorkflowFiles)
	if stats.FirstScanned != nil && stats.LastScanned != nil {
		fmt.Fprintf(stdout, "scanned_at=%s..%s\n", stats.FirstScanned.Format("2006-01-02 15:04:05"), stats.LastScanned.Format("2006-01-02 15:04:05"))
	}
	fmt.Fprintf(stdout, "formats=%s\n", countMapText(stats.Formats))
	fmt.Fprintf(stdout, "sources=%s\n", countMapText(stats.Sources))
	if len(stats.TopTags) > 0 {
		fmt.Fprintln(stdout, "top_tags:")
		for _, tag := range stats.TopTags {
			fmt.Fprintf(stdout, "  %s=%d\n", tag.Tag, tag.Count)
		}
	}
	return nil
}

func recordSource(record store.ImageRecord) string {
	if len(record.Metadata) == 0 || record.Metadata[0].Source == "" {
		return "-"
	}
	return record.Metadata[0].Source
}

func recordPrompt(record store.ImageRecord) string {
	for _, meta := range record.Metadata {
		if strings.TrimSpace(meta.PositivePrompt) != "" {
			return meta.PositivePrompt
		}
	}
	return ""
}

func dimensions(record store.ImageRecord) string {
	if record.Width <= 0 || record.Height <= 0 {
		return "-"
	}
	return fmt.Sprintf("%dx%d", record.Width, record.Height)
}

func tagsText(tags []store.TagRecord, long bool) string {
	values := make([]string, 0, len(tags))
	for _, tag := range tags {
		value := strings.TrimSpace(tag.Tag)
		if value == "" {
			value = tag.Normalized
		}
		if value != "" {
			values = append(values, value)
		}
	}
	text := strings.Join(values, ", ")
	if long {
		return text
	}
	return preview(text, 48, false)
}

func preview(value string, max int, long bool) string {
	value = strings.Join(strings.Fields(value), " ")
	if long || max <= 0 {
		return value
	}
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	if max <= 3 {
		return string(runes[:max])
	}
	return string(runes[:max-3]) + "..."
}

func keyValueText(values map[string]any) string {
	if len(values) == 0 {
		return ""
	}
	keys := sortedKeys(values)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+valueText(values[key]))
	}
	return strings.Join(parts, " ")
}

func workflowText(values map[string]any) string {
	if len(values) == 0 {
		return ""
	}
	parts := []string{}
	if value, ok := values["node_count"]; ok && valueText(value) != "" && valueText(value) != "0" {
		parts = append(parts, "node_count="+valueText(value))
	}
	for _, key := range []string{"models", "checkpoints", "samplers", "schedulers"} {
		if text := listText(values[key]); text != "" {
			parts = append(parts, key+"="+text)
		}
	}
	return strings.Join(parts, " ")
}

func listText(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case []any:
		parts := make([]string, 0, len(typed))
		for _, item := range typed {
			if text := valueText(item); text != "" {
				parts = append(parts, text)
			}
		}
		return strings.Join(parts, ",")
	case []string:
		return strings.Join(typed, ",")
	default:
		return valueText(typed)
	}
}

func valueText(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case float64:
		if typed == float64(int64(typed)) {
			return fmt.Sprintf("%d", int64(typed))
		}
		return fmt.Sprintf("%g", typed)
	case int:
		return fmt.Sprintf("%d", typed)
	case int64:
		return fmt.Sprintf("%d", typed)
	case bool:
		if typed {
			return "true"
		}
		return "false"
	default:
		data, err := json.Marshal(typed)
		if err != nil {
			return ""
		}
		return string(data)
	}
}

func countMapText(values map[string]int) string {
	if len(values) == 0 {
		return "-"
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", key, values[key]))
	}
	return strings.Join(parts, " ")
}

func sortedKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
