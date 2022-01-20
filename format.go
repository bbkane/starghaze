package main

import (
	"bufio"
	"context"
	"database/sql"
	"embed"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/bbkane/warg/flag"
	"github.com/lestrrat-go/strftime"
	_ "modernc.org/sqlite"
)

//go:embed sqlite_migrations/*.sql
var migrationFS embed.FS

type Printer interface {
	Header() error
	Line(*starredRepositoryEdge) error
	Flush() error
}

// -- JSONPrinter

type JSONPrinter struct {
	w io.Writer
}

func NewJSONPrinter(w io.Writer) *JSONPrinter {
	return &JSONPrinter{
		w: w,
	}
}

func (JSONPrinter) Header() error {
	return nil
}

func (p *JSONPrinter) Line(sR *starredRepositoryEdge) error {

	buf, err := json.Marshal(sR)
	if err != nil {
		return fmt.Errorf("json marshall err: %w", err)
	}
	_, err = p.w.Write(buf)
	if err != nil {
		return fmt.Errorf("file write err: %w", err)
	}
	_, err = p.w.Write([]byte{'\n'})
	if err != nil {
		return fmt.Errorf("newline write err: %w", err)
	}
	return nil
}

func (JSONPrinter) Flush() error {
	return nil
}

// -- CSVPrinter

type CSVPrinter struct {
	writer *csv.Writer
	count  int
}

func NewCSVPrinter(w io.Writer) *CSVPrinter {
	return &CSVPrinter{
		writer: csv.NewWriter(w),
		count:  1,
	}
}

func (p *CSVPrinter) Header() error {
	err := p.writer.Write([]string{
		"Count",
		"Description",
		"HomepageURL",
		"NameWithOwner",
		"Languages",
		"PushedAt",
		"README",
		"StargazerCount",
		"StarredAt",
		"Topics",
		"UpdatedAt",
		"Url",
	})
	if err != nil {
		return fmt.Errorf("CSV header err: %w", err)
	}
	return nil
}

func (p *CSVPrinter) Line(sr *starredRepositoryEdge) error {

	topicsList := []string{}
	for i := range sr.Node.RepositoryTopics.Nodes {
		topicsList = append(topicsList, sr.Node.RepositoryTopics.Nodes[i].Topic.Name)
	}
	topics := strings.Join(topicsList, " ")

	pushedAt, err := sr.Node.PushedAt.FormatString()
	if err != nil {
		return err
	}
	starredAt, err := sr.StarredAt.FormatString()
	if err != nil {
		return nil
	}

	updatedAt, err := sr.Node.UpdatedAt.FormatString()
	if err != nil {
		return nil
	}

	languagesList := []string{}
	for i := range sr.Node.Languages.Edges {
		languagesList = append(languagesList, sr.Node.Languages.Edges[i].Node.Name)
	}
	languages := strings.Join(languagesList, " ")

	err = p.writer.Write([]string{
		strconv.Itoa(p.count),
		sr.Node.Description,
		sr.Node.HomepageURL,
		sr.Node.NameWithOwner,
		languages,
		pushedAt,
		sr.Node.Object.Blob.Text, // README
		strconv.Itoa(sr.Node.StargazerCount),
		starredAt,
		topics,
		updatedAt,
		sr.Node.Url,
	})
	p.count++
	if err != nil {
		return fmt.Errorf("CSV write err: %w", err)
	}
	return nil
}

func (p *CSVPrinter) Flush() error {
	p.writer.Flush()
	return p.writer.Error()
}

// -- SqlitePrinter

type SqlitePrinter struct {
	ctx context.Context
	db  *sql.DB
	err error
	// we're going to use one transaction for all writes
	// so we might as well cache it here
	tx *sql.Tx
}

func NewSqlitePrinter(dsn string) (*SqlitePrinter, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("db open error: %s: %w", dsn, err)
	}

	// Enable foreign key checks. For historical reasons, SQLite does not check
	// foreign key constraints by default... which is kinda insane. There's some
	// overhead on inserts to verify foreign key integrity but it's definitely
	// worth it.
	if _, err := db.Exec(`PRAGMA foreign_keys = ON;`); err != nil {
		return nil, fmt.Errorf("foreign keys pragma: %w", err)
	}
	if err := migrate(db, migrationFS); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	tx, err := db.Begin()
	if err != nil {
		err = fmt.Errorf("can't begin tx: %w", err)
		return nil, err
	}

	return &SqlitePrinter{
		ctx: context.Background(), // TODO: paramaterize
		db:  db,
		err: nil,
		tx:  tx,
	}, nil
}

