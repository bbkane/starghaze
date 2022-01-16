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

## Save Stars to Google Sheets

### Download Star Info

```bash
GITHUB_TOKEN=my_github_token starghaze download \
	--include-readmes true \
	--output stars.jsonl
```

### Format Downloaded Stars as CSV

````bash
starghaze format \
	--input stars.jsonl \
	--format csv \
	--include-readmes false \
	--output stars.csv
````

### Upload CSV to Google Sheets

```bash
GOOGLE_APPLICATION_CREDENTIALS=/path/to/keys.json starghaze gsheets upload \
    --csv-path stars.csv \
    --sheet-id 0 \
    --spreadsheet-id 15AXUtql31P62zxvEnqxNnb8ZcCWnBUYpROAsrtAhOV0 \
    --timeout 30s
```

## Save Stars to [Zinc](https://github.com/prabhatsharma/zinc)

### Download Star Info

```bash
GITHUB_TOKEN=my_github_token starghaze download \
	--include-readmes true \
	--output stars.jsonl
```

### Format Downloaded Stars as Zinc

```bash
starghaze format \
	--include-readmes true \
	--input stars.jsonl \
	--format zinc \
	--output stars.zinc
```

### Upload to Zinc

Using default settings - See [Zinc repo](https://github.com/prabhatsharma/zinc) for more details.

```bash
curl http://localhost:4080/api/_bulk -i -u admin:Complexpass#123 --data-binary "@stars.zinc"
```
