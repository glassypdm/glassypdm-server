// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: query.sql

package sqlcgen

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const checkProjectName = `-- name: CheckProjectName :one
SELECT COUNT(*) FROM project
WHERE teamid = $1 and title=$2 LIMIT 1
`

type CheckProjectNameParams struct {
	Teamid int32  `json:"teamid"`
	Title  string `json:"title"`
}

func (q *Queries) CheckProjectName(ctx context.Context, arg CheckProjectNameParams) (int64, error) {
	row := q.db.QueryRow(ctx, checkProjectName, arg.Teamid, arg.Title)
	var count int64
	err := row.Scan(&count)
	return count, err
}

const countFilesUpdatedSinceCommit = `-- name: CountFilesUpdatedSinceCommit :one
SELECT COUNT(distinct path) FROM filerevision WHERE
commitid > $1 AND projectid = $2
`

type CountFilesUpdatedSinceCommitParams struct {
	Commitid  int32 `json:"commitid"`
	Projectid int32 `json:"projectid"`
}

func (q *Queries) CountFilesUpdatedSinceCommit(ctx context.Context, arg CountFilesUpdatedSinceCommitParams) (int64, error) {
	row := q.db.QueryRow(ctx, countFilesUpdatedSinceCommit, arg.Commitid, arg.Projectid)
	var count int64
	err := row.Scan(&count)
	return count, err
}

const countProjectCommits = `-- name: CountProjectCommits :one
SELECT COUNT(commitid) FROM commit
WHERE projectid = $1
LIMIT 1
`

func (q *Queries) CountProjectCommits(ctx context.Context, projectid int32) (int64, error) {
	row := q.db.QueryRow(ctx, countProjectCommits, projectid)
	var count int64
	err := row.Scan(&count)
	return count, err
}

const deleteTeamPermission = `-- name: DeleteTeamPermission :one
DELETE FROM teampermission
WHERE userid = $1
RETURNING userid, teamid, level
`

func (q *Queries) DeleteTeamPermission(ctx context.Context, userid string) (Teampermission, error) {
	row := q.db.QueryRow(ctx, deleteTeamPermission, userid)
	var i Teampermission
	err := row.Scan(&i.Userid, &i.Teamid, &i.Level)
	return i, err
}

const findProjectInitCommit = `-- name: FindProjectInitCommit :one
SELECT commitid FROM commit
WHERE projectid = $1
ORDER BY commitid ASC LIMIT 1
`

func (q *Queries) FindProjectInitCommit(ctx context.Context, projectid int32) (int32, error) {
	row := q.db.QueryRow(ctx, findProjectInitCommit, projectid)
	var commitid int32
	err := row.Scan(&commitid)
	return commitid, err
}

const findTeamPermissions = `-- name: FindTeamPermissions :many
SELECT level FROM teampermission
WHERE userid = $1
`

