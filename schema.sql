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

CREATE TABLE IF NOT EXISTS permissiongroup(
    pgroupid INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
    teamid INTEGER NOT NULL,
    name TEXT NOT NULL,
    FOREIGN KEY(teamid) REFERENCES team(teamid)
);

CREATE TABLE IF NOT EXISTS pgmembership(
    pgroupid INTEGER NOT NULL,
    userid TEXT NOT NULL,
    PRIMARY KEY (pgroupid, userid),
    FOREIGN KEY(pgroupid) REFERENCES permissiongroup(pgroupid)
);

CREATE TABLE IF NOT EXISTS pgmapping(
    pgroupid INTEGER NOT NULL,
    projectid INTEGER NOT NULL,
    level INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (pgroupid, projectid),
    FOREIGN KEY(pgroupid) REFERENCES permissiongroup(pgroupid),
    FOREIGN KEY(projectid) REFERENCES project(projectid)
);

CREATE TABLE IF NOT EXISTS 'commit'(
    commitid INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
    projectid INTEGER NOT NULL,
    userid TEXT NOT NULL,
    comment TEXT NOT NULL,
    numfiles INTEGER NOT NULL,
    cno INTEGER,
    timestamp INTEGER DEFAULT (strftime('%s','now')) NOT NULL,
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
    filehash TEXT NOT NULL,
    changetype INTEGER NOT NULL,
    numchunks INTEGER NOT NULL,
    frno INTEGER,
    FOREIGN KEY(projectid) REFERENCES project(projectid),
    FOREIGN KEY(commitid) REFERENCES 'commit'(commitid)
);

CREATE TABLE IF NOT EXISTS chunk(
    chunkindex INTEGER NOT NULL,
    numchunks INTEGER NOT NULL,
    filehash TEXT NOT NULL,
    blockhash TEXT NOT NULL,
    blocksize INTEGER NOT NULL,
    PRIMARY KEY(chunkindex, blockhash),
    FOREIGN KEY(blockhash) REFERENCES block(blockhash),
    UNIQUE(filehash, chunkindex)
);

CREATE TABLE IF NOT EXISTS block(
    blockhash TEXT PRIMARY KEY NOT NULL,
    s3key TEXT NOT NULL,
    blocksize INTEGER NOT NULL
);


CREATE TRIGGER IF NOT EXISTS commitnumber AFTER INSERT ON 'commit' BEGIN
UPDATE 'commit' SET cno = (SELECT COUNT(*) FROM 'commit' WHERE projectid = NEW.projectid) WHERE commitid = NEW.commitid;
END;

CREATE TRIGGER IF NOT EXISTS frnumber AFTER INSERT ON filerevision BEGIN
UPDATE filerevision SET frno = (SELECT COUNT(*) FROM filerevision WHERE path = NEW.path AND projectid = NEW.projectid) WHERE frid = NEW.frid;
END;

CREATE TRIGGER IF NOT EXISTS crfile AFTER INSERT ON filerevision BEGIN
INSERT INTO file(projectid, path) VALUES (NEW.projectid, NEW.path)
ON CONFLICT(projectid, path) DO NOTHING;
END;