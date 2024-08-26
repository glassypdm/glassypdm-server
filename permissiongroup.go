package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/go-chi/chi/v5"
	"github.com/joshtenorio/glassypdm-server/sqlcgen"
)

func CreatePermissionGroup(w http.ResponseWriter, r *http.Request) {
	_, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"access": "unauthorized"}`))
		return
	}
}

type PGMappingRequest struct {
	ProjectID int `json:"project_id"`
	PGroupID  int `json:"pgroup_id"`
}

func CreatePGMapping(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	user, ok := clerk.SessionClaimsFromContext(r.Context())
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
	if checkPermissionByID(int(team), user.ID) < 2 {
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
