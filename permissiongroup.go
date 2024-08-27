package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

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
		// TODO if project doesnt exist return a different error
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

	PrintSuccess(w, "mapping successful")
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
	groups_json, err := json.Marshal(groups)
	if err != nil {
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
	err = queries.AddMemberToPermissionGroup(ctx,
		sqlcgen.AddMemberToPermissionGroupParams{Userid: request.Member, Pgroupid: int64(request.PGroupID)})
	if err != nil {
		PrintError(w, "db error")
		return
	}

	PrintSuccess(w, "user successfully added")
}
