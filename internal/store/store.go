package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct {
	sql *sql.DB
}

type ImageRecord struct {
	ID       int64            `json:"id"`
	Path     string           `json:"path"`
	Format   string           `json:"format"`
	Size     int64            `json:"size"`
	MTime    int64            `json:"mtime"`
	Width    int              `json:"width"`
	Height   int              `json:"height"`
	Scanned  time.Time        `json:"scanned_at"`
	Metadata []MetadataRecord `json:"metadata,omitempty"`
	Tags     []TagRecord      `json:"tags,omitempty"`
}

type MetadataRecord struct {
	Source          string         `json:"source"`
	PositivePrompt  string         `json:"positive_prompt,omitempty"`
	NegativePrompt  string         `json:"negative_prompt,omitempty"`
	Settings        map[string]any `json:"settings,omitempty"`
	WorkflowSummary map[string]any `json:"workflow_summary,omitempty"`
	Raw             map[string]any `json:"raw,omitempty"`
}

type TagRecord struct {
	Tag        string `json:"tag"`
	Normalized string `json:"normalized_tag"`
	Source     string `json:"source"`
	Kind       string `json:"kind"`
}

type FileInput struct {
	Path     string
	Format   string
	Size     int64
	MTime    int64
	Width    int
	Height   int
	Metadata []MetadataRecord
	Tags     []TagRecord
}

type SearchOptions struct {
	Tag         string
	Source      string
	Query       string
	Format      string
	HasWorkflow bool
	Limit       int
}

type TagSummaryOptions struct {
	Source string
	Query  string
	Limit  int
}

type TagSummary struct {
	Tag     string   `json:"tag"`
	Count   int      `json:"count"`
	Sources []string `json:"sources"`
	Example string   `json:"example"`
}

type Stats struct {
	TotalFiles    int            `json:"total_files"`
	Formats       map[string]int `json:"formats"`
	Sources       map[string]int `json:"sources"`
	WorkflowFiles int            `json:"workflow_files"`
	TopTags       []TagSummary   `json:"top_tags"`
	FirstScanned  *time.Time     `json:"first_scanned_at,omitempty"`
	LastScanned   *time.Time     `json:"last_scanned_at,omitempty"`
}

func Open(path string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}
	sqldb, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	sqldb.SetMaxOpenConns(1)
	db := &DB{sql: sqldb}
	if err := db.init(); err != nil {
		_ = sqldb.Close()
		return nil, err
	}
	return db, nil
}

func (db *DB) Close() error {
	return db.sql.Close()
}

