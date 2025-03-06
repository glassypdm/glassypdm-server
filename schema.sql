CREATE TABLE IF NOT EXISTS team(
    teamid SERIAL PRIMARY KEY NOT NULL,
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
    projectid SERIAL PRIMARY KEY NOT NULL,
    title TEXT NOT NULL,
    teamid INTEGER NOT NULL,
    FOREIGN KEY(teamid) REFERENCES team(teamid),
    UNIQUE(teamid, title)
);

CREATE TABLE IF NOT EXISTS permissiongroup(
    pgroupid SERIAL PRIMARY KEY NOT NULL,
    teamid INTEGER NOT NULL,
    name TEXT NOT NULL,
    FOREIGN KEY(teamid) REFERENCES team(teamid),
    UNIQUE(teamid, name)
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

CREATE TABLE IF NOT EXISTS commit(
    commitid SERIAL PRIMARY KEY NOT NULL,
    projectid INTEGER NOT NULL,
    userid TEXT NOT NULL,
    comment TEXT NOT NULL,
    numfiles INTEGER NOT NULL,
    cno INTEGER,
    timestamp TIMESTAMP DEFAULT NOW() NOT NULL,
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
    frid SERIAL PRIMARY KEY NOT NULL,
    projectid INTEGER NOT NULL,
    path TEXT NOT NULL,
    commitid INTEGER NOT NULL,
    filehash TEXT NOT NULL,
    changetype INTEGER NOT NULL,
    numchunks INTEGER NOT NULL,
    filesize INTEGER NOT NULL DEFAULT 0,
    frno INTEGER,
    FOREIGN KEY(projectid) REFERENCES project(projectid),
    FOREIGN KEY(commitid) REFERENCES commit(commitid)
);

CREATE TABLE IF NOT EXISTS block(
    blockhash TEXT PRIMARY KEY NOT NULL,
    s3key TEXT NOT NULL,
    blocksize INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS chunk(
    chunkindex INTEGER NOT NULL,
    numchunks INTEGER NOT NULL,
    filehash TEXT NOT NULL,
    blockhash TEXT NOT NULL,
    blocksize INTEGER NOT NULL,
    filesize INTEGER NOT NULL,
    PRIMARY KEY(filehash, chunkindex),
    FOREIGN KEY(blockhash) REFERENCES block(blockhash),
    UNIQUE(filehash, chunkindex)
);

/*
CREATE TABLE IF NOT EXISTS part(
    partid SERIAL PRIMARY KEY NOT NULL,
    projectid INTEGER NOT NULL,
    teamid INTEGER NOT NULL,
    partname TEXT NOT NULL,
    partnumber TEXT NOT NULL,
    parttype INTEGER NOT NULL,
    UNIQUE(teamid, partnumber)
);

CREATE TABLE IF NOT EXISTS partedithistory(
    partedithistoryid SERIAL PRIMARY KEY NOT NULL,
    partid INTEGER NOT NULL,
    edit TEXT NOT NULL,
    timestamp TIMESTAMP DEFAULT NOW() NOT NULL,
    FOREIGN KEY(partid) REFERENCES part(partid) 
);

CREATE TABLE IF NOT EXISTS partversion(
    partversionid SERIAL PRIMARY KEY NOT NULL,
    partid INTEGER NOT NULL,
    release BOOLEAN NOT NULL,
    locked BOOLEAN NOT NULL,
    lockedby TEXT NOT NULL,
    lockedtimestamp TEXT NOT NULL,
    pvno INTEGER,
    FOREIGN KEY(partid) REFERENCES part(partid) 
);

CREATE TABLE IF NOT EXISTS partfile(
    partfileid SERIAL PRIMARY KEY NOT NULL,
    partversionid INTEGER NOT NULL,
    filetype TEXT NOT NULL,
    path TEXT NOT NULL,
    frid INTEGER NOT NULL,
    FOREIGN KEY(partversionid) REFERENCES partversion(partversionid),
    FOREIGN KEY(frid) REFERENCES filerevision(frid),
    UNIQUE(partversionid, path)
);
*/
CREATE OR REPLACE FUNCTION update_commit_number()
RETURNS TRIGGER
LANGUAGE PLPGSQL
AS
$$
BEGIN
UPDATE commit SET cno = (SELECT COUNT(*) FROM commit WHERE projectid = NEW.projectid) WHERE commitid = NEW.commitid;
RETURN NEW;
END;
$$;

CREATE OR REPLACE FUNCTION audit_filerevision()
RETURNS TRIGGER
LANGUAGE PLPGSQL
AS
$$
BEGIN
INSERT INTO file(projectid, path) VALUES (NEW.projectid, NEW.path)
ON CONFLICT(projectid, path) DO NOTHING;

UPDATE filerevision SET frno = (SELECT COUNT(*) FROM filerevision WHERE path = NEW.path AND projectid = NEW.projectid) WHERE frid = NEW.frid;
UPDATE filerevision set filesize = (SELECT COALESCE(SUM(blocksize), 0) FROM chunk WHERE chunk.filehash = NEW.filehash) WHERE frid = NEW.frid;

RETURN NEW;
END;
$$;


CREATE OR REPLACE TRIGGER commitnumber AFTER INSERT ON commit FOR EACH ROW EXECUTE FUNCTION update_commit_number();
CREATE OR REPLACE TRIGGER filerevisionaudit AFTER INSERT ON filerevision FOR EACH ROW EXECUTE FUNCTION audit_filerevision();
