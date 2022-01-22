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

-- fts
-- https://kimsereylam.com/sqlite/2020/03/06/full-text-search-with-sqlite.html
-- https://www.sqlite.org/fts5.html#external_content_and_contentless_tables

CREATE VIRTUAL TABLE Repo_fts USING fts5(
    -- indexed fields
    Description,
    HomepageURL,
    Readme,
    NameWithOwner,
    -- unindexed fields
    StarredAt UNINDEXED,
    PushedAt UNINDEXED,
    StargazerCount UNINDEXED,
    UpdatedAt UNINDEXED,
    -- special args
    content='Repo',
    content_rowid='id'
);

-- Triggers to keep the FTS index up to date.
CREATE TRIGGER Repo_ai AFTER INSERT ON Repo BEGIN
    INSERT INTO Repo_fts(
        rowid,
        Description,
        HomepageURL,
        Readme,
        NameWithOwner
    ) VALUES (
        new.id,
        new.Description,
        new.HomepageURL,
        new.Readme,
        new.NameWithOwner
    );
END;
CREATE TRIGGER Repo_ad AFTER DELETE ON Repo BEGIN
    INSERT INTO Repo_fts(
        Repo_fts,
        rowid,
        Description,
        HomepageURL,
        Readme,
        NameWithOwner
    ) VALUES (
        'delete',
        old.id,
        old.Description,
        old.HomepageURL,
        old.Readme,
        old.NameWithOwner
    );
END;
CREATE TRIGGER Repo_au AFTER UPDATE ON Repo BEGIN
    INSERT INTO Repo_fts(
        Repo_fts,
        rowid,
        Description,
        HomepageURL,
        Readme,
        NameWithOwner
    ) VALUES(
        'delete',
        old.id,
        old.Description,
        old.HomepageURL,
        old.Readme,
        old.NameWithOwner
    );
    INSERT INTO Repo_fts(
        rowid,
        Description,
        HomepageURL,
        Readme,
        NameWithOwner
    ) VALUES (
        new.id,
        new.Description,
        new.HomepageURL,
        new.Readme,
        new.NameWithOwner
    );
END;