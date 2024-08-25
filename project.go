package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/go-chi/chi/v5"
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
		fmt.Println(err.Error())
		fmt.Fprintf(w, `{ "status": "dgb error", "db": "%s" }`, err.Error())
		return
	}
	teamName, err := query.GetTeamName(ctx, team)
	if err != nil {
		fmt.Println(err.Error())
		fmt.Fprintf(w, `{ "status": "db error", "db": "%s" }`, err.Error())
		return
	}
	cid, err := query.FindProjectInitCommit(ctx, int64(pid))
	if err != nil {
		fmt.Println(err.Error())
		if err.Error() == "sql: no rows in result set" {
			cid = -1
		} else {
			fmt.Fprintf(w, `{ "status": "db error", "db": "%s" }`, err.Error())
			return
		}

	}

	permission, err := query.GetTeamPermission(ctx, sqlcgen.GetTeamPermissionParams{Teamid: team, Userid: claims.Subject})
	if err != nil {
		fmt.Println("gtp")
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
		fmt.Fprintf(w, `{ "status": "incorrect format" }`)
	}

	if getProjectPermissionByID(claims.Subject, projectId) < 1 {
		fmt.Fprintf(w, `{
		"status": "no permission"
		}`)
		return
	}

	// get project state
	query := UseQueries()

	output, err := query.GetProjectState(ctx, int64(projectId))
	if err != nil {
		fmt.Fprintf(w, `{ "status": "db error" }`)
		return
	}
	if len(output) == 0 {
		fmt.Println("empty")
	}

	outputjson, err := json.Marshal(output)
	if err != nil {
		fmt.Fprintf(w, `{ "status": "db error" }`)
		return
	}

	fmt.Fprintf(w, `{
		"status": "success",
		"project": %v
	}`, string(outputjson))
}
