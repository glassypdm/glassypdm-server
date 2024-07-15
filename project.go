package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/joshtenorio/glassypdm-server/sqlcgen"
)

type Project struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
	Team string `json:"team"`
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

	queries := UseQueries()

	// get user's projects
	teams, err := queries.FindUserTeams(ctx, user)
	_ = err
	projects := []Project{}
	for _, team := range teams {
		TeamProjects, err := queries.FindUserProjects(ctx, team.Teamid)
		if err != nil {
			fmt.Println(err)
		}
		for _, tp := range TeamProjects {
			projects = append(projects, Project{Id: int(tp.Projectid), Name: tp.Title, Team: tp.Name})
		}
	}

	// get user's managed teams
	managedTeams, _ := queries.FindUserManagedTeams(ctx, user)
	managed := []Team{}
	for _, team := range managedTeams {
		managed = append(managed, Team{Id: int(team.Teamid), Name: team.Name})
	}
	projectsJson, _ := json.Marshal(projects)
	managedJson, _ := json.Marshal(managed)
	fmt.Fprintf(w, `
	{
		"user_id": "%s",
		"projects": %s,
		"managed_teams": %s
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

	query := UseQueries()

	var request ProjectCreationRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		fmt.Fprintf(w, `{ "status": "json error" }`)
		return
	}

	// check permission level in team
	fmt.Println(request)
	permission, err := query.GetTeamPermission(ctx, sqlcgen.GetTeamPermissionParams{Teamid: int64(request.TeamID), Userid: claims.Subject})
	if err != nil {
		fmt.Println("a ...any")
		fmt.Fprintf(w, `{ "status": "db error" }`)
		return
	}
	level := int(permission)
	if level < 2 {
		fmt.Fprintf(w, `{ "status": "no permission" }`)
		return
	}

	// check for unique name
	// FIXME might be unnecessary now
	count, err := query.CheckProjectName(ctx, sqlcgen.CheckProjectNameParams{Teamid: int64(request.TeamID), Title: request.Name})
	if err != nil {
		fmt.Println("project name")
		fmt.Fprintf(w, `{ "status": "db error" }`)
		return
	} else if count > 0 {
		fmt.Fprintf(w, `{ "status": "project name exists" }`)
		return
	}

	pid, err := query.InsertProject(ctx, sqlcgen.InsertProjectParams{Teamid: int64(request.TeamID), Title: request.Name})
	if err != nil {
		fmt.Fprintf(w, `{ "status": "db error" }`)
		return
	}
	_, err = query.InsertCommit(ctx, sqlcgen.InsertCommitParams{Projectid: pid, Userid: claims.Subject, Comment: "Initial commit", Numfiles: 0})
	if err != nil {
		fmt.Fprintf(w, `{ "status": "db error" }`)
		return
	}
	fmt.Fprintf(w, `{ "status": "success" }`)
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
		fmt.Fprintf(w, `{ "status": "no pid supplied" }`)
		return
	}

	pid, err := strconv.Atoi(r.URL.Query().Get("pid"))
	_ = err

	query := UseQueries()
	projectname, err := query.GetProjectInfo(ctx, int64(pid))
	if err != nil {
		fmt.Fprintf(w, `{ "status": "db error", "db": "%s" }`, err.Error())
		return
	}
	team, err := query.GetTeamFromProject(ctx, int64(pid))
	if err != nil {
		fmt.Fprintf(w, `{ "status": "db error", "db": "%s" }`, err.Error())
		return
	}
	teamName, err := query.GetTeamName(ctx, team)
	if err != nil {
		fmt.Fprintf(w, `{ "status": "db error", "db": "%s" }`, err.Error())
		return
	}
	cid, err := query.FindProjectInitCommit(ctx, int64(pid))
	if err != nil {
		fmt.Fprintf(w, `{ "status": "db error", "db": "%s" }`, err.Error())
		return
	}

	permission, err := query.GetTeamPermission(ctx, sqlcgen.GetTeamPermissionParams{Teamid: team, Userid: claims.Subject})
	if err != nil {
		fmt.Fprintf(w, `{ "status": "db error", "db": "%s" }`, err.Error())
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
		"title": "%s",
		"teamId": %v,
		"teamName": "%s",
		"initCommit": %v,
		"canManage": %v
	}
	`, projectname, team, teamName, cid, CanManage)
}

