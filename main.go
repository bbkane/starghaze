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
	Line(*starredRepository) error
	Flush() error
}

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

func (p *JSONPrinter) Line(sR *starredRepository) error {
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

type CSVPrinter struct {
	writer *csv.Writer
}

func NewCSVPrinter(w io.Writer) *CSVPrinter {
	return &CSVPrinter{
		writer: csv.NewWriter(w),
	}
}

func (p *CSVPrinter) Header() error {
	err := p.writer.Write([]string{
		"Description",
		"HomepageURL",
		"NameWithOwner",
		"PushedAt",
		"StarGazerCount",
		"Topics",
		"UpdatedAt",
		"Url",
	})
	if err != nil {
		return fmt.Errorf("CSV header err: %w", err)
	}
	return nil
}

func (p *CSVPrinter) Line(sr *starredRepository) error {
	topics := []string{}
	for i := range sr.RepositoryTopics.Nodes {
		topics = append(topics, sr.RepositoryTopics.Nodes[i].Topic.Name)
	}
	starGazerCount := strconv.Itoa(sr.Stargazers.TotalCount)
	err := p.writer.Write([]string{
		sr.Description,
		sr.HomepageURL,
		sr.NameWithOwner,
		sr.PushedAt,
		starGazerCount,
		strings.Join(topics, " "),
		sr.UpdatedAt,
		sr.Url,
	})
	if err != nil {
		return fmt.Errorf("CSV write err: %w", err)
	}
	return nil
}

func (p *CSVPrinter) Flush() error {
	p.writer.Flush()
	return p.writer.Error()
}

type starredRepository struct {
	Description      string
	HomepageURL      string
	NameWithOwner    string
	PushedAt         string
	RepositoryTopics struct {
		Nodes []struct {
			URL   string
			Topic struct {
				Name string
			}
		}
	} `graphql:"repositoryTopics(first: 10)"`
	Stargazers struct {
		TotalCount int
	}
	UpdatedAt string
	Url       string
}

func info(pf flag.PassedFlags) error {
	token := pf["--token"].(string)
	pageSize := pf["--page-size"].(int)
	maxPages := pf["--max-pages"].(int)
	output := pf["--output"].(string)
	format := pf["--format"].(string)
	timeout := pf["--timeout"].(time.Duration)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	httpClient := oauth2.NewClient(ctx, src)

	client := githubv4.NewClient(httpClient)

	fp, err := os.Create(output)
	if err != nil {
		return fmt.Errorf("file open err: %w", err)
	}
	defer fp.Close()

	buf := bufio.NewWriter(fp)
	defer buf.Flush()

	var p Printer

	switch format {
	case "csv":
		p = NewCSVPrinter(buf)
	case "json":
		p = NewJSONPrinter(buf)
	default:
		return fmt.Errorf("unknown output format: %s", format)
	}

	err = p.Header()
	if err != nil {
		return err
	}

	defer p.Flush()

	var query struct {
		Viewer struct {
			StarredRepositories struct {
				Nodes    []starredRepository
				PageInfo struct {
					EndCursor   githubv4.String
					HasNextPage githubv4.Boolean
				}
			} `graphql:"starredRepositories(first: $starredRepositoryPageSize, orderBy: {field:STARRED_AT, direction:DESC}, after: $starredRepositoriesCursor)"`
		}
	}

	variables := map[string]interface{}{
		"starredRepositoriesCursor": (*githubv4.String)(nil),
		"starredRepositoryPageSize": githubv4.NewInt(githubv4.Int(pageSize)),
	}

	for i := 0; i < maxPages; i++ {
		err := client.Query(ctx, &query, variables)
		if err != nil {
			return fmt.Errorf("query err: %w", err)
		}

		for i := range query.Viewer.StarredRepositories.Nodes {
			p.Line(&query.Viewer.StarredRepositories.Nodes[i])
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
	app := warg.New(
		"starghaze",
		section.New(
			"Save GitHub Starred Repos",
			section.Command(
				"info",
				"Save starred repo information",
				info,
				command.Flag(
					"--format",
					"Output format",
					value.StringEnum("csv", "json"),
					flag.Default("csv"),
					flag.Required(),
				),
				command.Flag(
					"--max-pages",
					"Max number of pages to fetch",
					value.Int,
					flag.Default("1"),
					flag.Required(),
				),
				command.Flag(
					"--output",
					"output file",
					value.Path,
					// TODO: this won't workk on Windows
					flag.Default("/dev/stdout"),
					flag.Required(),
				),
				command.Flag(
					"--page-size",
					"Number of starred repos in page",
					value.Int,
					flag.Default("100"),
					flag.Required(),
				),
				command.Flag(
					"--timeout",
					"Timeout for a run. Use https://pkg.go.dev/time#Duration to build it",
					value.Duration,
					flag.Default("10m"),
					flag.Required(),
				),
				command.Flag(
					"--token",
					"Github PAT",
					value.String,
					flag.EnvVars("STARGHAZE_GITHUB_TOKEN", "GITHUB_TOKEN"),
					flag.Required(),
				),
			),
			section.Command(
				"version",
				"Print version",
				printVersion,
			),

			section.Section(
				"gsheets",
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
					flag.Default("0"),
					flag.Required(),
				),
				section.Flag(
					"--spreadsheet-id",
					"ID for the whole spreadsheet. Viewable from URL",
					value.String,
					flag.Default("15AXUtql31P62zxvEnqxNnb8ZcCWnBUYpROAsrtAhOV0"),
					flag.Required(),
				),
			),
		),
	)
	app.MustRun(os.Args, os.LookupEnv)
}
