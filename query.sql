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
WHERE teamid = $1;

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

-- name: InsertChunk :exec
INSERT INTO chunk(chunkindex, numchunks, filehash, blockhash, blocksize)
VALUES ($1, $2, $3, $4, $5);

-- name: InsertHash :exec
INSERT INTO block(blockhash, s3key, blocksize)
VALUES ($1, $2, $3);

-- name: RemoveHash :exec
DELETE FROM block WHERE blockhash = $1;

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

-- name: GetS3Key :one
SELECT s3key FROM block
WHERE blockhash = $1 LIMIT 1;

-- name: GetFileHash :one
SELECT filehash FROM filerevision
WHERE projectid = $1 AND path = $2 AND commitid = $3 LIMIT 1;

-- name: GetFileChunks :many
SELECT blockhash, chunkindex FROM chunk
WHERE filehash = $1 ORDER BY chunkindex ASC;

-- name: GetProjectState :many
SELECT a.frid, a.path, a.commitid, a.filehash, a.changetype, a.filesize as blocksize FROM filerevision a
INNER JOIN ( SELECT path, MAX(frid) frid FROM filerevision GROUP BY path ) b
ON a.path = b.path AND a.frid = b.frid
WHERE a.projectid = $1;

-- name: GetProjectLivingFiles :many
SELECT a.frid, a.path FROM filerevision a
INNER JOIN ( SELECT path, MAX(frid) frid FROM filerevision GROUP BY path ) b
ON a.path = b.path AND a.frid = b.frid
WHERE a.projectid = $1 and changetype != 3;

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
  numfiles
FROM commit
WHERE
  commitid = $1 LIMIT 1;

-- name: GetFileRevisionsByCommitId :many
SELECT frid, path, frno, changetype, filesize
FROM filerevision
WHERE commitid = $1;

-- name: CreatePermissionGroup :exec
INSERT INTO permissiongroup(teamid, name) VALUES($1, $2);

-- name: AddMemberToPermissionGroup :exec
INSERT INTO pgmembership(pgroupid, userid) VALUES($1, $2);

-- name: MapProjectToPermissionGroup :exec
INSERT INTO pgmapping(pgroupid, projectid) VALUES($1, $2);

-- name: ListPermissionGroupForTeam :many
SELECT pg.pgroupid, pg.name, count(pgm.userid) as count
FROM permissiongroup pg LEFT JOIN pgmembership pgm ON pg.pgroupid = pgm.pgroupid
WHERE pg.teamid = $1 GROUP BY pg.pgroupid;

-- name: ListPermissionGroupMembership :many
SELECT userid FROM pgmembership WHERE pgroupid = $1;

-- name: GetPermissionGroupMapping :many
SELECT p.projectid, p.title FROM pgmapping pg, project p
WHERE pg.pgroupid = $1 AND pg.projectid = p.projectid;

-- name: RemoveMemberFromPermissionGroup :exec
DELETE FROM pgmembership WHERE pgroupid = $1 AND userid = $2;

-- name: RemoveProjectFromPermissionGroup :exec
DELETE FROM pgmapping WHERE pgroupid = $1 AND projectid = $2;

-- name: DropPermissionGroupMembership :exec
DELETE FROM pgmembership WHERE pgroupid = $1;

-- name: DropPermissionGroupMapping :exec
DELETE FROM pgmapping WHERE pgroupid = $1;

-- name: DeletePermissionGroup :exec
DELETE FROM permissiongroup WHERE pgroupid = $1;

-- name: FindUserInPermissionGroup :many
SELECT pgroupid FROM pgmembership WHERE
userid = $1;

-- name: IsUserInPermissionGroup :one
SELECT userid FROM pgmembership pgme, pgmapping pgma WHERE
pgme.userid = $1 AND pgma.projectid = $2 AND pgma.pgroupid = pgme.pgroupid;

-- name: GetTeamFromPGroup :one
SELECT teamid FROM permissiongroup WHERE
pgroupid = $1 LIMIT 1;