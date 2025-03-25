package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/charmbracelet/log"
	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/user"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/joshtenorio/glassypdm-server/internal/dal"
	"github.com/joshtenorio/glassypdm-server/internal/sqlcgen"
)

type Project struct {
	Id     int    `json:"id"`
	Name   string `json:"name"`
	Team   string `json:"team"`
	TeamId int    `json:"team_id"`
}

type Team struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

/*
*
body:
- projectid, teamid
- proposed commit number
- commit msg
- files: [
{
filepath
size
number of chunks
list of hashes
}
]
*/
type File struct {
	Path       string `json:"path"`
	Hash       string `json:"hash"`
	ChangeType int    `json:"changetype"`
}

type CommitRequest struct {
	ProjectId int    `json:"projectId"`
	Message   string `json:"message"`
	Files     []File `json:"files"`
}

type ProjectCreationRequest struct {
	Name   string `json:"name"`
	TeamID int    `json:"teamId"`
}

func GetProjectsForUser(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"access": "unauthorized"}`))
		return
	}
	user := claims.Subject

	// get user's projects
	teams, err := dal.Queries.FindUserTeams(ctx, user)
	if err != nil {
		log.Error("couldn't retrieve user's teams", "user", user, "err", err.Error())
	}
	projects := []Project{}
	for _, team := range teams {
		TeamProjects, err := dal.Queries.FindTeamProjects(ctx, team.Teamid)
		if err != nil {
			log.Error("couldn't retrieve team's projects", "teamid", team.Teamid, "err", err.Error())
		}
		for _, tp := range TeamProjects {
			projects = append(projects, Project{Id: int(tp.Projectid), Name: tp.Title, Team: tp.Name, TeamId: int(team.Teamid)})
		}
	}

	// get user's managed teams
	managedTeams, _ := dal.Queries.FindUserManagedTeams(ctx, user)
	managed := []Team{}
	for _, team := range managedTeams {
		managed = append(managed, Team{Id: int(team.Teamid), Name: team.Name})
	}
	projectsJson, _ := json.Marshal(projects)
	managedJson, _ := json.Marshal(managed)
	fmt.Fprintf(w, `
	{
		"response": "success",
		"body": {
			"user_id": "%s",
			"projects": %s,
			"managed_teams": %s
		}
	}
	`, user, string(projectsJson), string(managedJson))
}

func CreateProject(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"access": "unauthorized"}`))
		return
	}

	var request ProjectCreationRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		WriteCustomError(w, "incorrect format")
		return
	}

	// check permission level in team
	permission, err := dal.Queries.GetTeamPermission(ctx, sqlcgen.GetTeamPermissionParams{Teamid: int32(request.TeamID), Userid: claims.Subject})
	if err != nil {
		log.Error("couldn't get team permission", "team", request.TeamID, "user", claims.Subject)
		WriteCustomError(w, "db error")
		return
	}
	level := int(permission)
	if level < 2 {
		log.Error("insufficient permission for creating project", "team", request.TeamID, "user", claims.Subject)
		WriteCustomError(w, "insufficient permission")
		return
	}

	pid, err := dal.Queries.InsertProject(ctx, sqlcgen.InsertProjectParams{Teamid: int32(request.TeamID), Title: request.Name})
	if err != nil {
		log.Error("insufficient permission for creating project", "db error", err)
		WriteCustomError(w, "db error")
		return
	}
	_, err = dal.Queries.InsertCommit(ctx, sqlcgen.InsertCommitParams{Projectid: pid, Userid: claims.Subject, Comment: "Initial commit", Numfiles: 0})
	if err != nil {
		log.Error("couldn't insert commit", "db error", err)
		WriteCustomError(w, "db error")
		return
	}
	log.Info("succesfully created project", "project ID", pid, "name", request.Name)
	WriteDefaultSuccess(w, "project created")
}

