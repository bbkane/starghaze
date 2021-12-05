package main

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"os"
	"runtime/debug"

	"gopkg.in/gorp.v1"
	_ "modernc.org/sqlite"

	"github.com/google/go-github/v41/github"
	migrate "github.com/rubenv/sql-migrate"
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

//go:embed migrations
var migrations embed.FS

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

func init_(pf flag.PassedFlags) error {
	dbPath := pf["--db"].(string)
	token := pf["--token"].(string)

	// HACK to use modernc/sqlite instead of mattn's CGO version
	migrate.MigrationDialects["sqlite"] = gorp.SqliteDialect{}

	migrations := &migrate.EmbedFileSystemMigrationSource{
		FileSystem: migrations,
		Root:       "migrations",
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("open error: %s: %w", dbPath, err)
	}

	n, err := migrate.Exec(db, "sqlite", migrations, migrate.Up)
	if err != nil {

		return fmt.Errorf("migrate error: %s: %w", dbPath, err)
	}
	fmt.Printf("Applied %d migrations!\n", n)

	// -- github API

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	starred := []*github.StarredRepository{}
	starredOpt := &github.ActivityListStarredOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}
	for {
		repos, resp, err := client.Activity.ListStarred(
			ctx,
			"", // current user
			starredOpt,
		)

		if err != nil {
			return fmt.Errorf("get-starred error: %w", err)
		}

		starred = append(starred, repos...)
		if resp.NextPage == 0 {
			break
		}
		starredOpt.Page = resp.NextPage
	}

	fmt.Printf("len starred: %v\n", len(starred))

	return nil
}

func queryGH(pf flag.PassedFlags) error {
	token := pf["--token"].(string)

	ctx := context.Background()
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	httpClient := oauth2.NewClient(ctx, src)

	client := githubv4.NewClient(httpClient)

	var query struct {
		Viewer struct {
			Login     githubv4.String
			CreatedAt githubv4.DateTime
		}
	}

	err := client.Query(ctx, &query, nil)
	if err != nil {
		return fmt.Errorf("query err: %w", err)
	}
	fmt.Println("    Login:", query.Viewer.Login)
	fmt.Println("CreatedAt:", query.Viewer.CreatedAt)
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
		"--db": flag.New(
			"Path to db file",
			value.Path,
			flag.Default("starghaze.db"),
			flag.Required(),
		),
	}
	app := warg.New(
		"starghaze",
		section.New(
			"Save GitHub Starred Repos to a SQLite3 DB",
			section.Command(
				"init",
				"Download stargazer links to db",
				init_,
				command.ExistingFlags(
					sharedFlags,
				),
			),
			section.Command(
				"query",
				"Download stargazer links to db",
				queryGH,
				command.ExistingFlags(
					sharedFlags,
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
