Thanks to https://github.com/yks0000/starred-repo-toc for the inspiration and most of the GitHub code.

https://blog.carlmjohnson.net/post/2021/how-to-use-go-embed/ for go:embed stuff

# Outline

All commands should migrate the db

GITHUB_TOKEN='' starghaze init --user bkane

- add/update stargazers from API - INSERT INTO ON CONFLICT UPDATE

starghaze topics add
starghaze commit-info add
starghaze query --query 'SELECT * FROM starred'

# Mergestat - ABANDONED

use mergestat with SQLite3 to do all of this and then GitHub Pages or netlify to make a datatable with it.

https://docs.mergestat.com/reference/github-tables

installed with brew.

Making a new token with https://github.com/settings/tokens

```
echo "SELECT name_with_owner, url FROM github_starred_repos('bbkane') LIMIT 30;" | mergestat --format csv
```

And... there's no way to get tags for starred repos like this - Back to the other method