func GetProjectInfo(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"access": "unauthorized"}`))
		return
	}
	if r.URL.Query().Get("pid") == "" {
		WriteCustomError(w, "incorrect format")
		return
	}

	pid, err := strconv.Atoi(r.URL.Query().Get("pid"))
	if err != nil {
		WriteCustomError(w, "incorrect format")
		return
	}

	projectname, err := dal.Queries.GetProjectInfo(ctx, int32(pid))
	if err != nil {
		fmt.Fprintf(w, `{ "response": "db error", "db": "%s" }`, err.Error())
		return
	}
	team, err := dal.Queries.GetTeamFromProject(ctx, int32(pid))
	if err != nil {
		log.Error("db error", "err", err.Error())
		fmt.Fprintf(w, `{ "response": "db error", "db": "%s" }`, err.Error())
		return
	}
	teamName, err := dal.Queries.GetTeamName(ctx, team)
	if err != nil {
		log.Error("db error", "err", err.Error())
		fmt.Fprintf(w, `{ "response": "db error", "db": "%s" }`, err.Error())
		return
	}
	cid, err := dal.Queries.FindProjectInitCommit(ctx, int32(pid))
	if err != nil {
		log.Error("db error", "err", err.Error())
		if err.Error() == "sql: no rows in result set" {
			cid = -1
		} else {
			fmt.Fprintf(w, `{ "response": "db error", "db": "%s" }`, err.Error())
			return
		}

	}

	permission, err := dal.Queries.GetTeamPermission(ctx, sqlcgen.GetTeamPermissionParams{Teamid: team, Userid: claims.Subject})
	if err != nil {
		log.Error("db error", "err", err.Error())
		fmt.Fprintf(w, `{ "response": "db error", "db": "%s" }`, err.Error())
		return
	}
	var CanManage bool
	if permission > 1 {
		CanManage = true
	} else {
		CanManage = false
	}
	fmt.Fprintf(w, `
	{
		"response": "success",
		"body": {
			"title": "%s",
			"teamId": %v,
			"teamName": "%s",
			"initCommit": %v,
			"canManage": %v
		}
	}
	`, projectname, team, teamName, cid, CanManage)
}

// 0 (not found and not in team): no permission at all
// 1 (not found but in team): read only
// 2 (found): write access
// 3 (manager): manager, can add write access
func GetProjectPermissionByID(userId string, projectId int) int {
	ctx := context.Background()

	teamId, err := dal.Queries.GetTeamByProject(ctx, int32(projectId))
	if err != nil {
		log.Warn("db error", "err", err.Error())
		return 0
	}

	teamPermission := CheckPermissionByID(int(teamId), userId)
	// not in team: < 1
	if teamPermission < 1 {
		return 0
	} else if teamPermission >= 2 {
		return 3
	}

	membership, err := dal.Queries.IsUserInPermissionGroup(ctx, sqlcgen.IsUserInPermissionGroupParams{Userid: userId, Projectid: int32(projectId)})
	if err != nil {
		if err.Error() == "no rows in result set" {
			log.Debug("user not found in permission group")
			return 1 // read only
		}

		log.Error("error grabbing project permission for", "user", userId, "project", projectId)
		log.Debug("no permission")
		return 0 // general error/no permission
	}

	if membership == userId {
		log.Debug("write permission")
		return 2
	}

	log.Error("unhandled case when grabbing project permission", userId, projectId, err)
	// if we are here, something went wrong
	return 0
}

// TODO remove after 0.7.2 is released, lmao
func GetProjectState(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"access": "unauthorized"}`))
		return
	}

	// make sure we have permission to read the project
	projectIdStr := chi.URLParam(r, "project-id")
	projectId, err := strconv.Atoi(projectIdStr)
	if err != nil {
		WriteCustomError(w, "incorrect format")
	}

	// something useful comment
	UseLatest := true
	var CommitId int32
	commitstr := chi.URLParam(r, "commit-no")
	log.Debug("getting project state:", "c", commitstr)
	if commitstr != "latest" && commitstr != "" {
		UseLatest = false
		commitno, err := strconv.Atoi(commitstr)
		if err != nil {
			log.Error("commit number url param is malformed")
			WriteCustomError(w, "incorrect format")
			return
		}
		CommitId, err = dal.Queries.GetCommitIdFromNo(ctx, sqlcgen.GetCommitIdFromNoParams{Projectid: int32(projectId), Cno: pgtype.Int4{Valid: true, Int32: int32(commitno)}})
		if err != nil {
			log.Error("couldn't find commit number", "cno", commitno, "project", projectId)
			WriteCustomError(w, "what")
			return
		}
		log.Debug("found commit id for cno:", "cid", CommitId, "cno", commitno)
	}

	if GetProjectPermissionByID(claims.Subject, projectId) < 1 {
		log.Warn("insufficient permission", "user", claims.Subject, "projectId", projectId)
		WriteCustomError(w, "insufficient permission")
		return
	}

	// get latest project state
	// TODO refactor this mess lmao
	if UseLatest {
		output, err := dal.Queries.GetProjectState(ctx, int32(projectId))
		if err != nil {
			log.Error("db error", "project", projectId, "err", err.Error())
			WriteCustomError(w, "db error")
			return
		}
		if len(output) == 0 {
			log.Warn("project state output is empty")
		}

		OutputBytes, _ := json.Marshal(output)

		WriteSuccess(w, string(OutputBytes))
		return
	} else {
		output, err := dal.Queries.GetProjectStateAtCommit(ctx, sqlcgen.GetProjectStateAtCommitParams{Projectid: int32(projectId), Commitid: int32(CommitId)})
		if err != nil {
			log.Error("db error", "project", projectId, "err", err.Error())
			WriteCustomError(w, "db error")
			return
		}
		if len(output) == 0 {
			log.Warn("project state output is empty")
		}

		OutputBytes, _ := json.Marshal(output)

		WriteSuccess(w, string(OutputBytes))
		return
	}
}

