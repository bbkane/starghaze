package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strconv"

	"go.bbkane.com/warg/command"
	"go.bbkane.com/warg/help"
	_ "modernc.org/sqlite"
)

type searchResult struct {
	Link           string
	StarredAt      string
	StargazerCount int
	Description    string
}

func search(ctx command.Context) error {
	dsn := ctx.Flags["--sqlite-dsn"].(string)
	limit := ctx.Flags["--limit"].(int)
	term := ctx.Flags["--term"].(string)

	col, err := help.ConditionallyEnableColor(ctx.Flags, os.Stdout)
	if err != nil {
		return fmt.Errorf("error enabling color: %w", err)
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return fmt.Errorf("db open error: %s: %w", dsn, err)
	}

	query := `
  SELECT
	'https://github.com/' || NameWithOwner AS link,
	StarredAt,
	StargazerCount,
	CASE
	  WHEN Description = '' THEN SUBSTR(Readme, 0, 50) || '...'
	  ELSE Description
	END AS Description
  FROM
	Repo_fts
  WHERE
	Repo_fts MATCH ?
  ORDER BY
	RANK
  LIMIT
	?
`

	rows, err := db.QueryContext(context.Background(), query, term, limit)
	if err != nil {
		return fmt.Errorf("error querying: %w", err)
	}
	defer rows.Close()

	var s searchResult
	for rows.Next() {
		err := rows.Scan(&s.Link, &s.StarredAt, &s.StargazerCount, &s.Description)
		if err != nil {
			return fmt.Errorf("error scanning result: %w", err)
		}

		fmt.Println(col.Add(col.Bold, "Link") + ": " + s.Link)
		fmt.Println(col.Add(col.Bold+col.FgGreenBright, "StarredAt") + ": " + s.StarredAt)
		fmt.Println(col.Add(col.Bold+col.FgCyanBright, "StargazerCount") + ": " + strconv.Itoa(s.StargazerCount))
		fmt.Println(col.Add(col.Bold+col.FgYellowBright, "Description") + ": " + s.Description)
		fmt.Println()
	}
	err = rows.Err()
	if err != nil {
		return fmt.Errorf("error at end of scan: %w", err)
	}

	return nil
}
