package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/bbkane/warg/flag"
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

type Query struct {
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

func githubStarsDownload(pf flag.PassedFlags) error {
	token := pf["--token"].(string)
	pageSize := pf["--page-size"].(int)
	maxPages := pf["--max-pages"].(int)
	timeout := pf["--timeout"].(time.Duration)
	includeReadmes := pf["--include-readmes"].(bool)
	maxLanguages := pf["--max-languages"].(int)
	maxRepoTopics := pf["--max-repo-topics"].(int)

	var afterPtr *string = nil
	afterStr, afterExists := pf["--after-cursor"].(string)
	if afterExists {
		afterPtr = &afterStr
	}

	outputPath := pf["--output"].(string)
	// https://pkg.go.dev/os?utm_source=gopls#pkg-constants
	// return error if the file exists - NOTE: this kind of screws with any plans to append
	fp, err := os.OpenFile(outputPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0666)
	if err != nil {
		return fmt.Errorf("file open err: %w", err)
	}
	defer fp.Close()

	buf := bufio.NewWriter(fp)
	defer buf.Flush()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	httpClient := oauth2.NewClient(ctx, src)
	client := githubv4.NewClient(httpClient)

	var query Query

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
