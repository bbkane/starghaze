package main

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"

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

func queryGH(pf flag.PassedFlags) error {
	token := pf["--token"].(string)
	pageSize := pf["--page-size"].(int)
	maxPages := pf["--max-pages"].(int)

	ctx := context.Background()
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	httpClient := oauth2.NewClient(ctx, src)

	client := githubv4.NewClient(httpClient)

	type starredRepository struct {
		Description      githubv4.String
		HomepageURL      githubv4.String
		NameWithOwner    githubv4.String
		PushedAt         githubv4.DateTime
		RepositoryTopics struct {
			Nodes []struct {
				URL   githubv4.String
				Topic struct {
					Name githubv4.String
				}
			}
		} `graphql:"repositoryTopics(first: 10)"`
		Stargazers struct {
			TotalCount githubv4.Int
		}
		UpdatedAt githubv4.DateTime
		Url       githubv4.String
	}

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
		fmt.Printf("NameWithOwner: %s\n", query.Viewer.StarredRepositories.Nodes[0].NameWithOwner)

		if !query.Viewer.StarredRepositories.PageInfo.HasNextPage {
			break
		}
		variables["starredRepositoriesCursor"] = githubv4.NewString(query.Viewer.StarredRepositories.PageInfo.EndCursor)
	}

	return nil

}

func main() {
	sharedFlags := flag.FlagMap{
		"--token": flag.New(
			"Github PAT",
			value.String,
			flag.EnvVars("STARGHAZE_GITHUB_TOKEN", "GITHUB_TOKEN"),
			flag.Required(),
		),
	}
	app := warg.New(
		"starghaze",
		section.New(
			"Save GitHub Starred Repos",
			section.Command(
				"query",
				"Save the starred Repo information",
				queryGH,
				command.ExistingFlags(
					sharedFlags,
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
