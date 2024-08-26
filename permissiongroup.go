package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/go-chi/chi/v5"
)

func CreatePermissionGroup(w http.ResponseWriter, r *http.Request) {
	_, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"access": "unauthorized"}`))
		return
	}
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
