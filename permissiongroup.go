package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/go-chi/chi/v5"
	"github.com/joshtenorio/glassypdm-server/sqlcgen"
)

type PGCreationRequest struct {
	TeamID     int    `json:"team_id"`
	PGroupName string `json:"pgroup_name"`
}

func CreatePermissionGroup(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"access": "unauthorized"}`))
		return
	}

	var request PGCreationRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		PrintError(w, "bad json")
		return
	}

	// check if user has permission to create pgroup for team
	level := checkPermissionByID(request.TeamID, string(claims.Subject))
	if level < 2 {
		PrintError(w, "insufficient permission")
		return
	}

	// attempt to create permission group
	queries := UseQueries()
	err = queries.CreatePermissionGroup(ctx,
		sqlcgen.CreatePermissionGroupParams{Teamid: int64(request.TeamID), Name: request.PGroupName})
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			PrintError(w, "permission group exists")
		} else {
			PrintError(w, "db error")
		}
		return
	}

	output := DefaultSuccessOutput{Message: "permission group created"}
	output_bytes, err := json.Marshal(output)
	if err != nil {
		PrintError(w, "json error but creation successful")
		return
	}
	PrintSuccess(w, string(output_bytes))
}

type PGMappingRequest struct {
	ProjectID int `json:"project_id"`
	PGroupID  int `json:"pgroup_id"`
}

func CreatePGMapping(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"access": "unauthorized"}`))
		return
	}

	var request PGMappingRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		PrintError(w, "bad json")
		return
	}

	queries := UseQueries()
	team, err := queries.GetTeamFromProject(ctx, int64(request.ProjectID))
	if err != nil {
		if err == sql.ErrNoRows {
			log.Error("db: team not found", "project", request.ProjectID)
			PrintError(w, "team not found")
		}
		PrintError(w, "db error")
		return
	}
	// check that user is a manager or owner
	// TODO double check numbers
	if checkPermissionByID(int(team), claims.Subject) < 2 {
		PrintError(w, "insufficient permission")
		return
	}

	// create mapping
	err = queries.MapProjectToPermissionGroup(ctx,
		sqlcgen.MapProjectToPermissionGroupParams{Projectid: int64(request.ProjectID), Pgroupid: int64(request.PGroupID)})
	if err != nil {
		// TODO if foreign key constraint, return different error
		PrintError(w, "db error")
		return
	}

	var output DefaultSuccessOutput
	output.Message = "mapping successful"
	output_bytes, _ := json.Marshal(output)
	PrintSuccess(w, string(output_bytes))
}

func GetPermissionGroups(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	_, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"access": "unauthorized"}`))
		return
	}
	teamIdStr := chi.URLParam(r, "team-id")
	teamId, err := strconv.Atoi(teamIdStr)
	if err != nil {
		PrintError(w, "incorrect format")
		return
	}

	queries := UseQueries()
	groups, err := queries.ListPermissionGroupForTeam(ctx, int64(teamId))
	if err != nil {
		PrintError(w, "db error")
		return
	}
	log.Debug("permission groups:", "groups", groups)
	groups_json, err := json.Marshal(groups)
	if err != nil {
		log.Error("couldn't convert json", "groups", groups)
		PrintError(w, "db error: couldn't convert to json")
		return
	}
	PrintSuccess(w, string(groups_json))
}

type AddUserToPGroupRequest struct {
	Member   string `json:"member"`
	PGroupID int    `json:"pgroup_id"`
}

func AddUserToPG(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"access": "unauthorized"}`))
		return
	}

	var request AddUserToPGroupRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		PrintError(w, "bad json")
		return
	}

	queries := UseQueries()

	// check if user has permission to manage permission groups
	// i.e. is a manager
	team, err := queries.GetTeamFromPGroup(ctx, int64(request.PGroupID))
	if err != nil {
		// TODO if project doesnt exist return a different error
		PrintError(w, "db error")
		return
	}
	level := checkPermissionByID(int(team), claims.Subject)
	if level < 2 {
		PrintError(w, "insufficient permission")
		return
	}

	// check if member is in team
	_, err = queries.GetTeamPermission(ctx, sqlcgen.GetTeamPermissionParams{Teamid: team, Userid: request.Member})
	if err != nil {
		if err == sql.ErrNoRows {
			PrintError(w, "user not found in team")
		} else {
			PrintError(w, "db error")
		}
		return
	}

	// at this point member is in team, so add them to the permission group
	// TODO change query to remove them if they are in the pgroup already
	err = queries.AddMemberToPermissionGroup(ctx,
		sqlcgen.AddMemberToPermissionGroupParams{Userid: request.Member, Pgroupid: int64(request.PGroupID)})
	if err != nil {
		PrintError(w, "db error")
		return
	}
	var output DefaultSuccessOutput
	output.Message = "user successfully added"
	output_bytes, _ := json.Marshal(output)

	PrintSuccess(w, string(output_bytes))
}

