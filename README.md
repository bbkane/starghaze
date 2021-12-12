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

- Google Sheets integration
  - add open command to open url
- `readmes` command

# Google Sheets Auth Setup

Making a project and a service account so my app has permission to upload to a Google Sheet

Open CloudTerminal from the button on the top right of the  [console](https://console.cloud.google.com/home/dashboard) or [directly](https://shell.cloud.google.com/)

Use “gcloud config set project [PROJECT_ID]” to change to a different project.

## [Create a Project](https://cloud.google.com/sdk/gcloud/reference/projects/create)

```
$ gcloud projects create starghaze
Create in progress for [https://cloudresourcemanager.googleapis.com/v1/projects/starghaze].
Waiting for [operations/cp.7358087792659517101] to finish...done.
Enabling service [cloudapis.googleapis.com] on project [starghaze]...
Operation "operations/acf.p2-824192962629-dbb00832-15b2-480b-91dc-cf3ceb5220e3" finished successfully.
```

## Change to `starghaze`

```
$ gcloud config set project starghaze
Updated property [core/project].
```

## [Enable API Access for `starghaze`](https://cloud.google.com/sdk/gcloud/reference/services/enable)

```
$ gcloud services enable sheets.googleapis.com
Operation "operations/acf.p2-824192962629-37a88c62-c546-46b5-9087-9fa97a68c58c" finished successfully.
```

## [Create a Service Account](https://cloud.google.com/docs/authentication/production#command-line)

```
$ gcloud iam service-accounts create starghaze-sa
Created service account [starghaze-sa].
```

```
$ gcloud projects add-iam-policy-binding starghaze --member="serviceAccount:starghaze-sa@starghaze.iam.gserviceaccount.com" --role="roles/owner"
Updated IAM policy for project [starghaze].
bindings:
- members:
  - serviceAccount:starghaze-sa@starghaze.iam.gserviceaccount.com
  - user:<my email>
  role: roles/owner
etag: BwXSy6ZN_FM=
version: 1
```

```
$ gcloud iam service-accounts keys create starghaze-sa-keys.json --iam-account=starghaze-sa@starghaze.iam.gserviceaccount.com
created key [6bd47f6450cc29401df0a5e4632f32e68dca14d3] of type [json] as [starghaze-sa-keys.json] for [starghaze-sa@starghaze.iam.gserviceaccount.com]
```

## [Download Keys](https://cloud.google.com/shell/docs/uploading-and-downloading-files)

```
$ cloudshell download starghaze-sa-keys.json
```

## Create and Share [a Google Sheet](https://docs.google.com/spreadsheets/d/15AXUtql31P62zxvEnqxNnb8ZcCWnBUYpROAsrtAhOV0/edit#gid=0) with `stargaze-sa`

![image-20211210064316079](share-starghaze-stats-gsheet.png)

---

```
$ GOOGLE_APPLICATION_CREDENTIALS=starghaze-sa-keys.json go run . gsheets upload
No data found.
```
