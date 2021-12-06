Save information about starred GitHub repos to a CSV or JSON file.

Thanks to https://github.com/yks0000/starred-repo-toc for the inspiration!

GraphQL notes at [./learning_graphql.md](./learning_graphql.md)

# Install

- Homebrew: `brew install bbkane/tap/starghaze`
- Download Mac/Linux/Windows executable: [GitHub releases](https://github.com/bbkane/starghaze/releases)
- Go: `go install github.com/bbkane/starghaze@latest`
- Build with [goreleaser](https://goreleaser.com/) after cloning: `goreleaser --snapshot --skip-publish --rm-dist`

# TODO

- get starred_at edge property. Unfortunately, this means I can't use the nodes property directly.

```
{
  viewer {
    login
    starredRepositories(first: 2, orderBy: {field:STARRED_AT, direction:DESC}, after:"Y3Vyc29yOnYyOpK5MjAyMS0xMi0wNFQxMjo1ODo1OC0wODowMM4STfEt") {
      totalCount
      edges {
        starredAt
        node {
          nameWithOwner
        }
      }
      pageInfo {
        endCursor
        hasNextPage
      }
    }
  }
}
```

- markdown table output
- customizable dates
- turn context into a duration flag
