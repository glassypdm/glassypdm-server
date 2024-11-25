package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/charmbracelet/log"
	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/go-chi/chi/v5"
	"github.com/joshtenorio/glassypdm-server/sqlcgen"
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
	teams, err := queries.FindUserTeams(ctx, user)
	if err != nil {
		log.Error("couldn't retrieve user's teams", "user", user, "err", err.Error())
	}
	projects := []Project{}
	for _, team := range teams {
		TeamProjects, err := queries.FindTeamProjects(ctx, team.Teamid)
		if err != nil {
			log.Error("couldn't retrieve team's projects", "teamid", team.Teamid, "err", err.Error())
		}
		for _, tp := range TeamProjects {
			projects = append(projects, Project{Id: int(tp.Projectid), Name: tp.Title, Team: tp.Name, TeamId: int(team.Teamid)})
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
		WriteError(w, "incorrect format")
		return
	}

	// check permission level in team
	permission, err := queries.GetTeamPermission(ctx, sqlcgen.GetTeamPermissionParams{Teamid: int32(request.TeamID), Userid: claims.Subject})
	if err != nil {
		log.Error("couldn't get team permission", "team", request.TeamID, "user", claims.Subject)
		WriteError(w, "db error")
		return
	}
	level := int(permission)
	if level < 2 {
		log.Error("insufficient permission for creating project", "team", request.TeamID, "user", claims.Subject)
		WriteError(w, "insufficient permission")
		return
	}

	pid, err := queries.InsertProject(ctx, sqlcgen.InsertProjectParams{Teamid: int32(request.TeamID), Title: request.Name})
	if err != nil {
		log.Error("insufficient permission for creating project", "db error", err)
		WriteError(w, "db error")
		return
	}
	_, err = queries.InsertCommit(ctx, sqlcgen.InsertCommitParams{Projectid: pid, Userid: claims.Subject, Comment: "Initial commit", Numfiles: 0})
	if err != nil {
		log.Error("couldn't insert commit", "db error", err)
		WriteError(w, "db error")
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
		WriteError(w, "incorrect format")
		return
	}

	pid, err := strconv.Atoi(r.URL.Query().Get("pid"))
	if err != nil {
		WriteError(w, "incorrect format")
		return
	}

	projectname, err := queries.GetProjectInfo(ctx, int32(pid))
	if err != nil {
		fmt.Fprintf(w, `{ "response": "db error", "db": "%s" }`, err.Error())
		return
	}
	team, err := queries.GetTeamFromProject(ctx, int32(pid))
	if err != nil {
		log.Error("db error", "err", err.Error())
		fmt.Fprintf(w, `{ "response": "db error", "db": "%s" }`, err.Error())
		return
	}
	teamName, err := queries.GetTeamName(ctx, team)
	if err != nil {
		log.Error("db error", "err", err.Error())
		fmt.Fprintf(w, `{ "response": "db error", "db": "%s" }`, err.Error())
		return
	}
	cid, err := queries.FindProjectInitCommit(ctx, int32(pid))
	if err != nil {
		log.Error("db error", "err", err.Error())
		if err.Error() == "sql: no rows in result set" {
			cid = -1
		} else {
			fmt.Fprintf(w, `{ "response": "db error", "db": "%s" }`, err.Error())
			return
		}

	}

	permission, err := queries.GetTeamPermission(ctx, sqlcgen.GetTeamPermissionParams{Teamid: team, Userid: claims.Subject})
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
// 4 (owner): can set managers
func getProjectPermissionByID(userId string, projectId int) int {
	ctx := context.Background()

	teamId, err := queries.GetTeamByProject(ctx, int32(projectId))
	if err != nil {
		log.Warn("db error", "err", err.Error())
		return 0
	}

	teamPermission := checkPermissionByID(int(teamId), userId)
	// not in team: < 1
	if teamPermission < 1 {
		return 0
	} else if teamPermission >= 2 {
		return 3
	}

	membership, err := queries.IsUserInPermissionGroup(ctx, sqlcgen.IsUserInPermissionGroupParams{Userid: userId, Projectid: int32(projectId)})
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
		WriteError(w, "incorrect format")
	}

	if getProjectPermissionByID(claims.Subject, projectId) < 1 {
		log.Warn("insufficient permission", "user", claims.Subject, "projectId", projectId)
		WriteError(w, "insufficient permission")
		return
	}

	// get project state
	output, err := queries.GetProjectState(ctx, int32(projectId))
	if err != nil {
		log.Error("db error", "project", projectId, "err", err.Error())
		WriteError(w, "db error")
		return
	}
	if len(output) == 0 {
		log.Warn("project state output is empty")
	}

	OutputBytes, _ := json.Marshal(output)

	WriteSuccess(w, string(OutputBytes))
}