type RestoreProjectRequest struct {
	CommitId  int `json:"commit_id"`  // commit id to restore project to
	ProjectId int `json:"project_id"` // project id
}

func RouteProjectRestore(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"access": "unauthorized"}`))
		return
	}
	var request RestoreProjectRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		WriteCustomError(w, "bad json")
		return
	}

	// verify that user is at least a team manager
	// TODO enum or use TeamRole instead of projectpermission
	userId := claims.Subject
	projectPermission := GetProjectPermissionByID(userId, request.ProjectId)
	if projectPermission < 3 {
		log.Warn("user does not have permission to restore project state", "levl", projectPermission)
		WriteCustomError(w, "no permission")
		return
	}

	// get number of files that would be updated
	FileCount, err := dal.Queries.CountFilesUpdatedSinceCommit(ctx, sqlcgen.CountFilesUpdatedSinceCommitParams{
		Commitid:  int32(request.CommitId),
		Projectid: int32(request.ProjectId),
	})
	if err != nil {
		log.Error("couldn't get number of files updated since", "error", err)
		WriteCustomError(w, "db error")
		return
	}

	// get commit number
	info, err := dal.Queries.GetCommitInfo(ctx, int32(request.CommitId))
	if err != nil {
		log.Error("couldn't get commit number for message", "error", err)
		WriteCustomError(w, "db error")
		return
	}
	// start transaction
	tx, err := dal.DbPool.Begin(ctx)
	if err != nil {
		log.Error("couldn't create transaction", "error", err)
		WriteCustomError(w, "db error")
		return
	}
	defer tx.Rollback(ctx)
	qtx := dal.Queries.WithTx(tx)

	// create commit
	NewCommitId, err := qtx.InsertCommit(ctx, sqlcgen.InsertCommitParams{
		Projectid: int32(request.ProjectId),
		Userid:    userId,
		Comment:   "Restoring project state to Project Update " + strconv.Itoa(int(info.Cno.Int32)),
		Numfiles:  int32(FileCount)})
	if err != nil {
		log.Error("db couldn't create commit", "db err", err)
		WriteCustomError(w, "db error")
		return
	}

	// insert new filerevisions
	err = qtx.RestoreProjectToCommit(ctx, sqlcgen.RestoreProjectToCommitParams{Projectid: int32(request.ProjectId), Commitid: int32(request.CommitId), NewCommit: NewCommitId})
	if err != nil {
		log.Debug("params", "projectid", int32(request.ProjectId), "commit", int32(request.CommitId), "newcommit", NewCommitId)
		log.Error("couldnt restore project due to database error", "db err", err)
		WriteCustomError(w, "db error")
		return
	}

	// commit transaction
	tx.Commit(ctx)

	WriteDefaultSuccess(w, "pogchamp")
}

func RouteGetProjectCommit(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"access": "unauthorized"}`))
		return
	}
	if r.URL.Query().Get("pid") == "" {
		WriteCustomError(w, "incorrect format")
		return
	}

	if r.URL.Query().Get("cno") == "" {
		WriteCustomError(w, "incorrect format")
		return
	}
	projectId, err := strconv.Atoi(r.URL.Query().Get("pid"))
	if err != nil {
		WriteCustomError(w, "incorrect format")
		return
	}
	cno, err := strconv.Atoi(r.URL.Query().Get("cno"))
	if err != nil {
		WriteCustomError(w, "incorrect format")
		return
	}

	// check permissions
	if GetProjectPermissionByID(claims.Subject, projectId) < 1 {
		log.Warn("insufficient permission", "user", claims.Subject, "projectId", projectId)
		WriteCustomError(w, "insufficient permission")
		return
	}
	CommitId, err := dal.Queries.GetCommitIdFromNo(
		ctx,
		sqlcgen.GetCommitIdFromNoParams{Projectid: int32(projectId), Cno: pgtype.Int4{Valid: true, Int32: int32(cno)}})
	if err != nil {
		WriteCustomError(w, "db error")
		return
	}

	// TODO refactor lmao bc this is copy pasta'd
	// get commit info for cno
	CommitInfoDto, err := dal.Queries.GetCommitInfo(ctx, int32(CommitId))
	if err != nil {
		WriteCustomError(w, "db error")
		log.Warn("encountered db error when getting commit info", "db", err, "commit-id", CommitId)
		return
	}

	// get file revisions
	Files, err := dal.Queries.GetFileRevisionsByCommitId(ctx, int32(CommitId))
	if err != nil {
		WriteCustomError(w, "db error")
		log.Warn("encountered db error when getting file revisions for commit", "db", err, "commit-id", CommitId)
		return
	}

	var Output CommitInformation
	Output.FilesChanged = Files

	usr, err := user.Get(ctx, CommitInfoDto.Userid)
	name := ""
	if err != nil {
		WriteCustomError(w, "invalid user id")
		return
	}
	name = *usr.FirstName + " " + *usr.LastName

	Output.Description = CommitDescription{
		CommitId:     int(CommitId),
		CommitNumber: int(CommitInfoDto.Cno.Int32),
		NumFiles:     int(CommitInfoDto.Numfiles),
		Comment:      CommitInfoDto.Comment,
		Timestamp:    CommitInfoDto.Timestamp.Time.UnixNano() / 1000000000,
		Author:       name,
	}

	OutputJson, err := json.Marshal(Output)
	if err != nil {
		WriteCustomError(w, "json error")
		return
	}
	WriteSuccess(w, string(OutputJson))
}

func GetProjectLatestCommit(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	_, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"access": "unauthorized"}`))
		return
	}
	if r.URL.Query().Get("pid") == "" {
		WriteCustomError(w, "incorrect format")
		return
	}
	pid, err := strconv.Atoi(r.URL.Query().Get("pid"))
	if err != nil {
		WriteCustomError(w, "incorrect format")
		return
	}
	hehez, err := dal.Queries.GetLatestCommit(ctx, int32(pid))
	if err != nil {
		WriteCustomError(w, "db error")
		return
	}
	WriteSuccess(w, strconv.Itoa(int(hehez)))
}