func (SqlitePrinter) Header() error {
	return nil
}

func (p *SqlitePrinter) Line(sr *starredRepositoryEdge) error {
	// we need to set p.err if needed so we don't commit the tx later

	// Repo
	var repoID int
	{
		starredAt, err := sr.StarredAt.Time()
		if err != nil {
			err = fmt.Errorf("StarredAt time err: %w", err)
			p.err = err
			return err
		}

		pushedAt, err := sr.Node.PushedAt.Time()
		if err != nil {
			err = fmt.Errorf("PushedAt time err: %w", err)
			p.err = err
			return err
		}

		updatedAt, err := sr.Node.UpdatedAt.Time()
		if err != nil {
			err = fmt.Errorf("UpdatedAt time err: %w", err)
			p.err = err
			return err
		}
		err = p.tx.QueryRowContext(
			p.ctx,
			`
			INSERT INTO Repo (
				StarredAt,
				Description,
				HomepageURL,
				NameWithOwner,
				Readme,
				PushedAt,
				StargazerCount,
				UpdatedAt,
				Url
			)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			RETURNING id
			`,
			(*NullTime)(&starredAt),
			sr.Node.Description,
			sr.Node.HomepageURL,
			sr.Node.NameWithOwner,
			sr.Node.Object.Blob.Text,
			(*NullTime)(&pushedAt),
			sr.Node.StargazerCount,
			(*NullTime)(&updatedAt),
			sr.Node.Url,
		).Scan(&repoID)
		if err != nil {
			p.err = err
			return err
		}
	}

	return nil
}

func (p *SqlitePrinter) Flush() error {

	if p.err != nil {
		p.tx.Rollback()
	}
	err := p.tx.Commit()
	if err != nil {
		err = fmt.Errorf("commit err: %w", err)
		return err
	}

	return p.db.Close()
}

var _ Printer = new(SqlitePrinter)

// -- ZincPrinter

type ZincPrinter struct {
	w         io.Writer
	indexName string
}

func NewZincPrinter(w io.Writer, indexName string) *ZincPrinter {
	return &ZincPrinter{
		w:         w,
		indexName: indexName,
	}
}

func (ZincPrinter) Header() error {
	return nil
}

func (p *ZincPrinter) Line(sr *starredRepositoryEdge) error {

	_, err := p.w.Write([]byte(`{ "index" : { "_index" : "` + p.indexName + `" } }` + "\n"))
	if err != nil {
		return fmt.Errorf("header write err: %w", err)
	}

	topics := []string{}
	for i := range sr.Node.RepositoryTopics.Nodes {
		topics = append(topics, sr.Node.RepositoryTopics.Nodes[i].Topic.Name)
	}
	topicsStr := strings.Join(topics, " ")
	pushedAt, err := sr.Node.PushedAt.FormatString()
	if err != nil {
		return err
	}
	starredAt, err := sr.StarredAt.FormatString()
	if err != nil {
		return nil
	}

	updatedAt, err := sr.Node.UpdatedAt.FormatString()
	if err != nil {
		return nil
	}

	languagesList := []string{}
	for i := range sr.Node.Languages.Edges {
		languagesList = append(languagesList, sr.Node.Languages.Edges[i].Node.Name)
	}
	languages := strings.Join(languagesList, " ")

	item := map[string]interface{}{
		"Description":    sr.Node.Description,
		"HomepageURL":    sr.Node.HomepageURL,
		"NameWithOwner":  sr.Node.NameWithOwner,
		"Languages":      languages,
		"PushedAt":       pushedAt,
		"StargazerCount": sr.Node.StargazerCount,
		"StarredAt":      starredAt,
		"Topics":         topicsStr,
		"UpdatedAt":      updatedAt,
		"Url":            sr.Node.Url,
		"README":         sr.Node.Object.Blob.Text,
	}

	buf, err := json.Marshal(item)
	if err != nil {
		return fmt.Errorf("json marshall err: %w", err)
	}
	_, err = p.w.Write(buf)
	if err != nil {
		return fmt.Errorf("file write err: %w", err)
	}
	_, err = p.w.Write([]byte{'\n'})
	if err != nil {
		return fmt.Errorf("newline write err: %w", err)
	}
	return nil
}

