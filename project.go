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
	Path      string   `json:"path"`
	Size      int      `json:"size"`
	NumChunks int      `json:"num_chunks"`
	Hashes    []string `json:"hashes"`
}

type CommitRequest struct {
	ProjectId         int    `json:"projectId"`
	TeamId            int    `json:"teamId"`
	Message           string `json:"message"`
	TentativeCommitId int    `json:"tentativeCommitId"`
	Files             []File `json:"files"`
}

type ProjectCreationRequest struct {
	Name   string `json:"name"`
	TeamID int    `json:"teamId"`
}

func getProjectsForUser(w http.ResponseWriter, r *http.Request) {
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
			projects = append(projects, Project{Id: int(tp.Pid), Name: tp.Title, Team: tp.Name})
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

func createProject(w http.ResponseWriter, r *http.Request) {
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
		fmt.Fprintf(w, `{ "status": "db error" }`)
		return
	}
	level := int(permission)
	if level < 2 {
		fmt.Fprintf(w, `{ "status": "no permission" }`)
		return
	}

	// check for unique name
	count, err := query.CheckProjectName(ctx, sqlcgen.CheckProjectNameParams{Teamid: int64(request.TeamID), Title: request.Name})
	if err != nil {
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

func getProjectInfo(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	_, ok := clerk.SessionClaimsFromContext(r.Context())
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
	fmt.Fprintf(w, `
	{
		"title": "%s"
	}
	`, projectname)
}

// 0 (not found and not in team): no permission at all
// 1 (not found but in team): read only
// 2 (found): write access
// 3 (manager): manager, can add write access
// 4 (owner): can set managers
func getProjectPermissionByID(userId string, projectId int, teamId int) int {
	teamPermission := checkPermissionByID(teamId, userId)
	// not in team: < 1
	if teamPermission < 1 {
		return 0
	}

	db := createDB()
	defer db.Close()

	queryresult := db.QueryRow("SELECT level FROM projectpermission WHERE userid = ? AND projectid = ?", userId, projectId)
	var level int
	err := queryresult.Scan(&level)
	if err == sql.ErrNoRows {
		return 1 // read only
	} else if err != nil {
		return 0 // general error/no permission
	}

	return level
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
func commit(w http.ResponseWriter, r *http.Request) {
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
	}

	// check permission
	projectPermission := getProjectPermissionByID(userId, request.ProjectId, request.TeamId)
	if projectPermission < 2 {
		fmt.Fprintf(w, `{ "response": "no permission" }`)
		return
	}

	// TODO
	// iterate through hashes to see if we have it in S3 (can see thru block table)
	// if we need hashes, return nb
	// otherwise, commit
	var hashesMissing []string
	/*
		for _, file := range request.Files {

		}
	*/
	if len(hashesMissing) > 0 {
		// respond with nb
		return
	}

	// TODO
	// no hashes missing, so commit
	// make an entry in the commit, file, and filerevision tables
}

// given a project id, returns the newest commit id used
func getLatestCommit(w http.ResponseWriter, r *http.Request) {
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

	db := createDB()
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