func (q *Queries) FindTeamPermissions(ctx context.Context, userid string) ([]int32, error) {
	rows, err := q.db.Query(ctx, findTeamPermissions, userid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []int32
	for rows.Next() {
		var level int32
		if err := rows.Scan(&level); err != nil {
			return nil, err
		}
		items = append(items, level)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const findTeamProjects = `-- name: FindTeamProjects :many
SELECT projectid, title, name FROM project INNER JOIN team ON team.teamid = project.teamid
WHERE project.teamid = $1
`

type FindTeamProjectsRow struct {
	Projectid int32  `json:"projectid"`
	Title     string `json:"title"`
	Name      string `json:"name"`
}

func (q *Queries) FindTeamProjects(ctx context.Context, teamid int32) ([]FindTeamProjectsRow, error) {
	rows, err := q.db.Query(ctx, findTeamProjects, teamid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []FindTeamProjectsRow
	for rows.Next() {
		var i FindTeamProjectsRow
		if err := rows.Scan(&i.Projectid, &i.Title, &i.Name); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const findUserManagedTeams = `-- name: FindUserManagedTeams :many
SELECT DISTINCT team.teamid, name FROM team INNER JOIN teampermission as tp ON team.teamid = tp.teamid
WHERE tp.userid = $1 AND tp.level >= 2
`

type FindUserManagedTeamsRow struct {
	Teamid int32  `json:"teamid"`
	Name   string `json:"name"`
}

func (q *Queries) FindUserManagedTeams(ctx context.Context, userid string) ([]FindUserManagedTeamsRow, error) {
	rows, err := q.db.Query(ctx, findUserManagedTeams, userid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []FindUserManagedTeamsRow
	for rows.Next() {
		var i FindUserManagedTeamsRow
		if err := rows.Scan(&i.Teamid, &i.Name); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const findUserTeams = `-- name: FindUserTeams :many
SELECT DISTINCT team.teamid, name FROM team INNER JOIN teampermission AS tp ON team.teamid = tp.teamid
WHERE tp.userid = $1
`

type FindUserTeamsRow struct {
	Teamid int32  `json:"teamid"`
	Name   string `json:"name"`
}

func (q *Queries) FindUserTeams(ctx context.Context, userid string) ([]FindUserTeamsRow, error) {
	rows, err := q.db.Query(ctx, findUserTeams, userid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []FindUserTeamsRow
	for rows.Next() {
		var i FindUserTeamsRow
		if err := rows.Scan(&i.Teamid, &i.Name); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getCommitInfo = `-- name: GetCommitInfo :one
SELECT
  cno,
  userid,
  timestamp,
  comment,
  numfiles,
  projectid
FROM commit
WHERE
  commitid = $1 LIMIT 1
`

type GetCommitInfoRow struct {
	Cno       pgtype.Int4      `json:"cno"`
	Userid    string           `json:"userid"`
	Timestamp pgtype.Timestamp `json:"timestamp"`
	Comment   string           `json:"comment"`
	Numfiles  int32            `json:"numfiles"`
	Projectid int32            `json:"projectid"`
}

func (q *Queries) GetCommitInfo(ctx context.Context, commitid int32) (GetCommitInfoRow, error) {
	row := q.db.QueryRow(ctx, getCommitInfo, commitid)
	var i GetCommitInfoRow
	err := row.Scan(
		&i.Cno,
		&i.Userid,
		&i.Timestamp,
		&i.Comment,
		&i.Numfiles,
		&i.Projectid,
	)
	return i, err
}

const getFileHash = `-- name: GetFileHash :one
SELECT filehash FROM filerevision
WHERE projectid = $1 AND path = $2 AND commitid = $3 LIMIT 1
`

type GetFileHashParams struct {
	Projectid int32  `json:"projectid"`
	Path      string `json:"path"`
	Commitid  int32  `json:"commitid"`
}

func (q *Queries) GetFileHash(ctx context.Context, arg GetFileHashParams) (string, error) {
	row := q.db.QueryRow(ctx, getFileHash, arg.Projectid, arg.Path, arg.Commitid)
	var filehash string
	err := row.Scan(&filehash)
	return filehash, err
}

const getFileRevisionsByCommitId = `-- name: GetFileRevisionsByCommitId :many
SELECT frid as filerevision_id, path, frno as filerevision_number, changetype, filesize, commitid as commit_id, projectid as project_id
FROM filerevision
WHERE commitid = $1
`

type GetFileRevisionsByCommitIdRow struct {
	FilerevisionID     int32       `json:"filerevision_id"`
	Path               string      `json:"path"`
	FilerevisionNumber pgtype.Int4 `json:"filerevision_number"`
	Changetype         int32       `json:"changetype"`
	Filesize           int32       `json:"filesize"`
	CommitID           int32       `json:"commit_id"`
	ProjectID          int32       `json:"project_id"`
}

func (q *Queries) GetFileRevisionsByCommitId(ctx context.Context, commitid int32) ([]GetFileRevisionsByCommitIdRow, error) {
	rows, err := q.db.Query(ctx, getFileRevisionsByCommitId, commitid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetFileRevisionsByCommitIdRow
	for rows.Next() {
		var i GetFileRevisionsByCommitIdRow
		if err := rows.Scan(
			&i.FilerevisionID,
			&i.Path,
			&i.FilerevisionNumber,
			&i.Changetype,
			&i.Filesize,
			&i.CommitID,
			&i.ProjectID,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getLatestCommit = `-- name: GetLatestCommit :one
SELECT MAX(commitid) FROM commit
WHERE projectid = $1 LIMIT 1
`

func (q *Queries) GetLatestCommit(ctx context.Context, projectid int32) (interface{}, error) {
	row := q.db.QueryRow(ctx, getLatestCommit, projectid)
	var max interface{}
	err := row.Scan(&max)
	return max, err
}

const getProjectDiffBetweenCommits = `-- name: GetProjectDiffBetweenCommits :many
SELECT a.frid, a.path, a.commitid, a.filehash, a.changetype, a.filesize as blocksize FROM filerevision a
INNER JOIN ( SELECT path, MAX(frid) frid FROM filerevision GROUP BY path ) b
ON a.path = b.path AND a.frid = b.frid
WHERE a.projectid = $1 AND a.commitid <= $2
INTERSECT
SELECT a.frid, a.path, a.commitid, a.filehash, a.changetype, a.filesize as blocksize FROM filerevision a
INNER JOIN ( SELECT path, MAX(frid) frid FROM filerevision GROUP BY path ) b
ON a.path = b.path AND a.frid = b.frid
WHERE a.projectid = $1 AND a.commitid <= $3
`

type GetProjectDiffBetweenCommitsParams struct {
	Projectid  int32 `json:"projectid"`
	Commitid   int32 `json:"commitid"`
	Commitid_2 int32 `json:"commitid_2"`
}

type GetProjectDiffBetweenCommitsRow struct {
	Frid       int32  `json:"frid"`
	Path       string `json:"path"`
	Commitid   int32  `json:"commitid"`
	Filehash   string `json:"filehash"`
	Changetype int32  `json:"changetype"`
	Blocksize  int32  `json:"blocksize"`
}

func (q *Queries) GetProjectDiffBetweenCommits(ctx context.Context, arg GetProjectDiffBetweenCommitsParams) ([]GetProjectDiffBetweenCommitsRow, error) {
	rows, err := q.db.Query(ctx, getProjectDiffBetweenCommits, arg.Projectid, arg.Commitid, arg.Commitid_2)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetProjectDiffBetweenCommitsRow
	for rows.Next() {
		var i GetProjectDiffBetweenCommitsRow
		if err := rows.Scan(
			&i.Frid,
			&i.Path,
			&i.Commitid,
			&i.Filehash,
			&i.Changetype,
			&i.Blocksize,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getProjectInfo = `-- name: GetProjectInfo :one
SELECT title FROM project
WHERE projectid = $1 LIMIT 1
`

func (q *Queries) GetProjectInfo(ctx context.Context, projectid int32) (string, error) {
	row := q.db.QueryRow(ctx, getProjectInfo, projectid)
	var title string
	err := row.Scan(&title)
	return title, err
}

const getProjectLivingFiles = `-- name: GetProjectLivingFiles :many
SELECT a.frid, a.path FROM filerevision a
INNER JOIN ( SELECT path, MAX(frid) frid FROM filerevision GROUP BY path ) b
ON a.path = b.path AND a.frid = b.frid
WHERE a.projectid = $1 AND changetype != 3
`

type GetProjectLivingFilesRow struct {
	Frid int32  `json:"frid"`
	Path string `json:"path"`
}

func (q *Queries) GetProjectLivingFiles(ctx context.Context, projectid int32) ([]GetProjectLivingFilesRow, error) {
	rows, err := q.db.Query(ctx, getProjectLivingFiles, projectid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetProjectLivingFilesRow
	for rows.Next() {
		var i GetProjectLivingFilesRow
		if err := rows.Scan(&i.Frid, &i.Path); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getProjectState = `-- name: GetProjectState :many
SELECT a.frid, a.path, a.commitid, a.filehash, a.changetype, a.filesize as blocksize FROM filerevision a
INNER JOIN ( SELECT path, MAX(frid) frid FROM filerevision GROUP BY path ) b
ON a.path = b.path AND a.frid = b.frid
WHERE a.projectid = $1
`

type GetProjectStateRow struct {
	Frid       int32  `json:"frid"`
	Path       string `json:"path"`
	Commitid   int32  `json:"commitid"`
	Filehash   string `json:"filehash"`
	Changetype int32  `json:"changetype"`
	Blocksize  int32  `json:"blocksize"`
}

func (q *Queries) GetProjectState(ctx context.Context, projectid int32) ([]GetProjectStateRow, error) {
	rows, err := q.db.Query(ctx, getProjectState, projectid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetProjectStateRow
	for rows.Next() {
		var i GetProjectStateRow
		if err := rows.Scan(
			&i.Frid,
			&i.Path,
			&i.Commitid,
			&i.Filehash,
			&i.Changetype,
			&i.Blocksize,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getProjectStateAtCommit = `-- name: GetProjectStateAtCommit :many
SELECT a.frid, a.path, a.commitid, a.filehash, a.changetype, a.filesize as blocksize FROM filerevision a
INNER JOIN ( SELECT path, MAX(frid) frid FROM filerevision GROUP BY path ) b
ON a.path = b.path AND a.frid = b.frid
WHERE a.projectid = $1 AND a.commitid <= $2
`

type GetProjectStateAtCommitParams struct {
	Projectid int32 `json:"projectid"`
	Commitid  int32 `json:"commitid"`
}

type GetProjectStateAtCommitRow struct {
	Frid       int32  `json:"frid"`
	Path       string `json:"path"`
	Commitid   int32  `json:"commitid"`
	Filehash   string `json:"filehash"`
	Changetype int32  `json:"changetype"`
	Blocksize  int32  `json:"blocksize"`
}

func (q *Queries) GetProjectStateAtCommit(ctx context.Context, arg GetProjectStateAtCommitParams) ([]GetProjectStateAtCommitRow, error) {
	rows, err := q.db.Query(ctx, getProjectStateAtCommit, arg.Projectid, arg.Commitid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetProjectStateAtCommitRow
	for rows.Next() {
		var i GetProjectStateAtCommitRow
		if err := rows.Scan(
			&i.Frid,
			&i.Path,
			&i.Commitid,
			&i.Filehash,
			&i.Changetype,
			&i.Blocksize,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getTeamByProject = `-- name: GetTeamByProject :one
SELECT teamid FROM project
WHERE projectid = $1 LIMIT 1
`

func (q *Queries) GetTeamByProject(ctx context.Context, projectid int32) (int32, error) {
	row := q.db.QueryRow(ctx, getTeamByProject, projectid)
	var teamid int32
	err := row.Scan(&teamid)
	return teamid, err
}

const getTeamFromName = `-- name: GetTeamFromName :one
SELECT teamid FROM team
WHERE name = $1 LIMIT 1
`

func (q *Queries) GetTeamFromName(ctx context.Context, name string) (int32, error) {
	row := q.db.QueryRow(ctx, getTeamFromName, name)
	var teamid int32
	err := row.Scan(&teamid)
	return teamid, err
}

const getTeamFromProject = `-- name: GetTeamFromProject :one
SELECT teamid FROM project
WHERE projectid = $1 LIMIT 1
`

func (q *Queries) GetTeamFromProject(ctx context.Context, projectid int32) (int32, error) {
	row := q.db.QueryRow(ctx, getTeamFromProject, projectid)
	var teamid int32
	err := row.Scan(&teamid)
	return teamid, err
}

const getTeamMembership = `-- name: GetTeamMembership :many
SELECT userid, level FROM teampermission
WHERE teamid = $1
ORDER by level desc
`

type GetTeamMembershipRow struct {
	Userid string `json:"userid"`
	Level  int32  `json:"level"`
}

func (q *Queries) GetTeamMembership(ctx context.Context, teamid int32) ([]GetTeamMembershipRow, error) {
	rows, err := q.db.Query(ctx, getTeamMembership, teamid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetTeamMembershipRow
	for rows.Next() {
		var i GetTeamMembershipRow
		if err := rows.Scan(&i.Userid, &i.Level); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getTeamName = `-- name: GetTeamName :one
SELECT name FROM team
WHERE teamid = $1 LIMIT 1
`

func (q *Queries) GetTeamName(ctx context.Context, teamid int32) (string, error) {
	row := q.db.QueryRow(ctx, getTeamName, teamid)
	var name string
	err := row.Scan(&name)
	return name, err
}

const getTeamPermission = `-- name: GetTeamPermission :one
SELECT level FROM teampermission
WHERE teamid = $1 AND userid = $2
LIMIT 1
`

type GetTeamPermissionParams struct {
	Teamid int32  `json:"teamid"`
	Userid string `json:"userid"`
}

func (q *Queries) GetTeamPermission(ctx context.Context, arg GetTeamPermissionParams) (int32, error) {
	row := q.db.QueryRow(ctx, getTeamPermission, arg.Teamid, arg.Userid)
	var level int32
	err := row.Scan(&level)
	return level, err
}

const getUploadPermission = `-- name: GetUploadPermission :one
SELECT COUNT(*) FROM teampermission
WHERE userid = $1 LIMIT 1
`

func (q *Queries) GetUploadPermission(ctx context.Context, userid string) (int64, error) {
	row := q.db.QueryRow(ctx, getUploadPermission, userid)
	var count int64
	err := row.Scan(&count)
	return count, err
}

const insertCommit = `-- name: InsertCommit :one
INSERT INTO commit(projectid, userid, comment, numfiles)
VALUES ($1, $2, $3, $4)
RETURNING commitid
`

type InsertCommitParams struct {
	Projectid int32  `json:"projectid"`
	Userid    string `json:"userid"`
	Comment   string `json:"comment"`
	Numfiles  int32  `json:"numfiles"`
}

func (q *Queries) InsertCommit(ctx context.Context, arg InsertCommitParams) (int32, error) {
	row := q.db.QueryRow(ctx, insertCommit,
		arg.Projectid,
		arg.Userid,
		arg.Comment,
		arg.Numfiles,
	)
	var commitid int32
	err := row.Scan(&commitid)
	return commitid, err
}

const insertFile = `-- name: InsertFile :exec
INSERT INTO file(projectid, path)
VALUES ($1, $2)
`

type InsertFileParams struct {
	Projectid int32  `json:"projectid"`
	Path      string `json:"path"`
}

func (q *Queries) InsertFile(ctx context.Context, arg InsertFileParams) error {
	_, err := q.db.Exec(ctx, insertFile, arg.Projectid, arg.Path)
	return err
}

const insertFileRevision = `-- name: InsertFileRevision :exec
INSERT INTO filerevision(projectid, path, commitid, filehash, numchunks, changetype)
VALUES ($1, $2, $3, $4, $5, $6)
`

type InsertFileRevisionParams struct {
	Projectid  int32  `json:"projectid"`
	Path       string `json:"path"`
	Commitid   int32  `json:"commitid"`
	Filehash   string `json:"filehash"`
	Numchunks  int32  `json:"numchunks"`
	Changetype int32  `json:"changetype"`
}

func (q *Queries) InsertFileRevision(ctx context.Context, arg InsertFileRevisionParams) error {
	_, err := q.db.Exec(ctx, insertFileRevision,
		arg.Projectid,
		arg.Path,
		arg.Commitid,
		arg.Filehash,
		arg.Numchunks,
		arg.Changetype,
	)
	return err
}

const insertProject = `-- name: InsertProject :one
INSERT INTO project(title, teamid)
VALUES ($1, $2)
RETURNING projectid
`

type InsertProjectParams struct {
	Title  string `json:"title"`
	Teamid int32  `json:"teamid"`
}

func (q *Queries) InsertProject(ctx context.Context, arg InsertProjectParams) (int32, error) {
	row := q.db.QueryRow(ctx, insertProject, arg.Title, arg.Teamid)
	var projectid int32
	err := row.Scan(&projectid)
	return projectid, err
}

const insertTeam = `-- name: InsertTeam :one
INSERT INTO team(name)
VALUES ($1)
RETURNING teamid
`

func (q *Queries) InsertTeam(ctx context.Context, name string) (int32, error) {
	row := q.db.QueryRow(ctx, insertTeam, name)
	var teamid int32
	err := row.Scan(&teamid)
	return teamid, err
}

const insertTwoFileRevisions = `-- name: InsertTwoFileRevisions :exec
INSERT INTO filerevision(projectid, path, commitid, filehash, numchunks, changetype)
VALUES ($1, $2, $3, $4, $5, $6), ($7, $8, $9, $10, $11, $12)
`

type InsertTwoFileRevisionsParams struct {
	Projectid    int32  `json:"projectid"`
	Path         string `json:"path"`
	Commitid     int32  `json:"commitid"`
	Filehash     string `json:"filehash"`
	Numchunks    int32  `json:"numchunks"`
	Changetype   int32  `json:"changetype"`
	Projectid_2  int32  `json:"projectid_2"`
	Path_2       string `json:"path_2"`
	Commitid_2   int32  `json:"commitid_2"`
	Filehash_2   string `json:"filehash_2"`
	Numchunks_2  int32  `json:"numchunks_2"`
	Changetype_2 int32  `json:"changetype_2"`
}

func (q *Queries) InsertTwoFileRevisions(ctx context.Context, arg InsertTwoFileRevisionsParams) error {
	_, err := q.db.Exec(ctx, insertTwoFileRevisions,
		arg.Projectid,
		arg.Path,
		arg.Commitid,
		arg.Filehash,
		arg.Numchunks,
		arg.Changetype,
		arg.Projectid_2,
		arg.Path_2,
		arg.Commitid_2,
		arg.Filehash_2,
		arg.Numchunks_2,
		arg.Changetype_2,
	)
	return err
}

const listProjectCommits = `-- name: ListProjectCommits :many
SELECT cno, numfiles, userid, comment, commitid, timestamp FROM commit
WHERE projectid = $1
ORDER BY commitid DESC
LIMIT 5 OFFSET $2
`

type ListProjectCommitsParams struct {
	Projectid int32 `json:"projectid"`
	Offset    int32 `json:"offset"`
}

type ListProjectCommitsRow struct {
	Cno       pgtype.Int4      `json:"cno"`
	Numfiles  int32            `json:"numfiles"`
	Userid    string           `json:"userid"`
	Comment   string           `json:"comment"`
	Commitid  int32            `json:"commitid"`
	Timestamp pgtype.Timestamp `json:"timestamp"`
}

func (q *Queries) ListProjectCommits(ctx context.Context, arg ListProjectCommitsParams) ([]ListProjectCommitsRow, error) {
	rows, err := q.db.Query(ctx, listProjectCommits, arg.Projectid, arg.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ListProjectCommitsRow
	for rows.Next() {
		var i ListProjectCommitsRow
		if err := rows.Scan(
			&i.Cno,
			&i.Numfiles,
			&i.Userid,
			&i.Comment,
			&i.Commitid,
			&i.Timestamp,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const restoreProjectToCommit = `-- name: RestoreProjectToCommit :exec
SELECT commitid, projectid, userid, comment, numfiles, cno, timestamp FROM commit WHERE numfiles = $3 and projectid = $2 and commitid = $1
`

type RestoreProjectToCommitParams struct {
	Commitid  int32 `json:"commitid"`
	Projectid int32 `json:"projectid"`
	NewCommit int32 `json:"new_commit"`
}

// TODO
func (q *Queries) RestoreProjectToCommit(ctx context.Context, arg RestoreProjectToCommitParams) error {
	_, err := q.db.Exec(ctx, restoreProjectToCommit, arg.Commitid, arg.Projectid, arg.NewCommit)
	return err
}

const setTeamPermission = `-- name: SetTeamPermission :one
INSERT INTO teampermission(userid, teamid, level)
VALUES($1, $2, $3) ON CONFLICT(userid, teamid) DO UPDATE SET level=excluded.level
RETURNING userid, teamid, level
`

type SetTeamPermissionParams struct {
	Userid string `json:"userid"`
	Teamid int32  `json:"teamid"`
	Level  int32  `json:"level"`
}

func (q *Queries) SetTeamPermission(ctx context.Context, arg SetTeamPermissionParams) (Teampermission, error) {
	row := q.db.QueryRow(ctx, setTeamPermission, arg.Userid, arg.Teamid, arg.Level)
	var i Teampermission
	err := row.Scan(&i.Userid, &i.Teamid, &i.Level)
	return i, err
}