func (db *DB) init() error {
	schema := []string{
		`create table if not exists migrations (version integer primary key, applied_at text not null);`,
		`create table if not exists files (
			id integer primary key autoincrement,
			path text not null unique,
			format text not null,
			size integer not null,
			mtime integer not null,
			width integer not null default 0,
			height integer not null default 0,
			scanned_at text not null
		);`,
		`create table if not exists metadata (
			file_id integer not null,
			source text not null,
			positive_prompt text not null default '',
			negative_prompt text not null default '',
			settings_json text not null default '{}',
			workflow_summary_json text not null default '{}',
			raw_json text not null default '{}',
			foreign key(file_id) references files(id) on delete cascade
		);`,
		`create table if not exists tags (
			file_id integer not null,
			tag text not null,
			normalized_tag text not null,
			source text not null,
			kind text not null,
			foreign key(file_id) references files(id) on delete cascade
		);`,
		`create index if not exists idx_files_path on files(path);`,
		`create index if not exists idx_tags_normalized on tags(normalized_tag);`,
		`create index if not exists idx_metadata_source on metadata(source);`,
	}
	for _, stmt := range schema {
		if _, err := db.sql.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

func (db *DB) IsUnchanged(path string, size int64, mtime int64) (bool, error) {
	var count int
	err := db.sql.QueryRow(`select count(*) from files where path = ? and size = ? and mtime = ?`, path, size, mtime).Scan(&count)
	return count > 0, err
}

func (db *DB) UpsertFile(ctx context.Context, input FileInput) (int64, error) {
	tx, err := db.sql.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err = tx.ExecContext(ctx, `insert into files(path, format, size, mtime, width, height, scanned_at)
		values(?, ?, ?, ?, ?, ?, ?)
		on conflict(path) do update set
			format = excluded.format,
			size = excluded.size,
			mtime = excluded.mtime,
			width = excluded.width,
			height = excluded.height,
			scanned_at = excluded.scanned_at`,
		input.Path, input.Format, input.Size, input.MTime, input.Width, input.Height, now)
	if err != nil {
		return 0, err
	}

	var id int64
	if err := tx.QueryRowContext(ctx, `select id from files where path = ?`, input.Path).Scan(&id); err != nil {
		return 0, err
	}
	if _, err := tx.ExecContext(ctx, `delete from metadata where file_id = ?`, id); err != nil {
		return 0, err
	}
	if _, err := tx.ExecContext(ctx, `delete from tags where file_id = ?`, id); err != nil {
		return 0, err
	}
	for _, meta := range input.Metadata {
		settingsJSON, err := jsonString(meta.Settings)
		if err != nil {
			return 0, err
		}
		workflowJSON, err := jsonString(meta.WorkflowSummary)
		if err != nil {
			return 0, err
		}
		rawJSON, err := jsonString(meta.Raw)
		if err != nil {
			return 0, err
		}
		if _, err := tx.ExecContext(ctx, `insert into metadata(file_id, source, positive_prompt, negative_prompt, settings_json, workflow_summary_json, raw_json)
			values(?, ?, ?, ?, ?, ?, ?)`, id, meta.Source, meta.PositivePrompt, meta.NegativePrompt, settingsJSON, workflowJSON, rawJSON); err != nil {
			return 0, err
		}
	}
	for _, tag := range input.Tags {
		if _, err := tx.ExecContext(ctx, `insert into tags(file_id, tag, normalized_tag, source, kind) values(?, ?, ?, ?, ?)`,
			id, tag.Tag, tag.Normalized, tag.Source, tag.Kind); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return id, nil
}

func (db *DB) GetByID(id int64, includeRaw bool) (ImageRecord, error) {
	return db.get(`f.id = ?`, includeRaw, id)
}

func (db *DB) GetByPath(path string, includeRaw bool) (ImageRecord, error) {
	return db.get(`f.path = ?`, includeRaw, path)
}

func (db *DB) Search(opts SearchOptions) ([]ImageRecord, error) {
	if opts.Limit <= 0 {
		opts.Limit = 50
	}
	clauses := []string{"1 = 1"}
	args := []any{}
	if opts.Tag != "" {
		clauses = append(clauses, `exists(select 1 from tags t where t.file_id = f.id and t.normalized_tag = ?)`)
		args = append(args, NormalizeTag(opts.Tag))
	}
	if opts.Source != "" {
		clauses = append(clauses, `exists(select 1 from metadata m where m.file_id = f.id and m.source = ?)`)
		args = append(args, opts.Source)
	}
	if strings.TrimSpace(opts.Format) != "" {
		clauses = append(clauses, `f.format = ?`)
		args = append(args, strings.ToLower(strings.TrimSpace(opts.Format)))
	}
	if opts.HasWorkflow {
		clauses = append(clauses, `exists(select 1 from metadata m where m.file_id = f.id and `+workflowPredicate("m")+`)`)
	}
	if opts.Query != "" {
		like := "%" + opts.Query + "%"
		clauses = append(clauses, `exists(select 1 from metadata m where m.file_id = f.id and (m.positive_prompt like ? or m.negative_prompt like ? or m.settings_json like ? or m.workflow_summary_json like ? or m.raw_json like ?))`)
		args = append(args, like, like, like, like, like)
	}
	args = append(args, opts.Limit)

	rows, err := db.sql.Query(`select f.id from files f where `+strings.Join(clauses, " and ")+` order by f.path limit ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}

	var records []ImageRecord
	for _, id := range ids {
		record, err := db.GetByID(id, false)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, nil
}

func (db *DB) Export() ([]ImageRecord, error) {
	rows, err := db.sql.Query(`select id from files order by path`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}

	var records []ImageRecord
	for _, id := range ids {
		record, err := db.GetByID(id, true)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, nil
}

func (db *DB) TagsSummary(opts TagSummaryOptions) ([]TagSummary, error) {
	if opts.Limit <= 0 {
		opts.Limit = 100
	}
	clauses := []string{"1 = 1"}
	args := []any{}
	if opts.Source != "" {
		clauses = append(clauses, `t.source = ?`)
		args = append(args, opts.Source)
	}
	if opts.Query != "" {
		like := "%" + NormalizeTag(opts.Query) + "%"
		clauses = append(clauses, `(t.normalized_tag like ? or lower(t.tag) like ?)`)
		args = append(args, like, like)
	}
	args = append(args, opts.Limit)

	rows, err := db.sql.Query(`select t.normalized_tag, min(t.tag), count(distinct t.file_id)
		from tags t
		where `+strings.Join(clauses, " and ")+`
		group by t.normalized_tag
		order by count(distinct t.file_id) desc, t.normalized_tag
		limit ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []TagSummary
	for rows.Next() {
		var item TagSummary
		if err := rows.Scan(&item.Tag, &item.Example, &item.Count); err != nil {
			return nil, err
		}
		summaries = append(summaries, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}

	for i := range summaries {
		sources, err := db.tagSources(summaries[i].Tag, opts.Source)
		if err != nil {
			return nil, err
		}
		summaries[i].Sources = sources
	}
	return summaries, nil
}

func (db *DB) Stats() (Stats, error) {
	stats := Stats{
		Formats: map[string]int{},
		Sources: map[string]int{},
	}
	var first, last sql.NullString
	if err := db.sql.QueryRow(`select count(*), min(scanned_at), max(scanned_at) from files`).Scan(&stats.TotalFiles, &first, &last); err != nil {
		return Stats{}, err
	}
	stats.FirstScanned = parseTimePtr(first)
	stats.LastScanned = parseTimePtr(last)

	formats, err := db.countsBy(`select format, count(*) from files group by format order by format`)
	if err != nil {
		return Stats{}, err
	}
	stats.Formats = formats

	sources, err := db.countsBy(`select source, count(distinct file_id) from metadata group by source order by source`)
	if err != nil {
		return Stats{}, err
	}
	stats.Sources = sources

	if err := db.sql.QueryRow(`select count(distinct f.id) from files f where exists(select 1 from metadata m where m.file_id = f.id and ` + workflowPredicate("m") + `)`).Scan(&stats.WorkflowFiles); err != nil {
		return Stats{}, err
	}
	topTags, err := db.TagsSummary(TagSummaryOptions{Limit: 10})
	if err != nil {
		return Stats{}, err
	}
	stats.TopTags = topTags
	return stats, nil
}

func (db *DB) UpdatePath(ctx context.Context, id int64, newPath string) error {
	_, err := db.sql.ExecContext(ctx, `update files set path = ? where id = ?`, newPath, id)
	return err
}

func (db *DB) RecordsByTag(tag string) ([]ImageRecord, error) {
	return db.Search(SearchOptions{Tag: tag, Limit: 1000000})
}

func (db *DB) tagSources(tag, sourceFilter string) ([]string, error) {
	clauses := []string{`normalized_tag = ?`}
	args := []any{tag}
	if sourceFilter != "" {
		clauses = append(clauses, `source = ?`)
		args = append(args, sourceFilter)
	}
	rows, err := db.sql.Query(`select distinct source from tags where `+strings.Join(clauses, " and ")+` order by source`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var sources []string
	for rows.Next() {
		var source string
		if err := rows.Scan(&source); err != nil {
			return nil, err
		}
		sources = append(sources, source)
	}
	return sources, rows.Err()
}

func (db *DB) countsBy(query string) (map[string]int, error) {
	rows, err := db.sql.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	counts := map[string]int{}
	for rows.Next() {
		var key string
		var count int
		if err := rows.Scan(&key, &count); err != nil {
			return nil, err
		}
		counts[key] = count
	}
	return counts, rows.Err()
}

func workflowPredicate(alias string) string {
	return alias + `.workflow_summary_json <> '{}' and ` + alias + `.workflow_summary_json not like '%"node_count":0%'`
}

func parseTimePtr(value sql.NullString) *time.Time {
	if !value.Valid || strings.TrimSpace(value.String) == "" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, value.String)
	if err != nil {
		return nil
	}
	return &parsed
}

func (db *DB) get(where string, includeRaw bool, args ...any) (ImageRecord, error) {
	row := db.sql.QueryRow(`select f.id, f.path, f.format, f.size, f.mtime, f.width, f.height, f.scanned_at from files f where `+where, args...)
	var record ImageRecord
	var scanned string
	if err := row.Scan(&record.ID, &record.Path, &record.Format, &record.Size, &record.MTime, &record.Width, &record.Height, &scanned); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ImageRecord{}, fmt.Errorf("record not found")
		}
		return ImageRecord{}, err
	}
	record.Scanned, _ = time.Parse(time.RFC3339Nano, scanned)

	metadata, err := db.loadMetadata(record.ID, includeRaw)
	if err != nil {
		return ImageRecord{}, err
	}
	tags, err := db.loadTags(record.ID)
	if err != nil {
		return ImageRecord{}, err
	}
	record.Metadata = metadata
	record.Tags = tags
	return record, nil
}

func (db *DB) loadMetadata(fileID int64, includeRaw bool) ([]MetadataRecord, error) {
	rows, err := db.sql.Query(`select source, positive_prompt, negative_prompt, settings_json, workflow_summary_json, raw_json from metadata where file_id = ? order by source`, fileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []MetadataRecord
	for rows.Next() {
		var item MetadataRecord
		var settingsJSON, workflowJSON, rawJSON string
		if err := rows.Scan(&item.Source, &item.PositivePrompt, &item.NegativePrompt, &settingsJSON, &workflowJSON, &rawJSON); err != nil {
			return nil, err
		}
		item.Settings = parseMap(settingsJSON)
		item.WorkflowSummary = parseMap(workflowJSON)
		if includeRaw {
			item.Raw = parseMap(rawJSON)
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (db *DB) loadTags(fileID int64) ([]TagRecord, error) {
	rows, err := db.sql.Query(`select tag, normalized_tag, source, kind from tags where file_id = ? order by normalized_tag`, fileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []TagRecord
	for rows.Next() {
		var tag TagRecord
		if err := rows.Scan(&tag.Tag, &tag.Normalized, &tag.Source, &tag.Kind); err != nil {
			return nil, err
		}
		result = append(result, tag)
	}
	return result, rows.Err()
}

func jsonString(v map[string]any) (string, error) {
	if v == nil {
		v = map[string]any{}
	}
	data, err := json.Marshal(v)
	return string(data), err
}

func parseMap(raw string) map[string]any {
	out := map[string]any{}
	_ = json.Unmarshal([]byte(raw), &out)
	return out
}

func NormalizeTag(tag string) string {
	return strings.ToLower(strings.TrimSpace(tag))
}

func SplitPromptTags(prompt string) []TagRecord {
	parts := strings.Split(prompt, ",")
	seen := map[string]bool{}
	var tags []TagRecord
	for _, part := range parts {
		tag := strings.TrimSpace(part)
		normalized := NormalizeTag(tag)
		if normalized == "" || seen[normalized] {
			continue
		}
		seen[normalized] = true
		tags = append(tags, TagRecord{Tag: tag, Normalized: normalized, Kind: "prompt"})
	}
	sort.Slice(tags, func(i, j int) bool {
		return tags[i].Normalized < tags[j].Normalized
	})
	return tags
}
