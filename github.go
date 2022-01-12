package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/bbkane/warg/flag"
	"github.com/lestrrat-go/strftime"
	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

type starredRepositoryEdge struct {
	StarredAt formattedDate
	Node      struct {
		Description string
		HomepageURL string
		Languages   struct {
			Edges []struct {
				Size int
				Node struct {
					Name string
				}
			}
		} `graphql:"languages(first: $maxLanguages)"`
		NameWithOwner string
		Object        struct {
			Blob struct {
				Text string
			} `graphql:"... on Blob"`
		} `graphql:"object(expression: \"HEAD:README.md\") @include(if: $includeREADME)"`
		PushedAt         formattedDate
		RepositoryTopics struct {
			Nodes []struct {
				URL   string
				Topic struct {
					Name string
				}
			}
		} `graphql:"repositoryTopics(first: $maxRepositoryTopics)"`
		StargazerCount int
		UpdatedAt      formattedDate
		Url            string
	}
}

func githubStarsDownload(pf flag.PassedFlags) error {
	token := pf["--token"].(string)
	pageSize := pf["--page-size"].(int)
	maxPages := pf["--max-pages"].(int)
	timeout := pf["--timeout"].(time.Duration)
	includeReadmes := pf["--include-readmes"].(bool)
	maxLanguages := pf["--max-languages"].(int)
	maxRepoTopics := pf["--max-repo-topics"].(int)

	var afterPtr *string = nil
	afterStr, afterExists := pf["--after"].(string)
	if afterExists {
		afterPtr = &afterStr
	}

	output, outputExists := pf["--output"].(string)
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
		"starredRepositoriesCursor": (*githubv4.String)(afterPtr),
		"starredRepositoryPageSize": githubv4.NewInt(githubv4.Int(pageSize)),
		"includeREADME":             githubv4.Boolean(includeReadmes),
		"maxLanguages":              githubv4.Int(maxLanguages),
		"maxRepositoryTopics":       githubv4.Int(maxRepoTopics),
	}

	for i := 0; i < maxPages; i++ {
		err := client.Query(ctx, &query, variables)
		if err != nil {
			return fmt.Errorf(
				"afterToken: %v , query err: %w",
				variables["starredRepositoriesCursor"],
				err,
			)
		}

		view, err := json.Marshal(&query)
		if err != nil {
			return fmt.Errorf("json marshall err: %w", err)
		}
		view = append(view, byte('\n'))
		_, err = buf.Write(view)
		if err != nil {
			return fmt.Errorf("file write err: %w", err)
		}

		if !query.Viewer.StarredRepositories.PageInfo.HasNextPage {
			break
		}
		variables["starredRepositoriesCursor"] = githubv4.NewString(query.Viewer.StarredRepositories.PageInfo.EndCursor)
	}
	return nil
}

func stats(pf flag.PassedFlags) error {
	token := pf["--token"].(string)
	pageSize := pf["--page-size"].(int)
	maxPages := pf["--max-pages"].(int)
	output, outputExists := pf["--output"].(string)
	format := pf["--format"].(string)
	timeout := pf["--timeout"].(time.Duration)
	includeReadmes := pf["--include-readmes"].(bool)
	zincIndexName := pf["--zinc-index-name"].(string)
	maxLanguages := pf["--max-languages"].(int)
	maxRepoTopics := pf["--max-repo-topics"].(int)

	dateFormatStr, dateFormatStrExists := pf["--date-format"].(string)
	var dateFormat *strftime.Strftime
	var err error

	if dateFormatStrExists {
		dateFormat, err = strftime.New(dateFormatStr)
		if err != nil {
			return fmt.Errorf("--date-format error: %w", err)
		}
	}

	var afterPtr *string = nil
	afterStr, afterExists := pf["--after"].(string)
	if afterExists {
		afterPtr = &afterStr
	}

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
		"starredRepositoriesCursor": (*githubv4.String)(afterPtr),
		"starredRepositoryPageSize": githubv4.NewInt(githubv4.Int(pageSize)),
		"includeREADME":             githubv4.Boolean(includeReadmes),
		"maxLanguages":              githubv4.Int(maxLanguages),
		"maxRepositoryTopics":       githubv4.Int(maxRepoTopics),
	}

	for i := 0; i < maxPages; i++ {
		err := client.Query(ctx, &query, variables)
		if err != nil {
			return fmt.Errorf(
				"afterToken: %v , query err: %w",
				variables["starredRepositoriesCursor"],
				err,
			)
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
