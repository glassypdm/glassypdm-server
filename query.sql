-- name: FindTeamPermissions :many
SELECT level FROM teampermission
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

-- name: FindTeamProjects :many
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

-- name: InsertChunk :exec
INSERT INTO chunk(chunkindex, numchunks, filehash, blockhash, blocksize)
VALUES (?, ?, ?, ?, ?);

-- name: InsertHash :exec
INSERT INTO block(blockhash, s3key, blocksize)
VALUES (?, ?, ?);

-- name: RemoveHash :exec
DELETE FROM block WHERE blockhash = ?;

-- name: InsertFile :exec
INSERT INTO file(projectid, path)
VALUES (?, ?);

-- name: InsertFileRevision :exec
INSERT INTO filerevision(projectid, path, commitid, filehash, numchunks, changetype)
VALUES (?, ?, ?, ?, ?, ?);

-- name: InsertTwoFileRevisions :exec
INSERT INTO filerevision(projectid, path, commitid, filehash, numchunks, changetype)
VALUES (?, ?, ?, ?, ?, ?), (?, ?, ?, ?, ?, ?);

-- name: GetTeamByProject :one
SELECT teamid FROM project
WHERE projectid = ? LIMIT 1;

-- name: GetS3Key :one
SELECT s3key FROM block
WHERE blockhash = ? LIMIT 1;

-- name: GetFileHash :one
SELECT filehash FROM filerevision
WHERE projectid = ? AND path = ? AND commitid = ? LIMIT 1;

-- name: GetFileChunks :many
SELECT blockhash, chunkindex FROM chunk
WHERE filehash = ?;

-- name: GetProjectState :many
SELECT a.frid, a.path, a.commitid, a.filehash, a.changetype, chunk.blocksize FROM chunk, filerevision a
INNER JOIN ( SELECT path, MAX(frid) frid FROM filerevision GROUP BY path ) b
ON a.path = b.path AND a.frid = b.frid
WHERE a.projectid = ? AND a.filehash = chunk.filehash;

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

-- TODO Fix
-- name: GetCommitInfo :one
SELECT
  a.cno,
  a.userid,
  a.timestamp,
  a.comment,
  a.numfiles,
  hehe.path,
  hehe.frno,
  hehe.filehash,
  hehe.blocksize
FROM
  'commit' a
  INNER JOIN (
    SELECT
      b.filehash,
      b.blocksize,
      fr.path,
      fr.frno,
      fr.commitid
    FROM
      filerevision fr
      INNER JOIN chunk b ON fr.filehash = b.filehash
    WHERE fr.commitid = ?
  ) hehe ON a.commitid = hehe.commitid
WHERE
  a.commitid = ? LIMIT 1;

-- name: CreatePermissionGroup :exec
INSERT INTO permissiongroup(teamid, name) VALUES(?, ?);

-- name: AddMemberToPermissionGroup :exec
INSERT INTO pgmembership(pgroupid, userid) VALUES(?, ?);

-- name: MapProjectToPermissionGroup :exec
INSERT INTO pgmapping(pgroupid, projectid) VALUES(?, ?);

-- name: ListPermissionGroupForTeam :many
SELECT pg.pgroupid, pg.name, count(pgm.userid) as count
FROM permissiongroup pg LEFT JOIN pgmembership pgm ON pg.pgroupid = pgm.pgroupid
WHERE pg.teamid = ? GROUP BY pg.pgroupid;

-- name: ListPermissionGroupMembership :many
SELECT userid FROM pgmembership WHERE pgroupid = ?;

-- name: GetPermissionGroupMapping :many
SELECT p.projectid, p.title FROM pgmapping pg, project p
WHERE pg.pgroupid = ? AND pg.projectid = p.projectid;

-- name: RemoveMemberFromPermissionGroup :exec
DELETE FROM pgmembership WHERE pgroupid = ? AND userid = ?;

-- name: RemoveProjectFromPermissionGroup :exec
DELETE FROM pgmapping WHERE pgroupid = ? AND projectid = ?;

-- name: DropPermissionGroupMembership :exec
DELETE FROM pgmembership WHERE pgroupid = ?;

-- name: DropPermissionGroupMapping :exec
DELETE FROM pgmapping WHERE pgroupid = ?;

-- name: DeletePermissionGroup :exec
DELETE FROM permissiongroup WHERE pgroupid = ?;

-- name: FindUserInPermissionGroup :many
SELECT pgroupid FROM pgmembership WHERE
userid = ?;

-- name: IsUserInPermissionGroup :one
SELECT userid FROM pgmembership pgme, pgmapping pgma WHERE
pgme.userid = ? AND pgma.projectid = ? AND pgma.pgroupid = pgme.pgroupid;

-- name: GetTeamFromPGroup :one
SELECT teamid FROM permissiongroup WHERE
pgroupid = ? LIMIT 1;