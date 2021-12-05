package main

import (
	"database/sql"
	"embed"
	"fmt"
	"os"
	"runtime/debug"

	"gopkg.in/gorp.v1"
	_ "modernc.org/sqlite"

	migrate "github.com/rubenv/sql-migrate"

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

	return nil
}

func main() {
	sharedFlags := flag.FlagMap{
		"--token": flag.New(
			"Github PAT",
			value.String,
			flag.EnvVars("GITHUB_TOKEN"),
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
				"version",
				"Print version",
				printVersion,
			),
		),
	)
	app.MustRun(os.Args, os.LookupEnv)
}
