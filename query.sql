-- name: FindTeamPermissions :many
SELECT level FROM teampermission
WHERE userid = ?;

-- name: FindProjectPermissions :many
SELECT level FROM projectpermission
WHERE userid = ?;

-- name: GetTeamName :one
SELECT name FROM team
WHERE teamid = ? LIMIT 1;
-- name: GetTeamPermission :one
SELECT level FROM teampermission
WHERE teamid = ? AND userid = ?
LIMIT 1;

-- name: SetTeamPermission :one
INSERT INTO teampermission(userid, teamid, level)
VALUES(?, ?, ?) ON CONFLICT(userid, teamid) DO UPDATE SET level=excluded.level
RETURNING *;

-- name: FindUserTeams :many
SELECT DISTINCT team.teamid, name FROM team INNER JOIN teampermission AS tp ON team.teamid = tp.teamid
WHERE tp.userid = ?;

-- name: GetTeamMembership :many
SELECT userid, level FROM teampermission
WHERE teamid = ?;

-- name: FindUserProjects :many
SELECT pid, title, name FROM project INNER JOIN team
WHERE team.teamid = ? AND project.teamid = ?;

-- name: FindUserManagedTeams :many
SELECT DISTINCT team.teamid, name FROM team INNER JOIN teampermission as tp
WHERE tp.userid = ? AND tp.level >= 2;

-- name: CheckProjectName :one
SELECT COUNT(*) FROM project
WHERE teamid = ? and title=? LIMIT 1;

-- name: InsertProject :exec
INSERT INTO project(title, teamid)
VALUES (?, ?);

-- name: GetProjectInfo :one
SELECT title FROM project
WHERE pid = ? LIMIT 1;

-- name: GetProjectPermission :one
SELECT level FROM projectpermission
WHERE userid = ? AND projectid = ? LIMIT 1;

-- name: GetUploadPermission :one
SELECT COUNT(*) FROM teampermission
WHERE userid = ? LIMIT 1;

-- name: GetLatestCommit :one
SELECT MAX(cid) FROM 'commit'
WHERE projectid = ? LIMIT 1;

-- name: InsertTeam :one
INSERT INTO team(name)
VALUES (?)
RETURNING teamid;