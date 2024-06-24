CREATE TABLE IF NOT EXISTS team(
    teamid INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
    name TEXT NOT NULL UNIQUE,
    planid INTEGER DEFAULT 0
);

CREATE TABLE IF NOT EXISTS teampermission(
    userid TEXT NOT NULL,
    teamid INTEGER NOT NULL,
    level INTEGER NOT NULL,
    PRIMARY KEY (userid, teamid)
);

CREATE TABLE IF NOT EXISTS project(
    projectid INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
    title TEXT NOT NULL,
    teamid INTEGER NOT NULL,
    FOREIGN KEY(teamid) REFERENCES team(teamid),
    UNIQUE(teamid, title)
);

CREATE TABLE IF NOT EXISTS projectpermission(
    userid TEXT NOT NULL,
    projectid INTEGER NOT NULL,
    level INTEGER NOT NULL,
    PRIMARY KEY (userid, projectid)
);

CREATE TABLE IF NOT EXISTS block(
    hash TEXT PRIMARY KEY NOT NULL,
    s3key TEXT NOT NULL,
    size INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS 'commit'(
    commitid INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
    projectid INTEGER NOT NULL,
    userid TEXT NOT NULL,
    comment TEXT NOT NULL DEFAULT "",
    numfiles INTEGER NOT NULL,
    cno INTEGER,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL,
    FOREIGN KEY(projectid) REFERENCES project(projectid)
);

CREATE TABLE IF NOT EXISTS file(
    projectid INTEGER NOT NULL,
    path TEXT NOT NULL,
    locked INTEGER NOT NULL DEFAULT 0,
    lockownerid TEXT,
    PRIMARY KEY(projectid, path),
    FOREIGN KEY(projectid) REFERENCES project(projectid),
    UNIQUE(projectid, path)
);

CREATE TABLE IF NOT EXISTS filerevision(
    frid INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
    projectid INTEGER NOT NULL,
    path TEXT NOT NULL,
    commitid INTEGER NOT NULL,
    hash TEXT NOT NULL,
    frno INTEGER,
    changetype INTEGER NOT NULL,
    FOREIGN KEY(path) REFERENCES file(path),
    FOREIGN KEY(projectid) REFERENCES project(projectid),
    FOREIGN KEY(commitid) REFERENCES 'commit'(commitid),
    FOREIGN KEY(hash) REFERENCES block(hash)
);



CREATE TRIGGER IF NOT EXISTS commitnumber AFTER INSERT ON 'commit' BEGIN
UPDATE 'commit' SET cno = (SELECT COUNT(*) FROM 'commit' WHERE projectid = NEW.projectid) WHERE commitid = NEW.commitid;
END;

CREATE TRIGGER IF NOT EXISTS frnumber AFTER INSERT ON filerevision BEGIN
UPDATE filerevision SET frno = (SELECT COUNT(*) FROM filerevision WHERE path = NEW.path AND projectid = NEW.projectid) WHERE frid = NEW.frid;
END;