// 0 (not found and not in team): no permission at all
// 1 (not found but in team): read only
// 2 (found): write access
// 3 (manager): manager, can add write access
// 4 (owner): can set managers
func getProjectPermissionByID(userId string, projectId int) int {
	ctx := context.Background()

	queries := UseQueries()

	teamId, err := queries.GetTeamByProject(ctx, int64(projectId))
	if err != nil {
		fmt.Println(err)
		return 0
	}

	teamPermission := checkPermissionByID(int(teamId), userId)
	// not in team: < 1
	if teamPermission < 1 {
		return 0
	} else if teamPermission >= 2 {
		return 3
	}

	// TODO test
	level, err := queries.GetProjectPermission(ctx, sqlcgen.GetProjectPermissionParams{Userid: userId, Projectid: int64(projectId)})
	if err == sql.ErrNoRows {
		return 1 // read only
	} else if err != nil {
		return 0 // general error/no permission
	}

	return int(level)
}

/*
*
body:
- projectid, teamid
- CreateCommit msg
- files: [
{
filepath
hash
changetype
}
]
*/
func CreateCommit(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"access": "unauthorized"}`))
		return
	}
	userId := claims.Subject
	var request CommitRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		fmt.Fprintf(w, `{ "response": "bad json" }`)
		return
	}

	// check permission
	projectPermission := getProjectPermissionByID(userId, request.ProjectId)
	if projectPermission < 2 {
		fmt.Fprintf(w, `{ "response": "no permission" }`)
		return
	}

	tx, qtx := useTxQueries()
	defer tx.Rollback()
	// make commit, get new commitid
	cid, err := qtx.InsertCommit(ctx, sqlcgen.InsertCommitParams{
		Projectid: int64(request.ProjectId),
		Userid:    userId,
		Comment:   request.Message,
		Numfiles:  int64(len(request.Files))})

	// FIXME if we create a new file entry
	// we don't see it when we have a filerevision
	var hashesMissing []string
	for _, file := range request.Files {
		_ = file
		// insert into file
		err = qtx.InsertFile(ctx, sqlcgen.InsertFileParams{Projectid: int64(request.ProjectId), Path: file.Path})
		if err != nil {
			// TODO do we need to handle anything here?
			fmt.Println("uwuwuwu")

			fmt.Printf("err: %v\n", err)
		}

		// add filerevision
		// error if we fail a unique thing (hopefully)
		err = qtx.InsertFileRevision(ctx, sqlcgen.InsertFileRevisionParams{
			Projectid:  int64(request.ProjectId),
			Path:       file.Path,
			Commitid:   cid,
			Hash:       file.Hash,
			Changetype: int64(file.ChangeType)})
		if err != nil {
			// TODO confirm error
			fmt.Printf("error %v\n", err)
			hashesMissing = append(hashesMissing, file.Hash)
			continue
		}

	}
	if len(hashesMissing) > 0 {
		// respond with nb
		fmt.Fprintf(w, `
			{
			"status": "nb",
			"hashes": %v
			}`, hashesMissing)
		return
	}

	// no hashes missing, so commit the transaction
	// we should consider returning more info too
	tx.Commit()
	fmt.Fprintf(w, `{
	"status": "success",
	"commitid": %v
	}`, cid)
}

// given a project id, returns the newest commit id used
func GetLatestCommit(w http.ResponseWriter, r *http.Request) {
	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"access": "unauthorized"}`))
		return
	}
	userId := claims.Subject
	project := r.URL.Query().Get("projectId")
	pid, err := strconv.Atoi(project)
	if err != nil {
		fmt.Fprintf(w, `{ "response": "incorrect format" }`)
		return
	}

	db := CreateDB()
	defer db.Close()

	// check user permissions
	// needs at least read permission
	rows, err := db.Query("SELECT COUNT(*) FROM teampermission WHERE userid = ?", userId)
	if err != nil {
		fmt.Fprintf(w, `{ "response": "database issue" }`)
		return
	}
	var count int
	for rows.Next() {
		if err := rows.Scan(&count); err != nil {
			fmt.Fprintf(w, `{ "response": "database issue" }`)
			return
		}
	}
	if count < 1 {
		fmt.Fprintf(w, `{ "response": "invalid permission" }`)
	}

	// get latest commit for pid
	rows, err = db.Query("SELECT MAX(cid) FROM 'commit' WHERE projectid = ?", pid)
	if err != nil {
		fmt.Fprintf(w, `{ "response": "database issue" }`)
		return
	}
	var commit int
	for rows.Next() {
		rows.Scan(&commit)
	}
	fmt.Fprintf(w, `
	{
		"response": "valid",
		"newestCommit": %d
	}`, commit)
}
