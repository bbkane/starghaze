# Learning GraphQL

Instead of making a couple thousand REST calls, I can make a GraphQL call:

https://docs.github.com/en/graphql

https://docs.github.com/en/graphql/overview/explorer

https://github.blog/2016-09-14-the-github-graphql-api/ has a starred REPO example

```
labels(first:2) {
  edges {
    node {
      description
      createdAt
      id
      name
    }
  }
}
```

This appears to refer to issue labels - I don't think I care about it

I don't think I need whatever `object` is :)

So far I have:


```
{
  viewer { login
    starredRepositories(first: 2) {
      totalCount
      edges {
        cursor
        node {
          name
          stargazers {
            totalCount
          }
          description
          homepageUrl
          id
          name
          nameWithOwner
          pushedAt
          repositoryTopics(first: 10) {
            nodes {
              url
              topic {
                name
              }
            }
          }
          updatedAt
          url
        }
      }
    }
  }
  rateLimit {
    limit
    cost
    remaining
    resetAt
  }
}
```

Let's read more about pagination

https://www.apollographql.com/blog/graphql/explaining-graphql-connections/ - explains how to page (first: 2, after: *id*)

https://javascript.plainenglish.io/graphql-pagination-using-edges-vs-nodes-in-connections-f2ddb8edffa0 - it looks like `nodes` is a shorthand in case you don't need information from the edge relationship - it also looks like it doesn't support pagination? yes - because that's a property of the edge, not the nodes


```
{
  viewer {
    login
    starredRepositories(first: 2, orderBy: {field:STARRED_AT, direction:DESC}, after:"Y3Vyc29yOnYyOpK5MjAyMS0xMi0wNFQyMToxOTo1Ny0wODowMM4STnrd") {
      totalCount
      edges {
        cursor
        node {
          nameWithOwner
        }
      }
    }
  }
}
```

So you include a cursor, then each note gets one. Grab the last one and use the after field in the next query.

On to https://github.com/shurcooL/githubv4 to make this work in Go - TODO: port to Elm? I can cache responses client side

https://docs.github.com/en/graphql/reference/queries

https://docs.github.com/en/graphql/guides/forming-calls-with-graphql

https://graphql.org/learn/pagination/ explains the edges and cursor pagination really well. - can also get pageInfo.hasNextPage

It looks like github's pageInfo does include endCursor to get to the next page - let's try it

```
{
  viewer {
    starredRepositories(first: 2, orderBy: {field:STARRED_AT, direction:DESC}, after:"Y3Vyc29yOnYyOpK5MjAyMS0xMi0wNFQxMjo1ODo1OC0wODowMM4STfEt") {
      nodes {
        nameWithOwner
      }
      pageInfo {
        endCursor
        hasNextPage
      }
    }
  }
}
```

And that works
