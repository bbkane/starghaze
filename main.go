package main

import (
	"bufio"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"

	"github.com/bbkane/warg"
	"github.com/bbkane/warg/command"
	"github.com/bbkane/warg/flag"
	"github.com/bbkane/warg/section"
	"github.com/bbkane/warg/value"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"

	"github.com/lestrrat-go/strftime"
)

// This will be overriden by goreleaser
var version = "unkown version: error reading goreleaser info"

func getVersion() string {
	// If installed via `go install`, we'll be able to read runtime version info
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown version: error reading build info"
	}
	if info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	// If built via GoReleaser, we'll be able to read this from the linker flags
	return version
}

func printVersion(_ flag.PassedFlags) error {
	fmt.Println(getVersion())
	return nil
}

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
		"PushedAt",
		"StargazerCount",
		"StarredAt",
		"Topics",
		"UpdatedAt",
		"Url",
		"README",
	})
	if err != nil {
		return fmt.Errorf("CSV header err: %w", err)
	}
	return nil
}

func (p *CSVPrinter) Line(sr *starredRepositoryEdge) error {

	topics := []string{}
	for i := range sr.Node.RepositoryTopics.Nodes {
		topics = append(topics, sr.Node.RepositoryTopics.Nodes[i].Topic.Name)
	}
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

	err = p.writer.Write([]string{
		strconv.Itoa(p.count),
		sr.Node.Description,
		sr.Node.HomepageURL,
		sr.Node.NameWithOwner,
		pushedAt,
		strconv.Itoa(sr.Node.StargazerCount),
		starredAt,
		strings.Join(topics, " "),
		updatedAt,
		sr.Node.Url,
		sr.Node.Object.Blob.Text,
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

	item := map[string]interface{}{
		"Description":    sr.Node.Description,
		"HomepageURL":    sr.Node.HomepageURL,
		"NameWithOwner":  sr.Node.NameWithOwner,
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

// FormatString formats d with the given format.
// If the format is nil, it jsut returns d
func (d *formattedDate) FormatString() (string, error) {
	if d.Format == nil {
		return d.datetime, nil
	}
	t, err := time.Parse(time.RFC3339, d.datetime)
	if err != nil {
		return "", err
	}
	return d.Format.FormatString(t), nil
}

type starredRepositoryEdge struct {
	StarredAt formattedDate
	Node      struct {
		Description string
		HomepageURL string
		Object      struct {
			Blob struct {
				Text string
			} `graphql:"... on Blob"`
		} `graphql:"object(expression: \"HEAD:README.md\") @include(if: $includeREADME)"`
		NameWithOwner    string
		PushedAt         formattedDate
		RepositoryTopics struct {
			Nodes []struct {
				URL   string
				Topic struct {
					Name string
				}
			}
		} `graphql:"repositoryTopics(first: 10)"`
		StargazerCount int
		UpdatedAt      formattedDate
		Url            string
	}
}

func stats(pf flag.PassedFlags) error {
	token := pf["--token"].(string)
	pageSize := pf["--page-size"].(int)
	maxPages := pf["--max-pages"].(int)
	output, outputExists := pf["--output"].(string)
	format := pf["--format"].(string)
	timeout := pf["--timeout"].(time.Duration)
	includeReadmes := pf["--include-readmes"].(bool)

	dateFormatStr, dateFormatStrExists := pf["--date-format"].(string)

	var dateFormat *strftime.Strftime
	var err error

	if dateFormatStrExists {
		dateFormat, err = strftime.New(dateFormatStr)
		if err != nil {
			return fmt.Errorf("--date-format error: %w", err)
		}
	}

	zincIndexName := pf["--zinc-index-name"].(string)

	fp := os.Stdout
	if outputExists {
		newFP, err := os.Create(output)
		if err != nil {
			return fmt.Errorf("file open err: %w", err)
		}
		fp = newFP
		defer newFP.Close()
	}

	buf := bufio.NewWriter(fp)
	defer buf.Flush()

	var p Printer
	switch format {
	case "csv":
		p = NewCSVPrinter(buf)
	case "jsonl":
		p = NewJSONPrinter(buf)
	case "zinc":
		p = NewZincPrinter(buf, zincIndexName)
	default:
		return fmt.Errorf("unknown output format: %s", format)
	}

	defer p.Flush()

	err = p.Header()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	httpClient := oauth2.NewClient(ctx, src)
	client := githubv4.NewClient(httpClient)

	var query struct {
		Viewer struct {
			StarredRepositories struct {
				Edges    []starredRepositoryEdge
				PageInfo struct {
					EndCursor   githubv4.String
					HasNextPage githubv4.Boolean
				}
			} `graphql:"starredRepositories(first: $starredRepositoryPageSize, orderBy: {field:STARRED_AT, direction:ASC}, after: $starredRepositoriesCursor)"`
		}
	}

	variables := map[string]interface{}{
		"starredRepositoriesCursor": (*githubv4.String)(nil),
		"starredRepositoryPageSize": githubv4.NewInt(githubv4.Int(pageSize)),
		"includeREADME":             githubv4.Boolean(includeReadmes),
	}

	for i := 0; i < maxPages; i++ {
		err := client.Query(ctx, &query, variables)
		if err != nil {
			return fmt.Errorf("query err: %w", err)
		}

		for i := range query.Viewer.StarredRepositories.Edges {
			edge := query.Viewer.StarredRepositories.Edges[i]
			edge.StarredAt.Format = dateFormat
			edge.Node.PushedAt.Format = dateFormat
			edge.Node.UpdatedAt.Format = dateFormat
			p.Line(&edge)
		}

		if !query.Viewer.StarredRepositories.PageInfo.HasNextPage {
			break
		}
		variables["starredRepositoriesCursor"] = githubv4.NewString(query.Viewer.StarredRepositories.PageInfo.EndCursor)
	}
	return nil
}

func gSheetsOpen(pf flag.PassedFlags) error {
	spreadsheetId := pf["--spreadsheet-id"].(string)
	sheetID := pf["--sheet-id"].(int)

	link := fmt.Sprintf(
		"https://docs.google.com/spreadsheets/d/%s/edit#gid=%d",
		spreadsheetId,
		sheetID,
	)
	fmt.Printf("Opening: %s\n", link)

	// https://stackoverflow.com/a/39324149/2958070
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}
	args = append(args, link)
	exec.Command(cmd, args...).Start()
	return nil
}

func gSheetsUpload(pf flag.PassedFlags) error {
	csvPath := pf["--csv-path"].(string)
	spreadsheetId := pf["--spreadsheet-id"].(string)
	sheetID := pf["--sheet-id"].(int)
	timeout := pf["--timeout"].(time.Duration)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	csvBytes, err := ioutil.ReadFile(csvPath)
	if err != nil {
		return fmt.Errorf("csv read error: %s: %w", csvPath, err)
	}
	csvStr := string(csvBytes)

	creds, err := google.FindDefaultCredentials(ctx, sheets.SpreadsheetsScope)
	if err != nil {
		return fmt.Errorf("can't find default credentials: %w", err)
	}

	srv, err := sheets.NewService(ctx, option.WithCredentials(creds))
	if err != nil {
		return fmt.Errorf("unable to retrieve Sheets client: %w", err)
	}

	requests := []*sheets.Request{
		// Erase current cells
		// https://stackoverflow.com/q/37928947/2958070
		{
			UpdateCells: &sheets.UpdateCellsRequest{
				Fields: "*",
				Range: &sheets.GridRange{
					SheetId: 0,
				},
			},
		},
		// https://stackoverflow.com/q/42362702/2958070
		{
			PasteData: &sheets.PasteDataRequest{
				Coordinate: &sheets.GridCoordinate{
					ColumnIndex: 0,
					RowIndex:    0,
					// https://developers.google.com/sheets/api/guides/concepts
					SheetId: int64(sheetID), // TODO: prefer reading an int64 flag, not casting :)
				},
				Data:      csvStr,
				Delimiter: ",",
				Type:      "PASTE_NORMAL",
			},
		},
	}

	rb := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: requests,
	}

	resp, err := srv.Spreadsheets.BatchUpdate(
		spreadsheetId,
		rb,
	).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("batch error failure: %w", err)
	}

	fmt.Printf("Status Code: %d\n", resp.HTTPStatusCode)
	return nil
}

func main() {

	githubSection := section.New(
		"GitHub commands",
		// section.Command(
		// 	"readmes",
		// 	"Save starred repo READMEs",
		// 	readmes,
		// ),
		section.Command(
			"stats",
			"Save starred repo information",
			stats,
			command.Flag(
				"--format",
				"Output format",
				value.StringEnum("csv", "jsonl", "zinc"),
				flag.Default("csv"),
				flag.Required(),
			),
			command.Flag(
				"--date-format",
				"Datetime output format. See https://github.com/lestrrat-go/strftime for details. If not passed, the GitHub default is RFC 3339. Consider using '%b %d, %Y' for csv format",
				value.String,
			),
			command.Flag(
				"--zinc-index-name",
				"Only valid for --format zinc.",
				value.String,
				flag.Default("starghaze"),
			),
			command.Flag(
				"--include-readmes",
				"Search for README.md.",
				value.Bool,
				flag.Default("false"),
			),
		),
		section.Flag(
			"--max-pages",
			"Max number of pages to fetch",
			value.Int,
			flag.Default("1"),
			flag.Required(),
		),
		section.Flag(
			"--output",
			"output file. Prints to stdout if not passed",
			value.Path,
		),
		section.Flag(
			"--page-size",
			"Number of starred repos in page",
			value.Int,
			flag.Default("100"),
			flag.Required(),
		),
		section.Flag(
			"--timeout",
			"Timeout for a run. Use https://pkg.go.dev/time#Duration to build it",
			value.Duration,
			flag.Default("10m"),
			flag.Required(),
		),
		section.Flag(
			"--token",
			"Github PAT",
			value.String,
			flag.EnvVars("STARGHAZE_GITHUB_TOKEN", "GITHUB_TOKEN"),
			flag.Required(),
		),
	)

	gsheetsSection := section.New(
		"Google Sheets commands",
		section.Command(
			"open",
			"Open spreadsheet in browser",
			gSheetsOpen,
		),
		section.Command(
			"upload",
			"Upload CSV to Google Sheets. This will overwrite whatever is in the spreadsheet",
			gSheetsUpload,
			command.Flag(
				"--csv-path",
				"CSV file to upload",
				value.Path,
				flag.Required(),
			),
			command.Flag(
				"--timeout",
				"Timeout for a run. Use https://pkg.go.dev/time#Duration to build it",
				value.Duration,
				flag.Default("10m"),
				flag.Required(),
			),
		),
		section.Flag(
			"--sheet-id",
			"ID For the particulare sheet. Viewable from `gid` URL param",
			value.Int,
			flag.EnvVars("STARGHAZE_SHEET_ID"),
			flag.Required(),
		),
		section.Flag(
			"--spreadsheet-id",
			"ID for the whole spreadsheet. Viewable from URL",
			value.String,
			flag.EnvVars("STARGHAZE_SPREADSHEET_ID"),
			flag.Required(),
		),
	)

	app := warg.New(
		"starghaze",
		section.New(
			"Save GitHub Starred Repos",
			section.ExistingSection("github", githubSection),
			section.ExistingSection("gsheets", gsheetsSection),
			section.Command(
				"version",
				"Print version",
				printVersion,
			),
		),
	)
	app.MustRun(os.Args, os.LookupEnv)
}
