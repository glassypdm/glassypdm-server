-- name: FindTeamPermissions :many
SELECT level FROM teampermission
WHERE userid = $1;

-- name: FindProjectInitCommit :one
SELECT commitid FROM commit
WHERE projectid = $1
ORDER BY commitid ASC LIMIT 1;

-- name: GetTeamName :one
SELECT name FROM team
WHERE teamid = $1 LIMIT 1;

-- name: GetTeamFromName :one
SELECT teamid FROM team
WHERE name = $1 LIMIT 1;

-- name: GetTeamFromProject :one
SELECT teamid FROM project
WHERE projectid = $1 LIMIT 1;

-- name: GetTeamPermission :one
SELECT level FROM teampermission
WHERE teamid = $1 AND userid = $2
LIMIT 1;

-- name: SetTeamPermission :one
INSERT INTO teampermission(userid, teamid, level)
VALUES($1, $2, $3) ON CONFLICT(userid, teamid) DO UPDATE SET level=excluded.level
RETURNING *;

-- name: DeleteTeamPermission :one
DELETE FROM teampermission
WHERE userid = $1
RETURNING *;

-- name: FindUserTeams :many
SELECT DISTINCT team.teamid, name FROM team INNER JOIN teampermission AS tp ON team.teamid = tp.teamid
WHERE tp.userid = $1;

-- name: GetTeamMembership :many
SELECT userid, level FROM teampermission
WHERE teamid = $1
ORDER by level desc;

-- name: FindTeamProjects :many
SELECT projectid, title, name FROM project INNER JOIN team ON team.teamid = project.teamid
WHERE project.teamid = $1;

-- name: FindUserManagedTeams :many
SELECT DISTINCT team.teamid, name FROM team INNER JOIN teampermission as tp ON team.teamid = tp.teamid
WHERE tp.userid = $1 AND tp.level >= 2;

-- name: CheckProjectName :one
SELECT COUNT(*) FROM project
WHERE teamid = $1 and title=$2 LIMIT 1;

-- name: InsertProject :one
INSERT INTO project(title, teamid)
VALUES ($1, $2)
RETURNING projectid;

-- name: GetProjectInfo :one
SELECT title FROM project
WHERE projectid = $1 LIMIT 1;

-- name: GetUploadPermission :one
SELECT COUNT(*) FROM teampermission
WHERE userid = $1 LIMIT 1;

-- name: GetLatestCommit :one
SELECT MAX(commitid) FROM commit
WHERE projectid = $1 LIMIT 1;

-- name: InsertTeam :one
INSERT INTO team(name)
VALUES ($1)
RETURNING teamid;

-- name: InsertCommit :one
INSERT INTO commit(projectid, userid, comment, numfiles)
VALUES ($1, $2, $3, $4)
RETURNING commitid;

-- name: InsertFile :exec
INSERT INTO file(projectid, path)
VALUES ($1, $2);

-- name: InsertFileRevision :exec
INSERT INTO filerevision(projectid, path, commitid, filehash, numchunks, changetype)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: InsertTwoFileRevisions :exec
INSERT INTO filerevision(projectid, path, commitid, filehash, numchunks, changetype)
VALUES ($1, $2, $3, $4, $5, $6), ($7, $8, $9, $10, $11, $12);

-- name: GetTeamByProject :one
SELECT teamid FROM project
WHERE projectid = $1 LIMIT 1;

-- name: GetFileHash :one
SELECT filehash FROM filerevision
WHERE projectid = $1 AND path = $2 AND commitid = $3 LIMIT 1;

-- name: GetProjectState :many
SELECT a.frid, a.path, a.commitid, a.filehash, a.changetype, a.filesize as blocksize FROM filerevision a
INNER JOIN ( SELECT path, MAX(frid) frid FROM filerevision GROUP BY path ) b
ON a.path = b.path AND a.frid = b.frid
WHERE a.projectid = $1;

-- name: GetProjectStateAtCommit :many
SELECT a.frid, a.path, a.commitid, a.filehash, a.changetype, a.filesize as blocksize FROM filerevision a
INNER JOIN ( SELECT path, MAX(frid) frid FROM filerevision GROUP BY path ) b
ON a.path = b.path AND a.frid = b.frid
WHERE a.projectid = $1 AND a.commitid <= $2;

