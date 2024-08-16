-- name: FindTeamPermissions :many
SELECT level FROM teampermission
WHERE userid = ?;

-- name: FindProjectPermissions :many
SELECT level FROM projectpermission
WHERE userid = ?;

-- name: FindProjectInitCommit :one
SELECT commitid FROM 'commit'
WHERE projectid = ?
ORDER BY commitid ASC LIMIT 1;

-- name: GetTeamName :one
SELECT name FROM team
WHERE teamid = ? LIMIT 1;

-- name: GetTeamFromProject :one
SELECT teamid FROM project
WHERE projectid = ? LIMIT 1;

-- name: GetTeamPermission :one
SELECT level FROM teampermission
WHERE teamid = ? AND userid = ?
LIMIT 1;

-- name: SetTeamPermission :one
INSERT INTO teampermission(userid, teamid, level)
VALUES(?, ?, ?) ON CONFLICT(userid, teamid) DO UPDATE SET level=excluded.level
RETURNING *;

-- name: DeleteTeamPermission :one
DELETE FROM teampermission
WHERE userid = ?
RETURNING *;

-- name: FindUserTeams :many
SELECT DISTINCT team.teamid, name FROM team INNER JOIN teampermission AS tp ON team.teamid = tp.teamid
WHERE tp.userid = ?;

-- name: GetTeamMembership :many
SELECT userid, level FROM teampermission
WHERE teamid = ?;

-- name: FindUserProjects :many
SELECT projectid, title, name FROM project INNER JOIN team ON team.teamid = project.teamid
WHERE project.teamid = ?;

-- name: FindUserManagedTeams :many
SELECT DISTINCT team.teamid, name FROM team INNER JOIN teampermission as tp ON team.teamid = tp.teamid
WHERE tp.userid = ? AND tp.level >= 2;

-- name: CheckProjectName :one
SELECT COUNT(*) FROM project
WHERE teamid = ? and title=? LIMIT 1;

-- name: InsertProject :one
INSERT INTO project(title, teamid)
VALUES (?, ?)
RETURNING projectid;

-- name: GetProjectInfo :one
SELECT title FROM project
WHERE projectid = ? LIMIT 1;

-- name: GetProjectPermission :one
SELECT level FROM projectpermission
WHERE userid = ? AND projectid = ? LIMIT 1;

-- name: GetUploadPermission :one
SELECT COUNT(*) FROM teampermission
WHERE userid = ? LIMIT 1;

-- name: GetLatestCommit :one
SELECT MAX(commitid) FROM 'commit'
WHERE projectid = ? LIMIT 1;

-- name: InsertTeam :one
INSERT INTO team(name)
VALUES (?)
RETURNING teamid;

-- name: InsertCommit :one
INSERT INTO 'commit'(projectid, userid, comment, numfiles)
VALUES (?, ?, ?, ?)
RETURNING commitid;

-- name: FindHash :one
SELECT * FROM block
WHERE hash = ?
LIMIT 1;

-- name: InsertHash :exec
INSERT INTO block(hash, s3key, size)
VALUES (?, ?, ?);

-- name: RemoveHash :exec
DELETE FROM block WHERE hash = ?;

-- name: InsertFile :exec
INSERT INTO file(projectid, path)
VALUES (?, ?);

-- name: InsertFileRevision :exec
INSERT INTO filerevision(projectid, path, commitid, hash, changetype)
VALUES (?, ?, ?, ?, ?);

-- name: InsertTwoFileRevisions :exec
INSERT INTO filerevision(projectid, path, commitid, hash, changetype)
VALUES (?, ?, ?, ?, ?), (?, ?, ?, ?, ?);

-- name: GetTeamByProject :one
SELECT teamid FROM project
WHERE projectid = ? LIMIT 1;

-- name: GetS3Key :one
SELECT s3key FROM block
WHERE hash = ? LIMIT 1;

-- name: GetHash :one
SELECT hash FROM filerevision
WHERE projectid = ? AND path = ? AND commitid = ? LIMIT 1;

-- name: GetProjectState :many
SELECT a.frid, a.path, a.commitid, a.hash, a.changetype, block.size FROM block, filerevision a
INNER JOIN ( SELECT path, MAX(frid) frid FROM filerevision GROUP BY path ) b
ON a.path = b.path AND a.frid = b.frid
WHERE a.projectid = ? AND a.hash = block.hash;

-- name: GetProjectLivingFiles :many
SELECT a.frid, a.path FROM filerevision a
INNER JOIN ( SELECT path, MAX(frid) frid FROM filerevision GROUP BY path ) b
ON a.path = b.path AND a.frid = b.frid
WHERE a.projectid = ? and changetype != 3;

-- name: ListProjectCommits :many
SELECT cno, numfiles, userid, comment, commitid, timestamp FROM 'commit'
WHERE projectid = ?
ORDER BY commitid DESC
LIMIT 5 OFFSET ?;

-- name: CountProjectCommits :one
SELECT COUNT(commitid) FROM 'commit'
WHERE projectid = ?
LIMIT 1;

-- name: GetCommitInfo :one
SELECT
  a.cno,
  a.userid,
  a.timestamp,
  a.comment,
  a.numfiles,
  hehe.path,
  hehe.frno,
  hehe.hash,
  hehe.size
FROM
  'commit' a
  INNER JOIN (
    SELECT
      b.hash,
      b.size,
      fr.path,
      fr.frno,
      fr.commitid
    FROM
      filerevision fr
      INNER JOIN block b ON fr.hash = b.hash
    WHERE fr.commitid = ?
  ) hehe ON a.commitid = hehe.commitid
WHERE
  a.commitid = ? LIMIT 1;