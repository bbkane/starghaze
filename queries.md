Interrogating my data!

# 10 most popular languages by repo

```bash
$ sqlite3 starghaze.db '
SELECT
    l.Name ,
    COUNT(lr.Language_id) as Repo_Count
FROM
    Language_Repo lr JOIN Language l ON lr.Language_id = l.id
GROUP BY Language_id
ORDER BY Repo_Count DESC
LIMIT 10
'
-- Loading resources from /Users/bbkane/.sqliterc
┌────────────┬────────────┐
│    Name    │ Repo_Count │
├────────────┼────────────┤
│ Shell      │ 939        │
│ JavaScript │ 617        │
│ HTML       │ 598        │
│ Python     │ 540        │
│ Makefile   │ 519        │
│ CSS        │ 432        │
│ Dockerfile │ 403        │
│ Go         │ 367        │
│ C          │ 305        │
│ C++        │ 230        │
└────────────┴────────────┘
```

# 10 most popular languages by lines

Note that these are lines in the repo, which includes generated code

```
starghaze.db> SELECT l.Name language_name, SUM(lr.Size) AS lines FROM `Language` l JOIN `Language_Repo` lr ON l.id = lr.`Language_id` GROUP BY l.`Name` ORDER BY lines DESC LIMIT 10;
language_name     lines
C++               942739556
Jupyter Notebook  804633765
Python            518129708
C                 442379463
Go                366579078
TypeScript        323614022
JavaScript        319470060
Java              306837166
HTML              196157314
C#                193605672
```

# What Repo has the most C++ code?

```
starghaze.db> SELECT l.Name AS language_name, lr.Size, "https://github.com/" || r.NameWithOwner AS link FROM `Language` l JOIN `Language_Repo` lr ON l.id = lr.`Language_id` JOIN Repo r ON lr.`Repo_id` = r.id WHERE l.`Name` = 'C++' ORDER BY lr.`Size` DESC LIMIT 10;
language_name    Size       link
C++              119581200  https://github.com/bloomberg/bde
C++              89578048   https://github.com/microsoft/service-fabric
C++              57547160   https://github.com/duckdb/duckdb
C++              40643968   https://github.com/mapsme/omim
C++              36784149   https://github.com/godotengine/godot
C++              31984347   https://github.com/arangodb/arangodb
C++              29233152   https://github.com/vespa-engine/vespa
C++              23346282   https://github.com/organicmaps/organicmaps
C++              23113037   https://github.com/SerenityOS/serenity
C++              20847917   https://github.com/Z3Prover/z3
10 rows in set
``````

# How Many Repos do I star per month

```
$ sqlite3 -cmd '.mode tabs' starghaze.db '
SELECT strftime("%Y-%m", r.StarredAt) as month, COUNT(r.NameWithOwner) as [count] FROM `Repo` r GROUP BY month ORDER BY month ASC;
' | tablegraph --firstline timechart
```

# How have my topics changed over time?

Let's see count by topic by month? Something like:

2020-01 C++ 0
2020-02 C++ 10
2020-02 Python 10

```
starghaze.db> SELECT strftime("%Y-%m", r.StarredAt) as month , l.Name, COUNT(r.NameWithOwner) as name_count FROM `Repo` r JOIN `
              Language_Repo` lr ON r.id = lr.`Repo_id` JOIN `Language` l ON lr.`Language_id` = l.id GROUP BY month ORDER BY mont
              h ASC;
```
