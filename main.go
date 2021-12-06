package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"strconv"
	"strings"

	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"

	"github.com/bbkane/warg"
	"github.com/bbkane/warg/command"
	"github.com/bbkane/warg/flag"
	"github.com/bbkane/warg/section"
	"github.com/bbkane/warg/value"
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

func queryGH(pf flag.PassedFlags) error {
	token := pf["--token"].(string)
	pageSize := pf["--page-size"].(int)
	maxPages := pf["--max-pages"].(int)
	output := pf["--output"].(string)
	format := pf["--format"].(string)

	ctx := context.Background() // TODO: paramaterize
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

	var p Printer

	switch format {
	case "csv":
		p = NewCSVPrinter(fp)
	case "json":
		p = NewJSONPrinter(fp)
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

func main() {
	app := warg.New(
		"starghaze",
		section.New(
			"Save GitHub Starred Repos",
			section.Command(
				"query",
				"Save the starred Repo information",
				queryGH,

				command.Flag(
					"--token",
					"Github PAT",
					value.String,
					flag.EnvVars("STARGHAZE_GITHUB_TOKEN", "GITHUB_TOKEN"),
					flag.Required(),
				),
				command.Flag(
					"--page-size",
					"Number of starred repos in page",
					value.Int,
					flag.Default("2"),
					flag.Required(),
				),
				command.Flag(
					"--max-pages",
					"Max number of pages to fetch",
					value.Int,
					flag.Default("2"),
					flag.Required(),
				),
				command.Flag(
					"--output",
					"output file",
					value.Path,
					flag.Default("/dev/stdout"),
					flag.Required(),
				),
				command.Flag(
					"--format",
					"Output format",
					value.StringEnum("csv", "json"),
					flag.Default("csv"),
					flag.Required(),
				),
			),
			section.Command(
				"version",
				"Print version",
				printVersion,
			),
		),
	)
	app.MustRun(os.Args, os.LookupEnv)
}
