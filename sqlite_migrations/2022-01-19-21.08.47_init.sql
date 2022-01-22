CREATE TABLE Repo (
    id INTEGER PRIMARY KEY NOT NULL,
    StarredAt TEXT NOT NULL,
    Description TEXT,
    HomepageURL TEXT,
    NameWithOwner TEXT NOT NULL,
    Readme TEXT,
    PushedAt TEXT NOT NULL,
    StargazerCount INTEGER NOT NULL,
    UpdatedAt TEXT NOT NULL,
    Url TEXT
) STRICT;

CREATE TABLE Language (
    id INTEGER PRIMARY KEY NOT NULL,
    Name TEXT,
    UNIQUE(Name)
) STRICT;

CREATE TABLE Language_Repo (
    Language_id INTEGER NOT NULL,
    Repo_id INTEGER NOT NULL,
    Size INTEGER NOT NULL,
    FOREIGN KEY (Language_id) REFERENCES Language(id) ON DELETE CASCADE,
    FOREIGN KEY (Repo_id) REFERENCES Repo(id) ON DELETE CASCADE,
    PRIMARY KEY (Language_id, Repo_id)
) STRICT;

CREATE TABLE Topic (
    id INTEGER PRIMARY KEY NOT NULL,
    Name TEXT NOT NULL,
    Url TEXT NOT NULL,
    UNIQUE(Name),
    UNIQUE(Url)
) STRICT;

CREATE TABLE Repo_Topic (
    Repo_id INTEGER NOT NULL,
    Topic_id INTEGER NOT NULL,
    FOREIGN KEY(Repo_id) REFERENCES Repo(id) ON DELETE CASCADE,
    FOREIGN KEY(Topic_id) REFERENCES Topic(id) ON DELETE CASCADE,
    PRIMARY KEY (Repo_id, Topic_id)
) STRICT;
