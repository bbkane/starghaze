# stargaze

Save information about starred GitHub repos to a CSV or JSON file, and upload CSVs to Google Sheets!

**Click here to see [My GitHub Stars Google Sheet](https://docs.google.com/spreadsheets/d/15AXUtql31P62zxvEnqxNnb8ZcCWnBUYpROAsrtAhOV0/edit?usp=sharing)**

![star-count-over-time.png](./star-count-over-time.png)

Thanks to https://github.com/yks0000/starred-repo-toc for the inspiration!

GraphQL and Google Sheets auth notes at [./dev_notes.md](./dev_notes.md)

## Install

- Homebrew: `brew install bbkane/tap/starghaze`
- Download Mac/Linux/Windows executable: [GitHub releases](https://github.com/bbkane/starghaze/releases)
- Go: `go install github.com/bbkane/starghaze@latest`
- Build with [goreleaser](https://goreleaser.com/) after cloning: `goreleaser --snapshot --skip-publish --rm-dist`

## TODO

- `readmes` command to download READMEs for starred repos,
- zinc command to push to [zinc](https://github.com/prabhatsharma/zinc) and see if I like zinc
