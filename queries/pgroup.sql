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

-- name: GetPermissionGroupsForUser :many
SELECT pg.pgroupid, pg.name
FROM permissiongroup pg
JOIN pgmembership pm ON pg.pgroupid = pm.pgroupid
WHERE pm.userid = $1 AND pg.teamid = $2;