func (ZincPrinter) Flush() error {
	return nil
}

// -- formattedDate

type formattedDate struct {
	datetime string
	// Format can be nil to do no processing on the datetime.
	Format *strftime.Strftime
}

func (d formattedDate) MarshalJSON() ([]byte, error) {
	str, err := d.FormatString()
	if err != nil {
		return nil, err
	}
	// https://www.programming-books.io/essential/go/custom-json-marshaling-468765d144a34e87b913c7674e66c3a4
	// NOTE: if you forget the enclosing quotes, MarshalJSON doesn't emit anything and doesn't error out
	return []byte(`"` + str + `"`), nil
}

func (d *formattedDate) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, &d.datetime)
}

func (d formattedDate) Time() (time.Time, error) {
	t, err := time.Parse(time.RFC3339, d.datetime)
	return t, err
}

// FormatString formats d with the given format.
// If the format is nil, it jsut returns d
func (d *formattedDate) FormatString() (string, error) {
	if d.Format == nil {
		return d.datetime, nil
	}
	t, err := d.Time()
	if err != nil {
		return "", err
	}
	return d.Format.FormatString(t), nil
}

func format(pf flag.PassedFlags) error {
	format := pf["--format"].(string)
	includeReadmes := pf["--include-readmes"].(bool)
	maxLineSize := pf["--max-line-size"].(int)
	sqliteDSN := pf["--sqlite-dsn"].(string)
	zincIndexName := pf["--zinc-index-name"].(string)

	dateFormatStr, dateFormatStrExists := pf["--date-format"].(string)
	var dateFormat *strftime.Strftime
	var err error
	if dateFormatStrExists {
		dateFormat, err = strftime.New(dateFormatStr)
		if err != nil {
			return fmt.Errorf("--date-format error: %w", err)
		}
	}

	output, outputExists := pf["--output"].(string)
	outputFp := os.Stdout
	if outputExists {
		newFP, err := os.Create(output)
		if err != nil {
			return fmt.Errorf("file open err: %w", err)
		}
		outputFp = newFP
		defer newFP.Close()
	}

	outputBuf := bufio.NewWriter(outputFp)
	defer outputBuf.Flush()

	var p Printer
	switch format {
	case "csv":
		p = NewCSVPrinter(outputBuf)
	case "jsonl":
		p = NewJSONPrinter(outputBuf)
	case "sqlite":
		p, err = NewSqlitePrinter(sqliteDSN)
		if err != nil {
			return fmt.Errorf("sql open err: %w", err)
		}
	case "zinc":
		p = NewZincPrinter(outputBuf, zincIndexName)
	default:
		return fmt.Errorf("unknown output format: %s", format)
	}

	defer p.Flush()

	err = p.Header()
	if err != nil {
		return err
	}

	// https://stackoverflow.com/a/16615559/2958070
	input := pf["--input"].(string)
	inputFp, err := os.Open(input)
	if err != nil {
		return fmt.Errorf("file open err: %w", err)
	}
	defer inputFp.Close()

	scanner := bufio.NewScanner(inputFp)

	maxCapacity := maxLineSize * 1024 * 1024 // MB -> bytes
	scannerBuf := make([]byte, maxCapacity)
	scanner.Buffer(scannerBuf, maxCapacity)

	var query Query
	for scanner.Scan() {
		err = json.Unmarshal(scanner.Bytes(), &query)
		if err != nil {
			return fmt.Errorf("json Unmarshal error: %w", err)
		}

		for i := range query.Viewer.StarredRepositories.Edges {
			edge := query.Viewer.StarredRepositories.Edges[i]
			edge.StarredAt.Format = dateFormat
			edge.Node.PushedAt.Format = dateFormat
			edge.Node.UpdatedAt.Format = dateFormat
			if !includeReadmes {
				edge.Node.Object.Blob.Text = ""
			}
			err := p.Line(&edge)
			if err != nil {
				return fmt.Errorf("line print error: %w", err)
			}
		}
	}
	err = scanner.Err()
	if err != nil {
		return fmt.Errorf("scanner err: %w", err)
	}
	return nil
}