-- name: GetProjectDiffBetweenCommits :many
SELECT a.frid, a.path, a.commitid, a.filehash, a.changetype, a.filesize as blocksize FROM filerevision a
INNER JOIN ( SELECT path, MAX(frid) frid FROM filerevision GROUP BY path ) b
ON a.path = b.path AND a.frid = b.frid
WHERE a.projectid = $1 AND a.commitid <= $2
INTERSECT
SELECT a.frid, a.path, a.commitid, a.filehash, a.changetype, a.filesize as blocksize FROM filerevision a
INNER JOIN ( SELECT path, MAX(frid) frid FROM filerevision GROUP BY path ) b
ON a.path = b.path AND a.frid = b.frid
WHERE a.projectid = $1 AND a.commitid <= $3;

-- name: GetProjectLivingFiles :many
SELECT a.frid, a.path FROM filerevision a
INNER JOIN ( SELECT path, MAX(frid) frid FROM filerevision GROUP BY path ) b
ON a.path = b.path AND a.frid = b.frid
WHERE a.projectid = $1 AND changetype != 3;

-- name: ListProjectCommits :many
SELECT cno, numfiles, userid, comment, commitid, timestamp FROM commit
WHERE projectid = $1
ORDER BY commitid DESC
LIMIT 5 OFFSET $2;

-- name: CountProjectCommits :one
SELECT COUNT(commitid) FROM commit
WHERE projectid = $1
LIMIT 1;

-- name: GetCommitInfo :one
SELECT
  cno,
  userid,
  timestamp,
  comment,
  numfiles,
  projectid
FROM commit
WHERE
  commitid = $1 LIMIT 1;

-- name: GetFileRevisionsByCommitId :many
SELECT frid as filerevision_id, path, frno as filerevision_number, changetype, filesize, commitid as commit_id, projectid as project_id
FROM filerevision
WHERE commitid = $1;

-- name: RestoreProjectToCommit :exec
INSERT INTO filerevision (commitid, projectid, path, filehash, changetype, numchunks, filesize)
WITH 
changedpaths AS (
    SELECT DISTINCT path
    FROM filerevision
    WHERE projectid = $2
      AND commitid > CAST($1 AS INTEGER)
),
revertcommitstate AS (
    SELECT 
        projectid, 
        path, 
        filehash,
        changetype,
        numchunks,
        filesize,
        ROW_NUMBER() OVER (PARTITION BY projectid, path ORDER BY commitid DESC) as rn
    FROM filerevision
    WHERE projectid = $2
      AND path IN (SELECT path FROM changedpaths)
      AND commitid <= CAST($1 AS INTEGER)
)
SELECT DISTINCT
    CAST(sqlc.arg(new_commit) AS INTEGER) as commitid,
    revertcommitstate.projectid,
    revertcommitstate.path,
    revertcommitstate.filehash,
    CASE 
        WHEN revertcommitstate.path IN (
            SELECT path 
            FROM filerevision 
            WHERE projectid = $2 
              AND commitid > CAST($1 AS INTEGER)
        ) THEN 
            CASE 
                WHEN NOT EXISTS (
                    SELECT 1 
                    FROM revertcommitstate 
                    WHERE projectid = revertcommitstate.projectid 
                      AND path = revertcommitstate.path
                ) THEN 3  -- File was created after revert commit, so delete
                WHEN revertcommitstate.changetype = 3 THEN 1  -- If deleted, add back
                WHEN revertcommitstate.changetype = 1 THEN 3  -- If added, delete
                ELSE 2  -- Modified case
            END
        ELSE 2  -- Unchanged files remain as is
    END as changetype,
    revertcommitstate.numchunks,
    revertcommitstate.filesize
FROM revertcommitstate
WHERE revertcommitstate.rn = 1;

-- name: CountFilesUpdatedSinceCommit :one
SELECT COUNT(distinct path) FROM filerevision WHERE
commitid > $1 AND projectid = $2;