func GetPermissionGroupInfo(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"access": "unauthorized"}`))
		return
	}
	user := claims.Subject
	hehe := r.URL.Query().Get("pgroup_id")
	pgroup, err := strconv.Atoi(hehe)
	if err != nil {
		log.Error("incorrect query", "param", hehe)
		PrintError(w, "incorrect format")
		return
	}

	queries := UseQueries()

	// make sure user has permission to get information about the permission group
	team, err := queries.GetTeamFromPGroup(ctx, int64(pgroup))
	if err != nil {
		log.Error("error fetching team from permission group", "err", err.Error())
		PrintError(w, "db error")
		return
	}
	level := checkPermissionByID(int(team), user)
	if level <= 0 {
		log.Debug("user's permission was insufficient", "user", user, "level", level)
		PrintError(w, "insufficient permission")
		return
	}

	// fetch projects for team
	TeamProjects, err := queries.FindTeamProjects(ctx, team)
	if err != nil {
		PrintError(w, "db error")
		return
	}

	// fetch projects for permission group
	pgProjects, err := queries.GetPermissionGroupMapping(ctx, int64(pgroup))
	if err != nil {
		PrintError(w, "db error")
		return
	}

	// fetch membership for permission group
	pgMembership, err := queries.ListPermissionGroupMembership(ctx, int64(pgroup))
	if err != nil {
		PrintError(w, "db error")
		return
	}

	// fetch membership for team
	TeamMembership, err := queries.GetTeamMembership(ctx, team)
	if err != nil {
		PrintError(w, "db error")
		return
	}

	var output PermissionGroupInfo
	// initialize arrays so that they don't return as null
	output.TeamMembership = make([]User, 0)
	output.PGroupMembership = make([]User, 0)
	output.TeamProjects = make([]Project, 0)
	output.PGroupProjects = make([]Project, 0)
	for _, project := range TeamProjects {
		output.TeamProjects = append(output.TeamProjects,
			Project{Id: int(project.Projectid), Name: project.Title, Team: project.Name})
	}

	for _, project := range pgProjects {
		output.PGroupProjects = append(output.PGroupProjects,
			Project{Id: int(project.Projectid), Name: project.Title, Team: ""})
	}

	for _, user := range TeamMembership {
		usr, err := GetUserByID(user.Userid)
		if !err {
			log.Warn("couldn't find user", usr)
			continue
		}
		output.TeamMembership = append(output.TeamMembership, usr)
	}

	for _, user := range pgMembership {
		usr, err := GetUserByID(user)
		if !err {
			log.Warn("couldn't find user", usr)
			continue
		}
		output.PGroupMembership = append(output.PGroupMembership, usr)
	}

	output_bytes, _ := json.Marshal(output)
	PrintSuccess(w, string(output_bytes))
}

type PermissionGroupInfo struct {
	TeamProjects     []Project `json:"team_projects"`
	PGroupProjects   []Project `json:"pg_projects"`
	TeamMembership   []User    `json:"team_membership"`
	PGroupMembership []User    `json:"pg_membership"`
}
